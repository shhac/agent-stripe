package cli

import "strings"

func stripeSearchEquals(field, value string) string {
	return field + ":" + stripeSearchString(value)
}

func stripeSearchMetadataEquals(key, value string) string {
	return "metadata[" + stripeSearchString(key) + "]:" + stripeSearchString(value)
}

func stripeSearchString(value string) string {
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
