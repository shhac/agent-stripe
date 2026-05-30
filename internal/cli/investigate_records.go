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

const defaultMaxString = 800

func writeEvidence(records []evidenceRecord, format string, opts evidenceOptions) {
	records = normalizeEvidence(records, opts)
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

func normalizeEvidence(records []evidenceRecord, opts evidenceOptions) []evidenceRecord {
	if opts.maxString <= 0 {
		opts.maxString = defaultMaxString
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

func defaultEvidenceOptions() evidenceOptions {
	return evidenceOptions{maxString: defaultMaxString}
}

func normalizeRecord(record evidenceRecord, opts evidenceOptions, seen map[string]bool) (evidenceRecord, []evidenceRecord) {
	if record.Type != "entity" || record.Data == nil {
		return record, nil
	}
	result := normalizeValue(record.Data, record.Object, "", opts, seen)
	if normalized, ok := result.value.(map[string]any); ok {
		record.Data = normalized
	}
	record.ExtractedEntities = append(record.ExtractedEntities, result.notes...)
	record.TruncatedFields = append(record.TruncatedFields, result.truncated...)
	return record, result.records
}

type normalizeResult struct {
	value     any
	records   []evidenceRecord
	notes     []fieldNote
	truncated []truncatedNote
}

func normalizeValue(value any, parentObject, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	switch v := value.(type) {
	case map[string]any:
		if isStripeList(v) {
			return normalizeList(v, parentObject, path, opts, seen)
		}
		if isStripeEntity(v) && path != "" {
			return normalizeNestedEntity(v, path, opts, seen)
		}
		return normalizeMap(v, parentObject, path, opts, seen)
	case []any:
		return normalizeSlice(v, parentObject, path, opts, seen)
	case string:
		if shouldTruncate(path, v, opts) {
			shown := opts.maxString
			if shown > len(v) {
				shown = len(v)
			}
			return normalizeResult{value: v[:shown] + "...", truncated: []truncatedNote{{
				Path:          path,
				OriginalBytes: len(v),
				ShownBytes:    shown,
				ExpandHint:    "--expand-field " + path + " or --full",
			}}}
		}
		return normalizeResult{value: v}
	default:
		return normalizeResult{value: value}
	}
}

func normalizeNestedEntity(item map[string]any, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	object := mapString(item, "object")
	id := mapString(item, "id")
	result := normalizeResult{
		value: id,
		notes: []fieldNote{{Path: path, Object: object, ID: id}},
	}
	key := object + ":" + id
	if seen[key] {
		return result
	}
	seen[key] = true
	child, children := normalizeRecord(entityRecord(object, item), opts, seen)
	result.records = append([]evidenceRecord{child}, children...)
	return result
}

func normalizeMap(item map[string]any, parentObject, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	result := normalizeResult{value: make(map[string]any, len(item))}
	out := result.value.(map[string]any)
	for key, value := range item {
		child := normalizeValue(value, parentObject, joinPath(path, key), opts, seen)
		out[key] = child.value
		result.merge(child)
	}
	return result
}

func normalizeSlice(items []any, parentObject, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	result := normalizeResult{value: make([]any, len(items))}
	out := result.value.([]any)
	for idx, item := range items {
		child := normalizeValue(item, parentObject, fmt.Sprintf("%s[%d]", path, idx), opts, seen)
		out[idx] = child.value
		result.merge(child)
	}
	return result
}

func normalizeList(list map[string]any, parentObject, path string, opts evidenceOptions, seen map[string]bool) normalizeResult {
	out := make(map[string]any, len(list))
	for key, value := range list {
		if key != "data" {
			out[key] = value
		}
	}
	items, _ := list["data"].([]any)
	normalizedItems := make([]any, 0, len(items))
	result := normalizeResult{value: out}
	for idx, item := range items {
		itemPath := fmt.Sprintf("%s.data[%d]", path, idx)
		child := normalizeValue(item, parentObject, itemPath, opts, seen)
		normalizedItems = append(normalizedItems, child.value)
		result.merge(child)
	}
	out["data"] = normalizedItems
	return result
}

func (r *normalizeResult) merge(child normalizeResult) {
	r.records = append(r.records, child.records...)
	r.notes = append(r.notes, child.notes...)
	r.truncated = append(r.truncated, child.truncated...)
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
	if path == "id" || path == "object" || strings.HasSuffix(path, ".id") || strings.HasSuffix(path, ".object") {
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
