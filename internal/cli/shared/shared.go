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

type GlobalFlags struct {
	libcli.Globals // Format, TimeoutMS, Debug

	Profile    string
	Context    string
	APIKey     string
	BaseURL    string
	Expose     []string
	MaxRetries int
	APIVersion string
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

func WriteRawList(raw json.RawMessage, format string, redaction output.RedactionOptions) error {
	list, err := api.DecodeList(raw)
	if err != nil {
		return err
	}
	var pagination *output.Pagination
	if list.HasMore || list.NextPage != "" {
		pagination = &output.Pagination{
			HasMore:  list.HasMore,
			NextPage: list.NextPage,
		}
	}
	if output.ResolveFormat(format, output.FormatNDJSON) == output.FormatNDJSON {
		w := output.NewNDJSONWriter(output.Stdout())
		for _, item := range list.Data {
			decoded, err := redactedRawListItem(item, redaction)
			if err != nil {
				return err
			}
			_ = w.WriteItem(decoded)
		}
		if pagination != nil {
			_ = w.WritePagination(pagination)
		}
		return nil
	}
	items := make([]any, 0, len(list.Data))
	for _, item := range list.Data {
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

func RequireFlag(flag, value, hint string) bool {
	if value != "" {
		return true
	}
	err := agenterrors.Newf(agenterrors.FixableByAgent, "--%s is required", flag)
	if hint != "" {
		err = err.WithHint(hint)
	}
	output.WriteError(output.Stderr(), err)
	return false
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
