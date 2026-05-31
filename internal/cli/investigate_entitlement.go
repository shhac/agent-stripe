package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateEntitlement(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var customer, subscription, invoice, checkoutSession, metadata string
	var limit int
	cmd := &cobra.Command{
		Use:   "entitlement",
		Short: "Find subscription, invoice, or checkout product metadata for entitlement mismatches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
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

func (i investigator) entitlement(q entitlementQuery) ([]evidenceRecord, error) {
	records := []evidenceRecord{}
	if q.subscription != "" || q.customer != "" || q.metadata != "" {
		subs, err := i.findSubscriptions(q.subscription, q.customer, q.metadata, q.limit)
		if err != nil {
			return nil, err
		}
		for _, sub := range subs {
			records = i.appendEvidence(records, entityRecord("subscription", sub))
			if bundle, err := i.subscriptionItemsBundle(mapString(sub, "id")); err == nil {
				records = i.appendEvidenceAll(records, bundle.records)
			}
		}
	}
	if q.invoice != "" {
		if err := validateExpectedStripeID(q.invoice, "invoice"); err != nil {
			return nil, err
		}
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(q.invoice), url.Values{})
		if err != nil {
			return nil, err
		}
		records = i.appendEvidence(records, entityRecord("invoice", invoice))
		records = i.appendEvidenceAll(records, i.invoiceLineEntitlements(q.invoice))
	}
	if q.checkoutSession != "" {
		sessionRecords, err := i.checkoutSession(q.checkoutSession)
		if err != nil {
			return nil, err
		}
		records = i.appendEvidenceAll(records, sessionRecords)
	}
	if len(records) == 0 {
		return i.appendEvidence(nil, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Provide --subscription, --customer, --metadata, --invoice, or --checkout-session to investigate entitlements."}), nil
	}
	records = i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "Entitlement evidence gathered from subscription items, invoice lines, checkout line items, prices, and products. Prefer product/price metadata for internal product IDs."})
	return records, nil
}

func (i investigator) invoiceLineEntitlements(invoiceID string) []evidenceRecord {
	lines, err := i.list("/v1/invoices/"+url.PathEscape(invoiceID)+"/lines", url.Values{"limit": []string{"100"}})
	if err != nil {
		return []evidenceRecord{relatedWarning("invoice lines", err)}
	}
	return i.appendListRecords(nil, "line_item", lines)
}
