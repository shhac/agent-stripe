package cli

import (
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
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
	if strings.HasPrefix(value, "acct_") {
		return i.resolveAccount(value)
	}
	if isV2EventID(value) {
		return i.resolveV2Event(value)
	}
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

// resolveAccount answers the question the acct_ prefix cannot: which account
// namespace this ID lives in. It reads v2 first because a v2 account ID also
// answers on v1 endpoints, so a v1-first probe would never reveal the richer
// object.
func (i investigator) resolveAccount(accountID string) ([]evidenceRecord, error) {
	includes, err := v2AccountIncludeParams(nil)
	if err != nil {
		return nil, err
	}
	account, v2Err := i.get(v2AccountPath(accountID), includes)
	if v2Err == nil {
		return i.appendEvidence(nil,
			entityRecord(objectV2Account, account),
			evidenceRecord{
				Type:     "finding",
				Severity: "info",
				Summary: "Resolved " + accountID + " as an Accounts v2 account with configurations [" +
					strings.Join(v2AppliedConfigurations(account), ", ") + "]. Use the accounts-v2 commands, not accounts.",
				Command: "agent-stripe investigate account-health " + accountID,
				Data:    map[string]any{"namespace": namespaceV2},
			},
		), nil
	}
	if !isNotV2AccountError(v2Err) {
		return nil, v2Err
	}
	v1Account, err := i.get("/v1/accounts/"+url.PathEscape(accountID), url.Values{})
	if err != nil {
		return nil, err
	}
	return i.appendEvidence(nil,
		entityRecord("account", v1Account),
		evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  "Resolved " + accountID + " as a Connect v1 account. Use the accounts commands; accounts-v2 will reject this ID.",
			Command:  "agent-stripe investigate account-health " + accountID + " --namespace v1",
			Data:     map[string]any{"namespace": namespaceV1, "v2_error_code": api.ErrorCode(v2Err)},
		},
	), nil
}

func (i investigator) resolveV2Event(eventID string) ([]evidenceRecord, error) {
	event, err := i.get(v2EventPath(eventID), url.Values{})
	if err != nil {
		return nil, err
	}
	related := mapAnyMap(event, "related_object")
	return i.appendEvidence(nil,
		entityRecord(objectV2Event, event),
		evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary: "Resolved " + eventID + " as a v2 core event of type " + mapString(event, "type") +
				" for " + mapString(related, "type") + " " + mapString(related, "id") + ".",
			Command: "agent-stripe investigate webhook-event " + eventID,
			Data:    map[string]any{"namespace": namespaceV2},
		},
	), nil
}

func resolvePath(id string) (object, path, commandPrefix string) {
	kind, ok := classifyStripeID(id)
	if !ok {
		return "", "", ""
	}
	return kind.resolvedObject(), kind.APIPath, kind.resolveCommandPrefix()
}
