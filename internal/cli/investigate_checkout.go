package cli

import (
	"fmt"
	"net/url"
)

var checkoutSessionInvestigation = investigationSpec{
	use:   "checkout-session <checkout-session-id>",
	short: "Explain Checkout completion, line items, and resulting payment or subscription",
	run:   investigator.checkoutSession,
}

func (i investigator) checkoutSession(sessionID string) error {
	if err := validateExpectedStripeID(sessionID, "checkout_session"); err != nil {
		return err
	}
	session, err := i.get("/v1/checkout/sessions/"+url.PathEscape(sessionID), url.Values{})
	if err != nil {
		return err
	}
	i.checkoutLineItems(sessionID)
	i.relatedCheckoutObjects(session)
	i.add(checkoutSessionFinding(session))
	return nil
}

func (i investigator) checkoutLineItems(sessionID string) {
	// Fetched without auto-recording: Stripe calls these "item", the
	// investigation calls them "line_item", and letting both labels through
	// emitted the same object twice.
	items, err := i.fetchList("/v1/checkout/sessions/"+url.PathEscape(sessionID)+"/line_items", url.Values{"limit": []string{"100"}})
	if err != nil {
		i.add(relatedWarning("checkout line items", err))
		return
	}
	for _, item := range items {
		i.add(entityRecord("line_item", item))
		// The price is embedded in the line item rather than fetched, so this
		// add is its only writer.
		if price := mapAnyMap(item, "price"); len(price) > 0 {
			i.add(entityRecord("price", price))
			i.followRef(price, "product")
		}
	}
}

func (i investigator) relatedCheckoutObjects(session map[string]any) {
	i.followRef(session, "customer")
	if pi := i.followRef(session, "payment_intent"); pi != nil {
		i.addLatestCharge(pi)
	}
	if sub := i.followRef(session, "subscription"); sub != nil {
		i.subscriptionPaymentSummary(sub)
	}
	i.followRef(session, "invoice")
	i.followRef(session, "payment_link")
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
