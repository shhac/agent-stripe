package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateInvoicePayment(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "invoice-payment <invoice-id>",
		Short: "Explain how an invoice was paid, including card last4 when available",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.invoicePayment(args[0])
			})
		},
	}
}

func newInvestigateInvoiceMetadata(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var number string
	cmd := &cobra.Command{
		Use:   "invoice-metadata [invoice-id]",
		Short: "Find PaymentIntent metadata from an invoice ID or invoice number",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				invoiceID := ""
				if len(args) == 1 {
					invoiceID = args[0]
				}
				if invoiceID == "" {
					if err := shared.RequireFlag("number", number, "Use --number when the customer sent an invoice number instead of an invoice ID"); err != nil {
						return nil, err
					}
				}
				return inv.invoiceMetadata(invoiceID, number)
			})
		},
	}
	cmd.Flags().StringVar(&number, "number", "", "Invoice number from a customer copy")
	return cmd
}

func (i investigator) invoiceMetadata(invoiceID, number string) ([]evidenceRecord, error) {
	if invoiceID == "" {
		found, err := i.list("/v1/invoices/search", url.Values{"query": []string{stripeSearchEquals("number", number)}, "limit": []string{"1"}})
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return i.appendEvidence(nil, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No invoice matched number " + number + "."}), nil
		}
		invoiceID = mapString(found[0], "id")
	}
	if err := validateExpectedStripeID(invoiceID, "invoice"); err != nil {
		return nil, err
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := i.appendEvidence(nil, entityRecord("invoice", invoice))
	pi, err := i.paymentIntentForInvoice(invoice)
	if err != nil {
		return nil, err
	}
	if pi == nil {
		return i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Invoice has no PaymentIntent."}), nil
	}
	records = i.appendEvidence(records, entityRecord("payment_intent", pi))
	records = i.appendEvidence(records, paymentIntentMetadataFinding(pi))
	return records, nil
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

func (i investigator) invoicePayment(invoiceID string) ([]evidenceRecord, error) {
	if err := validateExpectedStripeID(invoiceID, "invoice"); err != nil {
		return nil, err
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return nil, err
	}
	records := i.appendEvidence(nil, entityRecord("invoice", invoice))
	pi, err := i.paymentIntentForInvoice(invoice)
	if err != nil {
		return nil, err
	}
	if pi == nil {
		return i.appendEvidence(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Invoice has no PaymentIntent, so no card details are available from a charge."}), nil
	}
	records = i.appendEvidence(records, entityRecord("payment_intent", pi))
	charge, err := i.latestChargeForPaymentIntent(pi)
	if err != nil {
		return nil, err
	}
	if charge != nil {
		records = i.appendEvidence(records, entityRecord("charge", charge))
	}
	records = i.appendEvidence(records, evidenceRecord{
		Type:     "finding",
		Severity: severityForPayment(pi, charge),
		Summary:  invoicePaymentSummary(invoice, pi, charge),
	})
	return records, nil
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
