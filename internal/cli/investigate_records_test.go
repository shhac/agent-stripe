package cli

import (
	"testing"

	"github.com/shhac/agent-stripe/internal/output"
)

// collectNormalized runs records through the collector, which is now the only
// normalization path, and returns what it kept.
func collectNormalized(records ...evidenceRecord) []evidenceRecord {
	return collectNormalizedWith(defaultEvidenceOptions(), records...)
}

func collectNormalizedWith(opts evidenceOptions, records ...evidenceRecord) []evidenceRecord {
	collector := newEvidenceCollector(string(output.FormatJSON), opts)
	collector.add(records...)
	return collector.records
}

func TestNormalizeEvidenceExtractsNestedStripeEntities(t *testing.T) {
	charge := map[string]any{
		"id":     "ch_123",
		"object": "charge",
		"payment_intent": map[string]any{
			"id":     "pi_123",
			"object": "payment_intent",
			"status": "succeeded",
			"customer": map[string]any{
				"id":     "cus_123",
				"object": "customer",
				"email":  "person@example.com",
			},
		},
	}

	records := collectNormalized(entityRecord("charge", charge))

	if len(records) != 3 {
		t.Fatalf("expected parent plus two extracted records, got %d: %#v", len(records), records)
	}
	parent := records[0]
	if got := parent.Data["payment_intent"]; got != "pi_123" {
		t.Fatalf("parent payment_intent = %#v, want pi_123", got)
	}
	if len(parent.ExtractedEntities) != 1 {
		t.Fatalf("parent extracted notes = %#v, want one note", parent.ExtractedEntities)
	}
	if parent.ExtractedEntities[0].Path != "payment_intent" {
		t.Fatalf("parent note path = %q, want payment_intent", parent.ExtractedEntities[0].Path)
	}
	if records[1].Object != "payment_intent" || records[1].ID != "pi_123" {
		t.Fatalf("first extracted record = %#v, want payment_intent pi_123", records[1])
	}
	if got := records[1].Data["customer"]; got != "cus_123" {
		t.Fatalf("extracted payment_intent customer = %#v, want cus_123", got)
	}
	if records[2].Object != "customer" || records[2].ID != "cus_123" {
		t.Fatalf("second extracted record = %#v, want customer cus_123", records[2])
	}
}

func TestNormalizeEvidenceDeduplicatesExtractedEntities(t *testing.T) {
	customer := map[string]any{"id": "cus_123", "object": "customer"}
	charge := map[string]any{
		"id":       "ch_123",
		"object":   "charge",
		"customer": customer,
		"billing_details": map[string]any{
			"customer": customer,
		},
	}

	records := collectNormalized(entityRecord("charge", charge))

	if len(records) != 2 {
		t.Fatalf("expected parent plus one extracted customer, got %d: %#v", len(records), records)
	}
	if got := records[0].Data["customer"]; got != "cus_123" {
		t.Fatalf("parent customer = %#v, want cus_123", got)
	}
	billing := records[0].Data["billing_details"].(map[string]any)
	if got := billing["customer"]; got != "cus_123" {
		t.Fatalf("billing customer = %#v, want cus_123", got)
	}
	if len(records[0].ExtractedEntities) != 2 {
		t.Fatalf("expected two notes for repeated customer references, got %#v", records[0].ExtractedEntities)
	}
}

func TestNormalizeEvidenceTruncatesExpandableFieldsButKeepsIDs(t *testing.T) {
	records := collectNormalizedWith(
		evidenceOptions{maxString: 10, expandFields: []string{"metadata"}},
		entityRecord("payment_intent", map[string]any{
			"id":          "pi_1234567890",
			"object":      "payment_intent",
			"description": "abcdefghijklmnopqrstuvwxyz",
			"metadata": map[string]any{
				"notes": "abcdefghijklmnopqrstuvwxyz",
			},
		}))

	data := records[0].Data
	if got := data["id"]; got != "pi_1234567890" {
		t.Fatalf("id = %#v, want full ID", got)
	}
	if got := data["description"]; got != "abcdefghij..." {
		t.Fatalf("description = %#v, want truncated string", got)
	}
	metadata := data["metadata"].(map[string]any)
	if got := metadata["notes"]; got != "abcdefghijklmnopqrstuvwxyz" {
		t.Fatalf("metadata.notes = %#v, want expanded full string", got)
	}
	if len(records[0].TruncatedFields) != 1 {
		t.Fatalf("truncated fields = %#v, want one note", records[0].TruncatedFields)
	}
	if records[0].TruncatedFields[0].ExpandHint != "--expand-field description or --full" {
		t.Fatalf("expand hint = %q", records[0].TruncatedFields[0].ExpandHint)
	}
}

func TestNormalizeEvidenceFullSkipsTruncation(t *testing.T) {
	records := collectNormalizedWith(
		evidenceOptions{full: true, maxString: 10},
		entityRecord("payment_intent", map[string]any{
			"id":          "pi_123",
			"object":      "payment_intent",
			"description": "abcdefghijklmnopqrstuvwxyz",
		}))

	if got := records[0].Data["description"]; got != "abcdefghijklmnopqrstuvwxyz" {
		t.Fatalf("description = %#v, want full string", got)
	}
	if len(records[0].TruncatedFields) != 0 {
		t.Fatalf("truncated fields = %#v, want none", records[0].TruncatedFields)
	}
}
