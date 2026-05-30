package main

import (
	"strings"
	"testing"
)

func TestCLIAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("events", "list", "--type", "charge.failed", "--limit", "1")
	runner.AssertContains(out, `"evt_mock_charge_failed"`, `"charge.failed"`)
}

func TestCLIDebugAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("--debug", "balance", "get")
	runner.AssertContains(out, `"@debug":"client"`, `"credential_source":"flag"`, `"@debug":"http"`, `"request_id":"req_mock_123"`)
	if strings.Contains(out, "sk_test_mock") {
		t.Fatalf("debug output leaked API key:\n%s", out)
	}
}

func TestCLISubscriptionsAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("subscriptions", "invoices", "sub_mock_past_due", "--status", "open")
	runner.AssertContains(out, `"in_mock_open_failed"`, `"payment_intent":"pi_mock_failed"`)
}

func TestCLIInvestigateInvoicePaymentAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "invoice-payment", "in_mock_paid")
	assertContains(t, out,
		`"object":"invoice"`,
		`"id":"in_mock_paid"`,
		`"object":"charge"`,
		`"id":"ch_mock_succeeded"`,
		`card ending 4242`,
	)
}

func TestCLIInvestigateCustomerCardPaymentAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "customer-card-payment", "--customer", "cus_mock_123", "--last4", "4242")
	assertContains(t, out, `"id":"ch_mock_succeeded"`, `Most recent payment`, `card ending 4242`)
}

func TestCLIInvestigateCollectionRiskAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "collection-risk", "--limit", "10")
	assertContains(t, out, `"id":"sub_mock_past_due"`, `outreach about payment details`, `"id":"sub_mock_expiring_card"`, `expiring soon`, `"id":"sub_mock_missing_pm"`, `no default payment method`)
}

func TestCLIInvestigateSubscriptionAndInvoiceMetadataAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "subscription-renewal", "--metadata", "tenant_id=acme")
	assertContains(t, out, `"id":"sub_mock_active"`, `"object":"invoice_preview"`, `preview amount is 4200 USD minor units`)

	out = runMockCLI(t, "investigate", "invoice-metadata", "--number", "MOCK-0001")
	assertContains(t, out, `"id":"pi_mock_succeeded"`, `"order_id":"order_123"`)
}

func TestCLIInvestigateConnectMovementAgainstMockStripe(t *testing.T) {
	out := runMockCLI(t, "investigate", "outgoing-payment", "po_mock_failed")
	assertContains(t, out, `"id":"po_mock_failed"`, `bank account has been closed`)

	out = runMockCLI(t, "investigate", "refund-recovery", "trr_mock_failed", "--transfer", "tr_mock_failed")
	assertContains(t, out, `"id":"trr_mock_failed"`, `failure_balance_transaction`)
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
			name:  "subscription item metadata",
			args:  []string{"investigate", "subscription-items", "--subscription", "sub_mock_active"},
			wants: []string{`"object":"subscription_item"`, `"id":"si_mock_basic"`, `"object":"product"`, `"internal_product_id":"prod_internal_basic"`},
		},
		{
			name:  "subscription amount change",
			args:  []string{"investigate", "subscription-amount-change", "--subscription", "sub_mock_past_due"},
			wants: []string{`"id":"sub_mock_past_due"`, `"id":"in_mock_open_failed"`, `"object":"invoice_preview"`, `current item subtotal is 29700 USD minor units`},
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
			assertContains(t, out, check.wants...)
		})
	}
}

func TestCLIDomainUsageAgainstMockStripe(t *testing.T) {
	checks := []struct {
		args  []string
		wants []string
	}{
		{[]string{"subscriptions", "usage"}, []string{`subscriptions — renewal`, `subscription-amount-change`}},
		{[]string{"invoices", "usage"}, []string{`invoices — invoice payment`, `invoice-metadata`}},
		{[]string{"payments", "usage"}, []string{`payments — PaymentIntent`, `customer-card-payment`}},
		{[]string{"connect", "usage"}, []string{`connect — connected-account`, `refund-recovery`}},
	}
	for _, check := range checks {
		out := runMockCLI(t, check.args...)
		assertContains(t, out, check.wants...)
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
		assertContains(t, out, check.want)
	}
}
