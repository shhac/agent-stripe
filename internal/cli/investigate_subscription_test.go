package cli

import (
	"strings"
	"testing"
	"time"
)

func TestSubscriptionItemsSubtotal(t *testing.T) {
	tests := []struct {
		name         string
		items        []map[string]any
		wantTotal    int64
		wantCurrency string
		wantOK       bool
	}{
		{name: "empty"},
		{
			name: "missing quantity defaults to one",
			items: []map[string]any{{
				"price": map[string]any{"unit_amount": float64(500), "currency": "usd"},
			}},
			wantTotal:    500,
			wantCurrency: "usd",
			wantOK:       true,
		},
		{
			name: "quantity multiplies unit amount",
			items: []map[string]any{{
				"quantity": float64(3),
				"price":    map[string]any{"unit_amount": float64(700), "currency": "gbp"},
			}},
			wantTotal:    2100,
			wantCurrency: "gbp",
			wantOK:       true,
		},
		{
			name: "missing unit amount is skipped",
			items: []map[string]any{{
				"quantity": float64(3),
				"price":    map[string]any{"currency": "gbp"},
			}},
			wantCurrency: "",
			wantOK:       false,
		},
		{
			name: "zero amount with currency is still known",
			items: []map[string]any{{
				"price": map[string]any{"unit_amount": float64(0), "currency": "usd"},
			}},
			wantCurrency: "usd",
			wantOK:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTotal, gotCurrency, gotOK := subscriptionItemsSubtotal(tt.items)
			if gotTotal != tt.wantTotal || gotCurrency != tt.wantCurrency || gotOK != tt.wantOK {
				t.Fatalf("subscriptionItemsSubtotal() = (%d, %q, %v), want (%d, %q, %v)", gotTotal, gotCurrency, gotOK, tt.wantTotal, tt.wantCurrency, tt.wantOK)
			}
		})
	}
}

func TestSubscriptionAmountFindingIncludesAmountEvidence(t *testing.T) {
	record := subscriptionAmountFinding(
		"sub_123",
		map[string]any{"id": "in_123", "amount_due": float64(3200), "currency": "usd"},
		map[string]any{"id": "upcoming_in_123", "amount_due": float64(4500), "currency": "usd"},
		[]map[string]any{{
			"quantity": float64(2),
			"price":    map[string]any{"unit_amount": float64(1500), "currency": "usd"},
		}},
	)

	if record.Type != "finding" || record.Severity != "info" {
		t.Fatalf("record = %#v, want info finding", record)
	}
	if !strings.Contains(record.Summary, "current item subtotal is 3000 USD") {
		t.Fatalf("summary missing subtotal evidence: %s", record.Summary)
	}
	assertDataField(t, record.Data, "subscription", "sub_123")
	assertDataField(t, record.Data, "item_count", 1)
	assertDataField(t, record.Data, "item_subtotal", int64(3000))
	assertDataField(t, record.Data, "item_currency", "usd")
	assertDataField(t, record.Data, "latest_invoice", "in_123")
	assertDataField(t, record.Data, "latest_invoice_amount_due", int64(3200))
	assertDataField(t, record.Data, "preview_amount_due", int64(4500))
}

func TestCardExpiresSoon(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		pm   map[string]any
		now  time.Time
		want bool
	}{
		{
			name: "missing card",
			pm:   map[string]any{},
			now:  now,
		},
		{
			name: "invalid month",
			pm:   map[string]any{"card": map[string]any{"exp_year": float64(2026), "exp_month": float64(13)}},
			now:  now,
		},
		{
			name: "already expired",
			pm:   map[string]any{"card": map[string]any{"exp_year": float64(2025), "exp_month": float64(12)}},
			now:  now,
			want: true,
		},
		{
			name: "exact threshold is not soon",
			pm:   map[string]any{"card": map[string]any{"exp_year": float64(2026), "exp_month": float64(2)}},
			now:  now,
		},
		{
			name: "before threshold is soon",
			pm:   map[string]any{"card": map[string]any{"exp_year": float64(2026), "exp_month": float64(2)}},
			now:  now.AddDate(0, 0, 1),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cardExpiresSoon(tt.pm, tt.now); got != tt.want {
				t.Fatalf("cardExpiresSoon() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectionRiskStatusUsesFirstMatch(t *testing.T) {
	ctx := collectionRiskContext{customerID: "cus_123", subscriptionID: "sub_123"}
	risk := statusCollectionRisk(map[string]any{"status": "past_due"}, ctx)
	if !strings.Contains(risk, "past_due status") {
		t.Fatalf("risk = %q, want past_due status", risk)
	}
}

func assertDataField(t *testing.T, data map[string]any, key string, want any) {
	t.Helper()
	if got := data[key]; got != want {
		t.Fatalf("data[%q] = %#v, want %#v", key, got, want)
	}
}
