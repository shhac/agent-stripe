package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

type cursorFlags struct {
	startingAfter string
	endingBefore  string
}

func (f *cursorFlags) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.startingAfter, "starting-after", "", "Stripe cursor")
	cmd.Flags().StringVar(&f.endingBefore, "ending-before", "", "Stripe cursor")
	markCursorFlagsMutuallyExclusive(cmd)
}

func (f *cursorFlags) AddTo(params url.Values) {
	shared.AddString(params, "starting_after", f.startingAfter)
	shared.AddString(params, "ending_before", f.endingBefore)
}

func markCursorFlagsMutuallyExclusive(cmd *cobra.Command) {
	cmd.MarkFlagsMutuallyExclusive("starting-after", "ending-before")
}
