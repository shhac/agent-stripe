package cli

import (
	"strings"
	"testing"
)

// stripePII mirrors what a live Stripe last_payment_error carries: the failed
// PaymentMethod, nested whole. None of it belongs in a finding summary, which
// is free text and never passes through the redaction policy.
func stripePII() map[string]any {
	return map[string]any{
		"code":         "card_declined",
		"decline_code": "insufficient_funds",
		"message":      "Your card has insufficient funds.",
		"payment_method": map[string]any{
			"id":     "pm_123",
			"object": "payment_method",
			"billing_details": map[string]any{
				"name":  "Jenny Rosen",
				"email": "jenny.rosen@example.com",
				"phone": "+15550101001",
			},
			"card": map[string]any{
				"last4":       "4242",
				"fingerprint": "fPrInT123456",
			},
		},
	}
}

func TestPaymentFailureSummaryKeepsPersonalDataOut(t *testing.T) {
	summary := paymentFailureSummary(map[string]any{
		"id":                 "pi_123",
		"object":             "payment_intent",
		"last_payment_error": stripePII(),
	}, nil)

	for _, leaked := range []string{"Jenny Rosen", "jenny.rosen@example.com", "+15550101001", "fPrInT123456", "pm_123"} {
		if strings.Contains(summary, leaked) {
			t.Fatalf("summary leaked %q: %s", leaked, summary)
		}
	}
	for _, want := range []string{"pi_123", "card_declined", "insufficient_funds", "Your card has insufficient funds."} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q: %s", want, summary)
		}
	}
}

func TestPaymentFailureSummaryReportsOutcomeFieldsOnly(t *testing.T) {
	summary := paymentFailureSummary(nil, map[string]any{
		"id":     "ch_123",
		"object": "charge",
		"status": "failed",
		"outcome": map[string]any{
			"type":           "issuer_declined",
			"network_status": "declined_by_network",
			"risk_level":     "normal",
			"seller_message": "The bank returned the decline code insufficient_funds.",
			"risk_score":     42,
		},
	})

	for _, want := range []string{"ch_123", "issuer_declined", "declined_by_network", "normal"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q: %s", want, summary)
		}
	}
	// risk_score is not a named field, so a Stripe addition cannot widen the summary.
	if strings.Contains(summary, "risk_score") || strings.Contains(summary, "42") {
		t.Fatalf("summary included an unnamed field: %s", summary)
	}
}

func TestDescribeFieldsHandlesAbsentDetails(t *testing.T) {
	if got := describeFields(map[string]any{}, "code"); got != "no details reported" {
		t.Fatalf("describeFields() = %q", got)
	}
}
