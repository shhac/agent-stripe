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

// AddIndexed encodes an array parameter the way Stripe's /v2 namespace requires
// it: include[0]=a&include[1]=b. The unindexed /v1 form is rejected there, even
// for a single value.
func AddIndexed(params url.Values, key string, values []string) {
	index := 0
	for _, value := range values {
		if value == "" {
			continue
		}
		params.Set(key+"["+strconv.Itoa(index)+"]", value)
		index++
	}
}
