package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateRefund(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "refund <refund-id|charge-id|payment-intent-id>",
		Short: "Explain refund state from a refund or its original payment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.refund(args[0])
			})
		},
	}
}

func (i investigator) refund(id string) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "refund", "charge", "payment_intent"); err != nil {
		return nil, err
	}
	if strings.HasPrefix(id, "re_") {
		return i.refundStatus(id)
	}
	records, err := i.incomingPayment(id)
	if err != nil {
		return nil, err
	}
	params := url.Values{"limit": []string{"10"}}
	if strings.HasPrefix(id, "ch_") {
		params.Set("charge", id)
	} else {
		params.Set("payment_intent", id)
	}
	refunds, err := i.list("/v1/refunds", params)
	if err != nil {
		return nil, err
	}
	records = i.appendListRecords(records, "refund", refunds)
	if len(refunds) == 0 {
		records = i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No refunds found for " + id + "."})
		return records, nil
	}
	records = i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "Refund evidence gathered for original payment " + id + "."})
	return records, nil
}
