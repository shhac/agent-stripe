package cli

import (
	"net/url"
	"strings"
)

var refundInvestigation = investigationSpec{
	use:   "refund <refund-id|charge-id|payment-intent-id>",
	short: "Explain refund state from a refund or its original payment",
	run:   investigator.refund,
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
		i.add(finding("warning", "No refunds found for "+id+"."))
		return nil
	}
	i.add(finding("info", "Refund evidence gathered for original payment "+id+"."))
	return nil
}
