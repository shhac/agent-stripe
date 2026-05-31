package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateDisputeResponse(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "dispute-response <dispute-id>",
		Short: "Summarize dispute response status, evidence due date, and related payment objects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.disputeResponse(args[0])
			})
		},
	}
}

func (i investigator) disputeResponse(disputeID string) ([]evidenceRecord, error) {
	if err := validateExpectedStripeID(disputeID, "dispute"); err != nil {
		return nil, err
	}
	dispute, err := i.get("/v1/disputes/"+url.PathEscape(disputeID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord("dispute", dispute)}
	if chargeID := idFromValue(dispute["charge"]); chargeID != "" {
		charge, err := i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{})
		if err == nil {
			records = append(records, entityRecord("charge", charge))
			if customerID := idFromValue(charge["customer"]); customerID != "" {
				customer, err := i.get("/v1/customers/"+url.PathEscape(customerID), url.Values{})
				if err == nil {
					records = append(records, entityRecord("customer", customer))
				}
			}
		}
	}
	if piID := idFromValue(dispute["payment_intent"]); piID != "" {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{})
		if err == nil {
			records = append(records, entityRecord("payment_intent", pi))
		}
	}
	details := mapAnyMap(dispute, "evidence_details")
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: disputeSeverity(mapString(dispute, "status")),
		Summary: fmt.Sprintf("Dispute %s is %s for reason %s; evidence due_by=%v.",
			mapString(dispute, "id"), mapString(dispute, "status"), mapString(dispute, "reason"), details["due_by"]),
		Data: map[string]any{
			"status":           mapString(dispute, "status"),
			"reason":           mapString(dispute, "reason"),
			"evidence_details": details,
		},
	})
	return records, nil
}

func disputeSeverity(status string) string {
	switch status {
	case "needs_response", "warning_needs_response":
		return "warning"
	default:
		return "info"
	}
}
