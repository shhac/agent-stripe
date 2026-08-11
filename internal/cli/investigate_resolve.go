package cli

import (
	"net/url"
	"strings"

	"github.com/shhac/agent-stripe/internal/api"
)

var resolveInvestigation = investigationSpec{
	use:   "resolve <stripe-id-or-invoice-number>",
	short: "Identify a Stripe object and suggest next investigation commands",
	run:   investigator.resolve,
}

func (i investigator) resolve(value string) error {
	if strings.HasPrefix(value, "acct_") {
		return i.resolveAccount(value)
	}
	if isV2EventID(value) {
		return i.resolveV2Event(value)
	}
	object, path, next := resolvePath(value)
	if path == "" {
		if object != "" {
			i.add(evidenceRecord{
				Type:     "finding",
				Severity: "warning",
				Summary:  "Resolved " + value + " as " + object + ", but it requires a parent object to retrieve directly.",
				Command:  next + value,
			})
			return nil
		}
		return i.resolveInvoiceNumber(value)
	}
	// Fetched for the record, not the value: the fetch is what streams the
	// object into the evidence stream, and its error is how a bad ID is caught.
	if _, err := i.get(path+"/"+url.PathEscape(value), url.Values{}); err != nil {
		return err
	}
	i.add(
		evidenceRecord{Type: "finding", Severity: "info", Summary: "Resolved " + value + " as " + object + ".", Command: next + value},
	)
	return nil
}

// resolveAccount answers the question the acct_ prefix cannot: which account
// namespace this ID lives in. It reads v2 first because a v2 account ID also
// answers on v1 endpoints, so a v1-first probe would never reveal the richer
// object.
func (i investigator) resolveAccount(accountID string) error {
	includes, err := v2AccountIncludeParams(nil)
	if err != nil {
		return err
	}
	account, v2Err := i.get(v2AccountPath(accountID), includes)
	if v2Err == nil {
		i.add(
			evidenceRecord{
				Type:     "finding",
				Severity: "info",
				Summary: "Resolved " + accountID + " as an Accounts v2 account with configurations [" +
					strings.Join(v2AppliedConfigurations(account), ", ") + "]. Use the accounts-v2 commands, not accounts.",
				Command: "agent-stripe investigate account-health " + accountID,
				Data:    map[string]any{"namespace": namespaceV2},
			},
		)
		return nil
	}
	if !isNotV2AccountError(v2Err) {
		return v2Err
	}
	// Fetched for the record, not the value.
	if _, err := i.get("/v1/accounts/"+url.PathEscape(accountID), url.Values{}); err != nil {
		return err
	}
	i.add(
		evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  "Resolved " + accountID + " as a Connect v1 account. Use the accounts commands; accounts-v2 will reject this ID.",
			Command:  "agent-stripe investigate account-health " + accountID + " --namespace v1",
			Data:     map[string]any{"namespace": namespaceV1, "v2_error_code": api.ErrorCode(v2Err)},
		},
	)
	return nil
}

func (i investigator) resolveV2Event(eventID string) error {
	event, err := i.get(v2EventPath(eventID), url.Values{})
	if err != nil {
		return err
	}
	related := mapAnyMap(event, "related_object")
	i.add(
		evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary: "Resolved " + eventID + " as a v2 core event of type " + mapString(event, "type") +
				" for " + mapString(related, "type") + " " + mapString(related, "id") + ".",
			Command: "agent-stripe investigate webhook-event " + eventID,
			Data:    map[string]any{"namespace": namespaceV2},
		},
	)
	return nil
}

// resolveInvoiceNumber is the fallback when the value is not a known Stripe ID
// prefix: customers quote invoice numbers, not IDs.
func (i investigator) resolveInvoiceNumber(value string) error {
	found, err := i.list("/v1/invoices/search", url.Values{"query": []string{stripeSearchEquals("number", value)}, "limit": []string{"1"}})
	if err != nil {
		return err
	}
	if len(found) == 0 {
		i.add(finding("warning", "Could not resolve value as a known Stripe ID prefix or invoice number."))
		return nil
	}
	invoice := found[0]
	i.add(
		evidenceRecord{Type: "finding", Severity: "info", Summary: "Resolved invoice number to invoice " + mapString(invoice, "id") + ".", Command: "agent-stripe investigate invoice-payment " + mapString(invoice, "id")},
	)
	return nil
}

func resolvePath(id string) (object, path, commandPrefix string) {
	kind, ok := classifyStripeID(id)
	if !ok {
		return "", "", ""
	}
	return kind.resolvedObject(), kind.APIPath, kind.resolveCommandPrefix()
}
