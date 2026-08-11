package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateCustomerContext(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "customer-context",
		Short: "Gather payment, invoice, subscription, and risk context for a customer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireFlag("customer", customer, "Provide a Customer ID such as cus_..."); err != nil {
				return err
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.customerContext(customer, limit)
			})
		},
	}
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum recent objects per collection")
	return cmd
}

func (i investigator) customerContext(customer string, limit int) error {
	if err := validateExpectedStripeID(customer, "customer"); err != nil {
		return err
	}
	customerObj, err := i.get("/v1/customers/"+url.PathEscape(customer), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("customer", customerObj))

	i.addRelatedList("payment_method", "/v1/payment_methods", valuesWithLimit(limit, "customer", customer, "type", "card"))
	i.addRelatedList("subscription", "/v1/subscriptions", valuesWithLimit(limit, "customer", customer, "status", "all"))
	i.addRelatedList("invoice", "/v1/invoices", valuesWithLimit(limit, "customer", customer))
	i.addRelatedList("payment_intent", "/v1/payment_intents", valuesWithLimit(limit, "customer", customer))
	for _, charge := range i.addRelatedList("charge", "/v1/charges", valuesWithLimit(limit, "customer", customer)) {
		i.addRelatedList("dispute", "/v1/disputes", valuesWithLimit(limit, "charge", mapString(charge, "id")))
		i.addRelatedList("refund", "/v1/refunds", valuesWithLimit(limit, "charge", mapString(charge, "id")))
	}
	i.add(evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  "Customer context gathered. Use entity records for recent payment methods, subscriptions, invoices, payment intents, charges, disputes, and refunds.",
		Data: map[string]any{
			"customer": customer,
		},
	})
	return nil
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

// relatedListWarning is relatedWarning with the path, so a consumer parsing
// degraded-evidence findings can tell which collection was missed.
func relatedListWarning(object, path string, err error) evidenceRecord {
	record := relatedWarning(object+" from "+path, err)
	record.Data["object"] = object
	record.Data["path"] = path
	return record
}

func valuesWithLimit(limit int, pairs ...string) url.Values {
	params := url.Values{}
	shared.AddLimit(params, limit)
	for idx := 0; idx+1 < len(pairs); idx += 2 {
		shared.AddString(params, pairs[idx], pairs[idx+1])
	}
	return params
}
