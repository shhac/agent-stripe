package cli

import (
	"fmt"
	"net/url"
	"strings"
)

var invoiceCollectionInvestigation = investigationSpec{
	use:          "invoice-collection <invoice-id|customer-id|subscription-id>",
	short:        "Explain failed or pending invoice collection and retry state",
	runWithLimit: investigator.invoiceCollection,
	limitDefault: 5,
	limitHelp:    "Maximum invoices to inspect for customer or subscription input",
}

func (i investigator) invoiceCollection(id string, limit int) error {
	if err := validateAllowedStripeID(id, "invoice", "customer", "subscription"); err != nil {
		return err
	}
	invoices := []map[string]any{}
	switch {
	case strings.HasPrefix(id, "in_"):
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return err
		}
		invoices = append(invoices, invoice)
	case strings.HasPrefix(id, "sub_"):
		found, err := i.list("/v1/invoices", valuesWithLimit(limit, "subscription", id))
		if err != nil {
			return err
		}
		invoices = found
	default:
		found, err := i.list("/v1/invoices", valuesWithLimit(limit, "customer", id))
		if err != nil {
			return err
		}
		invoices = found
	}

	for _, invoice := range invoices {
		if pi := i.addInvoicePaymentIntent(invoice); pi != nil {
			i.addLatestCharge(pi)
		}
		i.add(invoiceCollectionFinding(invoice))
	}
	if len(invoices) == 0 {
		i.add(finding("warning", "No invoices matched the supplied collection target."))
	}
	return nil
}

func invoiceCollectionFinding(invoice map[string]any) evidenceRecord {
	severity := "info"
	if !mapBool(invoice, "paid") || mapString(invoice, "status") == "open" || mapString(invoice, "status") == "uncollectible" {
		severity = "warning"
	}
	nextAttempt, _ := mapInt64(invoice, "next_payment_attempt")
	attemptCount, _ := mapInt64(invoice, "attempt_count")
	summary := fmt.Sprintf("Invoice %s status=%s paid=%t amount_due=%s attempt_count=%d.",
		mapString(invoice, "id"), mapString(invoice, "status"), mapBool(invoice, "paid"), formatAmount(invoice), attemptCount)
	if nextAttempt > 0 {
		summary += fmt.Sprintf(" Next payment attempt is at Unix time %d.", nextAttempt)
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"invoice":              mapString(invoice, "id"),
			"customer":             idFromValue(invoice["customer"]),
			"subscription":         idFromValue(invoice["subscription"]),
			"payment_intent":       idFromValue(invoice["payment_intent"]),
			"attempt_count":        attemptCount,
			"next_payment_attempt": nextAttempt,
			"hosted_invoice_url":   mapString(invoice, "hosted_invoice_url"),
		},
	}
}
