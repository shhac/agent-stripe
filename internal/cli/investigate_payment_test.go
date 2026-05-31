package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shhac/agent-stripe/internal/api"
)

func TestLatestChargeForPaymentIntentUsesExpandedCharge(t *testing.T) {
	expanded := map[string]any{"id": "ch_expanded", "object": "charge"}
	got, err := (investigator{}).latestChargeForPaymentIntent(map[string]any{
		"id":            "pi_expanded",
		"object":        "payment_intent",
		"latest_charge": expanded,
	})
	if err != nil {
		t.Fatalf("latestChargeForPaymentIntent() error = %v", err)
	}
	if mapString(got, "id") != "ch_expanded" {
		t.Fatalf("latest charge = %#v, want expanded charge", got)
	}
}

func TestIncomingPaymentFromPaymentIntentCollectsRelatedEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/payment_intents/pi_failed":
			fmt.Fprint(w, `{"id":"pi_failed","object":"payment_intent","status":"requires_payment_method","latest_charge":"ch_failed","last_payment_error":{"message":"card declined"}}`)
		case "/v1/charges/ch_failed":
			fmt.Fprint(w, `{"id":"ch_failed","object":"charge","status":"failed","paid":false,"amount":4200,"currency":"usd","payment_intent":"pi_failed","failure_message":"card declined","payment_method_details":{"card":{"last4":"4242"}}}`)
		case "/v1/disputes":
			fmt.Fprint(w, `{"object":"list","data":[{"id":"dp_failed","object":"dispute","charge":"ch_failed"}],"has_more":false}`)
		case "/v1/refunds":
			fmt.Fprint(w, `{"object":"list","data":[{"id":"re_failed","object":"refund","charge":"ch_failed"}],"has_more":false}`)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	records, err := testInvestigator(server).incomingPayment("pi_failed")
	if err != nil {
		t.Fatalf("incomingPayment() error = %v", err)
	}
	assertRecordObject(t, records, "payment_intent", "pi_failed")
	assertRecordObject(t, records, "charge", "ch_failed")
	assertRecordObject(t, records, "dispute", "dp_failed")
	assertRecordObject(t, records, "refund", "re_failed")
	finding := findFinding(records, "Charge ch_failed failed")
	if finding == nil || finding.Severity != "warning" {
		t.Fatalf("finding = %#v, want warning failure finding", finding)
	}
}

func testInvestigator(server *httptest.Server) investigator {
	return investigator{
		ctx: context.Background(),
		client: api.NewClient(api.Options{
			APIKey:  "sk_test_123",
			BaseURL: server.URL,
		}),
	}
}

func assertRecordObject(t *testing.T, records []evidenceRecord, object, id string) {
	t.Helper()
	for _, record := range records {
		if record.Object == object && record.ID == id {
			return
		}
	}
	t.Fatalf("records missing %s %s: %#v", object, id, records)
}
