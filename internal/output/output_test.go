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
