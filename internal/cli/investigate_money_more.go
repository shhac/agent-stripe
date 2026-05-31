package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateLedger(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "ledger <charge-id|payment-intent-id|refund-id|transfer-id|payout-id|balance-transaction-id|application-fee-id>",
		Short: "Gather balance transactions and related money-movement objects",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.ledger(args[0])
			})
		},
	}
}

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

func (i investigator) ledger(id string) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "charge", "payment_intent", "refund", "transfer", "payout", "balance_transaction", "application_fee"); err != nil {
		return nil, err
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
			return nil, err
		}
		return []evidenceRecord{entityRecord("balance_transaction", txn), ledgerFinding("balance_transaction", txn)}, nil
	}
}

func (i investigator) ledgerFromPaymentIntent(id string) ([]evidenceRecord, error) {
	pi, err := i.get("/v1/payment_intents/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord("payment_intent", pi)}
	if charge, err := i.latestChargeForPaymentIntent(pi); err == nil && charge != nil {
		records = append(records, i.ledgerFromCharge(charge)...)
	}
	return append(records, ledgerFinding("payment_intent", pi)), nil
}

func (i investigator) ledgerFromChargeID(id string) ([]evidenceRecord, error) {
	charge, err := i.get("/v1/charges/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return nil, err
	}
	return i.ledgerFromCharge(charge), nil
}

func (i investigator) ledgerFromCharge(charge map[string]any) []evidenceRecord {
	records := []evidenceRecord{entityRecord("charge", charge)}
	if txnID := idFromValue(charge["balance_transaction"]); txnID != "" {
		if txn, err := i.get("/v1/balance_transactions/"+url.PathEscape(txnID), url.Values{}); err == nil {
			records = append(records, entityRecord("balance_transaction", txn))
		}
	}
	if fees, err := i.list("/v1/application_fees", url.Values{"charge": []string{mapString(charge, "id")}, "limit": []string{"10"}}); err == nil {
		records = appendListRecords(records, "application_fee", fees)
	}
	if refunds, err := i.list("/v1/refunds", url.Values{"charge": []string{mapString(charge, "id")}, "limit": []string{"10"}}); err == nil {
		for _, refund := range refunds {
			records = append(records, i.ledgerFromRefund(refund)...)
		}
	}
	return append(records, ledgerFinding("charge", charge))
}

func (i investigator) ledgerFromRefundID(id string) ([]evidenceRecord, error) {
	refund, err := i.get("/v1/refunds/"+url.PathEscape(id), url.Values{})
	if err != nil {
		return nil, err
	}
	return i.ledgerFromRefund(refund), nil
}

func (i investigator) ledgerFromRefund(refund map[string]any) []evidenceRecord {
	records := []evidenceRecord{entityRecord("refund", refund)}
	if txnID := idFromValue(refund["balance_transaction"]); txnID != "" {
		if txn, err := i.get("/v1/balance_transactions/"+url.PathEscape(txnID), url.Values{}); err == nil {
			records = append(records, entityRecord("balance_transaction", txn))
		}
	}
	return append(records, ledgerFinding("refund", refund))
}

func (i investigator) ledgerFromSimpleObject(object, path string) ([]evidenceRecord, error) {
	item, err := i.get(path, url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord(object, item)}
	if txnID := idFromValue(item["balance_transaction"]); txnID != "" {
		if txn, err := i.get("/v1/balance_transactions/"+url.PathEscape(txnID), url.Values{}); err == nil {
			records = append(records, entityRecord("balance_transaction", txn))
		}
	}
	records = append(records, ledgerFinding(object, item))
	return records, nil
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
	records = appendListRecords(records, "refund", refunds)
	if len(refunds) == 0 {
		records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No refunds found for " + id + "."})
		return records, nil
	}
	records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "Refund evidence gathered for original payment " + id + "."})
	return records, nil
}
