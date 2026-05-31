package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func TestWriteErrorIncludesHintAndFixability(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, agenterrors.New("bad query", agenterrors.FixableByAgent).WithHint("try a narrower query"))

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("error output was not JSON: %v", err)
	}
	if got["error"] != "bad query" {
		t.Fatalf("error = %q", got["error"])
	}
	if got["fixable_by"] != "agent" {
		t.Fatalf("fixable_by = %q", got["fixable_by"])
	}
	if !strings.Contains(got["hint"], "narrower") {
		t.Fatalf("hint = %q", got["hint"])
	}
}

func TestParseFormatAcceptsNDJSONAlias(t *testing.T) {
	got, err := ParseFormat("ndjson")
	if err != nil {
		t.Fatalf("ParseFormat() error = %v", err)
	}
	if got != FormatNDJSON {
		t.Fatalf("ParseFormat() = %q", got)
	}
}

func TestRedactSensitiveFieldsByDefault(t *testing.T) {
	input := map[string]any{
		"id":            "pi_mock_123",
		"object":        "payment_intent",
		"client_secret": "pi_mock_123_secret_fake",
		"metadata": map[string]any{
			"internal_product_id": "prod_internal_basic",
			"support_email":       "support@example.test",
			"api_token":           "tok_fake",
		},
	}

	redacted := Redact(input, RedactionOptions{}).(map[string]any)
	if redacted["client_secret"] == "pi_mock_123_secret_fake" {
		t.Fatalf("client_secret was not redacted: %#v", redacted)
	}
	if redacted["metadata"].(map[string]any)["internal_product_id"] != "prod_internal_basic" {
		t.Fatalf("non-sensitive metadata was redacted: %#v", redacted)
	}
	if _, ok := redacted["@redacted"].([]any); !ok {
		t.Fatalf("@redacted note missing: %#v", redacted)
	}
}

func TestRedactHonorsExposeByPathOrKey(t *testing.T) {
	input := map[string]any{
		"client_secret": "pi_mock_123_secret_fake",
		"metadata": map[string]any{
			"api_token": "tok_fake",
		},
	}

	redacted := Redact(input, RedactionOptions{Expose: []string{"client_secret,metadata.api_token"}}).(map[string]any)
	if redacted["client_secret"] != "pi_mock_123_secret_fake" {
		t.Fatalf("client_secret = %#v", redacted["client_secret"])
	}
	if redacted["metadata"].(map[string]any)["api_token"] != "tok_fake" {
		t.Fatalf("metadata.api_token = %#v", redacted["metadata"])
	}
	if _, ok := redacted["@redacted"]; ok {
		t.Fatalf("@redacted notes present despite exposed fields: %#v", redacted)
	}
}
