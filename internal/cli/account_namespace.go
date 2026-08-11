package cli

import (
	"github.com/shhac/agent-stripe/internal/api"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

// The account-namespace vocabulary. Connect v1 and Accounts v2 share the acct_
// prefix, so deciding which model an ID belongs to is a cross-cutting concern:
// account-health, connect-readiness, account-events, resolve, webhook-event and
// outgoing-payment all depend on it. It lived in the account-health workflow
// file, which made it undiscoverable to the next namespace-aware command.

const (
	namespaceAuto = "auto"
	namespaceV1   = "v1"
	namespaceV2   = "v2"
)

func validateNamespace(namespace string) error {
	switch namespace {
	case namespaceAuto, namespaceV1, namespaceV2:
		return nil
	}
	return agenterrors.Newf(agenterrors.FixableByAgent, "unknown --namespace value %q", namespace).
		WithHint("Use auto (default), v1 for Connect v1 accounts, or v2 for Accounts v2")
}

// isNotV2AccountError matches the documented ways Stripe says "this ID or this
// platform is not on Accounts v2". Anything else (auth, rate limit, network)
// must surface rather than being masked by a v1 retry.
func isNotV2AccountError(err error) bool {
	switch api.ErrorCode(err) {
	case "v1_account_instead_of_v2_account",
		"account_not_yet_compatible_with_v2",
		"accounts_v2_access_blocked",
		"non_connect_platform_accounts_v2_access_blocked":
		return true
	}
	return api.ErrorStatus(err) == 404
}

func v2FallbackFinding(accountID string, err error) evidenceRecord {
	reason := api.ErrorCode(err)
	if reason == "" {
		reason = "not found in the v2 namespace"
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  "Account " + accountID + " is not an Accounts v2 account (" + reason + "); read from Connect v1 instead.",
		Command:  "agent-stripe accounts get " + accountID,
		Data:     map[string]any{"namespace": namespaceV1, "v2_error_code": api.ErrorCode(err)},
	}
}
