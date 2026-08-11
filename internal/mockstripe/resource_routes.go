package mockstripe

import (
	"net/http"
	"strings"
)

type mockResource struct {
	path       string
	objectName string
	items      func() []map[string]any
	searchable bool
	filters    []mockFilter
	// subLists maps a sub-path suffix to its fixtures, e.g. "line_items" for
	// /v1/checkout/sessions/<id>/line_items. Declaring them here keeps the
	// resource in the table instead of needing a hand-written handler.
	subLists map[string]func(id string) []map[string]any
	// nested serves <path>/<id>/<segment>/<sub-id>, which is how Stripe models
	// transfer reversals.
	nested       string
	nestedItems  func(parentID string) []map[string]any
	nestedObject string
	// subPaths marks a resource whose <path>/ space is served elsewhere because
	// it has sub-resources (accounts has persons, capabilities, external
	// accounts), so the table does not claim the pattern.
	subPaths bool
}

type mockFilter struct {
	query       string
	field       string
	kind        filterKind
	ignoreValue string
}

type filterKind string

const (
	filterString filterKind = "string"
	filterBool   filterKind = "bool"
)

func (s *Server) registerMockResource(resource mockResource) {
	if resource.searchable {
		s.mux.HandleFunc(resource.path+"/search", resource.handleSearch)
	}
	s.mux.HandleFunc(resource.path, resource.handleList)
	if !resource.subPaths {
		s.mux.HandleFunc(resource.path+"/", resource.handleGet)
	}
}

func (r mockResource) handleList(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	writeList(w, r.path, r.filteredItems(req), req)
}

func (r mockResource) handleSearch(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	if req.URL.Query().Get("query") == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "Missing required param: query")
		return
	}
	writeSearchList(w, r.path+"/search", r.items(), req)
}

func (r mockResource) handleGet(w http.ResponseWriter, req *http.Request) {
	if !requireGet(w, req) {
		return
	}
	rest := strings.TrimPrefix(req.URL.Path, r.path+"/")
	if r.nested != "" && strings.Contains(rest, "/"+r.nested+"/") {
		parent, child, _ := strings.Cut(rest, "/"+r.nested+"/")
		writeOneByID(w, r.nestedItems(parent), child, r.nestedObject)
		return
	}
	for suffix, items := range r.subLists {
		if !strings.HasSuffix(rest, "/"+suffix) {
			continue
		}
		id := strings.TrimSuffix(rest, "/"+suffix)
		writeList(w, r.path+"/"+id+"/"+suffix, items(id), req)
		return
	}
	writeOneByID(w, r.items(), rest, r.objectName)
}

func (r mockResource) filteredItems(req *http.Request) []map[string]any {
	items := r.items()
	for _, filter := range r.filters {
		value := req.URL.Query().Get(filter.query)
		if value == "" || value == filter.ignoreValue {
			continue
		}
		switch filter.kind {
		case filterBool:
			items = filterByBoolString(items, filter.field, value)
		default:
			items = filterByString(items, filter.field, value)
		}
	}
	return items
}

func stringFilter(query, field string) mockFilter {
	return mockFilter{query: query, field: field, kind: filterString}
}

func boolFilter(query, field string) mockFilter {
	return mockFilter{query: query, field: field, kind: filterBool}
}
