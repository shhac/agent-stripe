package shared

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/shhac/agent-stripe/internal/api"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
	libcli "github.com/shhac/lib-agent-cli/cli"
)

// GetEntities runs the multi-capable get contract for the stripe domain: it
// sets up one client, then resolves each id through getOne and streams the
// result per the shared get contract (NDJSON by default — one record or
// {"@unresolved":…} per id in input order; item-level misses stay on stdout,
// command-level failures bubble to the single sink). getOne returns the decoded
// record for an id, or a classified *agenterrors.APIError (fixable_by:agent)
// so a wrong-prefix / 404 becomes an @unresolved record rather than aborting
// the batch.
func GetEntities(flags *GlobalFlags, args []string, getOne func(ctx context.Context, client *api.Client, id string) (any, error)) error {
	return WithClient(flags, func(ctx context.Context, client *api.Client) error {
		return libcli.EntityGet(output.Stdout(), flags.Format, args, func(id string) (any, error) {
			return getOne(ctx, client, id)
		})
	})
}

type GlobalFlags struct {
	libcli.Globals // Format, TimeoutMS, Debug

	Profile      string
	Context      string
	APIKey       string
	BaseURL      string
	MaxRetries   int
	APIVersion   string
	V2APIVersion string
}

type GlobalsFunc = func() *GlobalFlags

func ToAnySlice[T any](s []T) []any {
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}

func WritePaginatedList(items []any, pagination *output.Pagination, format string) {
	f := output.ResolveFormat(format, output.FormatNDJSON)
	if f == output.FormatNDJSON {
		WriteNDJSONItems(items, pagination)
		return
	}
	result := map[string]any{"data": items}
	if pagination != nil {
		result["pagination"] = pagination
	}
	output.Print(result, f, true)
}

func WriteNDJSONItems(items []any, pagination *output.Pagination) {
	w := output.NewNDJSONWriter(output.Stdout())
	for _, item := range items {
		_ = w.WriteItem(item)
	}
	if pagination != nil {
		_ = w.WritePagination(pagination)
	}
}

func WriteItem(data any, format string) {
	f := output.ResolveFormat(format, output.FormatJSON)
	output.Print(data, f, true)
}

func RedactionOptions(flags *GlobalFlags) output.RedactionOptions {
	if flags == nil {
		return output.RedactionOptions{}
	}
	return output.RedactionOptions{Expose: flags.Expose}
}

func WriteRawItem(raw json.RawMessage, format string, redaction output.RedactionOptions) {
	f := output.ResolveFormat(format, output.FormatJSON)
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		output.WriteRawJSON(raw, f, true)
		return
	}
	output.Print(output.Redact(data, redaction), f, true)
}

// DecodeListPage turns either namespace's list envelope into the same pair:
// the raw items and the pagination record for them. The namespaces differ in
// exactly this one place — v1 has has_more plus cursor IDs, v2 has a next-page
// URL carrying a token — so everything above this decodes once and stops caring
// which namespace it is reading.
func DecodeListPage(path string, raw json.RawMessage) ([]json.RawMessage, *output.Pagination, error) {
	if api.IsV2Path(path) {
		list, err := api.DecodeV2List(raw)
		if err != nil {
			return nil, nil, err
		}
		return list.Data, v2Pagination(list), nil
	}
	list, err := api.DecodeList(raw)
	if err != nil {
		return nil, nil, err
	}
	return list.Data, v1Pagination(list), nil
}

func v1Pagination(list *api.ListResponse) *output.Pagination {
	if !list.HasMore && list.NextPage == "" {
		return nil
	}
	return &output.Pagination{HasMore: list.HasMore, NextPage: list.NextPage}
}

// v2Pagination reports the page token lifted out of next_page_url, which is the
// value --page takes.
func v2Pagination(list *api.V2ListResponse) *output.Pagination {
	if !list.HasMore() {
		return nil
	}
	return &output.Pagination{HasMore: true, NextPage: list.NextPageToken()}
}

// WriteRawList renders a list page of either namespace as redacted raw objects.
func WriteRawList(path string, raw json.RawMessage, format string, redaction output.RedactionOptions) error {
	data, pagination, err := DecodeListPage(path, raw)
	if err != nil {
		return err
	}
	items := make([]any, 0, len(data))
	for _, item := range data {
		decoded, err := redactedRawListItem(item, redaction)
		if err != nil {
			return err
		}
		items = append(items, decoded)
	}
	WritePaginatedList(items, pagination, format)
	return nil
}

func redactedRawListItem(item json.RawMessage, redaction output.RedactionOptions) (any, error) {
	var decoded any
	if err := json.Unmarshal(item, &decoded); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return output.Redact(decoded, redaction), nil
}

// RequireFlag returns nil when value is present, or a structured
// fixable_by:agent error when it is empty. Callers bubble the error so the
// single sink in libcli.Run renders it once and exits non-zero.
func RequireFlag(flag, value, hint string) error {
	if value != "" {
		return nil
	}
	err := agenterrors.Newf(agenterrors.FixableByAgent, "--%s is required", flag)
	if hint != "" {
		err = err.WithHint(hint)
	}
	return err
}

func ContextWithTimeout(parent context.Context, ms int) (context.Context, context.CancelFunc) {
	if ms <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, time.Duration(ms)*time.Millisecond)
}

func AddString(values url.Values, key, value string) {
	if value != "" {
		values.Set(key, value)
	}
}

func AddMulti(values url.Values, key string, items []string) {
	for _, item := range items {
		if item != "" {
			values.Add(key, item)
		}
	}
}
