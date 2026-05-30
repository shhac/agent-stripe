package shared

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"time"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

type GlobalFlags struct {
	Profile    string
	Context    string
	APIKey     string
	BaseURL    string
	Format     string
	Timeout    int
	Debug      bool
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
		w := output.NewNDJSONWriter(os.Stdout)
		for _, item := range items {
			_ = w.WriteItem(item)
		}
		if pagination != nil {
			_ = w.WritePagination(pagination)
		}
		return
	}
	result := map[string]any{"data": items}
	if pagination != nil {
		result["pagination"] = pagination
	}
	output.Print(result, f, true)
}

func WriteItem(data any, format string) {
	f := output.ResolveFormat(format, output.FormatJSON)
	output.Print(data, f, true)
}

func WriteRawItem(raw json.RawMessage, format string) {
	f := output.ResolveFormat(format, output.FormatJSON)
	output.WriteRawJSON(raw, f, true)
}

func WriteRawList(raw json.RawMessage, format string) error {
	var wrapper struct {
		HasMore  bool              `json:"has_more"`
		Data     []json.RawMessage `json:"data"`
		NextPage string            `json:"next_page"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	items := make([]any, 0, len(wrapper.Data))
	for _, item := range wrapper.Data {
		var decoded any
		if err := json.Unmarshal(item, &decoded); err != nil {
			return agenterrors.Wrap(err, agenterrors.FixableByAgent)
		}
		items = append(items, decoded)
	}
	var pagination *output.Pagination
	if wrapper.HasMore || wrapper.NextPage != "" {
		pagination = &output.Pagination{
			HasMore:  wrapper.HasMore,
			NextPage: wrapper.NextPage,
		}
	}
	WritePaginatedList(items, pagination, format)
	return nil
}

func RequireFlag(flag, value, hint string) bool {
	if value != "" {
		return true
	}
	err := agenterrors.Newf(agenterrors.FixableByAgent, "--%s is required", flag)
	if hint != "" {
		err = err.WithHint(hint)
	}
	output.WriteError(os.Stderr, err)
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
