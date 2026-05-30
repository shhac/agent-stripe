package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func newInvestigateOutgoingPayment(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "outgoing-payment <transfer-id|payout-id|account-id>",
		Short: "Explain what happened to money moving from you to a connected business",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.outgoingPayment(args[0])
			})
		},
	}
}

func newInvestigateRefundRecovery(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var transfer string
	cmd := &cobra.Command{
		Use:   "refund-recovery <refund-id|charge-id|payment-intent-id|transfer-reversal-id>",
		Short: "Explain failed refund funding or Connect transfer reversal recovery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.refundRecovery(args[0], transfer)
			})
		},
	}
	cmd.Flags().StringVar(&transfer, "transfer", "", "Parent transfer ID when investigating a transfer reversal ID")
	return cmd
}

func newInvestigateRefundStatus(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "refund-status <refund-id>",
		Short: "Explain refund state, related payment, and Connect reversal details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.refundStatus(args[0])
			})
		},
	}
}

func newInvestigatePayoutFailure(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "payout-failure <payout-id>",
		Short: "Explain payout failure details and related ledger movement",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.payoutFailure(args[0])
			})
		},
	}
}

func (i investigator) outgoingPayment(id string) ([]evidenceRecord, error) {
	switch {
	case strings.HasPrefix(id, "tr_"):
		transfer, err := i.get("/v1/transfers/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("transfer", transfer), moneyMovementFinding("transfer", transfer)}, nil
	case strings.HasPrefix(id, "po_"):
		payout, err := i.get("/v1/payouts/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("payout", payout), moneyMovementFinding("payout", payout)}, nil
	default:
		account, err := i.get("/v1/accounts/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records := []evidenceRecord{entityRecord("account", account)}
		if !mapBool(account, "payouts_enabled") || !mapBool(account, "charges_enabled") {
			records = append(records, evidenceRecord{
				Type:     "finding",
				Severity: "warning",
				Summary:  fmt.Sprintf("Connected account %s is not fully enabled for charges/payouts; inspect account requirements.", mapString(account, "id")),
				Data: map[string]any{
					"requirements": mapAnyMap(account, "requirements"),
				},
			})
		}
		transfers, _ := i.list("/v1/transfers", url.Values{"destination": []string{id}, "limit": []string{"5"}})
		for _, transfer := range transfers {
			records = append(records, entityRecord("transfer", transfer))
		}
		return records, nil
	}
}

func (i investigator) refundRecovery(id, transferID string) ([]evidenceRecord, error) {
	switch {
	case strings.HasPrefix(id, "re_"):
		refund, err := i.get("/v1/refunds/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("refund", refund), moneyMovementFinding("refund", refund)}, nil
	case strings.HasPrefix(id, "trr_"):
		if transferID == "" {
			return nil, agenterrors.New("--transfer is required for transfer reversal IDs", agenterrors.FixableByAgent).
				WithHint("Stripe transfer reversals are nested under /v1/transfers/{transfer}/reversals/{reversal}")
		}
		reversal, err := i.get("/v1/transfers/"+url.PathEscape(transferID)+"/reversals/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		return []evidenceRecord{entityRecord("transfer_reversal", reversal), moneyMovementFinding("transfer_reversal", reversal)}, nil
	default:
		return i.incomingPayment(id)
	}
}

func (i investigator) refundStatus(refundID string) ([]evidenceRecord, error) {
	refund, err := i.get("/v1/refunds/"+url.PathEscape(refundID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord("refund", refund)}
	if chargeID := idFromValue(refund["charge"]); chargeID != "" {
		charge, err := i.get("/v1/charges/"+url.PathEscape(chargeID), url.Values{})
		if err == nil {
			records = append(records, entityRecord("charge", charge))
		}
	}
	if piID := idFromValue(refund["payment_intent"]); piID != "" {
		pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{})
		if err == nil {
			records = append(records, entityRecord("payment_intent", pi))
		}
	}
	if transferID := idFromValue(refund["transfer"]); transferID != "" {
		transfer, err := i.get("/v1/transfers/"+url.PathEscape(transferID), url.Values{})
		if err == nil {
			records = append(records, entityRecord("transfer", transfer))
		}
		if reversalID := idFromValue(refund["transfer_reversal"]); reversalID != "" {
			reversal, err := i.get("/v1/transfers/"+url.PathEscape(transferID)+"/reversals/"+url.PathEscape(reversalID), url.Values{})
			if err == nil {
				records = append(records, entityRecord("transfer_reversal", reversal))
			}
		}
	}
	records = append(records, moneyMovementFinding("refund", refund))
	return records, nil
}

func (i investigator) payoutFailure(payoutID string) ([]evidenceRecord, error) {
	payout, err := i.get("/v1/payouts/"+url.PathEscape(payoutID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := []evidenceRecord{entityRecord("payout", payout)}
	if txnID := idFromValue(payout["balance_transaction"]); txnID != "" {
		txn, err := i.get("/v1/balance_transactions/"+url.PathEscape(txnID), url.Values{})
		if err == nil {
			records = append(records, entityRecord("balance_transaction", txn))
		}
	}
	records = append(records, moneyMovementFinding("payout", payout))
	return records, nil
}
