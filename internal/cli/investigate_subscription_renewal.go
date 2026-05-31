package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateSubscriptionRenewal(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var subscription string
	var metadata string
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "subscription-renewal",
		Short: "Summarize last and next payment for subscriptions found by ID, customer, or metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				subs, err := inv.findSubscriptions(subscription, customer, metadata, limit)
				if err != nil {
					return nil, err
				}
				if len(subs) == 0 {
					return inv.appendEvidence(nil, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No subscriptions matched the supplied filters."}), nil
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					records = inv.appendEvidence(records, entityRecord("subscription", sub))
					records = inv.appendEvidenceAll(records, inv.subscriptionPaymentSummary(sub))
				}
				return records, nil
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	cmd.Flags().StringVar(&metadata, "metadata", "", "Metadata equality filter as key=value")
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum matching subscriptions to inspect")
	return cmd
}

func (i investigator) subscriptionPaymentSummary(sub map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	if latestInvoiceID := idFromValue(sub["latest_invoice"]); latestInvoiceID != "" {
		invoiceRecords, err := i.invoicePayment(latestInvoiceID)
		if err == nil {
			records = i.appendEvidenceAll(records, invoiceRecords)
		} else {
			records = i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Could not retrieve latest invoice " + latestInvoiceID + ": " + err.Error()})
		}
	}
	nextAmount := "unknown"
	if preview, err := i.postForm("/v1/invoices/create_preview", url.Values{"subscription": []string{mapString(sub, "id")}}); err == nil {
		records = i.appendEvidence(records, entityRecord("invoice_preview", preview))
		nextAmount = formatAmount(preview)
	}
	records = i.appendEvidence(records, evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary: fmt.Sprintf("Subscription %s last invoice is %s; next renewal is at %v and preview amount is %s.",
			mapString(sub, "id"), idFromValue(sub["latest_invoice"]), mapValue(sub, "current_period_end"), nextAmount),
	})
	return records
}
