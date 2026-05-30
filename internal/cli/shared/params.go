package shared

import (
	"net/url"
	"strconv"
)

func AddLimit(params url.Values, limit int) {
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
}

func AddCreatedRange(params url.Values, gte, lte string) {
	if gte != "" {
		params.Set("created[gte]", gte)
	}
	if lte != "" {
		params.Set("created[lte]", lte)
	}
}

func AddExpand(params url.Values, expand []string) {
	for _, item := range expand {
		if item != "" {
			params.Add("expand[]", item)
		}
	}
}
