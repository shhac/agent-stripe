package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateCustomerContext(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "customer-context",
		Short: "Gather payment, invoice, subscription, and risk context for a customer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("customer", customer, "Provide a Customer ID such as cus_...") {
				return nil
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.customerContext(customer, limit)
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum recent objects per collection")
	return cmd
}

func (i investigator) customerContext(customer string, limit int) ([]evidenceRecord, error) {
	if err := validateExpectedStripeID(customer, "customer"); err != nil {
		return nil, err
	}
	records := []evidenceRecord{}
	customerObj, err := i.get("/v1/customers/"+url.PathEscape(customer), url.Values{})
	if err != nil {
		return nil, err
	}
	records = append(records, entityRecord("customer", customerObj))

	records, _ = i.appendRelatedList(records, "payment_method", "/v1/payment_methods", valuesWithLimit(limit, "customer", customer, "type", "card"))
	records, _ = i.appendRelatedList(records, "subscription", "/v1/subscriptions", valuesWithLimit(limit, "customer", customer, "status", "all"))
	records, _ = i.appendRelatedList(records, "invoice", "/v1/invoices", valuesWithLimit(limit, "customer", customer))
	records, _ = i.appendRelatedList(records, "payment_intent", "/v1/payment_intents", valuesWithLimit(limit, "customer", customer))
	records, charges := i.appendRelatedList(records, "charge", "/v1/charges", valuesWithLimit(limit, "customer", customer))
	for _, charge := range charges {
		records, _ = i.appendRelatedList(records, "dispute", "/v1/disputes", valuesWithLimit(limit, "charge", mapString(charge, "id")))
		records, _ = i.appendRelatedList(records, "refund", "/v1/refunds", valuesWithLimit(limit, "charge", mapString(charge, "id")))
	}
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  "Customer context gathered. Use entity records for recent payment methods, subscriptions, invoices, payment intents, charges, disputes, and refunds.",
		Data: map[string]any{
			"customer": customer,
		},
	})
	return records, nil
}

func (i investigator) appendRelatedList(records []evidenceRecord, object, path string, params url.Values) ([]evidenceRecord, []map[string]any) {
	items, err := i.list(path, params)
	if err != nil {
		return append(records, evidenceRecord{
			Type:     "finding",
			Severity: "warning",
			Summary:  "Could not gather " + object + " context from " + path + "; continuing with available evidence.",
			Data: map[string]any{
				"object": object,
				"path":   path,
				"error":  err.Error(),
			},
		}), nil
	}
	return appendListRecords(records, object, items), items
}

func appendListRecords(records []evidenceRecord, object string, items []map[string]any) []evidenceRecord {
	for _, item := range items {
		records = append(records, entityRecord(object, item))
	}
	return records
}

func relatedWarning(name string, err error) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Severity: "warning",
		Summary:  "Could not gather " + name + "; continuing with available evidence.",
		Data: map[string]any{
			"error": err.Error(),
		},
	}
}

func valuesWithLimit(limit int, pairs ...string) url.Values {
	params := url.Values{}
	shared.AddLimit(params, limit)
	for idx := 0; idx+1 < len(pairs); idx += 2 {
		shared.AddString(params, pairs[idx], pairs[idx+1])
	}
	return params
}
