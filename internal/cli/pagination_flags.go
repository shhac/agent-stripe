package cli

import "github.com/spf13/cobra"

func markCursorFlagsMutuallyExclusive(cmd *cobra.Command) {
	cmd.MarkFlagsMutuallyExclusive("starting-after", "ending-before")
}
