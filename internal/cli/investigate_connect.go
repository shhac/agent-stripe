package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func newInvestigateOutgoingPayment(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "outgoing-payment <transfer-id|payout-id|account-id>",
		Short: "Explain what happened to money moving from you to a connected business",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.outgoingPayment(args[0])
			})
		},
	}
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

func newInvestigatePayoutFailure(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "payout-failure <payout-id>",
		Short: "Explain payout failure details and related ledger movement",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.payoutFailure(args[0])
			})
		},
	}
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
