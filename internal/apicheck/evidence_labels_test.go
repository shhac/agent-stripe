//go:build apicheck

package apicheck

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/cli"
	"github.com/shhac/agent-stripe/internal/mockstripe"
	"github.com/shhac/agent-stripe/internal/output"
)

// TestEachObjectIsEmittedUnderOneLabel drives every command in the table and
// checks that no Stripe object reaches the evidence stream under two different
// `object` names. Entity records are dedup'd on object plus ID, so a caller
// labelling an object differently from its own `object` field does not collide
// with the auto-emitted record — it emits the same thing twice under two names.
//
// This is the defect class that produced `upcoming_in_mock` as both invoice and
// invoice_preview, and `radar.early_fraud_warning` alongside
// early_fraud_warning. Reading call sites missed the last one; running every
// command found it.
func TestEachObjectIsEmittedUnderOneLabel(t *testing.T) {
	server := httptest.NewServer(mockstripe.NewServer())
	defer server.Close()

	type record struct {
		Type   string `json:"type"`
		Object string `json:"object"`
		ID     string `json:"id"`
	}

	var conflicts []string
	for _, args := range commands() {
		var sink bytes.Buffer
		restore := output.SetWritersForTest(&sink, &sink)
		full := append([]string{"--api-key", "sk_test_mock", "--base-url", server.URL}, args...)
		cli.RunForTest("apicheck", full)
		restore()

		labels := map[string]map[string]bool{}
		for _, line := range strings.Split(sink.String(), "\n") {
			var rec record
			if err := json.Unmarshal([]byte(line), &rec); err != nil || rec.Type != "entity" || rec.ID == "" {
				continue
			}
			if labels[rec.ID] == nil {
				labels[rec.ID] = map[string]bool{}
			}
			labels[rec.ID][rec.Object] = true
		}
		for id, seen := range labels {
			if len(seen) < 2 {
				continue
			}
			names := make([]string, 0, len(seen))
			for name := range seen {
				names = append(names, name)
			}
			sort.Strings(names)
			conflicts = append(conflicts, strings.Join(args, " ")+": "+id+" emitted as "+strings.Join(names, " and "))
		}
	}
	sort.Strings(conflicts)
	if len(conflicts) > 0 {
		t.Errorf("objects emitted under more than one label:\n  %s", strings.Join(conflicts, "\n  "))
	}
}
