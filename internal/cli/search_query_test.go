package cli

import "testing"

func TestStripeSearchEqualsQuotesAndEscapesValues(t *testing.T) {
	got := stripeSearchEquals("number", `INV "north"\west`)
	want := `number:"INV \"north\"\\west"`
	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
}

func TestStripeSearchMetadataEqualsQuotesKeyAndValue(t *testing.T) {
	got := stripeSearchMetadataEquals(`tenant"id`, `acme\prod`)
	want := `metadata["tenant\"id"]:"acme\\prod"`
	if got != want {
		t.Fatalf("query = %q, want %q", got, want)
	}
}
