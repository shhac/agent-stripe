package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func registerRawAPI(root *cobra.Command, globals shared.GlobalsFunc) {
	var queryPairs []string

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Read-only raw Stripe API escape hatch",
	}
	get := &cobra.Command{
		Use:   "get <path>",
		Short: "GET a Stripe API path with active credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := parseQueryPairs(queryPairs)
			if err != nil {
				return err
			}
			return shared.GetRawItem(globals(), args[0], params)
		},
	}
	get.Flags().StringArrayVar(&queryPairs, "query", nil, "Query parameter as k=v; repeatable")
	apiCmd.AddCommand(get)
	root.AddCommand(apiCmd)
}

func parseQueryPairs(pairs []string) (url.Values, error) {
	values := url.Values{}
	for _, pair := range pairs {
		key, value, ok := splitQueryPair(pair)
		if !ok {
			return nil, sharedQueryError(pair)
		}
		values.Add(key, value)
	}
	return values, nil
}

func splitQueryPair(pair string) (string, string, bool) {
	for i, r := range pair {
		if r == '=' {
			return pair[:i], pair[i+1:], i > 0
		}
	}
	return "", "", false
}

func sharedQueryError(pair string) error {
	return agenterrors.Newf(agenterrors.FixableByAgent, "invalid --query value %q", pair).
		WithHint("Use --query key=value, for example --query expand[]=latest_charge")
}
