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
	var routes bool

	cmd := &cobra.Command{
		Use:   "mockstripe",
		Short: "Local mock Stripe API server for agent-stripe tests",
		Long:  "Local mock Stripe API server for agent-stripe tests.\n\nRoutes:\n" + routeHelp(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if routes {
				for _, line := range mockstripe.Routes() {
					fmt.Fprintln(cmd.OutOrStdout(), line)
				}
				return nil
			}
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
	cmd.Flags().BoolVar(&routes, "routes", false, "Print mock route map and exit")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func routeHelp() string {
	out := ""
	for _, line := range mockstripe.Routes() {
		out += "  " + line + "\n"
	}
	return out
}
