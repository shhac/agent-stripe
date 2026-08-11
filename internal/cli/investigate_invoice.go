package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

var invoicePaymentInvestigation = investigationSpec{
	use:   "invoice-payment <invoice-id>",
	short: "Explain how an invoice was paid, including card last4 when available",
	run:   investigator.invoicePayment,
}

func newInvestigateInvoiceMetadata(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var number string
	cmd := &cobra.Command{
		Use:   "invoice-metadata [invoice-id]",
		Short: "Find PaymentIntent metadata from an invoice ID or invoice number",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				invoiceID := ""
				if len(args) == 1 {
					invoiceID = args[0]
				}
				if invoiceID == "" {
					if err := shared.RequireFlag("number", number, "Use --number when the customer sent an invoice number instead of an invoice ID"); err != nil {
						return err
					}
				}
				return inv.invoiceMetadata(invoiceID, number)
			})
		},
	}
	cmd.Flags().StringVar(&number, "number", "", "Invoice number from a customer copy")
	return cmd
}

func (i investigator) invoiceMetadata(invoiceID, number string) error {
	if invoiceID == "" {
		found, err := i.list("/v1/invoices/search", url.Values{"query": []string{stripeSearchEquals("number", number)}, "limit": []string{"1"}})
		if err != nil {
			return err
		}
		if len(found) == 0 {
			i.add(finding("warning", "No invoice matched number "+number+"."))
			return nil
		}
		invoiceID = mapString(found[0], "id")
	}
	if err := validateExpectedStripeID(invoiceID, "invoice"); err != nil {
		return err
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return err
	}
	pi, err := i.paymentIntentForInvoice(invoice)
	if err != nil {
		return err
	}
	if pi == nil {
		i.add(finding("warning", "Invoice has no PaymentIntent."))
		return nil
	}
	i.add(paymentIntentMetadataFinding(pi))
	return nil
}

func paymentIntentMetadataFinding(pi map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  "PaymentIntent metadata is available for internal product lookup.",
		Data: map[string]any{
			"payment_intent": mapString(pi, "id"),
			"metadata":       mapAnyMap(pi, "metadata"),
		},
	}
}

func (i investigator) invoicePayment(invoiceID string) error {
	if err := validateExpectedStripeID(invoiceID, "invoice"); err != nil {
		return err
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return err
	}
	pi, err := i.paymentIntentForInvoice(invoice)
	if err != nil {
		return err
	}
	if pi == nil {
		i.add(finding("warning", "Invoice has no PaymentIntent, so no card details are available from a charge."))
		return nil
	}
	charge, err := i.latestChargeForPaymentIntent(pi)
	if err != nil {
		return err
	}
	i.add(finding(severityForPayment(pi, charge), invoicePaymentSummary(invoice, pi, charge)))
	return nil
}

func (i investigator) paymentIntentForInvoice(invoice map[string]any) (map[string]any, error) {
	piID := idFromValue(invoice["payment_intent"])
	if piID == "" {
		return nil, nil
	}
	if pi, ok := invoice["payment_intent"].(map[string]any); ok {
		return pi, nil
	}
	return i.get("/v1/payment_intents/"+url.PathEscape(piID), url.Values{})
}
