package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateResolve(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <stripe-id-or-invoice-number>",
		Short: "Identify a Stripe object and suggest next investigation commands",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.resolve(args[0])
			})
		},
	}
}

func (i investigator) resolve(value string) ([]evidenceRecord, error) {
	object, path, next := resolvePath(value)
	if path == "" {
		found, err := i.list("/v1/invoices/search", url.Values{"query": []string{stripeSearchEquals("number", value)}, "limit": []string{"1"}})
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return []evidenceRecord{{Type: "finding", Severity: "warning", Summary: "Could not resolve value as a known Stripe ID prefix or invoice number."}}, nil
		}
		invoice := found[0]
		return []evidenceRecord{
			entityRecord("invoice", invoice),
			{Type: "finding", Severity: "info", Summary: "Resolved invoice number to invoice " + mapString(invoice, "id") + ".", Command: "agent-stripe investigate invoice-payment " + mapString(invoice, "id")},
		}, nil
	}
	item, err := i.get(path+"/"+url.PathEscape(value), url.Values{})
	if err != nil {
		return nil, err
	}
	return []evidenceRecord{
		entityRecord(object, item),
		{Type: "finding", Severity: "info", Summary: "Resolved " + value + " as " + object + ".", Command: next + value},
	}, nil
}

func resolvePath(id string) (object, path, commandPrefix string) {
	switch {
	case strings.HasPrefix(id, "cus_"):
		return "customer", "/v1/customers", "agent-stripe investigate customer-context --customer "
	case strings.HasPrefix(id, "in_"):
		return "invoice", "/v1/invoices", "agent-stripe investigate invoice-payment "
	case strings.HasPrefix(id, "pi_"):
		return "payment_intent", "/v1/payment_intents", "agent-stripe investigate incoming-payment "
	case strings.HasPrefix(id, "ch_"):
		return "charge", "/v1/charges", "agent-stripe investigate incoming-payment "
	case strings.HasPrefix(id, "sub_"):
		return "subscription", "/v1/subscriptions", "agent-stripe investigate subscription-renewal --subscription "
	case strings.HasPrefix(id, "dp_"):
		return "dispute", "/v1/disputes", "agent-stripe investigate dispute-response "
	case strings.HasPrefix(id, "re_"):
		return "refund", "/v1/refunds", "agent-stripe investigate refund-status "
	case strings.HasPrefix(id, "tr_"):
		return "transfer", "/v1/transfers", "agent-stripe investigate outgoing-payment "
	case strings.HasPrefix(id, "po_"):
		return "payout", "/v1/payouts", "agent-stripe investigate payout-failure "
	case strings.HasPrefix(id, "acct_"):
		return "account", "/v1/accounts", "agent-stripe investigate outgoing-payment "
	case strings.HasPrefix(id, "evt_"):
		return "event", "/v1/events", "agent-stripe investigate webhook-event "
	case strings.HasPrefix(id, "pm_"):
		return "payment_method", "/v1/payment_methods", "agent-stripe payment-methods get "
	case strings.HasPrefix(id, "seti_"):
		return "setup_intent", "/v1/setup_intents", "agent-stripe setup-intents get "
	case strings.HasPrefix(id, "cs_"):
		return "checkout.session", "/v1/checkout/sessions", "agent-stripe checkout-sessions get "
	case strings.HasPrefix(id, "price_"):
		return "price", "/v1/prices", "agent-stripe prices get "
	case strings.HasPrefix(id, "prod_"):
		return "product", "/v1/products", "agent-stripe products get "
	default:
		return "", "", ""
	}
}
