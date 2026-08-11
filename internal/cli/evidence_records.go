package cli

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

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

// evidenceCollector is the single accumulator for an investigation. Every
// record enters through add, is normalized and deduped exactly once, and is
// then either streamed immediately (NDJSON) or buffered for one envelope at the
// end. Investigations used to also thread a []evidenceRecord slice through
// their call graph, but that slice was the collector's own list handed back to
// them, so the two mechanisms could — and did — disagree: the buffered path
// emitted a duplicate entity and dropped an auto-emitted one that the streaming
// path showed.
type evidenceCollector struct {
	writer    *output.NDJSONWriter
	format    string
	opts      evidenceOptions
	extracted map[string]bool
	emitted   map[string]bool
	records   []evidenceRecord
}

func newEvidenceCollector(format string, opts evidenceOptions) *evidenceCollector {
	if opts.maxString <= 0 {
		opts.maxString = defaultMaxString
	}
	collector := &evidenceCollector{
		format:    format,
		opts:      opts,
		extracted: map[string]bool{},
		emitted:   map[string]bool{},
	}
	if output.ResolveFormat(format, output.FormatNDJSON) == output.FormatNDJSON {
		collector.writer = output.NewNDJSONWriter(output.Stdout())
	}
	return collector
}

// add normalizes each record and keeps it unless an identical one was already
// kept. Normalization lifts nested Stripe objects into their own records; those
// go through the same dedup as their parent.
func (c *evidenceCollector) add(records ...evidenceRecord) {
	for _, record := range records {
		normalized, extracted := normalizeRecord(record, c.opts, c.extracted)
		c.keep(normalized)
		for _, child := range extracted {
			c.keep(child)
		}
	}
}

// addList adds one entity record per item of a Stripe list response.
func (c *evidenceCollector) addList(object string, items []map[string]any) {
	for _, item := range items {
		c.add(entityRecord(object, item))
	}
}

// count reports how many records have been kept, so a workflow can ask "did the
// step I just ran produce anything" by comparing counts around it.
func (c *evidenceCollector) count() int {
	return len(c.records)
}

// since returns the records kept after mark, so a step can post-process just
// the evidence it produced (the timeline sorts its own slice of it).
func (c *evidenceCollector) since(mark int) []evidenceRecord {
	if mark < 0 || mark > len(c.records) {
		return nil
	}
	return c.records[mark:]
}

func (c *evidenceCollector) keep(record evidenceRecord) {
	if key := evidenceRecordKey(record); key != "" {
		if c.emitted[key] {
			return
		}
		c.emitted[key] = true
	}
	c.records = append(c.records, record)
	if c.writer != nil {
		_ = c.writer.WriteItem(record)
	}
}

// flush writes the buffered records for the non-streaming formats. NDJSON has
// already been written record by record, so it is a no-op there.
func (c *evidenceCollector) flush() {
	if c.writer != nil {
		return
	}
	items := make([]any, len(c.records))
	for idx, record := range c.records {
		items[idx] = record
	}
	shared.WritePaginatedList(items, nil, c.format)
}

func evidenceRecordKey(record evidenceRecord) string {
	if record.Type == "entity" && record.Object != "" && record.ID != "" {
		return record.Type + ":" + record.Object + ":" + record.ID
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("record:%x", sha256.Sum256(raw))
}

func entityRecord(object string, data map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:   "entity",
		Object: object,
		ID:     mapString(data, "id"),
		Data:   data,
	}
}
