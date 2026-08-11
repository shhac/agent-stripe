package cli

import (
	"net/url"
	"strings"
)

var ledgerInvestigation = investigationSpec{
	use:   "ledger <charge-id|payment-intent-id|refund-id|transfer-id|payout-id|balance-transaction-id|application-fee-id>",
	short: "Gather balance transactions and related money-movement objects",
	run:   investigator.ledger,
}

func (i investigator) ledger(id string) error {
	if err := validateAllowedStripeID(id, "charge", "payment_intent", "refund", "transfer", "payout", "balance_transaction", "application_fee"); err != nil {
		return err
	}
	switch {
	case strings.HasPrefix(id, "pi_"):
		return i.ledgerFromPaymentIntent(id)
	case strings.HasPrefix(id, "ch_"):
		return i.ledgerFromChargeID(id)
	case strings.HasPrefix(id, "re_"):
		return i.ledgerFromRefundID(id)
	case strings.HasPrefix(id, "tr_"):
		return i.ledgerFromSimpleObject("transfer", "/v1/transfers/"+url.PathEscape(id))
	case strings.HasPrefix(id, "po_"):
		return i.ledgerFromSimpleObject("payout", "/v1/payouts/"+url.PathEscape(id))
	case strings.HasPrefix(id, "fee_"):
		return i.ledgerFromSimpleObject("application_fee", "/v1/application_fees/"+url.PathEscape(id))
	default:
		txn, err := i.get("/v1/balance_transactions/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(ledgerFinding("balance_transaction", txn))
		return nil
	}
}

func (i investigator) ledgerFromPaymentIntent(id string) error {
	pi, err := i.get("/v1/payment_intents/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return err
	}
	if charge := i.addLatestCharge(pi); charge != nil {
		i.ledgerFromCharge(charge)
	}
	i.add(ledgerFinding("payment_intent", pi))
	return nil
}

func (i investigator) ledgerFromChargeID(id string) error {
	charge, err := i.get("/v1/charges/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return err
	}
	i.ledgerFromCharge(charge)
	return nil
}

func (i investigator) ledgerFromCharge(charge map[string]any) {
	i.followRef(charge, "balance_transaction")
	if fees := i.listRelated("application fees", "/v1/application_fees", url.Values{"charge": []string{mapString(charge, "id")}, "limit": []string{"10"}}); fees != nil {
		i.addList("application_fee", fees)
	}
	if refunds := i.listRelated("refunds", "/v1/refunds", url.Values{"charge": []string{mapString(charge, "id")}, "limit": []string{"10"}}); refunds != nil {
		for _, refund := range refunds {
			i.ledgerFromRefund(refund)
		}
	}
	i.add(ledgerFinding("charge", charge))
}

func (i investigator) ledgerFromRefundID(id string) error {
	refund, err := i.get("/v1/refunds/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return err
	}
	i.ledgerFromRefund(refund)
	return nil
}

func (i investigator) ledgerFromRefund(refund map[string]any) {
	i.followRef(refund, "balance_transaction")
	i.add(ledgerFinding("refund", refund))
}

func (i investigator) ledgerFromSimpleObject(object, path string) error {
	item, err := i.get(path, url.Values{})
	if err != nil {
		return err
	}
	i.followRef(item, "balance_transaction")
	i.add(ledgerFinding(object, item))
	return nil
}

func ledgerFinding(object string, item map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  object + " " + mapString(item, "id") + " ledger evidence gathered. Use balance_transaction net/fee/amount fields to reconcile money movement.",
		Data: map[string]any{
			"object":              object,
			"id":                  mapString(item, "id"),
			"amount":              mapValue(item, "amount"),
			"currency":            mapString(item, "currency"),
			"balance_transaction": idFromValue(item["balance_transaction"]),
		},
	}
}
