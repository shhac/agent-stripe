package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateAccountHealth(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "account-health <account-id>",
		Short: "Explain connected account requirements, capabilities, and money movement blockers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.accountHealth(args[0])
			})
		},
	}
}

func (i investigator) accountHealth(accountID string) ([]evidenceRecord, error) {
	if err := validateExpectedStripeID(accountID, "account"); err != nil {
		return nil, err
	}
	account, err := i.get("/v1/accounts/"+url.PathEscape(accountID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := i.appendEvidence(nil, entityRecord("account", account), accountHealthFinding(account))
	if transfers, err := i.list("/v1/transfers", valuesWithLimit(5, "destination", accountID)); err == nil {
		records = i.appendListRecords(records, "transfer", transfers)
	}
	return records, nil
}

func accountHealthFinding(account map[string]any) evidenceRecord {
	severity := "info"
	blockers := []string{}
	if !mapBool(account, "charges_enabled") {
		blockers = append(blockers, "charges disabled")
	}
	if !mapBool(account, "payouts_enabled") {
		blockers = append(blockers, "payouts disabled")
	}
	requirements := mapAnyMap(account, "requirements")
	if len(requirements) > 0 {
		blockers = append(blockers, "requirements present")
	}
	if len(blockers) > 0 {
		severity = "warning"
	}
	summary := fmt.Sprintf("Account %s charges_enabled=%t payouts_enabled=%t.", mapString(account, "id"), mapBool(account, "charges_enabled"), mapBool(account, "payouts_enabled"))
	if len(blockers) > 0 {
		summary += " Blockers: " + strings.Join(blockers, ", ") + "."
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"account":             mapString(account, "id"),
			"charges_enabled":     mapBool(account, "charges_enabled"),
			"payouts_enabled":     mapBool(account, "payouts_enabled"),
			"requirements":        requirements,
			"capabilities":        mapAnyMap(account, "capabilities"),
			"future_requirements": mapAnyMap(account, "future_requirements"),
		},
	}
}
