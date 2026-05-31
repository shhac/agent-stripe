package cli

import "testing"

func TestParseQueryPairs(t *testing.T) {
	values, err := parseQueryPairs([]string{
		"expand[]=latest_charge",
		"expand[]=customer",
		"metadata=",
		"query=metadata['order_id']:'123=456'",
	})
	if err != nil {
		t.Fatalf("parseQueryPairs() error = %v", err)
	}
	if got := values["expand[]"]; len(got) != 2 || got[0] != "latest_charge" || got[1] != "customer" {
		t.Fatalf("expand[] = %#v, want repeated values preserved", got)
	}
	if got := values.Get("metadata"); got != "" {
		t.Fatalf("metadata = %q, want empty value", got)
	}
	if got := values.Get("query"); got != "metadata['order_id']:'123=456'" {
		t.Fatalf("query = %q, want value containing '='", got)
	}
}

func TestParseQueryPairsRejectsMissingKeyOrSeparator(t *testing.T) {
	for _, pair := range []string{"missing-separator", "=value"} {
		t.Run(pair, func(t *testing.T) {
			if _, err := parseQueryPairs([]string{pair}); err == nil {
				t.Fatalf("parseQueryPairs(%q) error = nil, want error", pair)
			}
		})
	}
}
