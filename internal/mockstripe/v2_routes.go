package mockstripe

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// v2IncludableFields are the top-level fields Stripe nulls out unless the
// caller names them in include[]. configuration is handled per-configuration
// (include=configuration.merchant), so it is listed separately.
var v2IncludableFields = []string{"identity", "requirements", "future_requirements", "defaults"}

var v2Configurations = []string{"customer", "merchant", "recipient"}

func (s *Server) handleV2Accounts(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	query := r.URL.Query()
	items := v2Accounts()
	closed := query.Get("closed")
	if closed == "" {
		closed = "false"
	}
	items = filterByBoolString(items, "closed", closed)
	if wanted := indexedValues(query, "applied_configurations"); len(wanted) > 0 {
		items = filterByAppliedConfigurations(items, wanted)
	}
	listed := make([]map[string]any, 0, len(items))
	for _, item := range items {
		listed = append(listed, v2AccountWithIncludes(item, nil))
	}
	writeV2List(w, r, "/v2/core/accounts", listed)
}

func (s *Server) handleV2AccountPath(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/v2/core/accounts/")
	accountID, personPath, hasPersons := strings.Cut(rest, "/persons")
	if accountID == "" {
		writeV2Error(w, http.StatusNotFound, "invalid_request_error", "not_found", "No such account")
		return
	}
	if hasPersons {
		s.writeV2Persons(w, r, accountID, strings.Trim(personPath, "/"))
		return
	}
	account, ok := findV2Account(accountID)
	if !ok {
		writeV2AccountLookupError(w, accountID)
		return
	}
	writeJSON(w, http.StatusOK, v2AccountWithIncludes(account, indexedValues(r.URL.Query(), "include")))
}

func (s *Server) writeV2Persons(w http.ResponseWriter, r *http.Request, accountID, personID string) {
	if _, ok := findV2Account(accountID); !ok {
		writeV2AccountLookupError(w, accountID)
		return
	}
	persons := v2AccountPersons(accountID)
	if personID == "" {
		writeV2List(w, r, "/v2/core/accounts/"+accountID+"/persons", persons)
		return
	}
	for _, person := range persons {
		if stringValue(person, "id") == personID {
			writeJSON(w, http.StatusOK, person)
			return
		}
	}
	writeV2Error(w, http.StatusNotFound, "invalid_request_error", "not_found", "No such person: '"+personID+"'")
}

func (s *Server) handleV2Events(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	query := r.URL.Query()
	items := v2Events()
	if objectID := query.Get("object_id"); objectID != "" {
		items = filterByRelatedObjectID(items, objectID)
	}
	if types := indexedValues(query, "types"); len(types) > 0 {
		items = filterByAnyStringValue(items, "type", types)
	}
	writeV2List(w, r, "/v2/core/events", items)
}

func (s *Server) handleV2Event(w http.ResponseWriter, r *http.Request) {
	if !requireGet(w, r) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v2/core/events/")
	for _, event := range v2Events() {
		if stringValue(event, "id") == id {
			writeJSON(w, http.StatusOK, event)
			return
		}
	}
	writeV2Error(w, http.StatusNotFound, "invalid_request_error", "not_found", "No such event: '"+id+"'")
}

func findV2Account(id string) (map[string]any, bool) {
	for _, account := range v2Accounts() {
		if stringValue(account, "id") == id {
			return account, true
		}
	}
	return nil, false
}

// writeV2AccountLookupError reproduces Stripe's namespace-interop behavior: a
// Connect v1 account ID is a 400 with a specific code, not a 404, and that
// distinction is what drives the CLI's v2 -> v1 fallback.
func writeV2AccountLookupError(w http.ResponseWriter, accountID string) {
	for _, account := range accounts() {
		if stringValue(account, "id") == accountID {
			writeV2Error(w, http.StatusBadRequest, "invalid_request_error", "v1_account_instead_of_v2_account",
				"V1 Account ID cannot be used in V2 Account APIs.")
			return
		}
	}
	writeV2Error(w, http.StatusNotFound, "invalid_request_error", "not_found", "No such account: '"+accountID+"'")
}

// v2AccountWithIncludes strips a fixture down to what Stripe would return for
// the requested include set. Omitted fields come back null rather than being
// absent, so a caller can tell "not requested" from "empty".
func v2AccountWithIncludes(account map[string]any, include []string) map[string]any {
	requested := map[string]bool{}
	for _, value := range include {
		requested[value] = true
	}
	result := map[string]any{}
	for key, value := range account {
		result[key] = value
	}
	for _, field := range v2IncludableFields {
		if !requested[field] {
			result[field] = nil
		}
	}
	configuration, _ := account["configuration"].(map[string]any)
	filtered := map[string]any{}
	for _, name := range v2Configurations {
		if !requested["configuration."+name] {
			continue
		}
		if value, ok := configuration[name]; ok {
			filtered[name] = value
		} else {
			filtered[name] = nil
		}
	}
	if len(filtered) == 0 {
		result["configuration"] = nil
		return result
	}
	result["configuration"] = filtered
	return result
}

func filterByAppliedConfigurations(items []map[string]any, wanted []string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		applied := map[string]bool{}
		for _, value := range anyStrings(item["applied_configurations"]) {
			applied[value] = true
		}
		matchesAll := true
		for _, want := range wanted {
			if !applied[want] {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterByRelatedObjectID(items []map[string]any, objectID string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		related, _ := item["related_object"].(map[string]any)
		if stringValue(related, "id") == objectID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterByAnyStringValue(items []map[string]any, key string, wanted []string) []map[string]any {
	allowed := map[string]bool{}
	for _, value := range wanted {
		allowed[value] = true
	}
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if allowed[stringValue(item, key)] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func anyStrings(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// indexedValues reads Stripe's /v2 array encoding (key[0], key[1], ...). The
// bare and [] forms are also accepted so a hand-written request still works
// against the mock.
func indexedValues(query url.Values, key string) []string {
	var values []string
	values = append(values, query[key]...)
	values = append(values, query[key+"[]"]...)
	for index := 0; ; index++ {
		value := query.Get(key + "[" + strconv.Itoa(index) + "]")
		if value == "" {
			break
		}
		values = append(values, value)
	}
	return values
}
