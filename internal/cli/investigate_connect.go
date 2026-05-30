package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func newInvestigateOutgoingPayment(globals shared.GlobalsFunc) *cobra.Command {
	return &cobra.Command{
		Use:   "outgoing-payment <transfer-id|payout-id|account-id>",
		Short: "Explain what happened to money moving from you to a connected business",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				return inv.outgoingPayment(args[0])
			})
		},
	}
}

func newInvestigateRefundRecovery(globals shared.GlobalsFunc) *cobra.Command {
	var transfer string
	cmd := &cobra.Command{
		Use:   "refund-recovery <refund-id|charge-id|payment-intent-id|transfer-reversal-id>",
		Short: "Explain failed refund funding or Connect transfer reversal recovery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				return inv.refundRecovery(args[0], transfer)
			})
		},
	}
	cmd.Flags().StringVar(&transfer, "transfer", "", "Parent transfer ID when investigating a transfer reversal ID")
	return cmd
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
