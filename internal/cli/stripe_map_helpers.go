package cli

import "github.com/shhac/lib-agent-cli/creds"

func idFromValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		return mapString(v, "id")
	default:
		return ""
	}
}

func mapAnyMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	value, _ := m[key].(map[string]any)
	return value
}

func mapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return value
}

func mapBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	value, _ := m[key].(bool)
	return value
}

func mapValue(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func mapInt64(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	switch value := m[key].(type) {
	case float64:
		return int64(value), true
	case int64:
		return value, true
	case int:
		return int64(value), true
	default:
		return 0, false
	}
}

// firstNonEmpty is the canonical helper from lib-agent-cli, aliased so the
// investigation code reads without a package qualifier.
var firstNonEmpty = creds.FirstNonEmpty

// Summary writers. These copy a field into a compact summary only when it is
// present and of the expected type, so a summary never carries a null or an
// unexpected shape.
func copyString(out, in map[string]any, key string) {
	value, ok := in[key].(string)
	if ok && value != "" {
		out[key] = value
	}
}

func copyBool(out, in map[string]any, key string) {
	value, ok := in[key].(bool)
	if ok {
		out[key] = value
	}
}

func copyNumber(out, in map[string]any, key string) {
	if value, ok := in[key].(float64); ok {
		out[key] = value
	}
}

func addArrayCount(out, in map[string]any, key string) {
	items, ok := in[key].([]any)
	if ok && len(items) > 0 {
		out[key+"_count"] = len(items)
	}
}

// copyStringAs copies under a different key, for the cases where the compact
// name differs from Stripe's.
func copyStringAs(out, in map[string]any, srcKey, dstKey string) {
	if value, ok := in[srcKey].(string); ok && value != "" {
		out[dstKey] = value
	}
}

// copySubset projects selected string fields of a nested object into a compact
// sub-map, attaching it only when something was found.
func copySubset(summary, in map[string]any, key string, fields ...string) {
	nested, ok := in[key].(map[string]any)
	if !ok {
		return
	}
	out := map[string]any{}
	for _, field := range fields {
		copyString(out, nested, field)
	}
	if len(out) > 0 {
		summary[key] = out
	}
}

func copyExpandableID(out, in map[string]any, key string) {
	switch value := in[key].(type) {
	case string:
		if value != "" {
			out[key] = value
		}
	case map[string]any:
		if id, ok := value["id"].(string); ok && id != "" {
			out[key] = id
		}
	}
}

func addListDataCount(out, in map[string]any, key string) {
	list, ok := in[key].(map[string]any)
	if !ok {
		return
	}
	data, ok := list["data"].([]any)
	if ok && len(data) > 0 {
		out[key+"_count"] = len(data)
	}
}
