package credential

import "testing"

func TestTypeClassifiesStripeKeyPrefixes(t *testing.T) {
	tests := map[string]string{
		"rk_live_123": "rk_live",
		"rk_test_123": "rk_test",
		"sk_live_123": "sk_live",
		"sk_test_123": "sk_test",
		"pk_live_123": "pk_live",
		"pk_test_123": "pk_test",
		"sk_org_123":  UnknownType,
		"not_stripe":  UnknownType,
		"":            UnknownType,
	}

	for key, want := range tests {
		if got := Type(key); got != want {
			t.Fatalf("Type(%q) = %q, want %q", key, got, want)
		}
	}
}
