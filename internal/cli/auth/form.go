package auth

import (
	"context"
	"fmt"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/lib-agent-cli/dialog"
)

func promptAPIKeyViaDialog(ctx context.Context, profile, apiKey string) (string, error) {
	if apiKey != "" {
		return apiKey, nil
	}

	spec := dialog.Spec{
		Title: fmt.Sprintf("agent-stripe credential: %s", profile),
		Items: []dialog.Field{
			{ID: "api_key", Label: "Stripe API key", InputType: dialog.Password},
		},
	}

	if err := dialog.Default.Available(); err != nil {
		return apiKey, classifyDialogErr(err, profile)
	}

	results, err := dialog.Default.Prompt(ctx, spec)
	if err != nil {
		return apiKey, classifyDialogErr(err, profile)
	}
	for _, result := range results {
		if result.ID == "api_key" {
			apiKey = result.Value
		}
	}
	return apiKey, nil
}

func classifyDialogErr(err error, profile string) error {
	cat, hint := dialog.ClassifyError(err)
	switch cat {
	case dialog.CategoryHuman:
		hint = "agent-stripe auth add --form requires a graphical desktop session. " +
			"Ask the user to run it on their local machine, or fall back to non-interactive: " +
			fmt.Sprintf("agent-stripe auth add %s --api-key <secret>", profile)
	case dialog.CategoryRetry:
		hint = "User cancelled the dialog. Re-run agent-stripe auth add --form to retry."
	}
	return agenterrors.Wrap(err, categoryToFixableBy(cat)).WithHint(hint)
}

func categoryToFixableBy(c dialog.Category) agenterrors.FixableBy {
	switch c {
	case dialog.CategoryHuman:
		return agenterrors.FixableByHuman
	case dialog.CategoryRetry:
		return agenterrors.FixableByRetry
	default:
		return agenterrors.FixableByAgent
	}
}
