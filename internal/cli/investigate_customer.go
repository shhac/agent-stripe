package cli

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateCustomerCardPayment(globals shared.GlobalsFunc) *cobra.Command {
	var customer string
	var last4 string
	var limit int
	cmd := &cobra.Command{
		Use:   "customer-card-payment",
		Short: "Find a customer's most recent charge by card last4",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("customer", customer, "Last4 is not unique; provide a Customer ID to keep the investigation bounded") {
				return nil
			}
			if !shared.RequireFlag("last4", last4, "Use the final four digits the customer supplied") {
				return nil
			}
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				params := url.Values{"customer": []string{customer}}
				api.AddLimit(params, limit)
				charges, err := inv.list("/v1/charges", params)
				if err != nil {
					return nil, err
				}
				for _, charge := range charges {
					if cardLast4(charge) != last4 {
						continue
					}
					return []evidenceRecord{
						entityRecord("charge", charge),
						{
							Type:     "finding",
							Severity: "info",
							Summary:  fmt.Sprintf("Most recent payment for customer %s using card ending %s is charge %s for %s.", customer, last4, mapString(charge, "id"), formatAmount(charge)),
						},
					}, nil
				}
				return []evidenceRecord{{
					Type:     "finding",
					Severity: "warning",
					Summary:  fmt.Sprintf("No recent charge for customer %s matched card ending %s in the first %d charges.", customer, last4, limit),
				}}, nil
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().StringVar(&last4, "last4", "", "Card last4")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum recent charges to inspect")
	return cmd
}
