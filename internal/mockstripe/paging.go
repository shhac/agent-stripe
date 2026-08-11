package mockstripe

import (
	"net/http"
	"net/url"
	"strconv"
)

// Cursor paging and the fixture filters the route tables drive.

type pagedList struct {
	items   []map[string]any
	hasMore bool
}

func listPage(items []map[string]any, r *http.Request) pagedList {
	items = applyCursor(items, r.URL.Query())
	n := 10
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			n = parsed
		}
	}
	if n >= len(items) {
		return pagedList{items: items}
	}
	return pagedList{items: items[:n], hasMore: true}
}

func applyCursor(items []map[string]any, query url.Values) []map[string]any {
	if startingAfter := query.Get("starting_after"); startingAfter != "" {
		if idx := indexByID(items, startingAfter); idx >= 0 {
			items = items[idx+1:]
		}
	}
	if endingBefore := query.Get("ending_before"); endingBefore != "" {
		if idx := indexByID(items, endingBefore); idx >= 0 {
			items = items[:idx]
		}
	}
	return items
}

func indexByID(items []map[string]any, id string) int {
	for idx, item := range items {
		if stringValue(item, "id") == id {
			return idx
		}
	}
	return -1
}

func stringValue(item map[string]any, key string) string {
	value, _ := item[key].(string)
	return value
}

func filterByString(items []map[string]any, key, want string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if got, _ := item[key].(string); got == want {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterByBoolString(items []map[string]any, key, want string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	wantBool := want == "true"
	for _, item := range items {
		if got, _ := item[key].(bool); got == wantBool {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
