package main

import "testing"

func TestCLIAccountsV2ListAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "list", "--limit", "2")
	assertContains(t, out,
		`"id":"acct_mock_v2_active"`,
		`"object":"v2.core.account"`,
		`"applied_configurations":["customer","merchant"]`,
		`"@pagination"`,
		`"next_page":"page_mock_2"`,
	)
	// The v2 list endpoint has no include support, so nothing pretends to know
	// capability state here.
	assertNotContains(t, out, `"capabilities"`)
}

func TestCLIAccountsV2ListFiltersByConfiguration(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "list", "--applied-configuration", "recipient")
	assertContains(t, out, `"id":"acct_mock_v2_recipient"`)
	assertNotContains(t, out, "acct_mock_v2_active", "acct_mock_v2_restricted")
}

func TestCLIAccountsV2GetRequestsEveryIncludeByDefault(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "get", "acct_mock_v2_restricted")
	assertContains(t, out,
		`"object":"v2.core.account"`,
		`"card_payments":{"status":"restricted"`,
		`"description":"identity.business_details.registered_name"`,
		`"entity_type":"company"`,
		`"fees_collector":"application"`,
	)
}

func TestCLIAccountsV2GetNarrowsIncludes(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "get", "acct_mock_v2_restricted", "--include", "requirements")
	assertContains(t, out, `"requirements":{`, `"identity":null`, `"configuration":null`)
}

func TestCLIAccountsV2PersonsRedactPersonalData(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "persons", "get", "acct_mock_v2_restricted", "person_mock_representative")
	assertContains(t, out,
		`"given_name":"[REDACTED]"`,
		`"surname":"[REDACTED]"`,
		`"email":"[REDACTED]"`,
		`"date_of_birth":"[REDACTED]"`,
		`"relationship":{"owner":true`,
	)
	assertNotContains(t, out, "Jenny", "Rosen", "jenny.rosen@example.com")
}

func TestCLIEventsV2ListAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("events-v2", "list", "--object-id", "acct_mock_v2_restricted")
	assertContains(t, out,
		`"object":"v2.core.event"`,
		`"type":"v2.core.account[configuration.merchant].capability_status_updated"`,
		`"related_object":{"id":"acct_mock_v2_restricted"`,
	)
	assertNotContains(t, out, "acct_mock_v2_active")
}

func TestCLIInvestigateAccountHealthReadsV2(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "account-health", "acct_mock_v2_restricted")
	assertContains(t, out,
		`"object":"v2.core.account"`,
		`"object":"v2.core.account_person"`,
		`Accounts v2 account acct_mock_v2_restricted`,
		`Capabilities not active: merchant.card_payments, merchant.stripe_balance.payouts`,
		`identity.business_details.registered_name`,
		`"namespace":"v2"`,
	)
	// A v2 account has no charges_enabled/payouts_enabled; reporting them would
	// be a fabricated blocker.
	assertNotContains(t, out, "charges_enabled", "payouts_enabled")
}

func TestCLIInvestigateAccountHealthFallsBackToV1(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "account-health", "acct_mock_connected")
	assertContains(t, out,
		`is not an Accounts v2 account (v1_account_instead_of_v2_account)`,
		`Connect v1 account acct_mock_connected charges_enabled=true payouts_enabled=false`,
		`"namespace":"v1"`,
	)
}

func TestCLIInvestigateAccountHealthHealthyV2AccountIsInfo(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "account-health", "acct_mock_v2_active")
	assertContains(t, out, `All capabilities are active`, `"severity":"info"`)
}

func TestCLIInvestigateAccountEventsAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "account-events", "acct_mock_v2_restricted")
	assertContains(t, out,
		`"object":"v2.core.event"`,
		`v2 event(s) between`,
		`current, not point-in-time`,
		`agent-stripe investigate account-health acct_mock_v2_restricted`,
	)
}

func TestCLIInvestigateResolveNamesAccountNamespace(t *testing.T) {
	runner := newMockCLIRunner(t)
	v2 := runner.Run("investigate", "resolve", "acct_mock_v2_active")
	assertContains(t, v2, `Resolved acct_mock_v2_active as an Accounts v2 account`, `"namespace":"v2"`)

	v1 := runner.Run("investigate", "resolve", "acct_mock_connected")
	assertContains(t, v1, `Resolved acct_mock_connected as a Connect v1 account`, `--namespace v1`)
}

func TestCLIInvestigateWebhookEventHandlesV2ThinEvent(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "webhook-event", "evt_test_mock_requirements_updated")
	assertContains(t, out,
		`"object":"v2.core.event"`,
		`v2 events are thin`,
		`"object":"v2.core.account"`,
	)
}

func TestCLIAccountsV2RejectsUnknownInclude(t *testing.T) {
	out := runMockCLIErr(t, "accounts-v2", "get", "acct_mock_v2_active", "--include", "capabilities")
	assertContains(t, out, `unknown --include value`, `"fixable_by":"agent"`, "configuration.merchant")
}

func TestCLIAccountsV2RejectsV1AccountID(t *testing.T) {
	// A v1 account ID is an item-level miss, so it stays on stdout as
	// @unresolved with the command that would have worked.
	out := runMockCLI(t, "accounts-v2", "get", "acct_mock_connected")
	assertContains(t, out,
		`"@unresolved"`,
		"v1_account_instead_of_v2_account",
		"agent-stripe accounts get <acct_id>",
	)
}

func TestCLIInvestigateReportsFailedRelatedLookups(t *testing.T) {
	runner := newMockCLIRunner(t)
	// The charge itself resolves; the refund lookup beside it fails. The report
	// must say so rather than reading as "this charge has no refunds".
	runner.ArmFault("/v1/refunds=500")
	out := runner.Run("investigate", "ledger", "ch_mock_succeeded")

	assertContains(t, out,
		`"object":"charge"`,
		`"severity":"warning"`,
		`Could not gather refunds`,
	)
}

func TestCLIRetriesRateLimitsThenSucceeds(t *testing.T) {
	runner := newMockCLIRunner(t)
	runner.ArmFault("/v1/charges=429x2")
	out := runner.Run("charges", "get", "ch_mock_succeeded")
	assertContains(t, out, `"id":"ch_mock_succeeded"`)
}

func TestCLIReportsNonJSONErrorBodyBounded(t *testing.T) {
	runner := newMockCLIRunner(t)
	runner.ArmFault("/v1/charges=garbage")
	out := runner.RunExpectingError("charges", "get", "ch_mock_succeeded")
	assertContains(t, out, "HTTP 502", "upstream gateway error")
}

func TestCLIInvestigationSummariesCarryNoPersonalData(t *testing.T) {
	runner := newMockCLIRunner(t)
	// pi_mock_failed's last_payment_error nests a PaymentMethod, as Stripe's
	// does. Entity data is redacted; summaries must not carry it in the first
	// place, since they never pass through the redaction policy.
	out := runner.Run("investigate", "incoming-payment", "pi_mock_failed")

	assertNotContains(t, out, "fPrInTdEcLiNeD", "buyer@example.com", "+15550101001")
	assertContains(t, out, "insufficient funds")
}

func TestCLIAccountSubresourcesAnswerRequirements(t *testing.T) {
	runner := newMockCLIRunner(t)

	// acct_mock_connected's requirements name "external_account". These are the
	// commands that say what it actually has and which capability is waiting.
	external := runner.Run("accounts", "external-accounts", "acct_mock_connected")
	assertContains(t, external, `"id":"ba_mock_closed"`, `"status":"errored"`, `"last4":"6789"`)

	capabilities := runner.Run("accounts", "capabilities", "acct_mock_connected")
	assertContains(t, capabilities, `"id":"transfers"`, `"status":"inactive"`, `"external_account"`)

	persons := runner.Run("accounts", "persons", "list", "acct_mock_connected")
	assertContains(t, persons, `"id":"person_mock_v1_rep"`, `"representative":true`, `verification.document`)
	assertNotContains(t, persons, "robin.vance@example.com")
}

func TestCLIAccountPersonsFilterByRelationship(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts", "persons", "list", "acct_mock_connected", "--relationship", "director")
	assertNotContains(t, out, "person_mock_v1_rep")

	bad := runMockCLIErr(t, "accounts", "persons", "list", "acct_mock_connected", "--relationship", "auditor")
	assertContains(t, bad, `unknown --relationship value`, "representative, owner, director, executive")
}

func TestCLIV2PayoutMethodsScopeByContext(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "payout-methods", "acct_mock_v2_recipient")
	assertContains(t, out,
		`"object":"v2.money_management.payout_method"`,
		`"last4":"3311"`,
		`"usage_status"`,
	)
}

func TestCLIConnectReadinessSweepsAccounts(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "connect-readiness", "--limit", "5")
	assertContains(t, out,
		`inspected connected accounts have blockers (v2)`,
		`acct_mock_v2_restricted`,
		`"blocked_count":2`,
	)
	// The healthy account must not be reported as blocked.
	assertNotContains(t, out, `"blocked":["acct_mock_v2_active"`)
}

func TestCLIConnectReadinessFallsBackToV1(t *testing.T) {
	runner := newMockCLIRunner(t)
	// With the v2 account list unavailable, the sweep must fall back rather
	// than reporting a platform with no accounts.
	runner.ArmFault("/v2/core/accounts=400")
	out := runner.Run("investigate", "connect-readiness", "--namespace", "v1", "--limit", "5")
	assertContains(t, out, `(v1)`, `acct_mock_connected`)
}
