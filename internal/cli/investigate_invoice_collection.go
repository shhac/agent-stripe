package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateInvoiceCollection(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "invoice-collection <invoice-id|customer-id|subscription-id>",
		Short: "Explain failed or pending invoice collection and retry state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.invoiceCollection(args[0], limit)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum invoices to inspect for customer or subscription input")
	return cmd
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
		i.add(entityRecord("invoice", invoice))
		if pi, err := i.paymentIntentForInvoice(invoice); err == nil && pi != nil {
			i.add(entityRecord("payment_intent", pi))
			if charge, err := i.latestChargeForPaymentIntent(pi); err == nil && charge != nil {
				i.add(entityRecord("charge", charge))
			}
		}
		i.add(invoiceCollectionFinding(invoice))
	}
	if len(invoices) == 0 {
		i.add(evidenceRecord{Type: "finding", Severity: "warning", Summary: "No invoices matched the supplied collection target."})
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
