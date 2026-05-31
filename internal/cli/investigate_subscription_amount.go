package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateSubscriptionAmountChange(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var subscription string
	cmd := &cobra.Command{
		Use:   "subscription-amount-change",
		Short: "Explain subscription invoice amount using latest invoice, preview, items, prices, and products",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("subscription", subscription, "Provide a Subscription ID such as sub_...") {
				return nil
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.subscriptionAmountChange(subscription)
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	return cmd
}

func (i investigator) subscriptionAmountChange(subscriptionID string) ([]evidenceRecord, error) {
	bundle, err := i.subscriptionItemsBundle(subscriptionID)
	if err != nil {
		return nil, err
	}
	records := i.appendEvidenceAll(nil, bundle.records)
	records = i.appendEvidence(records, subscriptionItemsFinding(subscriptionID, len(bundle.items)))

	latestInvoice, latestInvoiceRecords := i.latestInvoiceEvidence(bundle.sub)
	records = i.appendEvidenceAll(records, latestInvoiceRecords)

	preview, previewRecords := i.invoicePreviewEvidence(subscriptionID)
	records = i.appendEvidenceAll(records, previewRecords)

	records = i.appendEvidence(records, subscriptionAmountFinding(subscriptionID, latestInvoice, preview, bundle.items))
	return records, nil
}

func (i investigator) latestInvoiceEvidence(sub map[string]any) (map[string]any, []evidenceRecord) {
	latestInvoiceID := idFromValue(sub["latest_invoice"])
	if latestInvoiceID == "" {
		return nil, nil
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(latestInvoiceID), url.Values{})
	if err != nil {
		return nil, []evidenceRecord{{
			Type:     "finding",
			Severity: "warning",
			Summary:  "Could not retrieve latest invoice " + latestInvoiceID + ": " + err.Error(),
		}}
	}
	records := i.appendEvidence(nil, entityRecord("invoice", invoice))
	lines, err := i.list("/v1/invoices/"+url.PathEscape(latestInvoiceID)+"/lines", url.Values{"limit": []string{"100"}})
	if err == nil {
		records = i.appendListRecords(records, "line_item", lines)
	}
	return invoice, records
}

func (i investigator) invoicePreviewEvidence(subscriptionID string) (map[string]any, []evidenceRecord) {
	preview, err := i.postForm("/v1/invoices/create_preview", url.Values{"subscription": []string{subscriptionID}})
	if err != nil {
		return nil, nil
	}
	return preview, i.appendEvidence(nil, entityRecord("invoice_preview", preview))
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
