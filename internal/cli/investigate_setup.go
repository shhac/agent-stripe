package cli

import (
	"fmt"
	"net/url"
	"strings"
)

var setupInvestigation = investigationSpec{
	use:          "setup <setup-intent-id|payment-method-id|customer-id>",
	short:        "Explain saved-payment setup status and reusable payment method readiness",
	runWithLimit: investigator.setup,
	limitDefault: 5,
	limitHelp:    "Maximum SetupIntents to inspect for customer or payment method input",
}

func (i investigator) setup(id string, limit int) error {
	if err := validateAllowedStripeID(id, "setup_intent", "payment_method", "customer"); err != nil {
		return err
	}
	setupIntents := []map[string]any{}
	switch {
	case strings.HasPrefix(id, "seti_"):
		seti, err := i.get("/v1/setup_intents/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		setupIntents = append(setupIntents, seti)
	case strings.HasPrefix(id, "pm_"):
		pm, err := i.get("/v1/payment_methods/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("payment_method", pm))
		found, err := i.list("/v1/setup_intents", valuesWithLimit(limit, "payment_method", id))
		if err != nil {
			return err
		}
		setupIntents = found
	default:
		customer, err := i.get("/v1/customers/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		i.add(entityRecord("customer", customer))
		found, err := i.list("/v1/setup_intents", valuesWithLimit(limit, "customer", id))
		if err != nil {
			return err
		}
		setupIntents = found
	}
	for _, seti := range setupIntents {
		i.add(entityRecord("setup_intent", seti))
		if pm := i.followRef(seti, "payment_method"); pm != nil {
			i.add(entityRecord("payment_method", pm))
		}
		i.add(setupFinding(seti))
	}
	if len(setupIntents) == 0 {
		i.add(evidenceRecord{Type: "finding", Severity: "warning", Summary: "No SetupIntents found for " + id + "."})
	}
	return nil
}

func setupFinding(seti map[string]any) evidenceRecord {
	status := mapString(seti, "status")
	severity := "info"
	if status != "succeeded" {
		severity = "warning"
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: fmt.Sprintf("SetupIntent %s status=%s usage=%s.", mapString(seti, "id"), status, mapString(seti, "usage")), Data: map[string]any{
		"setup_intent":     mapString(seti, "id"),
		"customer":         idFromValue(seti["customer"]),
		"payment_method":   idFromValue(seti["payment_method"]),
		"status":           status,
		"usage":            mapString(seti, "usage"),
		"last_setup_error": mapAnyMap(seti, "last_setup_error"),
	}}
}
