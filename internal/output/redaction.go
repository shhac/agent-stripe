package output

import (
	"strings"

	out "github.com/shhac/lib-agent-output"
)

// RedactionOptions carries the per-call --expose allowlist. agent-stripe threads
// the global --expose flag through each Redact call rather than a package global.
type RedactionOptions struct {
	Expose []string
}

// RedactedString is the masked-value placeholder. It matches the shared
// out.RedactedPlaceholder so callers and tests can refer to either.
const RedactedString = out.RedactedPlaceholder

// RedactionNote re-exports the shared note shape (the @redacted list entry).
type RedactionNote = out.RedactionNote

// ParseExpose splits comma-joined --expose entries into normalized tokens. Kept
// for callers/tests; the shared out.Redact does the same normalization
// internally, so the raw Expose slice can also be passed straight through.
func ParseExpose(values []string) []string {
	var result []string
	seen := map[string]bool{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			normalized := normalizeExpose(part)
			if normalized == "" || seen[normalized] {
				continue
			}
			seen[normalized] = true
			result = append(result, normalized)
		}
	}
	return result
}

// Redact masks agent-stripe's sensitive fields using the shared redaction
// MECHANISM (the walk, the [REDACTED] placeholder, the @redacted notes, and
// --expose matching all live in lib-agent-output). What stays here is the
// POLICY — stripeSecrets() decides WHICH fields are secret.
func Redact(data any, opts RedactionOptions) any {
	cleaned, ok := toCleanAny(data, false)
	if !ok {
		return data
	}
	return out.Redact(cleaned, stripeSecrets(cleaned), opts.Expose)
}

// stripeSecrets is agent-stripe's redaction POLICY expressed as an
// out.RedactRule. It masks a fixed list of secret-named fields, any key that
// contains a secret-ish substring, and the object-context-sensitive "name"
// field (PII on customer/account objects — including their nested sub-objects —
// or under billing_details).
//
// The shared out.RedactRule only exposes the immediate enclosing map, but the
// "name" policy is inherited: a name anywhere inside a customer/account subtree
// is PII. So we precompute, in one read-only pass, the effective Stripe object
// for every path prefix and close the rule over it. The lib still owns the
// MECHANISM (the masking walk, [REDACTED] placeholder, @redacted notes, and
// --expose handling).
func stripeSecrets(decoded any) out.RedactRule {
	objects := map[string]string{}
	indexObjectContext(decoded, "", "", objects)
	return func(field out.RedactField) bool {
		return shouldRedactField(field.Key, field.Path, objects[field.Path])
	}
}

// indexObjectContext records, for each field path, the nearest enclosing Stripe
// object type, inheriting it into nested maps/arrays that lack their own
// "object" marker (mirroring the old walk's currentObject inheritance).
func indexObjectContext(value any, path, object string, into map[string]string) {
	switch val := value.(type) {
	case map[string]any:
		if o, ok := val["object"].(string); ok && o != "" {
			object = o
		}
		for key, child := range val {
			childPath := joinRedactionPath(path, key)
			into[childPath] = object
			indexObjectContext(child, childPath, object, into)
		}
	case []any:
		itemPath := path + "[]"
		for _, item := range val {
			indexObjectContext(item, itemPath, object, into)
		}
	}
}

func joinRedactionPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// alwaysRedacted are fields masked wherever they appear. Anything a substring
// rule below already covers is deliberately absent: this list is the policy a
// reader audits, so a key that cannot change the answer is noise.
var alwaysRedacted = map[string]bool{
	"email": true, "customer_email": true, "receipt_email": true, "contact_email": true,
	"phone": true, "contact_phone": true,
	"fingerprint": true, "iin": true, "network_transaction_id": true, "authorization_code": true,
	"receipt_url": true, "hosted_invoice_url": true, "invoice_pdf": true, "request_log_url": true,
	// Accounts v2 identity carries dates of birth on person objects; no triage
	// question needs the value rather than its presence.
	"date_of_birth": true,
}

// secretSubstrings catch the open-ended families — client_secret, api_key,
// refresh_token, a metadata key named api_token, and so on.
var secretSubstrings = []string{"secret", "password", "token", "api_key"}

func shouldRedactField(key, path, object string) bool {
	k := strings.ToLower(key)
	if alwaysRedacted[k] {
		return true
	}
	switch k {
	case "name", "given_name", "surname", "legal_name":
		return isAccountLikeObject(object) || strings.Contains(strings.ToLower(path), "billing_details.name")
	}
	for _, substring := range secretSubstrings {
		if strings.Contains(k, substring) {
			return true
		}
	}
	return false
}

// isAccountLikeObject covers the object types whose person-name fields are PII:
// v1 customers and accounts, and their Accounts v2 equivalents. display_name on
// a v2 account is a business-facing label, not a person, so it is not masked.
func isAccountLikeObject(object string) bool {
	switch object {
	case "customer", "account", "v2.core.account", "v2.core.account_person":
		return true
	}
	return false
}

func normalizeExpose(value string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(value)), ".")
}
