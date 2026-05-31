package cli

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateInvoiceCollection(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "invoice-collection <invoice-id|customer-id|subscription-id>",
		Short: "Explain failed or pending invoice collection and retry state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.invoiceCollection(args[0], limit)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum invoices to inspect for customer or subscription input")
	return cmd
}

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

func newInvestigateEntitlement(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var customer, subscription, invoice, checkoutSession, metadata string
	var limit int
	cmd := &cobra.Command{
		Use:   "entitlement",
		Short: "Find subscription, invoice, or checkout product metadata for entitlement mismatches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.entitlement(entitlementQuery{
					customer:        customer,
					subscription:    subscription,
					invoice:         invoice,
					checkoutSession: checkoutSession,
					metadata:        metadata,
					limit:           limit,
				})
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	cmd.Flags().StringVar(&invoice, "invoice", "", "Invoice ID")
	cmd.Flags().StringVar(&checkoutSession, "checkout-session", "", "Checkout Session ID")
	cmd.Flags().StringVar(&metadata, "metadata", "", "Subscription metadata equality filter as key=value")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum matching subscriptions or invoices to inspect")
	return cmd
}

func (i investigator) invoiceCollection(id string, limit int) ([]evidenceRecord, error) {
	if err := validateAllowedStripeID(id, "invoice", "customer", "subscription"); err != nil {
		return nil, err
	}
	invoices := []map[string]any{}
	switch {
	case strings.HasPrefix(id, "in_"):
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(id), url.Values{})
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	case strings.HasPrefix(id, "sub_"):
		found, err := i.list("/v1/invoices", valuesWithLimit(limit, "subscription", id))
		if err != nil {
			return nil, err
		}
		invoices = found
	default:
		found, err := i.list("/v1/invoices", valuesWithLimit(limit, "customer", id))
		if err != nil {
			return nil, err
		}
		invoices = found
	}

	records := []evidenceRecord{}
	for _, invoice := range invoices {
		records = append(records, entityRecord("invoice", invoice))
		if pi, err := i.paymentIntentForInvoice(invoice); err == nil && pi != nil {
			records = append(records, entityRecord("payment_intent", pi))
			if charge, err := i.latestChargeForPaymentIntent(pi); err == nil && charge != nil {
				records = append(records, entityRecord("charge", charge))
			}
		}
		records = append(records, invoiceCollectionFinding(invoice))
	}
	if len(records) == 0 {
		records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No invoices matched the supplied collection target."})
	}
	return records, nil
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
		records = append(records, entityRecord("customer", customer))
		methods, err := i.list("/v1/payment_methods", valuesWithLimit(10, "customer", id, "type", "card"))
		if err != nil {
			return nil, err
		}
		paymentMethods = methods
	}
	for _, pm := range paymentMethods {
		records = append(records, entityRecord("payment_method", pm))
		records = append(records, paymentMethodReadinessFinding(customerID, pm))
		if setupIntents, err := i.list("/v1/setup_intents", valuesWithLimit(3, "payment_method", mapString(pm, "id"))); err == nil {
			records = appendListRecords(records, "setup_intent", setupIntents)
		}
	}
	if len(paymentMethods) == 0 {
		records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "No visible saved card payment methods found for customer " + customerID + "."})
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

type entitlementQuery struct {
	customer        string
	subscription    string
	invoice         string
	checkoutSession string
	metadata        string
	limit           int
}

func (i investigator) entitlement(q entitlementQuery) ([]evidenceRecord, error) {
	records := []evidenceRecord{}
	if q.subscription != "" || q.customer != "" || q.metadata != "" {
		subs, err := i.findSubscriptions(q.subscription, q.customer, q.metadata, q.limit)
		if err != nil {
			return nil, err
		}
		for _, sub := range subs {
			records = append(records, entityRecord("subscription", sub))
			if bundle, err := i.subscriptionItemsBundle(mapString(sub, "id")); err == nil {
				records = append(records, bundle.records...)
			}
		}
	}
	if q.invoice != "" {
		if err := validateExpectedStripeID(q.invoice, "invoice"); err != nil {
			return nil, err
		}
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(q.invoice), url.Values{})
		if err != nil {
			return nil, err
		}
		records = append(records, entityRecord("invoice", invoice))
		records = append(records, i.invoiceLineEntitlements(q.invoice)...)
	}
	if q.checkoutSession != "" {
		sessionRecords, err := i.checkoutSession(q.checkoutSession)
		if err != nil {
			return nil, err
		}
		records = append(records, sessionRecords...)
	}
	if len(records) == 0 {
		return []evidenceRecord{{Type: "finding", Severity: "warning", Summary: "Provide --subscription, --customer, --metadata, --invoice, or --checkout-session to investigate entitlements."}}, nil
	}
	records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "Entitlement evidence gathered from subscription items, invoice lines, checkout line items, prices, and products. Prefer product/price metadata for internal product IDs."})
	return records, nil
}

func (i investigator) invoiceLineEntitlements(invoiceID string) []evidenceRecord {
	lines, err := i.list("/v1/invoices/"+url.PathEscape(invoiceID)+"/lines", url.Values{"limit": []string{"100"}})
	if err != nil {
		return []evidenceRecord{relatedWarning("invoice lines", err)}
	}
	return appendListRecords(nil, "line_item", lines)
}
