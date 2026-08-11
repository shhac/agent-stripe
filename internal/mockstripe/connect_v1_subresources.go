package mockstripe

import (
	"net/http"
	"strings"
)

// /v1/accounts/<id>/{persons,capabilities,external_accounts} plus the v2
// money-management payout methods. These are the endpoints that answer what an
// account's requirements are actually asking for.
func (s *Server) handleAccountSubresource(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v1/accounts/")
	accountID, sub, hasSub := strings.Cut(rest, "/")
	if !hasSub {
		writeOneByID(w, accounts(), accountID, "account")
		return
	}
	switch {
	case sub == "persons":
		writeList(w, r.URL.Path, filterPersonsByRelationship(accountPersons(accountID), r), r)
	case strings.HasPrefix(sub, "persons/"):
		writeOneByID(w, accountPersons(accountID), strings.TrimPrefix(sub, "persons/"), "person")
	case sub == "capabilities":
		writeList(w, r.URL.Path, accountCapabilities(accountID), r)
	case sub == "external_accounts":
		items := accountExternalAccounts(accountID)
		if object := r.URL.Query().Get("object"); object != "" {
			items = filterByString(items, "object", object)
		}
		writeList(w, r.URL.Path, items, r)
	default:
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "No such account sub-resource: "+sub)
	}
}

func filterPersonsByRelationship(items []map[string]any, r *http.Request) []map[string]any {
	for _, role := range []string{"representative", "owner", "director", "executive"} {
		if r.URL.Query().Get("relationship["+role+"]") != "true" {
			continue
		}
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			relationship, _ := item["relationship"].(map[string]any)
			if value, _ := relationship[role].(bool); value {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return items
}

func (s *Server) handleV2PayoutMethods(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	// Payout methods are addressed by Stripe-Context, not by a path segment.
	recipient := r.Header.Get("Stripe-Context")
	if recipient == "" {
		writeV2Error(w, http.StatusBadRequest, "invalid_request_error", "context_missing",
			"Stripe-Context must name the recipient account")
		return
	}
	writeV2List(w, r, "/v2/money_management/payout_methods", payoutMethods(recipient))
}

func accountPersons(accountID string) []map[string]any {
	if accountID != "acct_mock_connected" {
		return nil
	}
	return []map[string]any{
		{
			"id":         "person_mock_v1_rep",
			"object":     "person",
			"account":    accountID,
			"first_name": "Robin",
			"last_name":  "Vance",
			"email":      "robin.vance@example.com",
			"relationship": map[string]any{
				"representative":    true,
				"owner":             true,
				"director":          false,
				"executive":         false,
				"percent_ownership": 100,
				"title":             "Founder",
			},
			"requirements": map[string]any{
				"currently_due": []any{"verification.document"},
				"past_due":      []any{},
			},
			"verification": map[string]any{"status": "pending"},
		},
	}
}

func accountCapabilities(accountID string) []map[string]any {
	if accountID != "acct_mock_connected" {
		return nil
	}
	return []map[string]any{
		{
			"id":           "card_payments",
			"object":       "capability",
			"account":      accountID,
			"status":       "active",
			"requested":    true,
			"requirements": map[string]any{"currently_due": []any{}, "past_due": []any{}},
		},
		{
			"id":        "transfers",
			"object":    "capability",
			"account":   accountID,
			"status":    "inactive",
			"requested": true,
			"requirements": map[string]any{
				"currently_due":   []any{"external_account"},
				"past_due":        []any{"external_account"},
				"disabled_reason": "requirements.past_due",
			},
		},
	}
}

func accountExternalAccounts(accountID string) []map[string]any {
	if accountID != "acct_mock_connected" {
		return nil
	}
	return []map[string]any{
		{
			"id":                   "ba_mock_closed",
			"object":               "bank_account",
			"account":              accountID,
			"bank_name":            "MOCK CREDIT UNION",
			"country":              "US",
			"currency":             "usd",
			"last4":                "6789",
			"routing_number":       "110000000",
			"status":               "errored",
			"default_for_currency": true,
		},
	}
}

func payoutMethods(recipient string) []map[string]any {
	if recipient != "acct_mock_v2_recipient" {
		return nil
	}
	return []map[string]any{
		{
			"id":       "usba_mock_recipient",
			"object":   "v2.money_management.payout_method",
			"type":     "bank_account",
			"created":  "2026-06-19T16:10:00.000Z",
			"livemode": false,
			"bank_account": map[string]any{
				"country":        "gb",
				"currency":       "gbp",
				"last4":          "3311",
				"bank_name":      "MOCK BANK PLC",
				"routing_number": "20-00-00",
			},
			"usage_status": map[string]any{
				"payments":  "eligible",
				"transfers": "eligible",
			},
			"available_payout_speeds": []any{"standard"},
		},
	}
}
