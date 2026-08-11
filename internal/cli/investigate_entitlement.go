package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateEntitlement(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var customer, subscription, invoice, checkoutSession, metadata string
	var limit int
	cmd := &cobra.Command{
		Use:   "entitlement",
		Short: "Find subscription, invoice, or checkout product metadata for entitlement mismatches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.entitlement(entitlementQuery{
					customer:        customer,
					subscription:    subscription,
					invoice:         invoice,
					checkoutSession: checkoutSession,
					metadata:        metadata,
					limit:           limit,
				})
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	cmd.Flags().StringVar(&invoice, "invoice", "", "Invoice ID")
	cmd.Flags().StringVar(&checkoutSession, "checkout-session", "", "Checkout Session ID")
	cmd.Flags().StringVar(&metadata, "metadata", "", "Subscription metadata equality filter as key=value")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum matching subscriptions or invoices to inspect")
	return cmd
}

type entitlementQuery struct {
	customer        string
	subscription    string
	invoice         string
	checkoutSession string
	metadata        string
	limit           int
}

func (i investigator) entitlement(q entitlementQuery) error {
	before := i.count()
	if q.subscription != "" || q.customer != "" || q.metadata != "" {
		subs, err := i.findSubscriptions(q.subscription, q.customer, q.metadata, q.limit)
		if err != nil {
			return err
		}
		for _, sub := range subs {
			i.add(entityRecord("subscription", sub))
			if bundle, err := i.subscriptionItemsBundle(mapString(sub, "id")); err == nil {
				i.add(bundle.records...)
			}
		}
	}
	if q.invoice != "" {
		if err := validateExpectedStripeID(q.invoice, "invoice"); err != nil {
			return err
		}
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(q.invoice), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("invoice", invoice))
		i.invoiceLineEntitlements(q.invoice)
	}
	if q.checkoutSession != "" {
		if err := i.checkoutSession(q.checkoutSession); err != nil {
			return err
		}
	}
	if i.count() == before {
		i.add(evidenceRecord{Type: "finding", Severity: "warning", Summary: "Provide --subscription, --customer, --metadata, --invoice, or --checkout-session to investigate entitlements."})
		return nil
	}
	i.add(evidenceRecord{Type: "finding", Severity: "info", Summary: "Entitlement evidence gathered from subscription items, invoice lines, checkout line items, prices, and products. Prefer product/price metadata for internal product IDs."})
	return nil
}

func (i investigator) invoiceLineEntitlements(invoiceID string) {
	lines, err := i.list("/v1/invoices/"+url.PathEscape(invoiceID)+"/lines", url.Values{"limit": []string{"100"}})
	if err != nil {
		i.add(relatedWarning("invoice lines", err))
		return
	}
	i.addList("line_item", lines)
}
