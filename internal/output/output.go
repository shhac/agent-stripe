// Package output re-exports the shared output contract from lib-agent-output,
// keeping the internal/output import path while the wire mechanism (format
// parsing, JSON/YAML encoding, error rendering) lives in one place. What stays
// local is agent-stripe policy: the writer indirection used by tests, the
// Stripe-shaped pagination trailer, the NDJSON list writer, and the
// expose-aware redaction in redaction.go. (Migration shim.)
package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"

	out "github.com/shhac/lib-agent-output"
	"gopkg.in/yaml.v3"
)

var (
	writersMu sync.RWMutex
	stdout    io.Writer = os.Stdout
	stderr    io.Writer = os.Stderr
)

func Stdout() io.Writer {
	writersMu.RLock()
	defer writersMu.RUnlock()
	return stdout
}

func Stderr() io.Writer {
	writersMu.RLock()
	defer writersMu.RUnlock()
	return stderr
}

func SetWritersForTest(o, e io.Writer) func() {
	writersMu.Lock()
	previousOut := stdout
	previousErr := stderr
	if o != nil {
		stdout = o
	}
	if e != nil {
		stderr = e
	}
	writersMu.Unlock()
	return func() {
		writersMu.Lock()
		stdout = previousOut
		stderr = previousErr
		writersMu.Unlock()
	}
}

// Format and its values come from the shared contract; the NDJSON value is
// "jsonl" in both.
type Format = out.Format

const (
	FormatJSON   = out.FormatJSON
	FormatYAML   = out.FormatYAML
	FormatNDJSON = out.FormatNDJSON
)

// ParseFormat is the family's lenient parser (accepts "ndjson"/"yml",
// case-insensitive). Thin wrapper over the shared contract.
func ParseFormat(s string) (Format, error) { return out.ParseFormat(s) }

// WriteError renders err as the shared {error,fixable_by,hint} contract on w,
// wrapping a bare error as fixable_by:agent. Thin wrapper over the shared
// contract.
func WriteError(w io.Writer, err error) { out.WriteError(w, err) }

// ResolveFormat keeps agent-stripe's one-arg, error-swallowing contract (the
// shared out.ResolveFormat returns an error): an unparseable flag falls back to
// the default rather than surfacing.
func ResolveFormat(flagFormat string, defaultFormat Format) Format {
	f, err := out.ResolveFormat(flagFormat, defaultFormat)
	if err != nil {
		return defaultFormat
	}
	return f
}

// init registers agent-stripe's YAML encoder with lib-agent-output, so YAML
// support (and its yaml.v3 dependency) stays in this CLI while the core library
// remains dependency-free.
func init() {
	out.RegisterEncoder(out.FormatYAML, func(v any) ([]byte, error) {
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(v); err != nil {
			return nil, err
		}
		_ = enc.Close()
		return buf.Bytes(), nil
	})
}

// Print prunes nulls (opt-in) then encodes data in the given format via the
// shared encoder. Pruning is the only clean step here; expose-aware redaction
// is applied by callers via Redact before Print.
func Print(data any, format Format, prune bool) {
	cleaned, ok := toCleanAny(data, prune)
	if !ok {
		return
	}
	// Data is already cleaned, so pass a nil pruner — out.Print just encodes.
	_ = out.Print(Stdout(), cleaned, format, nil)
}

func WriteRawJSON(raw json.RawMessage, format Format, prune bool) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		_ = out.Print(Stdout(), raw, FormatJSON, nil)
		return
	}
	Print(data, format, prune)
}

func toCleanAny(data any, prune bool) (any, bool) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, false
	}
	var decoded any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return nil, false
	}
	if prune {
		decoded = out.PruneNils(decoded)
	}
	return decoded, true
}

// NDJSONWriter writes one record per line. It stays local because of the
// Stripe-shaped pagination trailer below.
type NDJSONWriter struct {
	enc *json.Encoder
}

func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &NDJSONWriter{enc: enc}
}

func (n *NDJSONWriter) WriteItem(item any) error {
	return n.enc.Encode(item)
}

// Pagination is Stripe-shaped (cursor + page hints), so it stays local rather
// than using out.Pagination.
type Pagination struct {
	HasMore    bool   `json:"has_more"`
	TotalItems int    `json:"total_items,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	NextPage   string `json:"next_page,omitempty"`
}

func (n *NDJSONWriter) WritePagination(p *Pagination) error {
	return n.enc.Encode(map[string]any{"@pagination": p})
}
