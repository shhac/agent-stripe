package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/output"
)

type evidenceRecord struct {
	Type              string          `json:"type"`
	Object            string          `json:"object,omitempty"`
	ID                string          `json:"id,omitempty"`
	Severity          string          `json:"severity,omitempty"`
	Summary           string          `json:"summary,omitempty"`
	Data              map[string]any  `json:"data,omitempty"`
	Command           string          `json:"command,omitempty"`
	ExtractedEntities []fieldNote     `json:"extracted_entities,omitempty"`
	TruncatedFields   []truncatedNote `json:"truncated_fields,omitempty"`
}

type fieldNote struct {
	Path   string `json:"path"`
	Object string `json:"object"`
	ID     string `json:"id"`
}

type truncatedNote struct {
	Path          string `json:"path"`
	OriginalBytes int    `json:"original_bytes"`
	ShownBytes    int    `json:"shown_bytes"`
	ExpandHint    string `json:"expand_hint"`
}

func writeEvidence(records []evidenceRecord, format string) {
	records = normalizeEvidence(records)
	f := output.ResolveFormat(format, output.FormatNDJSON)
	if f == output.FormatNDJSON {
		w := output.NewNDJSONWriter(os.Stdout)
		for _, record := range records {
			_ = w.WriteItem(record)
		}
		return
	}
	items := make([]any, len(records))
	for idx, record := range records {
		items[idx] = record
	}
	shared.WritePaginatedList(items, nil, format)
}

func entityRecord(object string, data map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:   "entity",
		Object: object,
		ID:     mapString(data, "id"),
		Data:   data,
	}
}

func normalizeEvidence(records []evidenceRecord) []evidenceRecord {
	opts := evidenceOptions{
		full:         flagInvestigationFull,
		expandFields: flagInvestigationExpandFields,
		maxString:    flagInvestigationMaxString,
	}
	if opts.maxString <= 0 {
		opts.maxString = 800
	}
	seen := map[string]bool{}
	out := make([]evidenceRecord, 0, len(records))
	for _, record := range records {
		normalized, extracted := normalizeRecord(record, opts, seen)
		out = append(out, normalized)
		out = append(out, extracted...)
	}
	return out
}

type evidenceOptions struct {
	full         bool
	expandFields []string
	maxString    int
}

func normalizeRecord(record evidenceRecord, opts evidenceOptions, seen map[string]bool) (evidenceRecord, []evidenceRecord) {
	if record.Type != "entity" || record.Data == nil {
		return record, nil
	}
	data, extracted, notes, truncated := normalizeValue(record.Data, record.Object, "", opts, seen)
	if normalized, ok := data.(map[string]any); ok {
		record.Data = normalized
	}
	record.ExtractedEntities = append(record.ExtractedEntities, notes...)
	record.TruncatedFields = append(record.TruncatedFields, truncated...)
	return record, extracted
}

func normalizeValue(value any, parentObject, path string, opts evidenceOptions, seen map[string]bool) (any, []evidenceRecord, []fieldNote, []truncatedNote) {
	switch v := value.(type) {
	case map[string]any:
		if isStripeEntity(v) && path != "" {
			object := mapString(v, "object")
			id := mapString(v, "id")
			key := object + ":" + id
			note := fieldNote{Path: path, Object: object, ID: id}
			if !seen[key] {
				seen[key] = true
				child, children := normalizeRecord(entityRecord(object, v), opts, seen)
				extracted := append([]evidenceRecord{child}, children...)
				return id, extracted, []fieldNote{note}, nil
			}
			return id, nil, []fieldNote{note}, nil
		}
		if isStripeList(v) {
			return normalizeList(v, parentObject, path, opts, seen)
		}
		out := make(map[string]any, len(v))
		var extracted []evidenceRecord
		var notes []fieldNote
		var truncated []truncatedNote
		for key, item := range v {
			nextPath := joinPath(path, key)
			normalized, childRecords, childNotes, childTruncated := normalizeValue(item, parentObject, nextPath, opts, seen)
			out[key] = normalized
			extracted = append(extracted, childRecords...)
			notes = append(notes, childNotes...)
			truncated = append(truncated, childTruncated...)
		}
		return out, extracted, notes, truncated
	case []any:
		out := make([]any, len(v))
		var extracted []evidenceRecord
		var notes []fieldNote
		var truncated []truncatedNote
		for idx, item := range v {
			nextPath := fmt.Sprintf("%s[%d]", path, idx)
			normalized, childRecords, childNotes, childTruncated := normalizeValue(item, parentObject, nextPath, opts, seen)
			out[idx] = normalized
			extracted = append(extracted, childRecords...)
			notes = append(notes, childNotes...)
			truncated = append(truncated, childTruncated...)
		}
		return out, extracted, notes, truncated
	case string:
		if shouldTruncate(path, v, opts) {
			shown := opts.maxString
			if shown > len(v) {
				shown = len(v)
			}
			return v[:shown] + "...", nil, nil, []truncatedNote{{
				Path:          path,
				OriginalBytes: len(v),
				ShownBytes:    shown,
				ExpandHint:    "--expand-field " + path + " or --full",
			}}
		}
		return v, nil, nil, nil
	default:
		return value, nil, nil, nil
	}
}

func normalizeList(list map[string]any, parentObject, path string, opts evidenceOptions, seen map[string]bool) (any, []evidenceRecord, []fieldNote, []truncatedNote) {
	out := make(map[string]any, len(list))
	for key, value := range list {
		if key != "data" {
			out[key] = value
		}
	}
	items, _ := list["data"].([]any)
	normalizedItems := make([]any, 0, len(items))
	var extracted []evidenceRecord
	var notes []fieldNote
	var truncated []truncatedNote
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s.data[%d]", path, idx)
		normalized, childRecords, childNotes, childTruncated := normalizeValue(item, parentObject, itemPath, opts, seen)
		normalizedItems = append(normalizedItems, normalized)
		extracted = append(extracted, childRecords...)
		notes = append(notes, childNotes...)
		truncated = append(truncated, childTruncated...)
	}
	out["data"] = normalizedItems
	return out, extracted, notes, truncated
}

func isStripeEntity(item map[string]any) bool {
	id := mapString(item, "id")
	object := mapString(item, "object")
	return id != "" && object != "" && object != "list" && object != "search_result"
}

func isStripeList(item map[string]any) bool {
	object := mapString(item, "object")
	return object == "list" || object == "search_result"
}

func shouldTruncate(path, value string, opts evidenceOptions) bool {
	if opts.full || len(value) <= opts.maxString {
		return false
	}
	if path == "id" || strings.HasSuffix(path, ".id") {
		return false
	}
	for _, expanded := range opts.expandFields {
		if expanded == path || strings.HasPrefix(path, expanded+".") {
			return false
		}
	}
	return true
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func invoicePaymentSummary(invoice, pi, charge map[string]any) string {
	card := cardLast4(charge)
	cardText := "card details unavailable"
	if card != "" {
		cardText = "card ending " + card
	}
	return fmt.Sprintf("Invoice %s paid %s with %s through PaymentIntent %s.", mapString(invoice, "id"), formatAmountPaid(invoice), cardText, mapString(pi, "id"))
}

func paymentFailureSummary(pi, charge map[string]any) string {
	if charge != nil {
		if message := mapString(charge, "failure_message"); message != "" {
			return fmt.Sprintf("Charge %s failed: %s", mapString(charge, "id"), message)
		}
		if outcome := mapAnyMap(charge, "outcome"); len(outcome) > 0 {
			return fmt.Sprintf("Charge %s status is %s; outcome=%v.", mapString(charge, "id"), mapString(charge, "status"), outcome)
		}
	}
	if pi != nil {
		if lastErr := mapAnyMap(pi, "last_payment_error"); len(lastErr) > 0 {
			return fmt.Sprintf("PaymentIntent %s failed: %v.", mapString(pi, "id"), lastErr)
		}
		return fmt.Sprintf("PaymentIntent %s status is %s.", mapString(pi, "id"), mapString(pi, "status"))
	}
	return "Payment status could not be determined."
}

func moneyMovementFinding(object string, item map[string]any) evidenceRecord {
	status := mapString(item, "status")
	severity := "info"
	if status == "failed" || status == "canceled" || status == "reversed" {
		severity = "warning"
	}
	summary := fmt.Sprintf("%s %s status is %s.", object, mapString(item, "id"), status)
	if failure := firstNonEmpty(mapString(item, "failure_message"), mapString(item, "failure_code"), mapString(item, "failure_reason"), mapString(item, "failure_balance_transaction")); failure != "" {
		summary += " Failure detail: " + failure + "."
		severity = "warning"
	}
	return evidenceRecord{Type: "finding", Severity: severity, Summary: summary}
}

func severityForPayment(pi, charge map[string]any) string {
	if charge != nil {
		if mapString(charge, "status") == "failed" || !mapBool(charge, "paid") {
			return "warning"
		}
	}
	if pi != nil {
		status := mapString(pi, "status")
		if status != "" && status != "succeeded" {
			return "warning"
		}
	}
	return "info"
}

func cardLast4(charge map[string]any) string {
	if charge == nil {
		return ""
	}
	pmd := mapAnyMap(charge, "payment_method_details")
	card, _ := pmd["card"].(map[string]any)
	return mapString(card, "last4")
}

func formatAmount(item map[string]any) string {
	amount, ok := mapInt64(item, "amount")
	if !ok {
		amount, ok = mapInt64(item, "amount_due")
	}
	if !ok {
		return "unknown amount"
	}
	currency := strings.ToUpper(mapString(item, "currency"))
	if currency == "" {
		currency = "UNKNOWN"
	}
	return fmt.Sprintf("%d %s minor units", amount, currency)
}

func formatAmountPaid(invoice map[string]any) string {
	amount, ok := mapInt64(invoice, "amount_paid")
	if !ok {
		return formatAmount(invoice)
	}
	currency := strings.ToUpper(mapString(invoice, "currency"))
	if currency == "" {
		currency = "UNKNOWN"
	}
	return fmt.Sprintf("%d %s minor units", amount, currency)
}

func idFromValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		return mapString(v, "id")
	default:
		return ""
	}
}

func mapAnyMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	value, _ := m[key].(map[string]any)
	return value
}

func mapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return value
}

func mapBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	value, _ := m[key].(bool)
	return value
}

func mapValue(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func mapInt64(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	switch value := m[key].(type) {
	case float64:
		return int64(value), true
	case int64:
		return value, true
	case int:
		return int64(value), true
	default:
		return 0, false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
