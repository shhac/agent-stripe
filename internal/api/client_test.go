package api

import (
	"encoding/json"
	"net/url"
	"testing"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func TestDecodeList(t *testing.T) {
	raw := json.RawMessage(`{"object":"list","has_more":true,"data":[{"id":"evt_1"}]}`)

	list, err := DecodeList(raw)
	if err != nil {
		t.Fatalf("DecodeList() error = %v", err)
	}
	if !list.HasMore {
		t.Fatalf("HasMore = false, want true")
	}
	if len(list.Data) != 1 {
		t.Fatalf("len(Data) = %d, want 1", len(list.Data))
	}
}

func TestClassifyHTTPErrorIncludesStripeHints(t *testing.T) {
	body := []byte(`{"error":{"message":"No such payment_intent","code":"resource_missing","request_log_url":"https://dashboard.stripe.com/test/logs/req_123"}}`)

	err := classifyHTTPError(404, "req_123", body)
	if err.FixableBy != agenterrors.FixableByAgent {
		t.Fatalf("FixableBy = %q", err.FixableBy)
	}
	if err.Hint == "" {
		t.Fatalf("Hint should be populated")
	}
	if err.Message == "" || err.Message == "HTTP 404" {
		t.Fatalf("Message = %q", err.Message)
	}
}

func TestAddHelpers(t *testing.T) {
	values := url.Values{}
	AddLimit(values, 12)
	AddCreatedRange(values, "100", "200")
	AddExpand(values, []string{"latest_charge", "customer"})

	if values.Get("limit") != "12" {
		t.Fatalf("limit = %q", values.Get("limit"))
	}
	if values.Get("created[gte]") != "100" || values.Get("created[lte]") != "200" {
		t.Fatalf("created range not set: %v", values)
	}
	if got := values["expand[]"]; len(got) != 2 {
		t.Fatalf("expand[] = %v", got)
	}
}
