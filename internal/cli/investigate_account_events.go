package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateAccountEvents(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var limit int
	var eventTypes []string
	cmd := &cobra.Command{
		Use:   "account-events <account-id>",
		Short: "Explain what recently changed on an Accounts v2 account",
		Long: "Reads /v2/core/events for the account. Accounts v2 capability and requirement\n" +
			"transitions are emitted only as v2 thin events, so /v1/events never shows them.\n" +
			"Thin events carry no snapshot: the account state emitted alongside is current state.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.accountEvents(args[0], limit, eventTypes)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum events to read")
	cmd.Flags().StringArrayVar(&eventTypes, "type", nil, "Event type filter; repeatable")
	return cmd
}

func (i investigator) accountEvents(accountID string, limit int, eventTypes []string) ([]evidenceRecord, error) {
	if err := validateExpectedStripeID(accountID, "account"); err != nil {
		return nil, err
	}
	var records []evidenceRecord
	includes, err := v2AccountIncludeParams(nil)
	if err != nil {
		return nil, err
	}
	account, accountErr := i.get(v2AccountPath(accountID), includes)
	if accountErr == nil {
		records = i.appendEvidence(records, entityRecord(objectV2Account, account))
	} else if isNotV2AccountError(accountErr) {
		records = i.appendEvidence(records, v2FallbackFinding(accountID, accountErr))
	} else {
		return nil, accountErr
	}

	params := v2EventListParams(limit, "", accountID, eventTypes, "", "")
	events, err := i.listV2("/v2/core/events", params)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		records = i.appendEvidence(records, entityRecord(objectV2Event, event))
	}
	return i.appendEvidence(records, accountEventsFinding(accountID, events, account)), nil
}

func accountEventsFinding(accountID string, events []map[string]any, account map[string]any) evidenceRecord {
	if len(events) == 0 {
		return evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  "No v2 core events for account " + accountID + " in the last 30 days. Connect v1 accounts never emit them; check 'agent-stripe events list' for v1 activity.",
			Command:  "agent-stripe investigate account-health " + accountID,
			Data:     map[string]any{"account": accountID, "event_count": 0},
		}
	}

	counts := map[string]int{}
	for _, event := range events {
		counts[mapString(event, "type")]++
	}
	types := make([]string, 0, len(counts))
	for eventType := range counts {
		types = append(types, eventType)
	}
	sort.Strings(types)

	latest := mapString(events[0], "created")
	earliest := mapString(events[len(events)-1], "created")
	summary := fmt.Sprintf("Account %s has %d v2 event(s) between %s and %s: %s.",
		accountID, len(events), earliest, latest, strings.Join(types, ", "))
	severity := "info"
	if capabilityEventCount(counts) > 0 || counts["v2.core.account[requirements].updated"] > 0 {
		severity = "warning"
		summary += " Capability or requirement state changed; the account state shown here is current, not point-in-time."
	}
	data := map[string]any{
		"account":      accountID,
		"event_count":  len(events),
		"event_types":  counts,
		"latest_event": latest,
	}
	if account != nil {
		data["current_capabilities_not_active"] = summarizeV2Capabilities(v2AccountCapabilities(account)).Restricted
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Command:  "agent-stripe investigate account-health " + accountID,
		Data:     data,
	}
}

func capabilityEventCount(counts map[string]int) int {
	total := 0
	for eventType, count := range counts {
		if strings.HasSuffix(eventType, "].capability_status_updated") {
			total += count
		}
	}
	return total
}
