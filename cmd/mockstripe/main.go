package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/mockstripe"
)

func main() {
	var addr string

	cmd := &cobra.Command{
		Use:   "mockstripe",
		Short: "Local mock Stripe API server for agent-stripe tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := &http.Server{
				Addr:    addr,
				Handler: mockstripe.NewServer(),
			}
			_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
				"status":   "listening",
				"base_url": "http://" + addr,
			})
			return server.ListenAndServe()
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:12111", "Address to listen on")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
