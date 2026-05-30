package main

import (
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/mockstripe"
)

func TestCLIAgainstMockStripe(t *testing.T) {
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/agent-stripe",
		"--api-key", "sk_test_mock",
		"--base-url", server.URL,
		"events", "list",
		"--type", "charge.failed",
		"--limit", "1",
	)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent-stripe failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, `"evt_mock_charge_failed"`) {
		t.Fatalf("output did not contain mock event: %s", text)
	}
	if !strings.Contains(text, `"charge.failed"`) {
		t.Fatalf("output did not contain event type: %s", text)
	}
}

func TestCLIDebugAgainstMockStripe(t *testing.T) {
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/agent-stripe",
		"--debug",
		"--api-key", "sk_test_mock",
		"--base-url", server.URL,
		"balance", "get",
	)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent-stripe failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{`"@debug":"client"`, `"credential_source":"flag"`, `"@debug":"http"`, `"request_id":"req_mock_123"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("debug output missing %s:\n%s", want, text)
		}
	}
	if strings.Contains(text, "sk_test_mock") {
		t.Fatalf("debug output leaked API key:\n%s", text)
	}
}

func TestCLISubscriptionsAgainstMockStripe(t *testing.T) {
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	cmd := exec.Command("go", "run", "./cmd/agent-stripe",
		"--api-key", "sk_test_mock",
		"--base-url", server.URL,
		"subscriptions", "invoices", "sub_mock_past_due",
		"--status", "open",
	)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent-stripe failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, `"in_mock_open_failed"`) {
		t.Fatalf("output did not contain mock invoice: %s", text)
	}
	if !strings.Contains(text, `"payment_intent":"pi_mock_failed"`) {
		t.Fatalf("output did not contain failed PaymentIntent: %s", text)
	}
}

func TestCLIInvestigateInvoicePaymentAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "invoice-payment", "in_mock_paid")
	for _, want := range []string{
		`"object":"invoice"`,
		`"id":"in_mock_paid"`,
		`"object":"charge"`,
		`"id":"ch_mock_succeeded"`,
		`card ending 4242`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("investigation output missing %s:\n%s", want, out)
		}
	}
}

func TestCLIInvestigateCustomerCardPaymentAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "customer-card-payment", "--customer", "cus_mock_123", "--last4", "4242")
	for _, want := range []string{`"id":"ch_mock_succeeded"`, `Most recent payment`, `card ending 4242`} {
		if !strings.Contains(out, want) {
			t.Fatalf("investigation output missing %s:\n%s", want, out)
		}
	}
}

func TestCLIInvestigateCollectionRiskAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "collection-risk", "--limit", "10")
	for _, want := range []string{`"id":"sub_mock_past_due"`, `outreach about payment details`, `"id":"sub_mock_expiring_card"`, `expiring soon`, `"id":"sub_mock_missing_pm"`, `no default payment method`} {
		if !strings.Contains(out, want) {
			t.Fatalf("collection-risk output missing %s:\n%s", want, out)
		}
	}
}

func TestCLIInvestigateSubscriptionAndInvoiceMetadataAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "subscription-renewal", "--metadata", "tenant_id=acme")
	for _, want := range []string{`"id":"sub_mock_active"`, `"object":"invoice_preview"`, `preview amount is 4200 USD minor units`} {
		if !strings.Contains(out, want) {
			t.Fatalf("subscription-renewal output missing %s:\n%s", want, out)
		}
	}

	out = runMockCLI(t, "investigate", "invoice-metadata", "--number", "MOCK-0001")
	for _, want := range []string{`"id":"pi_mock_succeeded"`, `"order_id":"order_123"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("invoice-metadata output missing %s:\n%s", want, out)
		}
	}
}

func TestCLIInvestigateConnectMovementAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "outgoing-payment", "po_mock_failed")
	for _, want := range []string{`"id":"po_mock_failed"`, `bank account has been closed`} {
		if !strings.Contains(out, want) {
			t.Fatalf("outgoing-payment output missing %s:\n%s", want, out)
		}
	}

	out = runMockCLI(t, "investigate", "refund-recovery", "trr_mock_failed", "--transfer", "tr_mock_failed")
	for _, want := range []string{`"id":"trr_mock_failed"`, `failure_balance_transaction`} {
		if !strings.Contains(out, want) {
			t.Fatalf("refund-recovery output missing %s:\n%s", want, out)
		}
	}
}

func TestCLIInvestigateNewWorkflowScenariosAgainstMockStripe(t *testing.T) {
	checks := []struct {
		name  string
		args  []string
		wants []string
	}{
		{
			name:  "resolve invoice number",
			args:  []string{"investigate", "resolve", "MOCK-0001"},
			wants: []string{`"id":"in_mock_paid"`, `Resolved invoice number`},
		},
		{
			name:  "customer context",
			args:  []string{"investigate", "customer-context", "--customer", "cus_mock_123", "--limit", "3"},
			wants: []string{`"object":"customer"`, `"object":"payment_method"`, `"object":"invoice"`, `"object":"charge"`},
		},
		{
			name:  "webhook event extraction",
			args:  []string{"investigate", "webhook-event", "evt_mock_charge_failed"},
			wants: []string{`"object":"event"`, `"path":"data.object"`, `"id":"ch_mock_failed"`, `"summary":"Event evt_mock_charge_failed is charge.failed`},
		},
		{
			name:  "dispute response",
			args:  []string{"investigate", "dispute-response", "dp_mock_needs_response"},
			wants: []string{`"id":"dp_mock_needs_response"`, `"severity":"warning"`, `evidence due_by`},
		},
		{
			name:  "refund status",
			args:  []string{"investigate", "refund-status", "re_mock_pending"},
			wants: []string{`"id":"re_mock_pending"`, `"object":"transfer_reversal"`, `Failure detail`},
		},
		{
			name:  "payout failure",
			args:  []string{"investigate", "payout-failure", "po_mock_failed"},
			wants: []string{`"id":"po_mock_failed"`, `"id":"txn_mock_payout_failed"`, `bank account has been closed`},
		},
		{
			name:  "subscription cancel risk",
			args:  []string{"investigate", "subscription-cancel-risk", "--limit", "10"},
			wants: []string{`"id":"sub_mock_canceling"`, `set to cancel at period end`},
		},
		{
			name:  "truncation controls",
			args:  []string{"investigate", "--max-string", "40", "resolve", "prod_mock_basic"},
			wants: []string{`"id":"prod_mock_basic"`, `"truncated_fields"`, `"path":"description"`, `--expand-field description or --full`},
		},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			out := runMockCLI(t, check.args...)
			for _, want := range check.wants {
				if !strings.Contains(out, want) {
					t.Fatalf("%s output missing %s:\n%s", check.name, want, out)
				}
			}
		})
	}
}

func TestCLINewResourcePrimitivesAgainstMockStripe(t *testing.T) {
	checks := []struct {
		args []string
		want string
	}{
		{[]string{"checkout-sessions", "list", "--customer", "cus_mock_123"}, `"cs_mock_paid"`},
		{[]string{"checkout-sessions", "line-items", "cs_mock_paid"}, `"price_mock_basic"`},
		{[]string{"customers", "list", "--email", "buyer@example.com"}, `"cus_mock_123"`},
		{[]string{"products", "list", "--active", "true"}, `"prod_mock_basic"`},
		{[]string{"prices", "list", "--product", "prod_mock_basic"}, `"price_mock_basic"`},
		{[]string{"invoices", "line-items", "in_mock_paid"}, `"internal_product_id":"prod_internal_basic"`},
		{[]string{"setup-intents", "list", "--customer", "cus_mock_123"}, `"seti_mock_succeeded"`},
		{[]string{"payment-methods", "list", "--customer", "cus_mock_123", "--type", "card"}, `"last4":"4242"`},
		{[]string{"refunds", "list", "--payment-intent", "pi_mock_succeeded"}, `"re_mock_pending"`},
		{[]string{"transfers", "list", "--destination", "acct_mock_connected"}, `"tr_mock_failed"`},
		{[]string{"balance-transactions", "get", "txn_mock_succeeded"}, `"txn_mock_succeeded"`},
		{[]string{"payment-links", "list", "--active", "true"}, `"plink_mock_basic"`},
		{[]string{"early-fraud-warnings", "list", "--charge", "ch_mock_succeeded"}, `"issfr_mock_123"`},
	}
	for _, check := range checks {
		out := runMockCLI(t, check.args...)
		if !strings.Contains(out, check.want) {
			t.Fatalf("%v output missing %s:\n%s", check.args, check.want, out)
		}
	}
}

func runMockCLI(t *testing.T, args ...string) string {
	t.Helper()
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	allArgs := []string{"run", "./cmd/agent-stripe", "--api-key", "sk_test_mock", "--base-url", server.URL}
	allArgs = append(allArgs, args...)
	cmd := exec.Command("go", allArgs...)
	cmd.Dir = "../.."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("agent-stripe %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}
