package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/output"
)

func TestInvestigatorGetStreamsDecodedEntity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	restore := output.SetWritersForTest(&stdout, &stderr)
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/payment_intents/pi_stream" {
			t.Fatalf("path = %s, want /v1/payment_intents/pi_stream", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"pi_stream","object":"payment_intent","status":"succeeded"}`)
	}))
	defer server.Close()

	inv := investigator{
		ctx: context.Background(),
		client: api.NewClient(api.Options{
			APIKey:  "sk_test_123",
			BaseURL: server.URL,
		}),
		stream: newEvidenceStreamer("jsonl", defaultEvidenceOptions()),
	}

	item, err := inv.get("/v1/payment_intents/pi_stream", nil)
	if err != nil {
		t.Fatalf("get() error = %v", err)
	}
	if got := mapString(item, "id"); got != "pi_stream" {
		t.Fatalf("id = %q, want pi_stream", got)
	}

	lines := ndjsonLines(stdout.String())
	if len(lines) != 1 {
		t.Fatalf("streamed %d lines, want 1: %s", len(lines), stdout.String())
	}
	var record evidenceRecord
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("decode streamed record: %v\nline: %s", err, lines[0])
	}
	if record.Type != "entity" || record.Object != "payment_intent" || record.ID != "pi_stream" {
		t.Fatalf("streamed record = %#v, want payment_intent pi_stream entity", record)
	}
}

func TestEvidenceStreamerDeduplicatesFinalEntity(t *testing.T) {
	var stdout, stderr bytes.Buffer
	restore := output.SetWritersForTest(&stdout, &stderr)
	defer restore()

	stream := newEvidenceStreamer("jsonl", defaultEvidenceOptions())
	entity := entityRecord("payment_intent", map[string]any{
		"id":     "pi_stream",
		"object": "payment_intent",
		"status": "succeeded",
	})
	stream.emit(entity)
	stream.writeRemaining([]evidenceRecord{
		entity,
		{Type: "finding", Severity: "info", Summary: "investigation complete"},
	})

	lines := ndjsonLines(stdout.String())
	if len(lines) != 2 {
		t.Fatalf("streamed %d lines, want entity plus finding: %s", len(lines), stdout.String())
	}
	entityLines := 0
	findingLines := 0
	for _, line := range lines {
		var record evidenceRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("decode streamed record: %v\nline: %s", err, line)
		}
		if record.Type == "entity" {
			entityLines++
		}
		if record.Type == "finding" {
			findingLines++
		}
	}
	if entityLines != 1 || findingLines != 1 {
		t.Fatalf("entity lines = %d, finding lines = %d, want 1 each\n%s", entityLines, findingLines, stdout.String())
	}
}

func ndjsonLines(out string) []string {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
