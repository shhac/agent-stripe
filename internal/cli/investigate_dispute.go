package cli

import (
	"fmt"
	"net/url"
)

var disputeResponseInvestigation = investigationSpec{
	use:   "dispute-response <dispute-id>",
	short: "Summarize dispute response status, evidence due date, and related payment objects",
	run:   investigator.disputeResponse,
}

func (i investigator) disputeResponse(disputeID string) error {
	if err := validateExpectedStripeID(disputeID, "dispute"); err != nil {
		return err
	}
	dispute, err := i.get("/v1/disputes/"+url.PathEscape(disputeID), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("dispute", dispute))
	if charge := i.followRef(dispute, "charge"); charge != nil {
		i.followRef(charge, "customer")
	}
	i.followRef(dispute, "payment_intent")
	details := mapAnyMap(dispute, "evidence_details")
	i.add(evidenceRecord{
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
	return nil
}

func disputeSeverity(status string) string {
	switch status {
	case "needs_response", "warning_needs_response":
		return "warning"
	default:
		return "info"
	}
}
