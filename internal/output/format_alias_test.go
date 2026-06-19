package output

import "testing"

// These pin the now-shared (intentionally lenient) ParseFormat: the "ndjson"
// and "yml" aliases and case-insensitivity resolve, while a junk value still
// errors fixable_by:agent. The NDJSON wire value stays "jsonl" across both the
// alias and the canonical spelling.
func TestParseFormatLenientAliases(t *testing.T) {
	cases := []struct {
		in   string
		want Format
	}{
		{"json", FormatJSON},
		{"JSON", FormatJSON},
		{"yaml", FormatYAML},
		{"yml", FormatYAML},
		{"YML", FormatYAML},
		{"jsonl", FormatNDJSON},
		{"ndjson", FormatNDJSON},
		{"NDJSON", FormatNDJSON},
		{"  jsonl  ", FormatNDJSON},
	}
	for _, c := range cases {
		got, err := ParseFormat(c.in)
		if err != nil {
			t.Fatalf("ParseFormat(%q) error = %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("ParseFormat(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseFormatRejectsUnknown(t *testing.T) {
	if _, err := ParseFormat("xml"); err == nil {
		t.Fatalf("ParseFormat(\"xml\") should error")
	}
}

// NDJSON's on-the-wire value is "jsonl" — unchanged by the migration.
func TestNDJSONWireValue(t *testing.T) {
	if string(FormatNDJSON) != "jsonl" {
		t.Fatalf("FormatNDJSON = %q, want jsonl", FormatNDJSON)
	}
}

// ResolveFormat keeps agent-stripe's error-swallowing fallback contract.
func TestResolveFormatFallsBackOnGarbage(t *testing.T) {
	if got := ResolveFormat("nonsense", FormatJSON); got != FormatJSON {
		t.Fatalf("ResolveFormat(garbage) = %q, want json fallback", got)
	}
	if got := ResolveFormat("", FormatNDJSON); got != FormatNDJSON {
		t.Fatalf("ResolveFormat(empty) = %q, want ndjson default", got)
	}
	if got := ResolveFormat("yaml", FormatJSON); got != FormatYAML {
		t.Fatalf("ResolveFormat(yaml) = %q, want yaml", got)
	}
}
