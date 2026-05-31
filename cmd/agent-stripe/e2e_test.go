package main

import (
	"os/exec"
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

func TestCLIRedactsSensitiveStripeFieldsByDefault(t *testing.T) {
	out := runMockCLI(t, "payment-intents", "get", "pi_mock_succeeded")
	assertContains(t, out, `"@redacted"`, `"client_secret": "[REDACTED]"`, `"api_token": "[REDACTED]"`, `"path": "client_secret"`, `"path": "metadata.api_token"`, `"order_id": "order_123"`)
	assertNotContains(t, out, "pi_mock_succeeded_secret_fake", "tok_fake_order")
	assertNotContains(t, out, `"@redacted": true`)

	out = runMockCLI(t, "customers", "get", "cus_mock_123")
	assertContains(t, out, `"email": "[REDACTED]"`, `"name": "[REDACTED]"`, `"phone": "[REDACTED]"`)
	assertContains(t, out, `"path": "email"`, `"path": "name"`, `"path": "phone"`)
	assertNotContains(t, out, "buyer@example.com", "Mock Buyer", "+15550101001")
}

func TestCLIExposeRevealsRequestedStripeFields(t *testing.T) {
	out := runMockCLI(t, "--expose", "client_secret,metadata.api_token", "payment-intents", "get", "pi_mock_succeeded")
	assertContains(t, out, `"client_secret": "pi_mock_succeeded_secret_fake"`, `"api_token": "tok_fake_order"`, `"order_id": "order_123"`)
	assertNotContains(t, out, `"@redacted"`)
}

func TestCLIDebugRedactsStripeResponseBodies(t *testing.T) {
	out := runMockCLI(t, "--debug", "payment-intents", "get", "pi_mock_succeeded")
	assertContains(t, out, `"@debug":"http"`, `"path":"client_secret"`)
	assertNotContains(t, out, "pi_mock_succeeded_secret_fake", "tok_fake_order")
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
			args:  []string{"investigate", "refund", "re_mock_pending"},
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
			name:  "checkout session completion",
			args:  []string{"investigate", "checkout-session", "cs_mock_paid"},
			wants: []string{`"object":"checkout.session"`, `"id":"cs_mock_paid"`, `"object":"line_item"`, `"id":"pi_mock_succeeded"`, `payment_status=paid`},
		},
		{
			name:  "webhook delivery",
			args:  []string{"investigate", "webhook-delivery", "evt_mock_charge_failed"},
			wants: []string{`"object":"event"`, `"object":"webhook_endpoint"`, `"pending_webhooks":1`, `Some configured endpoints may not have successfully acknowledged it yet`},
		},
		{
			name:  "invoice collection",
			args:  []string{"investigate", "invoice-collection", "in_mock_open_failed"},
			wants: []string{`"id":"in_mock_open_failed"`, `"id":"pi_mock_failed"`, `"id":"ch_mock_failed"`, `Next payment attempt is at Unix time`},
		},
		{
			name:  "payment method readiness",
			args:  []string{"investigate", "payment-method-readiness", "cus_mock_123"},
			wants: []string{`"object":"payment_method"`, `"id":"pm_mock_visa"`, `card last4=4242`},
		},
		{
			name:  "ledger",
			args:  []string{"investigate", "ledger", "ch_mock_succeeded"},
			wants: []string{`"object":"balance_transaction"`, `"id":"txn_mock_succeeded"`, `"object":"application_fee"`, `ledger evidence gathered`},
		},
		{
			name:  "account health",
			args:  []string{"investigate", "account-health", "acct_mock_connected"},
			wants: []string{`"object":"account"`, `"payouts_enabled":false`, `Blockers: payouts disabled`},
		},
		{
			name:  "refund",
			args:  []string{"investigate", "refund", "ch_mock_succeeded"},
			wants: []string{`"object":"refund"`, `"id":"re_mock_pending"`, `Refund evidence gathered`},
		},
		{
			name:  "dispute impact",
			args:  []string{"investigate", "dispute-impact", "dp_mock_needs_response"},
			wants: []string{`"object":"dispute"`, `"id":"dp_mock_needs_response"`, `Dispute dp_mock_needs_response status=needs_response`},
		},
		{
			name:  "entitlement",
			args:  []string{"investigate", "entitlement", "--subscription", "sub_mock_active"},
			wants: []string{`"object":"subscription_item"`, `"object":"product"`, `"internal_product_id":"prod_internal_basic"`, `Entitlement evidence gathered`},
		},
		{
			name:  "timeline",
			args:  []string{"investigate", "timeline", "cus_mock_123", "--limit", "3"},
			wants: []string{`Timeline gathered`, `payment_intent pi_mock_succeeded`, `invoice in_mock_paid`, `charge ch_mock_succeeded`},
		},
		{
			name:  "fraud review",
			args:  []string{"investigate", "fraud-review", "issfr_mock_123"},
			wants: []string{`"object":"early_fraud_warning"`, `"id":"issfr_mock_123"`, `"object":"dispute"`, `Fraud review evidence gathered`},
		},
		{
			name:  "setup",
			args:  []string{"investigate", "setup", "seti_mock_succeeded"},
			wants: []string{`"object":"setup_intent"`, `"id":"seti_mock_succeeded"`, `"object":"payment_method"`, `SetupIntent seti_mock_succeeded status=succeeded`},
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
		{[]string{"customers", "list", "--ending-before", "cus_mock_456"}, `"cus_mock_123"`},
		{[]string{"customers", "list", "--starting-after", "cus_mock_123"}, `"cus_mock_456"`},
		{[]string{"products", "list", "--active", "true"}, `"prod_mock_basic"`},
		{[]string{"prices", "list", "--product", "prod_mock_basic"}, `"price_mock_basic"`},
		{[]string{"invoices", "line-items", "in_mock_paid", "--ending-before", "li_mock_basic"}, `"internal_product_id":"prod_internal_basic"`},
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

func TestCLIWrongIDPrefixHintsAgainstMockStripe(t *testing.T) {
	checks := []struct {
		name  string
		args  []string
		wants []string
	}{
		{
			name:  "resource get rejects known wrong prefix",
			args:  []string{"invoices", "get", "pi_mock_succeeded"},
			wants: []string{`"fixable_by":"agent"`, `looks like a PaymentIntent ID`, `this command expects invoice ID`, `agent-stripe payment-intents get pi_mock_succeeded`, `agent-stripe investigate incoming-payment pi_mock_succeeded`},
		},
		{
			name:  "nested resource rejects known wrong prefix",
			args:  []string{"invoices", "line-items", "pi_mock_succeeded"},
			wants: []string{`"fixable_by":"agent"`, `looks like a PaymentIntent ID`, `this command expects invoice ID`},
		},
		{
			name:  "investigation rejects unrecoverable known prefix",
			args:  []string{"investigate", "incoming-payment", "sub_mock_active"},
			wants: []string{`"fixable_by":"agent"`, `looks like a subscription ID`, `this investigation expects invoice ID`, `agent-stripe subscriptions get sub_mock_active`, `agent-stripe investigate subscription-renewal --subscription sub_mock_active`},
		},
		{
			name:  "recoverable investigation still accepts related prefixes",
			args:  []string{"investigate", "incoming-payment", "ch_mock_succeeded"},
			wants: []string{`"object":"charge"`, `"id":"ch_mock_succeeded"`, `"last4":"4242"`},
		},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			out := runMockCLI(t, check.args...)
			assertContains(t, out, check.wants...)
		})
	}
}

func TestCLICursorFlagsAreMutuallyExclusiveAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	allArgs := []string{"run", "./cmd/agent-stripe", "--api-key", "sk_test_mock", "--base-url", runner.server.URL, "customers", "list", "--starting-after", "cus_mock_123", "--ending-before", "cus_mock_456"}
	cmd := exec.Command("go", allArgs...)
	cmd.Dir = "../.."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected mutually exclusive cursor flags to fail:\n%s", out)
	}
	assertContains(t, string(out), `if any flags in the group [starting-after ending-before] are set none of the others can be`)
}
