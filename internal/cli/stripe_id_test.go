package cli

import (
	"errors"
	"testing"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func TestClassifyStripeIDPrefixes(t *testing.T) {
	tests := []struct {
		id   string
		kind string
		ok   bool
	}{
		{id: "trr_123", kind: "transfer_reversal", ok: true},
		{id: "tr_123", kind: "transfer", ok: true},
		{id: "pi_123", kind: "payment_intent", ok: true},
		{id: "seti_123", kind: "setup_intent", ok: true},
		{id: "acct_123", kind: "account", ok: true},
		{id: "unknown_123", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got, ok := classifyStripeID(tt.id)
			if ok != tt.ok {
				t.Fatalf("ok = %t, want %t", ok, tt.ok)
			}
			if ok && got.Kind != tt.kind {
				t.Fatalf("kind = %q, want %q", got.Kind, tt.kind)
			}
		})
	}
}

func TestValidateExpectedStripeID(t *testing.T) {
	if err := validateExpectedStripeID("pi_123", "payment_intent"); err != nil {
		t.Fatalf("matching ID returned error: %v", err)
	}
	if err := validateExpectedStripeID("unknown_123", "payment_intent"); err != nil {
		t.Fatalf("unknown prefix should be accepted for Stripe to decide: %v", err)
	}
	err := validateExpectedStripeID("ch_123", "payment_intent")
	var apiErr *agenterrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %#v, want APIError", err)
	}
	if apiErr.FixableBy != agenterrors.FixableByAgent {
		t.Fatalf("FixableBy = %q, want agent", apiErr.FixableBy)
	}
}

func TestValidateAllowedStripeID(t *testing.T) {
	if err := validateAllowedStripeID("ch_123", "charge", "payment_intent"); err != nil {
		t.Fatalf("allowed ID returned error: %v", err)
	}
	if err := validateAllowedStripeID("unknown_123", "charge"); err != nil {
		t.Fatalf("unknown prefix should be accepted for Stripe to decide: %v", err)
	}
	err := validateAllowedStripeID("acct_123", "charge", "payment_intent")
	var apiErr *agenterrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %#v, want APIError", err)
	}
	if apiErr.FixableBy != agenterrors.FixableByAgent {
		t.Fatalf("FixableBy = %q, want agent", apiErr.FixableBy)
	}
}
