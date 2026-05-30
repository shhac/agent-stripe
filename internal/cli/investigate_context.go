package cli

import (
	"context"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateCustomerContext(globals shared.GlobalsFunc) *cobra.Command {
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "customer-context",
		Short: "Gather payment, invoice, subscription, and risk context for a customer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("customer", customer, "Provide a Customer ID such as cus_...") {
				return nil
			}
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				return inv.customerContext(customer, limit)
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum recent objects per collection")
	return cmd
}

func (i investigator) customerContext(customer string, limit int) ([]evidenceRecord, error) {
	records := []evidenceRecord{}
	customerObj, err := i.get("/v1/customers/"+url.PathEscape(customer), url.Values{})
	if err != nil {
		return nil, err
	}
	records = append(records, entityRecord("customer", customerObj))

	records = appendListRecords(records, "payment_method", i.mustList("/v1/payment_methods", valuesWithLimit(limit, "customer", customer, "type", "card")))
	records = appendListRecords(records, "subscription", i.mustList("/v1/subscriptions", valuesWithLimit(limit, "customer", customer, "status", "all")))
	records = appendListRecords(records, "invoice", i.mustList("/v1/invoices", valuesWithLimit(limit, "customer", customer)))
	records = appendListRecords(records, "payment_intent", i.mustList("/v1/payment_intents", valuesWithLimit(limit, "customer", customer)))
	charges := i.mustList("/v1/charges", valuesWithLimit(limit, "customer", customer))
	records = appendListRecords(records, "charge", charges)
	for _, charge := range charges {
		records = appendListRecords(records, "dispute", i.mustList("/v1/disputes", valuesWithLimit(limit, "charge", mapString(charge, "id"))))
		records = appendListRecords(records, "refund", i.mustList("/v1/refunds", valuesWithLimit(limit, "charge", mapString(charge, "id"))))
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

func (i investigator) mustList(path string, params url.Values) []map[string]any {
	items, err := i.list(path, params)
	if err != nil {
		return nil
	}
	return items
}

func appendListRecords(records []evidenceRecord, object string, items []map[string]any) []evidenceRecord {
	for _, item := range items {
		records = append(records, entityRecord(object, item))
	}
	return records
}

func valuesWithLimit(limit int, pairs ...string) url.Values {
	params := url.Values{}
	api.AddLimit(params, limit)
	for idx := 0; idx+1 < len(pairs); idx += 2 {
		shared.AddString(params, pairs[idx], pairs[idx+1])
	}
	return params
}
