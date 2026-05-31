package output

import (
	"strings"
)

type RedactionOptions struct {
	Expose []string
}

const RedactedString = "[REDACTED]"

type redactionNote struct {
	Path       string `json:"path"`
	Reason     string `json:"reason"`
	ExposeHint string `json:"expose_hint"`
}

func ParseExpose(values []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			normalized := normalizeExpose(part)
			if normalized == "" || seen[normalized] {
				continue
			}
			seen[normalized] = true
			out = append(out, normalized)
		}
	}
	return out
}

func Redact(data any, opts RedactionOptions) any {
	cleaned, ok := toCleanAny(data, false)
	if !ok {
		return data
	}
	expose := exposeSet(ParseExpose(opts.Expose))
	result, notes := redactValue(cleaned, "", "", expose)
	if len(notes) == 0 {
		return result
	}
	if item, ok := result.(map[string]any); ok {
		item["@redacted"] = notesAsAny(notes)
	}
	return result
}

func redactValue(value any, path, currentObject string, expose map[string]bool) (any, []redactionNote) {
	switch v := value.(type) {
	case map[string]any:
		return redactMap(v, path, currentObject, expose)
	case []any:
		return redactSlice(v, path, currentObject, expose)
	default:
		return value, nil
	}
}

func redactMap(item map[string]any, path, currentObject string, expose map[string]bool) (map[string]any, []redactionNote) {
	if object, ok := item["object"].(string); ok && object != "" {
		currentObject = object
	}
	out := make(map[string]any, len(item))
	var notes []redactionNote
	for key, value := range item {
		fieldPath := joinRedactionPath(path, key)
		if shouldRedactField(key, fieldPath, currentObject) && !isExposed(key, fieldPath, expose) {
			note := redactionNote{
				Path:       fieldPath,
				Reason:     "sensitive_field",
				ExposeHint: "--expose " + fieldPath,
			}
			out[key] = redactedPlaceholder(value)
			notes = append(notes, note)
			continue
		}
		redacted, childNotes := redactValue(value, fieldPath, currentObject, expose)
		out[key] = redacted
		notes = append(notes, childNotes...)
	}
	return out, notes
}

func redactedPlaceholder(value any) any {
	if value == nil {
		return nil
	}
	return RedactedString
}

func redactSlice(items []any, path, currentObject string, expose map[string]bool) ([]any, []redactionNote) {
	out := make([]any, len(items))
	var notes []redactionNote
	for idx, item := range items {
		itemPath := path + "[]"
		redacted, childNotes := redactValue(item, itemPath, currentObject, expose)
		out[idx] = redacted
		notes = append(notes, childNotes...)
	}
	return out, notes
}

func shouldRedactField(key, path, object string) bool {
	k := strings.ToLower(key)
	p := strings.ToLower(path)
	switch k {
	case "client_secret", "secret", "api_key", "access_token", "refresh_token", "password",
		"email", "customer_email", "receipt_email", "phone", "fingerprint", "iin",
		"network_transaction_id", "authorization_code", "receipt_url", "hosted_invoice_url",
		"invoice_pdf", "request_log_url":
		return true
	case "name":
		return object == "customer" || object == "account" || strings.Contains(p, "billing_details.name")
	}
	return strings.Contains(k, "secret") ||
		strings.Contains(k, "password") ||
		strings.Contains(k, "token") ||
		strings.Contains(k, "access_token") ||
		strings.Contains(k, "refresh_token") ||
		strings.Contains(k, "api_key")
}

func isExposed(key, path string, expose map[string]bool) bool {
	if len(expose) == 0 {
		return false
	}
	normalizedPath := normalizeExpose(path)
	normalizedKey := normalizeExpose(key)
	if expose["all"] || expose["*"] || expose[normalizedPath] || expose[normalizedKey] {
		return true
	}
	for allowed := range expose {
		if allowed != "" && strings.HasPrefix(normalizedPath, allowed+".") {
			return true
		}
	}
	return false
}

func exposeSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func normalizeExpose(value string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(value)), ".")
}

func notesAsAny(notes []redactionNote) []any {
	out := make([]any, len(notes))
	for idx, note := range notes {
		out[idx] = map[string]any{
			"path":        note.Path,
			"reason":      note.Reason,
			"expose_hint": note.ExposeHint,
		}
	}
	return out
}

func joinRedactionPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
