package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateRefund(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "refund <refund-id|charge-id|payment-intent-id>",
		Short: "Explain refund state from a refund or its original payment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.refund(args[0])
			})
		},
	}
}

func (i investigator) refund(id string) error {
	if err := validateAllowedStripeID(id, "refund", "charge", "payment_intent"); err != nil {
		return err
	}
	if strings.HasPrefix(id, "re_") {
		return i.refundStatus(id)
	}
	if err := i.incomingPayment(id); err != nil {
		return err
	}
	params := url.Values{"limit": []string{"10"}}
	if strings.HasPrefix(id, "ch_") {
		params.Set("charge", id)
	} else {
		params.Set("payment_intent", id)
	}
	refunds, err := i.list("/v1/refunds", params)
	if err != nil {
		return err
	}
	i.addList("refund", refunds)
	if len(refunds) == 0 {
		i.add(evidenceRecord{Type: "finding", Severity: "warning", Summary: "No refunds found for " + id + "."})
		return nil
	}
	i.add(evidenceRecord{Type: "finding", Severity: "info", Summary: "Refund evidence gathered for original payment " + id + "."})
	return nil
}
