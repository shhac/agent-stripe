package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

var outgoingPaymentInvestigation = investigationSpec{
	use:   "outgoing-payment <transfer-id|payout-id|account-id>",
	short: "Explain what happened to money moving from you to a connected business",
	run:   investigator.outgoingPayment,
}

func newInvestigateRefundRecovery(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var transfer string
	cmd := &cobra.Command{
		Use:   "refund-recovery <refund-id|charge-id|payment-intent-id|transfer-reversal-id>",
		Short: "Explain failed refund funding or Connect transfer reversal recovery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.refundRecovery(args[0], transfer)
			})
		},
	}
	cmd.Flags().StringVar(&transfer, "transfer", "", "Parent transfer ID when investigating a transfer reversal ID")
	return cmd
}

var payoutFailureInvestigation = investigationSpec{
	use:   "payout-failure <payout-id>",
	short: "Explain payout failure details and related ledger movement",
	run:   investigator.payoutFailure,
}

func (i investigator) outgoingPayment(id string) error {
	if err := validateAllowedStripeID(id, "transfer", "payout", "account"); err != nil {
		return err
	}
	switch {
	case strings.HasPrefix(id, "tr_"):
		transfer, err := i.get("/v1/transfers/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("transfer", transfer), moneyMovementFinding("transfer", transfer))
		return nil
	case strings.HasPrefix(id, "po_"):
		payout, err := i.get("/v1/payouts/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("payout", payout), moneyMovementFinding("payout", payout))
		return nil
	default:
		return i.accountHealth(id, namespaceAuto)
	}
}

func (i investigator) refundRecovery(id, transferID string) error {
	if err := validateAllowedStripeID(id, "refund", "transfer_reversal", "charge", "payment_intent"); err != nil {
		return err
	}
	switch {
	case strings.HasPrefix(id, "re_"):
		refund, err := i.get("/v1/refunds/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("refund", refund), moneyMovementFinding("refund", refund))
		i.addConnectRefundLiability(refund)
		return nil
	case strings.HasPrefix(id, "trr_"):
		if transferID == "" {
			return agenterrors.New("--transfer is required for transfer reversal IDs", agenterrors.FixableByAgent).
				WithHint("Stripe transfer reversals are nested under /v1/transfers/{transfer}/reversals/{reversal}")
		}
		reversal, err := i.get("/v1/transfers/"+url.PathEscape(transferID)+"/reversals/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("transfer_reversal", reversal), moneyMovementFinding("transfer_reversal", reversal))
		return nil
	default:
		return i.incomingPayment(id)
	}
}

// addConnectRefundLiability answers who is out of pocket after a refund on a
// Connect payment. Reversing the transfer pulls the money back from the
// connected account; refunding the application fee gives back the platform's
// cut. Which of those happened decides who absorbed the refund, and neither is
// visible on the refund object itself.
func (i investigator) addConnectRefundLiability(refund map[string]any) {
	transferID := idFromValue(refund["transfer"])
	reversalID := idFromValue(refund["transfer_reversal"])
	chargeID := idFromValue(refund["charge"])
	if transferID == "" && chargeID == "" {
		return
	}

	var reversal map[string]any
	if transferID != "" {
		i.fetchRelated("transfer", transferID)
		if reversalID != "" {
			found, err := i.get("/v1/transfers/"+url.PathEscape(transferID)+"/reversals/"+url.PathEscape(reversalID), url.Values{})
			if err != nil {
				i.add(relatedWarning("transfer reversal "+reversalID, err))
			} else {
				reversal = found
				i.add(entityRecord("transfer_reversal", reversal))
			}
		}
	}

	var fees []map[string]any
	if chargeID != "" {
		fees = i.listRelated("application fees", "/v1/application_fees", valuesWithLimit(5, "charge", chargeID))
		i.addList("application_fee", fees)
	}
	i.add(connectRefundLiabilityFinding(refund, reversal, fees))
}

func connectRefundLiabilityFinding(refund, reversal map[string]any, fees []map[string]any) evidenceRecord {
	amount, _ := mapInt64(refund, "amount")
	data := map[string]any{
		"refund":   mapString(refund, "id"),
		"amount":   amount,
		"currency": mapString(refund, "currency"),
		"transfer": idFromValue(refund["transfer"]),
	}

	parts := []string{}
	severity := "info"
	if reversal != nil {
		reversed, _ := mapInt64(reversal, "amount")
		data["transfer_reversal"] = mapString(reversal, "id")
		data["reversed_amount"] = reversed
		parts = append(parts, fmt.Sprintf("%d %s was pulled back from the connected account by reversal %s",
			reversed, mapString(reversal, "currency"), mapString(reversal, "id")))
	} else if idFromValue(refund["transfer"]) != "" {
		severity = "warning"
		parts = append(parts, "the transfer to the connected account was not reversed, so the platform absorbed this refund")
	}

	feeRefunded := int64(0)
	for _, fee := range fees {
		refundedAmount, _ := mapInt64(fee, "amount_refunded")
		feeRefunded += refundedAmount
	}
	if len(fees) > 0 {
		data["application_fees"] = len(fees)
		data["application_fee_refunded"] = feeRefunded
		if feeRefunded > 0 {
			parts = append(parts, fmt.Sprintf("%d of the application fee was refunded", feeRefunded))
		} else {
			parts = append(parts, "the application fee was kept")
		}
	}
	if len(parts) == 0 {
		return evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  "Refund " + mapString(refund, "id") + " has no Connect transfer or application fee attached; liability sits with the platform account alone.",
			Data:     data,
		}
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  fmt.Sprintf("Connect liability for refund %s: %s.", mapString(refund, "id"), joinAndTruncate(parts, 3)),
		Data:     data,
	}
}

func (i investigator) refundStatus(refundID string) error {
	if err := validateExpectedStripeID(refundID, "refund"); err != nil {
		return err
	}
	refund, err := i.get("/v1/refunds/"+url.PathEscape(refundID), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("refund", refund))
	if chargeID := idFromValue(refund["charge"]); chargeID != "" {
		charge, err := i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{})
		if err == nil {
			i.add(entityRecord("charge", charge))
		}
	}
	if piID := idFromValue(refund["payment_intent"]); piID != "" {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{})
		if err == nil {
			i.add(entityRecord("payment_intent", pi))
		}
	}
	if transferID := idFromValue(refund["transfer"]); transferID != "" {
		transfer, err := i.get("/v1/transfers/"+url.PathEscape(transferID), url.Values{})
		if err == nil {
			i.add(entityRecord("transfer", transfer))
		}
		if reversalID := idFromValue(refund["transfer_reversal"]); reversalID != "" {
			reversal, err := i.get("/v1/transfers/"+url.PathEscape(transferID)+"/reversals/"+url.PathEscape(reversalID), url.Values{})
			if err == nil {
				i.add(entityRecord("transfer_reversal", reversal))
			}
		}
	}
	i.add(moneyMovementFinding("refund", refund))
	return nil
}

func (i investigator) payoutFailure(payoutID string) error {
	if err := validateExpectedStripeID(payoutID, "payout"); err != nil {
		return err
	}
	payout, err := i.get("/v1/payouts/"+url.PathEscape(payoutID), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("payout", payout))
	if txnID := idFromValue(payout["balance_transaction"]); txnID != "" {
		txn, err := i.get("/v1/balance_transactions/"+url.PathEscape(txnID), url.Values{})
		if err == nil {
			i.add(entityRecord("balance_transaction", txn))
		}
	}
	i.add(moneyMovementFinding("payout", payout))
	return nil
}
