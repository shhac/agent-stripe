package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateCheckoutSession(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "checkout-session <checkout-session-id>",
		Short: "Explain Checkout completion, line items, and resulting payment or subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.checkoutSession(args[0])
			})
		},
	}
}

func (i investigator) checkoutSession(sessionID string) error {
	if err := validateExpectedStripeID(sessionID, "checkout_session"); err != nil {
		return err
	}
	session, err := i.get("/v1/checkout/sessions/"+url.PathEscape(sessionID), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("checkout.session", session))
	i.checkoutLineItems(sessionID)
	i.relatedCheckoutObjects(session)
	i.add(checkoutSessionFinding(session))
	return nil
}

func (i investigator) checkoutLineItems(sessionID string) {
	items, err := i.list("/v1/checkout/sessions/"+url.PathEscape(sessionID)+"/line_items", url.Values{"limit": []string{"100"}})
	if err != nil {
		i.add(relatedWarning("checkout line items", err))
		return
	}
	for _, item := range items {
		i.add(entityRecord("line_item", item))
		if price := mapAnyMap(item, "price"); len(price) > 0 {
			i.add(entityRecord("price", price))
			if productID := idFromValue(price["product"]); productID != "" {
				if product, err := i.get("/v1/products/"+url.PathEscape(productID), url.Values{}); err == nil {
					i.add(entityRecord("product", product))
				}
			}
		}
	}
}

func (i investigator) relatedCheckoutObjects(session map[string]any) {
	if customerID := idFromValue(session["customer"]); customerID != "" {
		if customer, err := i.get("/v1/customers/"+url.PathEscape(customerID), url.Values{}); err == nil {
			i.add(entityRecord("customer", customer))
		}
	}
	if piID := idFromValue(session["payment_intent"]); piID != "" {
		if pi, err := i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{}); err == nil {
			i.add(entityRecord("payment_intent", pi))
			if charge, err := i.latestChargeForPaymentIntent(pi); err == nil && charge != nil {
				i.add(entityRecord("charge", charge))
			}
		}
	}
	if subID := idFromValue(session["subscription"]); subID != "" {
		if sub, err := i.get("/v1/subscriptions/"+url.PathEscape(subID), url.Values{}); err == nil {
			i.add(entityRecord("subscription", sub))
			i.subscriptionPaymentSummary(sub)
		}
	}
	if invoiceID := idFromValue(session["invoice"]); invoiceID != "" {
		if invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{}); err == nil {
			i.add(entityRecord("invoice", invoice))
		}
	}
	if linkID := idFromValue(session["payment_link"]); linkID != "" {
		if link, err := i.get("/v1/payment_links/"+url.PathEscape(linkID), url.Values{}); err == nil {
			i.add(entityRecord("payment_link", link))
		}
	}
}

func checkoutSessionFinding(session map[string]any) evidenceRecord {
	severity := "info"
	if mapString(session, "payment_status") != "paid" && mapString(session, "status") != "complete" {
		severity = "warning"
	}
	summary := fmt.Sprintf("Checkout Session %s mode=%s status=%s payment_status=%s amount_total=%s.",
		mapString(session, "id"),
		mapString(session, "mode"),
		mapString(session, "status"),
		mapString(session, "payment_status"),
		formatAmount(map[string]any{"amount": mapValue(session, "amount_total"), "currency": mapValue(session, "currency")}),
	)
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"checkout_session": mapString(session, "id"),
			"customer":         idFromValue(session["customer"]),
			"payment_intent":   idFromValue(session["payment_intent"]),
			"subscription":     idFromValue(session["subscription"]),
			"invoice":          idFromValue(session["invoice"]),
			"payment_link":     idFromValue(session["payment_link"]),
		},
	}
}
