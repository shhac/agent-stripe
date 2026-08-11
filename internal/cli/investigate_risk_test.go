package cli

import (
	"strings"
	"testing"
	"time"
)

var riskNow = time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)

func TestCollectionRiskFlagsPastDueStatusWithoutLookups(t *testing.T) {
	// A past-due status is decided from the subscription alone; no fetch should
	// happen, so an empty route table proves it.
	server := routeServer(t, map[string]string{})

	risk, err := testInvestigator(server).collectionRiskAt(map[string]any{
		"id": "sub_1", "customer": "cus_1", "status": "past_due",
	}, riskNow)
	if err != nil {
		t.Fatalf("collectionRiskAt() error = %v", err)
	}
	if !strings.Contains(risk, "past_due status") {
		t.Fatalf("risk = %q", risk)
	}
}

func TestCollectionRiskSurfacesCustomerLookupFailure(t *testing.T) {
	// The customer fetch decides whether there is a default payment method.
	// Absorbing its failure reported "no default payment method visible" — a
	// risk that was never observed.
	server := routeServer(t, map[string]string{"/v1/customers/cus_1": failingRoute(500, "api_error")})

	risk, err := testInvestigator(server).collectionRiskAt(map[string]any{
		"id": "sub_1", "customer": "cus_1", "status": "active",
	}, riskNow)
	if err == nil {
		t.Fatalf("collectionRiskAt() error = nil, want the lookup failure surfaced (risk = %q)", risk)
	}
	if risk != "" {
		t.Fatalf("risk = %q, want no claim when the lookup failed", risk)
	}
}

func TestCollectionRiskSurfacesInvoiceLookupFailure(t *testing.T) {
	// Symmetrically: absorbing this one dropped a real risk and reported the
	// subscription as healthy.
	server := routeServer(t, map[string]string{
		"/v1/payment_methods/pm_1": `{"id":"pm_1","object":"payment_method","card":{"exp_month":12,"exp_year":2030}}`,
		"/v1/invoices/in_1":        failingRoute(500, "api_error"),
	})

	risk, err := testInvestigator(server).collectionRiskAt(map[string]any{
		"id": "sub_1", "customer": "cus_1", "status": "active",
		"default_payment_method": "pm_1",
		"latest_invoice":         "in_1",
	}, riskNow)
	if err == nil || risk != "" {
		t.Fatalf("collectionRiskAt() = %q, %v; want the failure surfaced", risk, err)
	}
}

func TestCollectionRiskReportsExpiringCard(t *testing.T) {
	server := routeServer(t, map[string]string{
		"/v1/payment_methods/pm_1": `{"id":"pm_1","object":"payment_method","card":{"exp_month":9,"exp_year":2026}}`,
	})

	risk, err := testInvestigator(server).collectionRiskAt(map[string]any{
		"id": "sub_1", "customer": "cus_1", "status": "active",
		"default_payment_method": "pm_1",
	}, riskNow)
	if err != nil {
		t.Fatalf("collectionRiskAt() error = %v", err)
	}
	if !strings.Contains(risk, "expiring soon") {
		t.Fatalf("risk = %q, want the expiring-card risk", risk)
	}
}

func TestSubscriptionRiskScanReportsAssessmentFailures(t *testing.T) {
	server := routeServer(t, map[string]string{
		"/v1/subscriptions":   `{"object":"list","data":[{"id":"sub_1","object":"subscription","customer":"cus_1","status":"active"}],"has_more":false}`,
		"/v1/customers/cus_1": failingRoute(500, "api_error"),
	})

	inv := testInvestigator(server)
	if err := inv.subscriptionRiskScan(30, 25, subscriptionRiskSpec{
		empty: "nothing found",
		risk:  investigator.collectionRisk,
	}); err != nil {
		t.Fatalf("subscriptionRiskScan() error = %v", err)
	}

	if findFinding(inv.records(), "Could not assess subscription sub_1") == nil {
		t.Fatalf("records = %#v, want an assessment-failure warning", inv.records())
	}
	// It must not also claim the window was clean.
	if findFinding(inv.records(), "nothing found") != nil {
		t.Fatalf("a failed assessment must not report an empty window")
	}
}

func TestEndpointHandlesEvent(t *testing.T) {
	cases := map[string]struct {
		enabled []any
		want    bool
	}{
		"exact match":      {[]any{"charge.failed", "invoice.paid"}, true},
		"wildcard":         {[]any{"*"}, true},
		"no match":         {[]any{"invoice.paid"}, false},
		"missing field":    {nil, false},
		"non-string entry": {[]any{42, "charge.failed"}, true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			endpoint := map[string]any{"id": "we_1"}
			if tc.enabled != nil {
				endpoint["enabled_events"] = tc.enabled
			}
			if got := endpointHandlesEvent(endpoint, "charge.failed"); got != tc.want {
				t.Fatalf("endpointHandlesEvent() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestWebhookEndpointFindingOmitsHandlesEventWithoutAnEventType(t *testing.T) {
	finding := webhookEndpointFinding(map[string]any{
		"id": "we_1", "status": "enabled", "enabled_events": []any{"invoice.paid"},
	}, "")
	if _, ok := finding.Data["handles_event"]; ok {
		t.Fatalf("handles_event must be omitted when nothing was matched against: %#v", finding.Data)
	}
	if strings.Contains(finding.Summary, "Handles") {
		t.Fatalf("summary should not claim handling: %s", finding.Summary)
	}
}
