package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/shhac/lib-agent-cli/dialog"
	"github.com/shhac/lib-agent-cli/dialog/dialogtest"
)

func TestPromptAPIKeyViaDialog(t *testing.T) {
	rec := &dialogtest.Recorder{
		PromptResults: []dialog.Result{{ID: "api_key", Value: "sk_test_secret"}},
	}
	restore := dialog.SetDefault(rec)
	t.Cleanup(restore)

	got, err := promptAPIKeyViaDialog(context.Background(), "sandbox", "")
	if err != nil {
		t.Fatalf("promptAPIKeyViaDialog() error = %v", err)
	}
	if got != "sk_test_secret" {
		t.Fatalf("api key = %q", got)
	}
	if len(rec.Calls) != 1 {
		t.Fatalf("dialog calls = %d, want 1", len(rec.Calls))
	}
	if rec.Calls[0].Items[0].InputType != dialog.Password {
		t.Fatalf("input type = %v, want Password", rec.Calls[0].Items[0].InputType)
	}
}

func TestPromptAPIKeyViaDialogSkipsWhenProvided(t *testing.T) {
	rec := &dialogtest.Recorder{}
	restore := dialog.SetDefault(rec)
	t.Cleanup(restore)

	got, err := promptAPIKeyViaDialog(context.Background(), "sandbox", "rk_test_existing")
	if err != nil {
		t.Fatalf("promptAPIKeyViaDialog() error = %v", err)
	}
	if got != "rk_test_existing" {
		t.Fatalf("api key = %q", got)
	}
	if len(rec.Calls) != 0 {
		t.Fatalf("dialog should not be called")
	}
}

func TestPromptAPIKeyViaDialogClassifiesNoGUI(t *testing.T) {
	rec := &dialogtest.Recorder{AvailableErr: dialog.ErrNoGUI}
	restore := dialog.SetDefault(rec)
	t.Cleanup(restore)

	_, err := promptAPIKeyViaDialog(context.Background(), "sandbox", "")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, dialog.ErrNoGUI) {
		t.Fatalf("error = %v, want ErrNoGUI", err)
	}
}
