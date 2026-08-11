// Package output re-exports the shared output contract from lib-agent-output,
// keeping the internal/output import path while the wire mechanism (format
// parsing, JSON encoding, error rendering) lives in one place; YAML encoding is
// supplied by the shared lib-agent-cli/yaml encoder. What stays
// local is agent-stripe policy: the writer indirection used by tests, the
// Stripe-shaped pagination trailer, the NDJSON list writer, and the
// expose-aware redaction in redaction.go. (Migration shim.)
package output

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	_ "github.com/shhac/lib-agent-cli/yaml" // registers the shared YAML encoder
	out "github.com/shhac/lib-agent-output"
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

// YAML support (and its yaml.v3 dependency) comes from the shared
// lib-agent-cli/yaml package, imported for its registration side effect above,
// so the core lib-agent-output library remains dependency-free.

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

// NDJSONWriter wraps the shared writer rather than reimplementing it. The
// previous local encoder skipped the library's colorization path, so a
// colorized `get` sat next to uncolorized lists and investigations even though
// the CLI registers --color. What is genuinely agent-stripe's is the shape of
// the pagination trailer, which rides the library's meta-line contract.
type NDJSONWriter struct {
	writer *out.NDJSONWriter
}

func NewNDJSONWriter(w io.Writer) *NDJSONWriter {
	return &NDJSONWriter{writer: out.NewNDJSONWriter(w)}
}

func (n *NDJSONWriter) WriteItem(item any) error {
	return n.writer.WriteItem(item)
}

// Pagination is Stripe-shaped: has_more plus the cursor or page token to hand
// back. TotalItems and NextCursor were copied from the library's shape and
// never set by anything here, so they are gone.
type Pagination struct {
	HasMore  bool   `json:"has_more"`
	NextPage string `json:"next_page,omitempty"`
}

func (n *NDJSONWriter) WritePagination(p *Pagination) error {
	return n.writer.WriteMetaLine(out.MetaKeyPagination, p)
}
