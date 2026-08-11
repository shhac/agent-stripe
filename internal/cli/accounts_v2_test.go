package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

const v2RestrictedAccountJSON = `{
  "id": "acct_v2_restricted",
  "object": "v2.core.account",
  "applied_configurations": ["customer", "merchant"],
  "closed": false,
  "dashboard": "express",
  "configuration": {
    "customer": {"capabilities": {"automatic_indirect_tax": {"status": "active", "status_details": []}}},
    "merchant": {
      "capabilities": {
        "card_payments": {"status": "restricted", "status_details": [{"code": "requirements_past_due", "resolution": "provide_info"}]},
        "stripe_balance": {"payouts": {"status": "restricted", "status_details": [{"code": "requirements_past_due"}]}}
      }
    },
    "recipient": null
  },
  "requirements": {
    "collector": "stripe",
    "entries": [
      {
        "id": "reqent_1",
        "description": "identity.business_details.registered_name",
        "awaiting_action_from": "user",
        "minimum_deadline": {"status": "past_due"},
        "errors": [],
        "impact": {"restricts_capabilities": [{"capability": "card_payments", "configuration": "merchant", "deadline": {"status": "past_due"}}]}
      },
      {
        "id": "reqent_2",
        "description": "identity.business_details.id_numbers.us_ein",
        "awaiting_action_from": "stripe",
        "minimum_deadline": {"status": "currently_due"},
        "errors": [{"code": "verification_document_name_mismatch", "description": "mismatch"}],
        "impact": {"restricts_capabilities": []}
      }
    ],
    "summary": {"minimum_deadline": {"status": "past_due", "time": "2026-07-30T00:00:00.000Z"}}
  }
}`

func decodeAccount(t *testing.T, raw string) map[string]any {
	t.Helper()
	var account map[string]any
	if err := json.Unmarshal([]byte(raw), &account); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	return account
}

func TestV2AccountCapabilitiesFlattensNestedLeaves(t *testing.T) {
	capabilities := v2AccountCapabilities(decodeAccount(t, v2RestrictedAccountJSON))

	got := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		got = append(got, capability.QualifiedName()+"="+capability.Status)
	}
	want := []string{
		"customer.automatic_indirect_tax=active",
		"merchant.card_payments=restricted",
		"merchant.stripe_balance.payouts=restricted",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("capabilities = %v, want %v", got, want)
	}
	if codes := capabilities[1].StatusCodes; len(codes) != 1 || codes[0] != "requirements_past_due" {
		t.Fatalf("status codes = %v", codes)
	}
}

func TestV2AccountRequirementsCarryImpactAndOwner(t *testing.T) {
	requirements := v2AccountRequirements(decodeAccount(t, v2RestrictedAccountJSON), "requirements")
	if len(requirements) != 2 {
		t.Fatalf("len(requirements) = %d, want 2", len(requirements))
	}

	first := requirements[0]
	if !first.BlocksUser() {
		t.Fatalf("past-due user requirement should block: %#v", first)
	}
	if len(first.Restricts) != 1 || first.Restricts[0] != "merchant.card_payments" {
		t.Fatalf("Restricts = %v, want the qualified capability", first.Restricts)
	}

	second := requirements[1]
	if second.BlocksUser() {
		t.Fatalf("a requirement awaiting Stripe is not the integrator's blocker: %#v", second)
	}
	if len(second.ErrorCodes) != 1 || second.ErrorCodes[0] != "verification_document_name_mismatch" {
		t.Fatalf("ErrorCodes = %v", second.ErrorCodes)
	}
}

func TestV2AccountHealthFindingUsesV2SignalsOnly(t *testing.T) {
	account := decodeAccount(t, v2RestrictedAccountJSON)
	finding := v2AccountHealthFinding(account, 2)

	if finding.Severity != "warning" {
		t.Fatalf("Severity = %q, want warning", finding.Severity)
	}
	for _, banned := range []string{"charges_enabled", "payouts_enabled", "details_submitted"} {
		if strings.Contains(finding.Summary, banned) {
			t.Fatalf("summary mentions v1-only field %q: %s", banned, finding.Summary)
		}
		if _, ok := finding.Data[banned]; ok {
			t.Fatalf("finding data carries v1-only field %q", banned)
		}
	}
	if finding.Data["namespace"] != namespaceV2 {
		t.Fatalf("namespace = %v, want v2", finding.Data["namespace"])
	}
	for _, want := range []string{
		"merchant.card_payments",
		"identity.business_details.registered_name",
		"1 requirement(s) need action from you",
		"A further 1 requirement(s) are awaiting Stripe",
	} {
		if !strings.Contains(finding.Summary, want) {
			t.Fatalf("summary missing %q: %s", want, finding.Summary)
		}
	}
	if count, _ := finding.Data["person_count"].(int); count != 2 {
		t.Fatalf("person_count = %v, want 2", finding.Data["person_count"])
	}
}

func TestV2AccountHealthSummaryCountsOnlyUserOwnedRequirements(t *testing.T) {
	// The fixture has two currently-due-or-worse entries, but only one is the
	// integrator's: the EIN entry is awaiting Stripe. Reporting the all-entries
	// deadline counts next to the user-owned headline told the reader to go
	// collect something nobody can supply.
	finding := v2AccountHealthFinding(decodeAccount(t, v2RestrictedAccountJSON), 0)

	if !strings.Contains(finding.Summary, "1 requirement(s) need action from you (1 past due, 0 currently due)") {
		t.Fatalf("summary should count only user-owned deadlines: %s", finding.Summary)
	}
	if !strings.Contains(finding.Summary, "A further 1 requirement(s) are awaiting Stripe") {
		t.Fatalf("summary should account for the Stripe-side entry: %s", finding.Summary)
	}
}

func TestV2AccountHealthSummaryWhenOnlyStripeIsBlocking(t *testing.T) {
	account := decodeAccount(t, `{
      "id": "acct_v2_review",
      "object": "v2.core.account",
      "applied_configurations": ["merchant"],
      "configuration": {"merchant": {"capabilities": {"card_payments": {"status": "pending", "status_details": []}}}},
      "requirements": {"entries": [
        {"description": "identity.business_details.id_numbers.us_ein", "awaiting_action_from": "stripe", "minimum_deadline": {"status": "currently_due"}, "errors": []}
      ]}
    }`)
	finding := v2AccountHealthFinding(account, 0)

	if !strings.Contains(finding.Summary, "Nothing is outstanding from you; 1 requirement(s) are awaiting Stripe") {
		t.Fatalf("summary = %s", finding.Summary)
	}
	if finding.Severity != "warning" {
		t.Fatalf("Severity = %q, want warning while a capability is not active", finding.Severity)
	}
}

func TestV2AccountHealthFindingStaysInfoWhenHealthy(t *testing.T) {
	account := decodeAccount(t, `{
      "id": "acct_v2_ok",
      "object": "v2.core.account",
      "applied_configurations": ["merchant"],
      "dashboard": "full",
      "configuration": {"merchant": {"capabilities": {"card_payments": {"status": "active", "status_details": []}}}},
      "requirements": {"entries": [], "summary": {"minimum_deadline": {"status": "eventually_due"}}}
    }`)
	finding := v2AccountHealthFinding(account, 0)
	if finding.Severity != "info" {
		t.Fatalf("Severity = %q, want info", finding.Severity)
	}
	if !strings.Contains(finding.Summary, "All capabilities are active") {
		t.Fatalf("summary = %s", finding.Summary)
	}
}

func TestV2AccountListSummaryKeepsNavigationAndRollups(t *testing.T) {
	summary := v2AccountListSummary(decodeAccount(t, v2RestrictedAccountJSON))

	if summary["id"] != "acct_v2_restricted" || summary["object"] != objectV2Account {
		t.Fatalf("summary identity = %#v", summary)
	}
	capabilities, ok := summary["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities rollup missing: %#v", summary)
	}
	if capabilities["restricted_count"] != 2 || capabilities["active_count"] != 1 {
		t.Fatalf("capability counts = %#v", capabilities)
	}
	requirements, ok := summary["requirements"].(map[string]any)
	if !ok {
		t.Fatalf("requirements rollup missing: %#v", summary)
	}
	if requirements["past_due_count"] != 1 || requirements["awaiting_user_count"] != 1 {
		t.Fatalf("requirement counts = %#v", requirements)
	}
	if requirements["collector"] != "stripe" {
		t.Fatalf("collector = %#v", requirements["collector"])
	}
}

func TestV2AccountIncludeParamsDefaultsToEveryIncludeIndexed(t *testing.T) {
	params, err := v2AccountIncludeParams(nil)
	if err != nil {
		t.Fatalf("v2AccountIncludeParams() error = %v", err)
	}
	if len(params) != len(v2AccountIncludes) {
		t.Fatalf("len(params) = %d, want %d", len(params), len(v2AccountIncludes))
	}
	if got := params.Get("include[0]"); got != "configuration.customer" {
		t.Fatalf("include[0] = %q, want indexed encoding", got)
	}
	if got := params.Get("include[6]"); got != "requirements" {
		t.Fatalf("include[6] = %q", got)
	}
	if params.Get("include[]") != "" {
		t.Fatalf("v2 must not use the v1 include[] form: %v", params)
	}
}

func TestV2AccountIncludeParamsRejectsUnknownInclude(t *testing.T) {
	if _, err := v2AccountIncludeParams([]string{"capabilities"}); err == nil {
		t.Fatalf("v2AccountIncludeParams() error = nil, want a validation error")
	}
	if err := validateV2Configurations([]string{"storer"}); err == nil {
		t.Fatalf("validateV2Configurations() error = nil, want a validation error")
	}
	if err := validateV2Configurations([]string{"merchant", "recipient"}); err != nil {
		t.Fatalf("validateV2Configurations() error = %v", err)
	}
}

func TestV2PersonListSummaryKeepsRelationshipNotPII(t *testing.T) {
	person := decodeAccount(t, `{
      "id": "person_1", "object": "v2.core.account_person", "account": "acct_v2_restricted",
      "given_name": "Jenny", "surname": "Rosen", "email": "jenny@example.com",
      "date_of_birth": {"day": 1, "month": 2, "year": 1990},
      "id_numbers": [{"type": "us_ssn_last_4"}],
      "relationship": {"owner": true, "representative": true, "title": "CEO", "percent_ownership": "0.8"}
    }`)
	summary := v2PersonListSummary(person)

	for _, banned := range []string{"given_name", "surname", "email", "date_of_birth"} {
		if _, ok := summary[banned]; ok {
			t.Fatalf("person list summary should not carry %q: %#v", banned, summary)
		}
	}
	relationship, ok := summary["relationship"].(map[string]any)
	if !ok || relationship["representative"] != true || relationship["title"] != "CEO" {
		t.Fatalf("relationship = %#v", summary["relationship"])
	}
	types, ok := summary["id_number_types"].([]string)
	if !ok || len(types) != 1 || types[0] != "us_ssn_last_4" {
		t.Fatalf("id_number_types = %#v", summary["id_number_types"])
	}
}
