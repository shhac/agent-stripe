package cli

import (
	"fmt"
	"net/url"
	"strings"
)

var paymentMethodReadinessInvestigation = investigationSpec{
	use:   "payment-method-readiness <customer-id|payment-method-id>",
	short: "Check whether a customer has usable saved payment details",
	run:   investigator.paymentMethodReadiness,
}

func (i investigator) paymentMethodReadiness(id string) error {
	if err := validateAllowedStripeID(id, "customer", "payment_method"); err != nil {
		return err
	}
	customerID := ""
	paymentMethods := []map[string]any{}
	if strings.HasPrefix(id, "pm_") {
		pm, err := i.get("/v1/payment_methods/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		paymentMethods = append(paymentMethods, pm)
		customerID = idFromValue(pm["customer"])
	} else {
		customerID = id
		customer, err := i.get("/v1/customers/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("customer", customer))
		methods, err := i.list("/v1/payment_methods", valuesWithLimit(10, "customer", id, "type", "card"))
		if err != nil {
			return err
		}
		paymentMethods = methods
	}
	for _, pm := range paymentMethods {
		i.add(entityRecord("payment_method", pm))
		i.add(paymentMethodReadinessFinding(customerID, pm))
		if setupIntents := i.listRelated("setup intents", "/v1/setup_intents", valuesWithLimit(3, "payment_method", mapString(pm, "id"))); setupIntents != nil {
			i.addList("setup_intent", setupIntents)
		}
	}
	if len(paymentMethods) == 0 {
		i.add(finding("warning", "No visible saved card payment methods found for customer "+customerID+"."))
	}
	return nil
}

func paymentMethodReadinessFinding(customerID string, pm map[string]any) evidenceRecord {
	card := mapAnyMap(pm, "card")
	expMonth, _ := mapInt64(card, "exp_month")
	expYear, _ := mapInt64(card, "exp_year")
	severity := "info"
	summary := fmt.Sprintf("PaymentMethod %s is attached to customer %s and card last4=%s exp=%02d/%d.",
		mapString(pm, "id"), firstNonEmpty(customerID, idFromValue(pm["customer"])), mapString(card, "last4"), expMonth, expYear)
	if mapString(pm, "customer") == "" && customerID == "" {
		severity = "warning"
		summary = "PaymentMethod " + mapString(pm, "id") + " is not attached to a customer."
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"customer":       firstNonEmpty(customerID, idFromValue(pm["customer"])),
			"payment_method": mapString(pm, "id"),
			"brand":          mapString(card, "brand"),
			"last4":          mapString(card, "last4"),
			"exp_month":      expMonth,
			"exp_year":       expYear,
		},
	}
}
