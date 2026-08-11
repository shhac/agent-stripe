package cli

import (
	"net/url"
	"strings"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

var incomingPaymentInvestigation = investigationSpec{
	use:   "incoming-payment <payment-intent-id|charge-id|invoice-id>",
	short: "Explain what happened to a customer payment to you",
	run:   investigator.incomingPayment,
}

func (i investigator) incomingPayment(id string) error {
	if err := validateAllowedStripeID(id, "invoice", "charge", "payment_intent"); err != nil {
		return err
	}
	switch {
	case strings.HasPrefix(id, "in_"):
		return i.invoicePayment(id)
	case strings.HasPrefix(id, "ch_"):
		charge, err := i.get("/v1/charges/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		return i.paymentIncidentFromCharge(charge)
	default:
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		return i.paymentIncidentFromPI(pi)
	}
}

func (i investigator) paymentIncidentFromPI(pi map[string]any) error {
	charge, err := i.latestChargeForPaymentIntent(pi)
	if err != nil {
		return err
	}
	i.relatedDisputesAndRefunds(pi, charge)
	i.add(finding(severityForPayment(pi, charge), paymentFailureSummary(pi, charge)))
	return nil
}

func (i investigator) paymentIncidentFromCharge(charge map[string]any) error {
	i.followRef(charge, "payment_intent")
	i.relatedDisputesAndRefunds(nil, charge)
	i.add(finding(severityForPayment(nil, charge), paymentFailureSummary(nil, charge)))
	return nil
}

func (i investigator) relatedDisputesAndRefunds(pi, charge map[string]any) {
	params := url.Values{}
	if charge != nil {
		shared.AddString(params, "charge", mapString(charge, "id"))
	}
	if pi != nil {
		shared.AddString(params, "payment_intent", mapString(pi, "id"))
	}
	if len(params) == 0 {
		return
	}
	i.listRelated("disputes", "/v1/disputes", params)
	i.listRelated("refunds", "/v1/refunds", params)
}

func (i investigator) latestChargeForPaymentIntent(pi map[string]any) (map[string]any, error) {
	if pi == nil {
		return nil, nil
	}
	if charge, ok := pi["latest_charge"].(map[string]any); ok {
		return charge, nil
	}
	chargeID := idFromValue(pi["latest_charge"])
	if chargeID != "" {
		return i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{})
	}
	charges, err := i.list("/v1/charges", url.Values{"payment_intent": []string{mapString(pi, "id")}, "limit": []string{"1"}})
	if err != nil || len(charges) == 0 {
		return nil, err
	}
	return charges[0], nil
}
