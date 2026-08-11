package main

import "testing"

func TestCLIDuplicateChargeClustersByWindow(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "duplicate-charge", "--customer", "cus_mock_123")

	runner.AssertContains(out,
		`"id":"ch_mock_dupe_a"`,
		`"id":"ch_mock_dupe_b"`,
		`2 charges of 2500 usd on card ending 4242`,
		`"seconds_apart":45`,
	)
	// Every inspected charge is emitted as evidence, as in the other scans, so
	// the claim to check is the cluster: a same-amount charge a month later is a
	// repeat customer, not a duplicate, and must not be in the finding.
	assertNotContains(t, out, `"charges":["ch_mock_dupe_a","ch_mock_dupe_b","ch_mock_dupe_far"]`)
	runner.AssertContains(out, `"charges":["ch_mock_dupe_a","ch_mock_dupe_b"]`)
}

func TestCLIDuplicateChargeRequiresACustomer(t *testing.T) {
	out := runMockCLIErr(t, "investigate", "duplicate-charge")
	assertContains(t, out, "--customer is required", "not unique")
}

func TestCLIStatementDescriptorMatchesPartialText(t *testing.T) {
	runner := newMockCLIRunner(t)
	// Banks truncate, so a fragment of the descriptor must still match.
	out := runner.Run("investigate", "statement-descriptor", "--descriptor", "FUREVER")
	runner.AssertContains(out, `matches charge ch_mock_dupe`, `"statement_text":"FUREVER GROOMING"`)

	miss := runner.Run("investigate", "statement-descriptor", "--descriptor", "NOTATHING")
	runner.AssertContains(miss, `No charge among the`, `try a shorter fragment`)
}

func TestCLIActionRequiredFindsStalledPayments(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "action-required")
	runner.AssertContains(out,
		`"id":"pi_mock_requires_action"`,
		`needs the customer to act`,
	)
	// The completion URL is redacted; the finding says it exists instead.
	assertNotContains(t, out, "https://invoices.stripe.example.test")
}

func TestCLIRefundSettlementSeparatesStripeFromTheBank(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "refund-settlement", "re_mock_settled")
	runner.AssertContains(out,
		`Stripe has sent it`,
		`up to their bank`,
		`24692167822600123456789`,
		`"acquirer_reference"`,
	)
}

func TestCLIInvoiceTotalReconcilesOrExplains(t *testing.T) {
	runner := newMockCLIRunner(t)
	ok := runner.Run("investigate", "invoice-total", "in_mock_paid")
	runner.AssertContains(ok, `The arithmetic reconciles`, `"severity":"info"`)

	bad := runner.Run("investigate", "invoice-total", "in_mock_tax_mismatch")
	runner.AssertContains(bad,
		`Does not reconcile`,
		`line items sum to 4500 but subtotal is 5000`,
		`subtotal 5000 minus discounts 250 plus tax 500 is 5250, but total is 6000`,
		`"severity":"warning"`,
	)
}

func TestCLIRefundRecoveryReportsConnectLiability(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "refund-recovery", "re_mock_pending")
	runner.AssertContains(out,
		`Connect liability for refund re_mock_pending`,
		`pulled back from the connected account by reversal trr_mock_failed`,
		`application fee was kept`,
	)
}
