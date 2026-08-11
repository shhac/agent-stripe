package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateFraudReview(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "fraud-review <early-fraud-warning-id|charge-id|payment-intent-id>",
		Short: "Gather Radar early fraud warnings, disputes, refunds, and risk outcome",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.fraudReview(args[0])
			})
		},
	}
}

func (i investigator) fraudReview(id string) error {
	before := i.count()
	if err := validateAllowedStripeID(id, "early_fraud_warning", "charge", "payment_intent"); err != nil {
		return err
	}
	paymentID := id
	if strings.HasPrefix(id, "issfr_") {
		efw, err := i.get("/v1/radar/early_fraud_warnings/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("early_fraud_warning", efw))
		// Dispatch on the payment the warning points at, but keep id as what the
		// caller passed so the not-found message names their input.
		paymentID = firstNonEmpty(idFromValue(efw["charge"]), idFromValue(efw["payment_intent"]))
	}
	if strings.HasPrefix(paymentID, "pi_") {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(paymentID), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("payment_intent", pi))
		if charge, err := i.latestChargeForPaymentIntent(pi); err == nil && charge != nil {
			i.fraudReviewForCharge(charge)
		}
	} else if strings.HasPrefix(paymentID, "ch_") {
		charge, err := i.get("/v1/charges/"+url.PathEscape(paymentID), url.Values{})
		if err != nil {
			return err
		}
		i.fraudReviewForCharge(charge)
	}
	if i.count() == before {
		i.add(evidenceRecord{Type: "finding", Severity: "warning", Summary: "No fraud-review evidence found for " + id + "."})
	}
	i.add(evidenceRecord{Type: "finding", Severity: "warning", Summary: "Fraud review evidence gathered. Check charge outcome risk fields, early fraud warnings, disputes, and refunds before deciding customer action."})
	return nil
}

func (i investigator) fraudReviewForCharge(charge map[string]any) {
	i.add(entityRecord("charge", charge))
	if warnings := i.listRelated("early fraud warnings", "/v1/radar/early_fraud_warnings", valuesWithLimit(10, "charge", mapString(charge, "id"))); warnings != nil {
		i.addList("early_fraud_warning", warnings)
	}
	i.relatedDisputesAndRefunds(nil, charge)
}
