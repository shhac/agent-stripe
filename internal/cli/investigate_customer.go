package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateCustomerCardPayment(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var customer string
	var last4 string
	var limit int
	cmd := &cobra.Command{
		Use:   "customer-card-payment",
		Short: "Find a customer's most recent charge by card last4",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireFlag("customer", customer, "Last4 is not unique; provide a Customer ID to keep the investigation bounded"); err != nil {
				return err
			}
			if err := shared.RequireFlag("last4", last4, "Use the final four digits the customer supplied"); err != nil {
				return err
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.customerCardPayment(customer, last4, limit)
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().StringVar(&last4, "last4", "", "Card last4")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum recent charges to inspect")
	return cmd
}

func (i investigator) customerCardPayment(customer, last4 string, limit int) error {
	if err := validateExpectedStripeID(customer, "customer"); err != nil {
		return err
	}
	params := url.Values{"customer": []string{customer}}
	shared.AddLimit(params, limit)
	charges, err := i.list("/v1/charges", params)
	if err != nil {
		return err
	}
	for _, charge := range charges {
		if cardLast4(charge) != last4 {
			continue
		}
		i.add(
			entityRecord("charge", charge),
			customerCardPaymentFinding(customer, last4, charge),
		)
		return nil
	}
	i.add(customerCardPaymentNotFound(customer, last4, limit))
	return nil
}

func customerCardPaymentFinding(customer, last4 string, charge map[string]any) evidenceRecord {
	return finding("info", fmt.Sprintf("Most recent payment for customer %s using card ending %s is charge %s for %s.", customer, last4, mapString(charge, "id"), formatAmount(charge)))
}

func customerCardPaymentNotFound(customer, last4 string, limit int) evidenceRecord {
	return finding("warning", fmt.Sprintf("No recent charge for customer %s matched card ending %s in the first %d charges.", customer, last4, limit))
}
