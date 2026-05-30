package cli

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
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
					return []evidenceRecord{{Type: "finding", Severity: "warning", Summary: "No subscriptions matched the supplied filters."}}, nil
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					records = append(records, entityRecord("subscription", sub))
					records = append(records, inv.subscriptionPaymentSummary(sub)...)
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

func newInvestigateCollectionRisk(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "collection-risk",
		Short: "Find customers likely to need payment-method outreach before upcoming collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				params := url.Values{"status": []string{"all"}}
				shared.AddLimit(params, limit)
				if days > 0 {
					params.Set("current_period_end[lte]", strconv.FormatInt(time.Now().Add(time.Duration(days)*24*time.Hour).Unix(), 10))
				}
				subs, err := inv.list("/v1/subscriptions", params)
				if err != nil {
					return nil, err
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					risk := inv.collectionRisk(sub)
					if risk == "" {
						continue
					}
					records = append(records, entityRecord("subscription", sub))
					records = append(records, evidenceRecord{
						Type:     "finding",
						Severity: "warning",
						Summary:  risk,
						Data: map[string]any{
							"customer":     mapString(sub, "customer"),
							"subscription": mapString(sub, "id"),
						},
					})
				}
				if len(records) == 0 {
					records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "No collection-risk subscriptions found in the inspected window."})
				}
				return records, nil
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Upcoming renewal window in days")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum subscriptions to inspect")
	return cmd
}

func newInvestigateSubscriptionCancelRisk(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "subscription-cancel-risk",
		Short: "Find subscriptions likely to cancel, end trial, or stop billing soon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				params := url.Values{"status": []string{"all"}}
				shared.AddLimit(params, limit)
				if days > 0 {
					params.Set("current_period_end[lte]", strconv.FormatInt(time.Now().Add(time.Duration(days)*24*time.Hour).Unix(), 10))
				}
				subs, err := inv.list("/v1/subscriptions", params)
				if err != nil {
					return nil, err
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					risk := subscriptionCancelRisk(sub)
					if risk == "" {
						continue
					}
					records = append(records, entityRecord("subscription", sub))
					records = append(records, evidenceRecord{
						Type:     "finding",
						Severity: "warning",
						Summary:  risk,
						Data: map[string]any{
							"customer":     mapString(sub, "customer"),
							"subscription": mapString(sub, "id"),
						},
					})
				}
				if len(records) == 0 {
					records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "No subscription cancellation risks found in the inspected window."})
				}
				return records, nil
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Upcoming window in days")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum subscriptions to inspect")
	return cmd
}

func newInvestigateSubscriptionItems(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var subscription string
	cmd := &cobra.Command{
		Use:   "subscription-items",
		Short: "Show subscription items, prices, products, and product metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("subscription", subscription, "Provide a Subscription ID such as sub_...") {
				return nil
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.subscriptionItemsEvidence(subscription)
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	return cmd
}

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

func (i investigator) findSubscriptions(subscription, customer, metadata string, limit int) ([]map[string]any, error) {
	switch {
	case subscription != "":
		sub, err := i.get("/v1/subscriptions/"+url.PathEscape(subscription), url.Values{})
		if err != nil {
			return nil, err
		}
		return []map[string]any{sub}, nil
	case metadata != "":
		key, value, ok := strings.Cut(metadata, "=")
		if !ok || key == "" || value == "" {
			return nil, agenterrors.New("--metadata must be key=value", agenterrors.FixableByAgent).
				WithHint("Example: --metadata tenant_id=acme")
		}
		params := url.Values{"query": []string{"metadata['" + key + "']:'" + value + "'"}}
		shared.AddLimit(params, limit)
		return i.list("/v1/subscriptions/search", params)
	default:
		params := url.Values{}
		shared.AddLimit(params, limit)
		shared.AddString(params, "customer", customer)
		return i.list("/v1/subscriptions", params)
	}
}

func (i investigator) subscriptionItemsEvidence(subscriptionID string) ([]evidenceRecord, error) {
	records := []evidenceRecord{}
	sub, err := i.get("/v1/subscriptions/"+url.PathEscape(subscriptionID), url.Values{})
	if err != nil {
		return nil, err
	}
	records = append(records, entityRecord("subscription", sub))

	items, err := i.list("/v1/subscription_items", url.Values{"subscription": []string{subscriptionID}, "limit": []string{"100"}})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		records = append(records, entityRecord("subscription_item", item))
		price := mapAnyMap(item, "price")
		if productID := idFromValue(price["product"]); productID != "" {
			if product, productErr := i.get("/v1/products/"+url.PathEscape(productID), url.Values{}); productErr == nil {
				records = append(records, entityRecord("product", product))
			}
		}
	}
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  fmt.Sprintf("Subscription %s has %d visible item(s). Use price/product metadata for internal product mapping.", subscriptionID, len(items)),
		Data: map[string]any{
			"subscription": subscriptionID,
			"item_count":   len(items),
		},
	})
	return records, nil
}

func (i investigator) subscriptionAmountChange(subscriptionID string) ([]evidenceRecord, error) {
	records, err := i.subscriptionItemsEvidence(subscriptionID)
	if err != nil {
		return nil, err
	}
	sub := firstEntityData(records, "subscription")
	if sub == nil {
		return records, nil
	}

	var latestInvoice map[string]any
	if latestInvoiceID := idFromValue(sub["latest_invoice"]); latestInvoiceID != "" {
		invoice, invoiceErr := i.get("/v1/invoices/"+url.PathEscape(latestInvoiceID), url.Values{})
		if invoiceErr != nil {
			records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Could not retrieve latest invoice " + latestInvoiceID + ": " + invoiceErr.Error()})
		} else {
			latestInvoice = invoice
			records = append(records, entityRecord("invoice", invoice))
			lines, lineErr := i.list("/v1/invoices/"+url.PathEscape(latestInvoiceID)+"/lines", url.Values{"limit": []string{"100"}})
			if lineErr == nil {
				records = appendListRecords(records, "line_item", lines)
			}
		}
	}

	var preview map[string]any
	if next, previewErr := i.postForm("/v1/invoices/create_preview", url.Values{"subscription": []string{subscriptionID}}); previewErr == nil {
		preview = next
		records = append(records, entityRecord("invoice_preview", next))
	}

	items := entityDataByObject(records, "subscription_item")
	records = append(records, subscriptionAmountFinding(subscriptionID, latestInvoice, preview, items))
	return records, nil
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

func firstEntityData(records []evidenceRecord, object string) map[string]any {
	for _, record := range records {
		if record.Type == "entity" && record.Object == object {
			return record.Data
		}
	}
	return nil
}

func entityDataByObject(records []evidenceRecord, object string) []map[string]any {
	items := []map[string]any{}
	for _, record := range records {
		if record.Type == "entity" && record.Object == object {
			items = append(items, record.Data)
		}
	}
	return items
}

func (i investigator) subscriptionPaymentSummary(sub map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	if latestInvoiceID := idFromValue(sub["latest_invoice"]); latestInvoiceID != "" {
		invoiceRecords, err := i.invoicePayment(latestInvoiceID)
		if err == nil {
			records = append(records, invoiceRecords...)
		} else {
			records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Could not retrieve latest invoice " + latestInvoiceID + ": " + err.Error()})
		}
	}
	nextAmount := "unknown"
	if preview, err := i.postForm("/v1/invoices/create_preview", url.Values{"subscription": []string{mapString(sub, "id")}}); err == nil {
		records = append(records, entityRecord("invoice_preview", preview))
		nextAmount = formatAmount(preview)
	}
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary: fmt.Sprintf("Subscription %s last invoice is %s; next renewal is at %v and preview amount is %s.",
			mapString(sub, "id"), idFromValue(sub["latest_invoice"]), mapValue(sub, "current_period_end"), nextAmount),
	})
	return records
}

func (i investigator) collectionRisk(sub map[string]any) string {
	status := mapString(sub, "status")
	if status == "past_due" || status == "unpaid" || status == "incomplete" {
		return fmt.Sprintf("Customer %s has subscription %s in %s status; outreach about payment details is recommended.", mapString(sub, "customer"), mapString(sub, "id"), status)
	}
	pmID := idFromValue(sub["default_payment_method"])
	if pmID == "" {
		if customer, err := i.get("/v1/customers/"+url.PathEscape(mapString(sub, "customer")), url.Values{}); err == nil {
			settings := mapAnyMap(customer, "invoice_settings")
			pmID = idFromValue(settings["default_payment_method"])
		}
		if pmID == "" {
			return fmt.Sprintf("Customer %s has subscription %s with no default payment method visible.", mapString(sub, "customer"), mapString(sub, "id"))
		}
	}
	if pmID != "" {
		if pm, err := i.get("/v1/payment_methods/"+url.PathEscape(pmID), url.Values{}); err == nil && cardExpiresSoon(pm, time.Now()) {
			return fmt.Sprintf("Customer %s has subscription %s with card payment method %s expiring soon.", mapString(sub, "customer"), mapString(sub, "id"), pmID)
		}
	}
	if invoiceID := idFromValue(sub["latest_invoice"]); invoiceID != "" {
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
		if err == nil && !mapBool(invoice, "paid") && mapString(invoice, "status") == "open" {
			return fmt.Sprintf("Customer %s has an open unpaid invoice %s for subscription %s.", mapString(sub, "customer"), invoiceID, mapString(sub, "id"))
		}
		if err == nil && idFromValue(invoice["payment_intent"]) != "" {
			pi, piErr := i.paymentIntentForInvoice(invoice)
			if piErr == nil && mapString(pi, "status") == "requires_action" {
				return fmt.Sprintf("Customer %s has subscription %s with latest invoice PaymentIntent requiring action.", mapString(sub, "customer"), mapString(sub, "id"))
			}
		}
	}
	return ""
}

func subscriptionCancelRisk(sub map[string]any) string {
	status := mapString(sub, "status")
	if status == "canceled" || status == "unpaid" {
		return fmt.Sprintf("Subscription %s for customer %s is %s.", mapString(sub, "id"), mapString(sub, "customer"), status)
	}
	if mapBool(sub, "cancel_at_period_end") {
		return fmt.Sprintf("Subscription %s for customer %s is set to cancel at period end %v.", mapString(sub, "id"), mapString(sub, "customer"), mapValue(sub, "current_period_end"))
	}
	if trialEnd, ok := mapInt64(sub, "trial_end"); ok && trialEnd > 0 {
		return fmt.Sprintf("Subscription %s for customer %s trial ends at %d.", mapString(sub, "id"), mapString(sub, "customer"), trialEnd)
	}
	return ""
}

func cardExpiresSoon(paymentMethod map[string]any, now time.Time) bool {
	card := mapAnyMap(paymentMethod, "card")
	year, yearOK := mapInt64(card, "exp_year")
	month, monthOK := mapInt64(card, "exp_month")
	if !yearOK || !monthOK || month < 1 || month > 12 {
		return false
	}
	expiry := time.Date(int(year), time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
	return expiry.Before(now.AddDate(0, 2, 0))
}
