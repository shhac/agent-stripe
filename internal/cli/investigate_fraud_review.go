package cli

import (
	"net/url"
	"strings"
)

var fraudReviewInvestigation = investigationSpec{
	use:   "fraud-review <early-fraud-warning-id|charge-id|payment-intent-id>",
	short: "Gather Radar early fraud warnings, disputes, refunds, and risk outcome",
	run:   investigator.fraudReview,
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
		// Dispatch on the payment the warning points at, but keep id as what the
		// caller passed so the not-found message names their input.
		paymentID = firstNonEmpty(idFromValue(efw["charge"]), idFromValue(efw["payment_intent"]))
	}
	if strings.HasPrefix(paymentID, "pi_") {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(paymentID), url.Values{})
		if err != nil {
			return err
		}
		if charge := i.addLatestCharge(pi); charge != nil {
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
		i.add(finding("warning", "No fraud-review evidence found for "+id+"."))
	}
	i.add(finding("warning", "Fraud review evidence gathered. Check charge outcome risk fields, early fraud warnings, disputes, and refunds before deciding customer action."))
	return nil
}

func (i investigator) fraudReviewForCharge(charge map[string]any) {
	i.addRelatedList("early_fraud_warning", "/v1/radar/early_fraud_warnings", valuesWithLimit(10, "charge", mapString(charge, "id")))
	i.relatedDisputesAndRefunds(nil, charge)
}
