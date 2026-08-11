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
			if product := i.followRef(price, "product"); product != nil {
				i.add(entityRecord("product", product))
			}
		}
	}
}

func (i investigator) relatedCheckoutObjects(session map[string]any) {
	if customer := i.followRef(session, "customer"); customer != nil {
		i.add(entityRecord("customer", customer))
	}
	if pi := i.followRef(session, "payment_intent"); pi != nil {
		i.add(entityRecord("payment_intent", pi))
		if charge, err := i.latestChargeForPaymentIntent(pi); err == nil && charge != nil {
			i.add(entityRecord("charge", charge))
		}
	}
	if sub := i.followRef(session, "subscription"); sub != nil {
		i.add(entityRecord("subscription", sub))
		i.subscriptionPaymentSummary(sub)
	}
	if invoice := i.followRef(session, "invoice"); invoice != nil {
		i.add(entityRecord("invoice", invoice))
	}
	if link := i.followRef(session, "payment_link"); link != nil {
		i.add(entityRecord("payment_link", link))
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
