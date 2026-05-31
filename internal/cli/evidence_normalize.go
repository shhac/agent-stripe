package cli

import (
	"fmt"
	"strings"

	"github.com/shhac/agent-stripe/internal/output"
)

func normalizeEvidence(records []evidenceRecord, opts evidenceOptions) []evidenceRecord {
	if opts.maxString <= 0 {
		opts.maxString = defaultMaxString
	}
	seen := map[string]bool{}
	out := make([]evidenceRecord, 0, len(records))
	for _, record := range records {
		normalized, extracted := normalizeRecord(record, opts, seen)
		out = append(out, normalized)
		out = append(out, extracted...)
	}
	return out
}

type evidenceOptions struct {
	full         bool
	expandFields []string
	maxString    int
	redaction    output.RedactionOptions
}

func defaultEvidenceOptions() evidenceOptions {
	return evidenceOptions{maxString: defaultMaxString}
}

func normalizeRecord(record evidenceRecord, opts evidenceOptions, seen map[string]bool) (evidenceRecord, []evidenceRecord) {
	if record.Type != "entity" || record.Data == nil {
		if record.Data != nil {
			if redacted, ok := output.Redact(record.Data, opts.redaction).(map[string]any); ok {
				record.Data = redacted
			}
		}
		return record, nil
	}
	result := normalizeValue(record.Data, "", opts, seen)
	if normalized, ok := result.value.(map[string]any); ok {
		if redacted, ok := output.Redact(normalized, opts.redaction).(map[string]any); ok {
			record.Data = redacted
		} else {
			record.Data = normalized
		}
	}
	record.ExtractedEntities = append(record.ExtractedEntities, result.notes...)
	record.TruncatedFields = append(record.TruncatedFields, result.truncated...)
	return record, result.records
}

type normalizeResult struct {
	value     any
	records   []evidenceRecord
	notes     []fieldNote
	truncated []truncatedNote
}

func normalizeValue(value any, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	switch v := value.(type) {
	case map[string]any:
		if isStripeList(v) {
			return normalizeList(v, path, opts, seen)
		}
		if isStripeEntity(v) && path != "" {
			return normalizeNestedEntity(v, path, opts, seen)
		}
		return normalizeMap(v, path, opts, seen)
	case []any:
		return normalizeSlice(v, path, opts, seen)
	case string:
		if shouldTruncate(path, v, opts) {
			shown := opts.maxString
			if shown > len(v) {
				shown = len(v)
			}
			return normalizeResult{value: v[:shown] + "...", truncated: []truncatedNote{{
				Path:          path,
				OriginalBytes: len(v),
				ShownBytes:    shown,
				ExpandHint:    "--expand-field " + path + " or --full",
			}}}
		}
		return normalizeResult{value: v}
	default:
		return normalizeResult{value: value}
	}
}

func normalizeNestedEntity(item map[string]any, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	object := mapString(item, "object")
	id := mapString(item, "id")
	result := normalizeResult{
		value: id,
		notes: []fieldNote{{Path: path, Object: object, ID: id}},
	}
	key := object + ":" + id
	if seen[key] {
		return result
	}
	seen[key] = true
	child, children := normalizeRecord(entityRecord(object, item), opts, seen)
	result.records = append([]evidenceRecord{child}, children...)
	return result
}

func normalizeMap(item map[string]any, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	result := normalizeResult{value: make(map[string]any, len(item))}
	out := result.value.(map[string]any)
	for key, value := range item {
		child := normalizeValue(value, joinPath(path, key), opts, seen)
		out[key] = child.value
		result.merge(child)
	}
	return result
}

func normalizeSlice(items []any, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	result := normalizeResult{value: make([]any, len(items))}
	out := result.value.([]any)
	for idx, item := range items {
		child := normalizeValue(item, fmt.Sprintf("%s[%d]", path, idx), opts, seen)
		out[idx] = child.value
		result.merge(child)
	}
	return result
}

func normalizeList(list map[string]any, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	out := make(map[string]any, len(list))
	for key, value := range list {
		if key != "data" {
			out[key] = value
		}
	}
	items, _ := list["data"].([]any)
	normalizedItems := make([]any, 0, len(items))
	result := normalizeResult{value: out}
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s.data[%d]", path, idx)
		child := normalizeValue(item, itemPath, opts, seen)
		normalizedItems = append(normalizedItems, child.value)
		result.merge(child)
	}
	out["data"] = normalizedItems
	return result
}

func (r *normalizeResult) merge(child normalizeResult) {
	r.records = append(r.records, child.records...)
	r.notes = append(r.notes, child.notes...)
	r.truncated = append(r.truncated, child.truncated...)
}

func isStripeEntity(item map[string]any) bool {
	id := mapString(item, "id")
	object := mapString(item, "object")
	return id != "" && object != "" && object != "list" && object != "search_result"
}

func isStripeList(item map[string]any) bool {
	object := mapString(item, "object")
	return object == "list" || object == "search_result"
}

func shouldTruncate(path, value string, opts evidenceOptions) bool {
	if opts.full || len(value) <= opts.maxString {
		return false
	}
	if path == "id" || path == "object" || strings.HasSuffix(path, ".id") || strings.HasSuffix(path, ".object") {
		return false
	}
	for _, expanded := range opts.expandFields {
		if expanded == path || strings.HasPrefix(path, expanded+".") {
			return false
		}
	}
	return true
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
