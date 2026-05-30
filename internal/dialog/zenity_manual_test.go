//go:build manual

// Run with: go test -tags=manual ./internal/dialog/...
//
// This test pops real native dialogs on the developer's screen. It is
// excluded from the default test run so CI never blocks on a popup.

package dialog_test

import (
	"context"
	"testing"

	"github.com/shhac/agent-stripe/internal/dialog"
)

func TestZenityPromptManually(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results, err := dialog.Default.Prompt(ctx, dialog.Spec{
		Title: "agent-stripe manual test",
		Items: []dialog.Field{
			{ID: "context", Label: "Type any Stripe context", InputType: dialog.Text},
			{ID: "api_key", Label: "Type any Stripe API key", InputType: dialog.Password},
		},
	})
	if err != nil {
		t.Fatalf("Prompt() returned %v — did you cancel?", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	t.Logf("you typed context=%q, api_key=(%d chars)", results[0].Value, len(results[1].Value))
}
