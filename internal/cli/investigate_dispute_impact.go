package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateDisputeImpact(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "dispute-impact <dispute-id|charge-id|customer-id>",
		Short: "Summarize dispute exposure and related payment/refund evidence",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.disputeImpact(args[0], limit)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum charges to inspect for customer input")
	return cmd
}

func (i investigator) disputeImpact(id string, limit int) error {
	if err := validateAllowedStripeID(id, "dispute", "charge", "customer"); err != nil {
		return err
	}
	disputes := []map[string]any{}
	switch {
	case strings.HasPrefix(id, "dp_"):
		dispute, err := i.get("/v1/disputes/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		disputes = append(disputes, dispute)
	case strings.HasPrefix(id, "ch_"):
		found, err := i.list("/v1/disputes", valuesWithLimit(10, "charge", id))
		if err != nil {
			return err
		}
		disputes = found
	default:
		charges, err := i.list("/v1/charges", valuesWithLimit(limit, "customer", id))
		if err != nil {
			return err
		}
		for _, charge := range charges {
			found, _ := i.list("/v1/disputes", valuesWithLimit(10, "charge", mapString(charge, "id")))
			disputes = append(disputes, found...)
		}
	}
	for _, dispute := range disputes {
		i.disputeImpactRecords(dispute)
	}
	if len(disputes) == 0 {
		i.add(evidenceRecord{Type: "finding", Severity: "info", Summary: "No dispute exposure found for " + id + "."})
	}
	return nil
}

func (i investigator) disputeImpactRecords(dispute map[string]any) {
	i.add(entityRecord("dispute", dispute), disputeImpactFinding(dispute))
	if chargeID := idFromValue(dispute["charge"]); chargeID != "" {
		if charge, err := i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{}); err == nil {
			i.add(entityRecord("charge", charge))
			if refunds, err := i.list("/v1/refunds", valuesWithLimit(5, "charge", chargeID)); err == nil {
				i.addList("refund", refunds)
			}
		}
	}
	if piID := idFromValue(dispute["payment_intent"]); piID != "" {
		if pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{}); err == nil {
			i.add(entityRecord("payment_intent", pi))
		}
	}
}

func disputeImpactFinding(dispute map[string]any) evidenceRecord {
	severity := "info"
	status := mapString(dispute, "status")
	if status == "needs_response" || status == "under_review" || status == "warning_needs_response" {
		severity = "warning"
	}
	details := mapAnyMap(dispute, "evidence_details")
	summary := fmt.Sprintf("Dispute %s status=%s reason=%s amount=%s.", mapString(dispute, "id"), status, mapString(dispute, "reason"), formatAmount(dispute))
	if dueBy, ok := mapInt64(details, "due_by"); ok && dueBy > 0 {
		summary += fmt.Sprintf(" Evidence is due by Unix time %d.", dueBy)
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: summary, Data: map[string]any{
		"dispute":          mapString(dispute, "id"),
		"charge":           idFromValue(dispute["charge"]),
		"payment_intent":   idFromValue(dispute["payment_intent"]),
		"amount":           mapValue(dispute, "amount"),
		"currency":         mapString(dispute, "currency"),
		"reason":           mapString(dispute, "reason"),
		"status":           status,
		"evidence_details": details,
	}}
}
