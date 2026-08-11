package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateSubscriptionAmountChange(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var subscription string
	cmd := &cobra.Command{
		Use:   "subscription-amount-change",
		Short: "Explain subscription invoice amount using latest invoice, preview, items, prices, and products",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireFlag("subscription", subscription, "Provide a Subscription ID such as sub_..."); err != nil {
				return err
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.subscriptionAmountChange(subscription)
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	return cmd
}

func (i investigator) subscriptionAmountChange(subscriptionID string) error {
	bundle, err := i.subscriptionItemsBundle(subscriptionID)
	if err != nil {
		return err
	}
	i.add(subscriptionItemsFinding(subscriptionID, len(bundle.items)))

	latestInvoice := i.latestInvoiceEvidence(bundle.sub)
	preview := i.invoicePreviewEvidence(subscriptionID)

	i.add(subscriptionAmountFinding(subscriptionID, latestInvoice, preview, bundle.items))
	return nil
}

func (i investigator) latestInvoiceEvidence(sub map[string]any) map[string]any {
	latestInvoiceID := idFromValue(sub["latest_invoice"])
	if latestInvoiceID == "" {
		return nil
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(latestInvoiceID), url.Values{})
	if err != nil {
		i.add(finding("warning", "Could not retrieve latest invoice "+latestInvoiceID+": "+err.Error()))
		return nil
	}
	i.add(entityRecord("invoice", invoice))
	lines, err := i.list("/v1/invoices/"+url.PathEscape(latestInvoiceID)+"/lines", url.Values{"limit": []string{"100"}})
	if err != nil {
		i.add(relatedWarning("invoice lines", err))
		return invoice
	}
	i.addList("line_item", lines)
	return invoice
}

func (i investigator) invoicePreviewEvidence(subscriptionID string) map[string]any {
	preview, err := i.postFormAs("invoice_preview", "/v1/invoices/create_preview", url.Values{"subscription": []string{subscriptionID}})
	if err != nil {
		i.add(relatedWarning("upcoming invoice preview", err))
		return nil
	}
	return preview
}

func subscriptionAmountFinding(subscriptionID string, latestInvoice, preview map[string]any, items []map[string]any) evidenceRecord {
	itemSubtotal, itemCurrency, subtotalOK := subscriptionItemsSubtotal(items)
	data := map[string]any{
		"subscription": subscriptionID,
		"item_count":   len(items),
	}
	if subtotalOK {
		data["item_subtotal"] = itemSubtotal
		data["item_currency"] = itemCurrency
	}
	if latestInvoice != nil {
		if amount, ok := mapInt64(latestInvoice, "amount_due"); ok {
			data["latest_invoice_amount_due"] = amount
		}
		data["latest_invoice"] = mapString(latestInvoice, "id")
	}
	if preview != nil {
		if amount, ok := mapInt64(preview, "amount_due"); ok {
			data["preview_amount_due"] = amount
		}
	}

	summary := fmt.Sprintf("Subscription %s amount evidence gathered.", subscriptionID)
	if subtotalOK {
		summary = fmt.Sprintf("Subscription %s current item subtotal is %d %s minor units.", subscriptionID, itemSubtotal, strings.ToUpper(itemCurrency))
	}
	if latestInvoice != nil && preview != nil {
		summary += fmt.Sprintf(" Latest invoice amount is %s; next preview amount is %s.", formatAmount(latestInvoice), formatAmount(preview))
	}
	return evidenceRecord{Type: "finding", Severity: "info", Summary: summary, Data: data}
}

func subscriptionItemsSubtotal(items []map[string]any) (int64, string, bool) {
	var total int64
	currency := ""
	for _, item := range items {
		price := mapAnyMap(item, "price")
		unitAmount, ok := mapInt64(price, "unit_amount")
		if !ok {
			continue
		}
		quantity, ok := mapInt64(item, "quantity")
		if !ok || quantity == 0 {
			quantity = 1
		}
		total += unitAmount * quantity
		if currency == "" {
			currency = mapString(price, "currency")
		}
	}
	return total, currency, currency != "" || total > 0
}
