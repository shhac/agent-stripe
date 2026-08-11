package cli

import (
	"net/url"
)

// fetchRelated retrieves an object referenced by another object, resolving the
// API path from the ID prefix rather than restating it at each call site.
//
// A failed lookup is recorded as a warning and returns nil. Investigations used
// to write `if x, err := i.get(...); err == nil { ... }` at ~30 sites, which
// dropped the evidence and left the report looking complete — the reader could
// not tell "this account has no disputes" from "the dispute lookup failed".
func (i investigator) fetchRelated(label, id string) map[string]any {
	if id == "" {
		return nil
	}
	path := stripeAPIPathForID(id)
	if path == "" {
		return nil
	}
	object, err := i.get(path+"/"+url.PathEscape(id), url.Values{})
	if err != nil {
		i.add(relatedWarning(label+" "+id, err))
		return nil
	}
	return object
}

// followRef is fetchRelated for an expandable field on a parent object, where
// Stripe returns either an ID string or the expanded object.
func (i investigator) followRef(parent map[string]any, field string) map[string]any {
	return i.fetchRelated(field, idFromValue(parent[field]))
}

// listRelated fetches a related collection, reporting a failure as a warning
// instead of returning silence that reads as "there are none".
func (i investigator) listRelated(label, path string, params url.Values) []map[string]any {
	items, err := i.list(path, params)
	if err != nil {
		i.add(relatedListWarning(label, path, err))
		return nil
	}
	return items
}

// addRelatedList is listRelated plus recording, for the callers that want the
// collection in the evidence stream as well as in hand.
func (i investigator) addRelatedList(object, path string, params url.Values) []map[string]any {
	items := i.listRelated(object, path, params)
	i.addList(object, items)
	return items
}

// addLatestCharge and addInvoicePaymentIntent are the reporting forms of the
// two chain helpers. Their (object, error) signature invited callers to write
// `err == nil && x != nil`, which drops the failure — five of eight call sites
// did exactly that. These report it and return nil, like fetchRelated.
func (i investigator) addLatestCharge(pi map[string]any) map[string]any {
	charge, err := i.latestChargeForPaymentIntent(pi)
	if err != nil {
		i.add(relatedWarning("latest charge for "+mapString(pi, "id"), err))
		return nil
	}
	return charge
}

func (i investigator) addInvoicePaymentIntent(invoice map[string]any) map[string]any {
	pi, err := i.paymentIntentForInvoice(invoice)
	if err != nil {
		i.add(relatedWarning("PaymentIntent for "+mapString(invoice, "id"), err))
		return nil
	}
	return pi
}

// stripeAPIPathForID resolves an ID prefix to its collection path. stripeIDKinds
// already carries this mapping for every object investigations fetch, so the
// path is derived rather than written out again.
func stripeAPIPathForID(id string) string {
	kind, ok := classifyStripeID(id)
	if !ok {
		return ""
	}
	return kind.APIPath
}
