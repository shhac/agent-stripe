package cli

import (
	"fmt"
	"net/url"
	"strings"
)

var refundSettlementInvestigation = investigationSpec{
	use:   "refund-settlement <refund-id|charge-id>",
	short: "Explain where a refund is between Stripe and the customer's bank",
	long: "Answers \"we refunded them but nothing has arrived\". Stripe issuing a refund and\n" +
		"the bank posting it are different events; this reports the refund's own status,\n" +
		"the destination the money is going to, the network reference the customer's bank\n" +
		"can be asked about, and the ledger entry that moved the money.",
	run: investigator.refundSettlement,
}

func (i investigator) refundSettlement(id string) error {
	if err := validateAllowedStripeID(id, "refund", "charge"); err != nil {
		return err
	}
	refunds, err := i.refundsForSettlement(id)
	if err != nil {
		return err
	}
	if len(refunds) == 0 {
		i.add(evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "No refunds found for " + id + ". If the customer expected one, none was issued.",
			Data:     map[string]any{"input": id},
		})
		return nil
	}
	for _, refund := range refunds {
		i.add(entityRecord("refund", refund))
		i.followRef(refund, "balance_transaction")
		charge := i.followRef(refund, "charge")
		i.add(refundSettlementFinding(refund, charge))
	}
	return nil
}

func (i investigator) refundsForSettlement(id string) ([]map[string]any, error) {
	if strings.HasPrefix(id, "re_") {
		refund, err := i.get("/v1/refunds/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []map[string]any{refund}, nil
	}
	return i.listRelated("refunds", "/v1/refunds", valuesWithLimit(10, "charge", id)), nil
}

// refundSettlementFinding separates "Stripe sent it" from "the bank posted it",
// which is the distinction the customer is actually asking about.
func refundSettlementFinding(refund, charge map[string]any) evidenceRecord {
	amount, _ := mapInt64(refund, "amount")
	status := mapString(refund, "status")
	destination := mapAnyMap(refund, "destination_details")
	card := mapAnyMap(destination, "card")

	data := map[string]any{
		"refund":              mapString(refund, "id"),
		"status":              status,
		"amount":              amount,
		"currency":            mapString(refund, "currency"),
		"charge":              idFromValue(refund["charge"]),
		"balance_transaction": idFromValue(refund["balance_transaction"]),
		"destination_type":    mapString(destination, "type"),
	}
	if reason := mapString(refund, "failure_reason"); reason != "" {
		data["failure_reason"] = reason
	}
	if charge != nil {
		data["card_last4"] = cardLast4(charge)
	}

	summary := fmt.Sprintf("Refund %s of %d %s is %s.", mapString(refund, "id"), amount, mapString(refund, "currency"), status)
	severity := "info"
	switch status {
	case "failed", "canceled":
		severity = "warning"
		summary += " The money did not leave Stripe; it was returned to your balance."
		if reason := mapString(refund, "failure_reason"); reason != "" {
			summary += " Reason: " + reason + "."
		}
	case "pending":
		severity = "warning"
		summary += " Stripe has not finished sending it, so nothing will appear on the customer's statement yet."
	case "succeeded":
		summary += " Stripe has sent it; posting to the customer's account is up to their bank and typically takes several business days."
	}

	// The acquirer reference is what a customer's bank can actually trace.
	if reference := mapString(card, "reference"); reference != "" {
		data["acquirer_reference"] = reference
		data["reference_status"] = mapString(card, "reference_status")
		summary += fmt.Sprintf(" Give the customer's bank reference %s.", reference)
	} else if status == "succeeded" {
		summary += " No acquirer reference is available yet, so the bank has nothing to trace by."
		if arrival := mapString(card, "reference_status"); arrival != "" {
			data["reference_status"] = arrival
		}
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Command:  "agent-stripe investigate ledger " + mapString(refund, "id"),
		Data:     data,
	}
}
