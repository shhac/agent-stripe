package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateSubscriptionRenewal(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var subscription string
	var metadata string
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "subscription-renewal",
		Short: "Summarize last and next payment for subscriptions found by ID, customer, or metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.subscriptionRenewal(subscription, customer, metadata, limit)
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	cmd.Flags().StringVar(&metadata, "metadata", "", "Metadata equality filter as key=value")
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum matching subscriptions to inspect")
	return cmd
}

func (i investigator) subscriptionRenewal(subscription, customer, metadata string, limit int) error {
	subs, err := i.findSubscriptions(subscription, customer, metadata, limit)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		i.add(finding("warning", "No subscriptions matched the supplied filters."))
		return nil
	}
	for _, sub := range subs {
		i.add(entityRecord("subscription", sub))
		i.subscriptionPaymentSummary(sub)
	}
	return nil
}

func (i investigator) subscriptionPaymentSummary(sub map[string]any) {
	if latestInvoiceID := idFromValue(sub["latest_invoice"]); latestInvoiceID != "" {
		if err := i.invoicePayment(latestInvoiceID); err != nil {
			i.add(finding("warning", "Could not retrieve latest invoice "+latestInvoiceID+": "+err.Error()))
		}
	}
	nextAmount := "unknown"
	preview, err := i.postForm("/v1/invoices/create_preview", url.Values{"subscription": []string{mapString(sub, "id")}})
	if err != nil {
		i.add(relatedWarning("upcoming invoice preview", err))
	} else {
		i.add(entityRecord("invoice_preview", preview))
		nextAmount = formatAmount(preview)
	}
	i.add(evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary: fmt.Sprintf("Subscription %s last invoice is %s; next renewal is at %v and preview amount is %s.",
			mapString(sub, "id"), idFromValue(sub["latest_invoice"]), mapValue(sub, "current_period_end"), nextAmount),
	})
}
