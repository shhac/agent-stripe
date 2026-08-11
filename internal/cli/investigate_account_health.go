package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateAccountHealth(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var namespace string
	cmd := &cobra.Command{
		Use:   "account-health <account-id>",
		Short: "Explain connected account requirements, capabilities, and money movement blockers",
		Long: "Explains why a connected account can or cannot take payments and get paid.\n" +
			"Both Connect v1 and Accounts v2 use the acct_ prefix, so --namespace auto (the default)\n" +
			"reads the v2 account first and falls back to v1 when Stripe says the ID is not a v2 account.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateNamespace(namespace); err != nil {
				return err
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.accountHealth(args[0], namespace)
			})
		},
	}
	cmd.Flags().StringVar(&namespace, "namespace", namespaceAuto, "Account namespace to read: auto, v1, or v2")
	return cmd
}

func (i investigator) accountHealth(accountID, namespace string) error {
	if err := validateExpectedStripeID(accountID, "account"); err != nil {
		return err
	}
	if namespace == "" {
		namespace = namespaceAuto
	}
	if namespace == namespaceV1 {
		return i.accountHealthV1(accountID, nil)
	}

	includes, err := v2AccountIncludeParams(nil)
	if err != nil {
		return err
	}
	account, err := i.get(v2AccountPath(accountID), includes)
	if err != nil {
		if namespace == namespaceV2 || !isNotV2AccountError(err) {
			return err
		}
		return i.accountHealthV1(accountID, []evidenceRecord{v2FallbackFinding(accountID, err)})
	}
	return i.accountHealthV2(accountID, account)
}

func (i investigator) accountHealthV1(accountID string, prefix []evidenceRecord) error {
	// The fallback note is emitted before the v1 read so a streaming reader
	// sees why the namespace changed before the v1-shaped account arrives.
	i.add(prefix...)
	account, err := i.get("/v1/accounts/"+url.PathEscape(accountID), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("account", account), accountHealthFinding(account))
	i.addAccountTransfers(accountID)
	return nil
}

func (i investigator) accountHealthV2(accountID string, account map[string]any) error {
	i.add(entityRecord(objectV2Account, account))
	persons, err := i.listV2(v2AccountPersonsPath(accountID), url.Values{})
	if err == nil {
		for _, person := range persons {
			i.add(entityRecord(objectV2Person, person))
		}
	}
	i.add(v2AccountHealthFinding(account, len(persons)))
	i.addAccountTransfers(accountID)
	return nil
}

// addAccountTransfers is shared by both namespaces: transfers live in /v1 for
// v1 and v2 accounts alike, and a v2 account ID is accepted there.
func (i investigator) addAccountTransfers(accountID string) {
	transfers, err := i.list("/v1/transfers", valuesWithLimit(5, "destination", accountID))
	if err != nil {
		i.add(relatedWarning("transfers to the account", err))
		return
	}
	i.addList("transfer", transfers)
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
	summary := fmt.Sprintf("Connect v1 account %s charges_enabled=%t payouts_enabled=%t.", mapString(account, "id"), mapBool(account, "charges_enabled"), mapBool(account, "payouts_enabled"))
	if len(blockers) > 0 {
		summary += " Blockers: " + strings.Join(blockers, ", ") + "."
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"namespace":           namespaceV1,
			"account":             mapString(account, "id"),
			"charges_enabled":     mapBool(account, "charges_enabled"),
			"payouts_enabled":     mapBool(account, "payouts_enabled"),
			"requirements":        requirements,
			"capabilities":        mapAnyMap(account, "capabilities"),
			"future_requirements": mapAnyMap(account, "future_requirements"),
		},
	}
}

// v2AccountHealthFinding reads only v2 signals. A v2.core.account has no
// charges_enabled/payouts_enabled — enablement is per-capability, and what is
// missing is a requirement entry rather than a field name in an array.
func v2AccountHealthFinding(account map[string]any, personCount int) evidenceRecord {
	capabilities := v2AccountCapabilities(account)
	capabilityRollup := summarizeV2Capabilities(capabilities)
	requirements := v2AccountRequirements(account, "requirements")
	requirementRollup := summarizeV2Requirements(requirements)

	severity := "info"
	if len(capabilityRollup.Restricted) > 0 || len(requirementRollup.Blocking) > 0 || mapBool(account, "closed") {
		severity = "warning"
	}

	summary := v2AccountHealthSummary(account, capabilityRollup, requirementRollup)
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Command:  "agent-stripe investigate account-events " + mapString(account, "id"),
		Data: map[string]any{
			"namespace":               namespaceV2,
			"account":                 mapString(account, "id"),
			"closed":                  mapBool(account, "closed"),
			"dashboard":               mapString(account, "dashboard"),
			"applied_configurations":  v2AppliedConfigurations(account),
			"capabilities":            v2CapabilityData(capabilities),
			"capabilities_not_active": capabilityRollup.Restricted,
			"requirements":            v2RequirementData(requirements),
			"requirements_summary":    v2RequirementSummary(account, "requirements"),
			"future_requirements":     v2RequirementData(v2AccountRequirements(account, "future_requirements")),
			"person_count":            personCount,
		},
	}
}

func v2AccountHealthSummary(account map[string]any, capabilities v2CapabilityRollup, requirements v2RequirementRollup) string {
	id := mapString(account, "id")
	configurations := v2AppliedConfigurations(account)
	summary := fmt.Sprintf("Accounts v2 account %s has configurations [%s]", id, strings.Join(configurations, ", "))
	if dashboard := mapString(account, "dashboard"); dashboard != "" {
		summary += " and dashboard " + dashboard
	}
	summary += "."
	if mapBool(account, "closed") {
		summary += " The account is closed."
	}
	if len(capabilities.Restricted) == 0 {
		summary += " All capabilities are active."
	} else {
		summary += " Capabilities not active: " + joinAndTruncate(capabilities.Restricted, 6) + "."
	}
	return summary + v2RequirementSentence(requirements)
}

// v2RequirementSentence reports only what the integrator can act on in its
// headline count and deadline breakdown, then mentions Stripe-side entries
// separately — they are outstanding, but nobody can supply anything for them.
func v2RequirementSentence(requirements v2RequirementRollup) string {
	if len(requirements.Blocking) == 0 {
		if requirements.Counts["eventually_due"] > 0 {
			return fmt.Sprintf(" No requirement needs action now; %d eventually due.", requirements.Counts["eventually_due"])
		}
		if requirements.AwaitingStripe > 0 {
			return fmt.Sprintf(" Nothing is outstanding from you; %d requirement(s) are awaiting Stripe.", requirements.AwaitingStripe)
		}
		return " No outstanding requirements need action."
	}
	sentence := fmt.Sprintf(" %d requirement(s) need action from you (%d past due, %d currently due): %s.",
		len(requirements.Blocking),
		requirements.UserCounts["past_due"],
		requirements.UserCounts["currently_due"],
		joinAndTruncate(requirements.Blocking, 6))
	if requirements.AwaitingStripe > 0 {
		sentence += fmt.Sprintf(" A further %d requirement(s) are awaiting Stripe, not you.", requirements.AwaitingStripe)
	}
	return sentence
}
