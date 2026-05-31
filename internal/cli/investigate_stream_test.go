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
		evidence: newEvidenceCollector(newEvidenceStreamer("jsonl", defaultEvidenceOptions())),
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

func TestEvidenceCollectorDeduplicatesRecords(t *testing.T) {
	var stdout, stderr bytes.Buffer
	restore := output.SetWritersForTest(&stdout, &stderr)
	defer restore()

	collector := newEvidenceCollector(newEvidenceStreamer("jsonl", defaultEvidenceOptions()))
	entity := entityRecord("payment_intent", map[string]any{
		"id":     "pi_stream",
		"object": "payment_intent",
		"status": "succeeded",
	})
	finding := evidenceRecord{Type: "finding", Severity: "info", Summary: "investigation complete"}
	records := collector.append(nil, entity, finding)
	records = collector.appendAll(records, []evidenceRecord{entity, finding})
	if len(records) != 2 {
		t.Fatalf("collector records = %d, want entity plus finding", len(records))
	}

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

func TestWorkflowFindingsStreamThroughCollector(t *testing.T) {
	var stdout, stderr bytes.Buffer
	restore := output.SetWritersForTest(&stdout, &stderr)
	defer restore()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/invoices/in_stream":
			fmt.Fprint(w, `{"id":"in_stream","object":"invoice","status":"paid","paid":true,"amount_paid":4200,"currency":"usd","payment_intent":"pi_stream"}`)
		case "/v1/payment_intents/pi_stream":
			fmt.Fprint(w, `{"id":"pi_stream","object":"payment_intent","status":"succeeded","latest_charge":"ch_stream"}`)
		case "/v1/charges/ch_stream":
			fmt.Fprint(w, `{"id":"ch_stream","object":"charge","status":"succeeded","paid":true,"amount":4200,"currency":"usd","payment_intent":"pi_stream","payment_method_details":{"card":{"last4":"4242"}}}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	inv := investigator{
		ctx: context.Background(),
		client: api.NewClient(api.Options{
			APIKey:  "sk_test_123",
			BaseURL: server.URL,
		}),
		evidence: newEvidenceCollector(newEvidenceStreamer("jsonl", defaultEvidenceOptions())),
	}

	if _, err := inv.invoicePayment("in_stream"); err != nil {
		t.Fatalf("invoicePayment() error = %v", err)
	}
	if !strings.Contains(stdout.String(), `"summary":"Invoice in_stream paid 4200 USD minor units with card ending 4242 through PaymentIntent pi_stream."`) {
		t.Fatalf("stdout missing streamed workflow finding:\n%s", stdout.String())
	}
}

func TestStreamingEvidenceNormalizesBeforeEmit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	restore := output.SetWritersForTest(&stdout, &stderr)
	defer restore()

	opts := defaultEvidenceOptions()
	opts.maxString = 12
	collector := newEvidenceCollector(newEvidenceStreamer("jsonl", opts))
	collector.append(nil, entityRecord("payment_intent", map[string]any{
		"id":            "pi_stream",
		"object":        "payment_intent",
		"client_secret": "pi_stream_secret_leak",
		"description":   "this description is intentionally long",
	}))

	lines := ndjsonLines(stdout.String())
	if len(lines) != 1 {
		t.Fatalf("streamed %d lines, want 1: %s", len(lines), stdout.String())
	}
	if strings.Contains(lines[0], "pi_stream_secret_leak") {
		t.Fatalf("stream leaked client_secret: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"client_secret":"[REDACTED]"`) || !strings.Contains(lines[0], `"@redacted"`) {
		t.Fatalf("stream did not redact sensitive field: %s", lines[0])
	}
	if !strings.Contains(lines[0], `"truncated_fields"`) || !strings.Contains(lines[0], `"path":"description"`) {
		t.Fatalf("stream did not include truncation note: %s", lines[0])
	}
}

func ndjsonLines(out string) []string {
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
