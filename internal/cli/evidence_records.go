package cli

import (
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
		w := output.NewNDJSONWriter(output.Stdout())
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

type evidenceStreamer struct {
	writer  *output.NDJSONWriter
	opts    evidenceOptions
	seen    map[string]bool
	emitted map[string]bool
}

func newEvidenceStreamer(format string, opts evidenceOptions) *evidenceStreamer {
	if output.ResolveFormat(format, output.FormatNDJSON) != output.FormatNDJSON {
		return nil
	}
	if opts.maxString <= 0 {
		opts.maxString = defaultMaxString
	}
	return &evidenceStreamer{
		writer:  output.NewNDJSONWriter(output.Stdout()),
		opts:    opts,
		seen:    map[string]bool{},
		emitted: map[string]bool{},
	}
}

func (s *evidenceStreamer) emit(record evidenceRecord) {
	if s == nil {
		return
	}
	normalized, extracted := normalizeRecord(record, s.opts, s.seen)
	s.write(normalized)
	for _, child := range extracted {
		s.write(child)
	}
}

func (s *evidenceStreamer) writeRemaining(records []evidenceRecord) {
	if s == nil {
		return
	}
	for _, record := range records {
		s.emit(record)
	}
}

func (s *evidenceStreamer) write(record evidenceRecord) {
	if key := evidenceRecordKey(record); key != "" {
		if s.emitted[key] {
			return
		}
		s.emitted[key] = true
	}
	_ = s.writer.WriteItem(record)
}

func evidenceRecordKey(record evidenceRecord) string {
	if record.Type != "entity" || record.Object == "" || record.ID == "" {
		return ""
	}
	return record.Type + ":" + record.Object + ":" + record.ID
}

func entityRecord(object string, data map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:   "entity",
		Object: object,
		ID:     mapString(data, "id"),
		Data:   data,
	}
}
