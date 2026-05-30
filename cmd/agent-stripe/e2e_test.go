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
	for _, want := range []string{`"id":"sub_mock_past_due"`, `outreach about payment details`} {
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

func TestCLINewResourcePrimitivesAgainstMockStripe(t *testing.T) {
	checks := []struct {
		args []string
		want string
	}{
		{[]string{"customers", "list", "--email", "buyer@example.com"}, `"cus_mock_123"`},
		{[]string{"invoices", "line-items", "in_mock_paid"}, `"internal_product_id":"prod_internal_basic"`},
		{[]string{"payment-methods", "list", "--customer", "cus_mock_123", "--type", "card"}, `"last4":"4242"`},
		{[]string{"refunds", "list", "--payment-intent", "pi_mock_succeeded"}, `"re_mock_pending"`},
		{[]string{"transfers", "list", "--destination", "acct_mock_connected"}, `"tr_mock_failed"`},
		{[]string{"balance-transactions", "get", "txn_mock_succeeded"}, `"txn_mock_succeeded"`},
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
