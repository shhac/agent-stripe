package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymentIntentsListDefaultsToCompactSummaries(t *testing.T) {
	h := newCLITestHarness(t)
	server := newCompactListServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "payment-intents", "list", "--limit", "1")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	item := firstNDJSONItem(t, stdout)
	if item["id"] != "pi_123" || item["latest_charge"] != "ch_123" {
		t.Fatalf("unexpected payment intent summary: %s", stdout)
	}
	for _, key := range []string{"client_secret", "metadata"} {
		if _, ok := item[key]; ok {
			t.Fatalf("compact PaymentIntent summary exposed %q in %s", key, stdout)
		}
	}
	lastErr, ok := item["last_payment_error"].(map[string]any)
	if !ok || lastErr["code"] != "card_declined" {
		t.Fatalf("last_payment_error summary = %#v", item["last_payment_error"])
	}
}

func TestPaymentIntentsListFullKeepsRawRedactedObjects(t *testing.T) {
	h := newCLITestHarness(t)
	server := newCompactListServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "payment-intents", "list", "--limit", "1", "--full")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	item := firstNDJSONItem(t, stdout)
	if item["client_secret"] != "[REDACTED]" {
		t.Fatalf("--full should preserve raw object with redacted client_secret: %s", stdout)
	}
	if _, ok := item["metadata"]; !ok {
		t.Fatalf("--full should preserve metadata object: %s", stdout)
	}
}

func TestEventsListDefaultsToCompactSummaries(t *testing.T) {
	h := newCLITestHarness(t)
	server := newCompactListServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "events", "list", "--limit", "1")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	item := firstNDJSONItem(t, stdout)
	if item["id"] != "evt_123" || item["type"] != "charge.failed" {
		t.Fatalf("unexpected event summary: %s", stdout)
	}
	if _, ok := item["data"]; ok {
		t.Fatalf("compact event summary exposed event data payload: %s", stdout)
	}
	dataObject, ok := item["data_object"].(map[string]any)
	if !ok || dataObject["id"] != "ch_123" || dataObject["object"] != "charge" {
		t.Fatalf("event data_object summary = %#v", item["data_object"])
	}
}

func TestEventsListFullKeepsRawRedactedObjects(t *testing.T) {
	h := newCLITestHarness(t)
	server := newCompactListServer(t)
	defer server.Close()

	stdout, stderr := h.run("--api-key", "sk_test_123", "--base-url", server.URL, "events", "list", "--limit", "1", "--full")

	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	item := firstNDJSONItem(t, stdout)
	if _, ok := item["data"]; !ok {
		t.Fatalf("--full should preserve event data payload: %s", stdout)
	}
	if !strings.Contains(stdout, "receipt_url") || !strings.Contains(stdout, "[REDACTED]") {
		t.Fatalf("--full should still redact nested event payload fields: %s", stdout)
	}
}

func TestCompactListRejectsExpandWithoutFull(t *testing.T) {
	h := newCLITestHarness(t)

	stdout, stderr := h.run("--api-key", "sk_test_123", "invoices", "list", "--expand", "payment_intent")

	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "--expand requires --full") {
		t.Fatalf("stderr = %q, want --expand/--full hint", stderr)
	}
}

func newCompactListServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Fatalf("limit = %q, want 1", got)
		}
		w.Header().Set("Request-Id", "req_test")
		switch r.URL.Path {
		case "/v1/payment_intents":
			_, _ = w.Write([]byte(`{
				"object": "list",
				"has_more": false,
				"data": [{
					"id": "pi_123",
					"object": "payment_intent",
					"created": 1700000000,
					"amount": 4200,
					"amount_received": 0,
					"currency": "usd",
					"status": "requires_payment_method",
					"customer": "cus_123",
					"payment_method": "pm_123",
					"latest_charge": {"id": "ch_123", "object": "charge"},
					"client_secret": "pi_123_secret_fake",
					"metadata": {"order_id": "order_123"},
					"last_payment_error": {
						"code": "card_declined",
						"decline_code": "insufficient_funds",
						"message": "Your card has insufficient funds."
					}
				}]
			}`))
		case "/v1/events":
			_, _ = w.Write([]byte(`{
				"object": "list",
				"has_more": false,
				"data": [{
					"id": "evt_123",
					"object": "event",
					"type": "charge.failed",
					"created": 1700000000,
					"livemode": false,
					"api_version": "2025-06-30.basil",
					"request": {"id": "req_123", "idempotency_key": "order_123"},
					"data": {"object": {
						"id": "ch_123",
						"object": "charge",
						"status": "failed",
						"customer": "cus_123",
						"receipt_url": "https://receipts.stripe.example.test/fake/ch_123"
					}}
				}]
			}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
}
