package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigatePaymentMethodReadiness(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "payment-method-readiness <customer-id|payment-method-id>",
		Short: "Check whether a customer has usable saved payment details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.paymentMethodReadiness(args[0])
			})
		},
	}
}

func (i investigator) paymentMethodReadiness(id string) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "customer", "payment_method"); err != nil {
		return nil, err
	}
	records := []evidenceRecord{}
	customerID := ""
	paymentMethods := []map[string]any{}
	if strings.HasPrefix(id, "pm_") {
		pm, err := i.get("/v1/payment_methods/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		paymentMethods = append(paymentMethods, pm)
		customerID = idFromValue(pm["customer"])
	} else {
		customerID = id
		customer, err := i.get("/v1/customers/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		records = i.appendEvidence(records, entityRecord("customer", customer))
		methods, err := i.list("/v1/payment_methods", valuesWithLimit(10, "customer", id, "type", "card"))
		if err != nil {
			return nil, err
		}
		paymentMethods = methods
	}
	for _, pm := range paymentMethods {
		records = i.appendEvidence(records, entityRecord("payment_method", pm))
		records = i.appendEvidence(records, paymentMethodReadinessFinding(customerID, pm))
		if setupIntents, err := i.list("/v1/setup_intents", valuesWithLimit(3, "payment_method", mapString(pm, "id"))); err == nil {
			records = i.appendListRecords(records, "setup_intent", setupIntents)
		}
	}
	if len(paymentMethods) == 0 {
		records = i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No visible saved card payment methods found for customer " + customerID + "."})
	}
	return records, nil
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
