package main

import "testing"

func TestCLIAccountsV2ListAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "list", "--limit", "2")
	runner.AssertContains(out,
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
	runner.AssertContains(out, `"id":"acct_mock_v2_recipient"`)
	assertNotContains(t, out, "acct_mock_v2_active", "acct_mock_v2_restricted")
}

func TestCLIAccountsV2GetRequestsEveryIncludeByDefault(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "get", "acct_mock_v2_restricted")
	runner.AssertContains(out,
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
	runner.AssertContains(out, `"requirements":{`, `"identity":null`, `"configuration":null`)
}

func TestCLIAccountsV2PersonsRedactPersonalData(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("accounts-v2", "persons", "get", "acct_mock_v2_restricted", "person_mock_representative")
	runner.AssertContains(out,
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
	runner.AssertContains(out,
		`"object":"v2.core.event"`,
		`"type":"v2.core.account[configuration.merchant].capability_status_updated"`,
		`"related_object":{"id":"acct_mock_v2_restricted"`,
	)
	assertNotContains(t, out, "acct_mock_v2_active")
}

func TestCLIInvestigateAccountHealthReadsV2(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "account-health", "acct_mock_v2_restricted")
	runner.AssertContains(out,
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
	runner.AssertContains(out,
		`is not an Accounts v2 account (v1_account_instead_of_v2_account)`,
		`Connect v1 account acct_mock_connected charges_enabled=true payouts_enabled=false`,
		`"namespace":"v1"`,
	)
}

func TestCLIInvestigateAccountHealthHealthyV2AccountIsInfo(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "account-health", "acct_mock_v2_active")
	runner.AssertContains(out, `All capabilities are active`, `"severity":"info"`)
}

func TestCLIInvestigateAccountEventsAgainstMockStripe(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "account-events", "acct_mock_v2_restricted")
	runner.AssertContains(out,
		`"object":"v2.core.event"`,
		`v2 event(s) between`,
		`current, not point-in-time`,
		`agent-stripe investigate account-health acct_mock_v2_restricted`,
	)
}

func TestCLIInvestigateResolveNamesAccountNamespace(t *testing.T) {
	runner := newMockCLIRunner(t)
	v2 := runner.Run("investigate", "resolve", "acct_mock_v2_active")
	runner.AssertContains(v2, `Resolved acct_mock_v2_active as an Accounts v2 account`, `"namespace":"v2"`)

	v1 := runner.Run("investigate", "resolve", "acct_mock_connected")
	runner.AssertContains(v1, `Resolved acct_mock_connected as a Connect v1 account`, `--namespace v1`)
}

func TestCLIInvestigateWebhookEventHandlesV2ThinEvent(t *testing.T) {
	runner := newMockCLIRunner(t)
	out := runner.Run("investigate", "webhook-event", "evt_test_mock_requirements_updated")
	runner.AssertContains(out,
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
