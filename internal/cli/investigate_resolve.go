package cli

import (
	"net/url"

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
		if object != "" {
			return i.appendEvidence(nil, evidenceRecord{
				Type:     "finding",
				Severity: "warning",
				Summary:  "Resolved " + value + " as " + object + ", but it requires a parent object to retrieve directly.",
				Command:  next + value,
			}), nil
		}
		found, err := i.list("/v1/invoices/search", url.Values{"query": []string{stripeSearchEquals("number", value)}, "limit": []string{"1"}})
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return i.appendEvidence(nil, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Could not resolve value as a known Stripe ID prefix or invoice number."}), nil
		}
		invoice := found[0]
		return i.appendEvidence(nil,
			entityRecord("invoice", invoice),
			evidenceRecord{Type: "finding", Severity: "info", Summary: "Resolved invoice number to invoice " + mapString(invoice, "id") + ".", Command: "agent-stripe investigate invoice-payment " + mapString(invoice, "id")},
		), nil
	}
	item, err := i.get(path+"/"+url.PathEscape(value), url.Values{})
	if err != nil {
		return nil, err
	}
	return i.appendEvidence(nil,
		entityRecord(object, item),
		evidenceRecord{Type: "finding", Severity: "info", Summary: "Resolved " + value + " as " + object + ".", Command: next + value},
	), nil
}

func resolvePath(id string) (object, path, commandPrefix string) {
	kind, ok := classifyStripeID(id)
	if !ok {
		return "", "", ""
	}
	return kind.resolvedObject(), kind.APIPath, kind.resolveCommandPrefix()
}
