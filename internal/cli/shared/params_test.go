package shared

import (
	"net/url"
	"testing"
)

func TestAddParams(t *testing.T) {
	values := url.Values{}
	AddLimit(values, 12)
	AddCreatedRange(values, "100", "200")
	AddExpand(values, []string{"latest_charge", "customer"})

	if values.Get("limit") != "12" {
		t.Fatalf("limit = %q", values.Get("limit"))
	}
	if values.Get("created[gte]") != "100" || values.Get("created[lte]") != "200" {
		t.Fatalf("created range not set: %v", values)
	}
	if got := values["expand[]"]; len(got) != 2 {
		t.Fatalf("expand[] = %v", got)
	}
}
