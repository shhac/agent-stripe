package output

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func TestWriteErrorIncludesHintAndFixability(t *testing.T) {
	var buf bytes.Buffer
	WriteError(&buf, agenterrors.New("bad query", agenterrors.FixableByAgent).WithHint("try a narrower query"))

	var got map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("error output was not JSON: %v", err)
	}
	if got["error"] != "bad query" {
		t.Fatalf("error = %q", got["error"])
	}
	if got["fixable_by"] != "agent" {
		t.Fatalf("fixable_by = %q", got["fixable_by"])
	}
	if !strings.Contains(got["hint"], "narrower") {
		t.Fatalf("hint = %q", got["hint"])
	}
}

func TestParseFormatAcceptsNDJSONAlias(t *testing.T) {
	got, err := ParseFormat("ndjson")
	if err != nil {
		t.Fatalf("ParseFormat() error = %v", err)
	}
	if got != FormatNDJSON {
		t.Fatalf("ParseFormat() = %q", got)
	}
}

func TestRedactSensitiveFieldsByDefault(t *testing.T) {
	input := map[string]any{
		"id":            "pi_mock_123",
		"object":        "payment_intent",
		"client_secret": "pi_mock_123_secret_fake",
		"metadata": map[string]any{
			"internal_product_id": "prod_internal_basic",
			"support_email":       "support@example.test",
			"api_token":           "tok_fake",
		},
	}

	redacted := Redact(input, RedactionOptions{}).(map[string]any)
	if redacted["client_secret"] != RedactedString {
		t.Fatalf("client_secret = %#v, want redacted marker", redacted["client_secret"])
	}
	metadata := redacted["metadata"].(map[string]any)
	if metadata["api_token"] != RedactedString {
		t.Fatalf("metadata.api_token = %#v, want redacted marker", metadata["api_token"])
	}
	if metadata["internal_product_id"] != "prod_internal_basic" {
		t.Fatalf("non-sensitive metadata was redacted: %#v", redacted)
	}
	if _, ok := metadata["@redacted"]; ok {
		t.Fatalf("nested @redacted note present: %#v", metadata)
	}
	notes, ok := redacted["@redacted"].([]RedactionNote)
	if !ok {
		t.Fatalf("@redacted note missing: %#v", redacted)
	}
	if len(notes) != 2 {
		t.Fatalf("@redacted note count = %d, want 2: %#v", len(notes), notes)
	}
	assertRedactionPath(t, notes, "client_secret")
	assertRedactionPath(t, notes, "metadata.api_token")
}

func TestRedactNameByObjectContext(t *testing.T) {
	cases := []struct {
		name       string
		input      map[string]any
		path       []string // path of keys to reach the "name" field
		wantRedact bool
	}{
		{
			name: "name on customer object is redacted",
			input: map[string]any{
				"object": "customer",
				"name":   "Jane Doe",
			},
			path:       []string{"name"},
			wantRedact: true,
		},
		{
			name: "name on account object is redacted",
			input: map[string]any{
				"object": "account",
				"name":   "Acme Inc",
			},
			path:       []string{"name"},
			wantRedact: true,
		},
		{
			name: "name in nested sub-object of customer inherits redaction",
			input: map[string]any{
				"object": "customer",
				"shipping": map[string]any{
					"name": "Jane Doe",
				},
			},
			path:       []string{"shipping", "name"},
			wantRedact: true,
		},
		{
			name: "name under billing_details is redacted",
			input: map[string]any{
				"object": "charge",
				"billing_details": map[string]any{
					"name": "Jane Doe",
				},
			},
			path:       []string{"billing_details", "name"},
			wantRedact: true,
		},
		{
			name: "name on a product object is not redacted",
			input: map[string]any{
				"object": "product",
				"name":   "Basic Plan",
			},
			path:       []string{"name"},
			wantRedact: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			redacted := Redact(tc.input, RedactionOptions{}).(map[string]any)
			got := nestedValue(t, redacted, tc.path)
			if tc.wantRedact && got != RedactedString {
				t.Fatalf("name = %#v, want redacted marker", got)
			}
			if !tc.wantRedact && got == RedactedString {
				t.Fatalf("name = %#v, want unredacted", got)
			}
		})
	}
}

func nestedValue(t *testing.T, m map[string]any, path []string) any {
	t.Helper()
	var cur any = m
	for _, key := range path {
		asMap, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("expected map at key %q, got %#v", key, cur)
		}
		cur = asMap[key]
	}
	return cur
}

func TestRedactHonorsExposeByPathOrKey(t *testing.T) {
	input := map[string]any{
		"client_secret": "pi_mock_123_secret_fake",
		"metadata": map[string]any{
			"api_token": "tok_fake",
		},
	}

	redacted := Redact(input, RedactionOptions{Expose: []string{"client_secret,metadata.api_token"}}).(map[string]any)
	if redacted["client_secret"] != "pi_mock_123_secret_fake" {
		t.Fatalf("client_secret = %#v", redacted["client_secret"])
	}
	if redacted["metadata"].(map[string]any)["api_token"] != "tok_fake" {
		t.Fatalf("metadata.api_token = %#v", redacted["metadata"])
	}
	if _, ok := redacted["@redacted"]; ok {
		t.Fatalf("@redacted notes present despite exposed fields: %#v", redacted)
	}
}

func assertRedactionPath(t *testing.T, notes []RedactionNote, path string) {
	t.Helper()
	for _, note := range notes {
		if note.Path == path {
			return
		}
	}
	t.Fatalf("@redacted missing path %q in %#v", path, notes)
}

func TestRedactAccountsV2IdentityFields(t *testing.T) {
	person := map[string]any{
		"object":        "v2.core.account_person",
		"id":            "person_123",
		"account":       "acct_123",
		"given_name":    "Jenny",
		"surname":       "Rosen",
		"email":         "jenny.rosen@example.com",
		"date_of_birth": map[string]any{"day": 28, "month": 1, "year": 1988},
		"id_numbers":    []any{map[string]any{"type": "us_ssn_last_4"}},
		"relationship":  map[string]any{"representative": true, "title": "CEO"},
	}

	got, ok := Redact(person, RedactionOptions{}).(map[string]any)
	if !ok {
		t.Fatalf("Redact() returned %T, want map", got)
	}
	for _, key := range []string{"given_name", "surname", "email", "date_of_birth"} {
		if got[key] != RedactedString {
			t.Fatalf("%s = %#v, want redacted", key, got[key])
		}
	}
	// Relationship and ID-number types are what triage needs, and Stripe never
	// returns the ID number itself.
	relationship, _ := got["relationship"].(map[string]any)
	if relationship["title"] != "CEO" || relationship["representative"] != true {
		t.Fatalf("relationship = %#v, want it kept visible", got["relationship"])
	}
	idNumbers, _ := got["id_numbers"].([]any)
	first, _ := idNumbers[0].(map[string]any)
	if first["type"] != "us_ssn_last_4" {
		t.Fatalf("id_numbers = %#v, want the type kept visible", got["id_numbers"])
	}
	if got["id"] != "person_123" || got["account"] != "acct_123" {
		t.Fatalf("navigation IDs should stay visible: %#v", got)
	}
}

func TestRedactKeepsV2AccountDisplayNameButMasksContact(t *testing.T) {
	account := map[string]any{
		"object":        "v2.core.account",
		"id":            "acct_123",
		"display_name":  "Furever Grooming",
		"contact_email": "owner@furever.example.com",
		"contact_phone": "+15550101001",
		"identity": map[string]any{
			"country":     "us",
			"entity_type": "company",
			"business_details": map[string]any{
				"registered_name": "Furever Inc",
			},
		},
	}

	got, ok := Redact(account, RedactionOptions{}).(map[string]any)
	if !ok {
		t.Fatalf("Redact() returned %T, want map", got)
	}
	// display_name is a business label, not a person.
	if got["display_name"] != "Furever Grooming" {
		t.Fatalf("display_name = %#v, want it visible", got["display_name"])
	}
	for _, key := range []string{"contact_email", "contact_phone"} {
		if got[key] != RedactedString {
			t.Fatalf("%s = %#v, want redacted", key, got[key])
		}
	}
	identity, _ := got["identity"].(map[string]any)
	if identity["country"] != "us" || identity["entity_type"] != "company" {
		t.Fatalf("identity = %#v, want country/entity_type visible", identity)
	}
}

// TestRedactionManifestIsOrdered pins the @redacted array to path order. It was
// built by walking a Go map, so repeated runs of the same command emitted the
// same masked object with the manifest shuffled, and diffing two runs of one
// command showed changes that were not there.
func TestRedactionManifestIsOrdered(t *testing.T) {
	person := map[string]any{
		"object": "person", "id": "person_1",
		"email": "a@b.c", "phone": "+44", "dob": "1990", "first_name": "A", "last_name": "B",
	}
	want := []string{"dob", "email", "first_name", "last_name", "phone"}

	// A single run cannot tell "sorted" from "this map happened to walk in
	// order", so assert the same order over enough runs that an accidental pass
	// is vanishingly unlikely.
	for range 50 {
		redacted, ok := Redact(person, RedactionOptions{}).(map[string]any)
		if !ok {
			t.Fatal("redacted document is not a map")
		}
		notes, ok := redacted["@redacted"].([]RedactionNote)
		if !ok {
			t.Fatalf("@redacted missing: %#v", redacted)
		}
		got := make([]string, 0, len(notes))
		for _, note := range notes {
			got = append(got, note.Path)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("manifest order = %v, want %v", got, want)
		}
	}
}
