package shared

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterUsage(parent *cobra.Command, verb, text string) {
	parent.AddCommand(&cobra.Command{
		Use:   "usage",
		Short: "Show detailed reference for " + verb,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(text)
		},
	})
}
