package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

// v2ListOptions declares a /v2 list command. It stays separate from
// registerResource because the namespaces are deliberately separate in the
// command surface — this only removes the repeated limit/page/full triple and
// the identical raw-versus-summary branch.
type v2ListOptions struct {
	use     string
	short   string
	long    string
	path    string
	pathFor func(id string) string // set instead of path for a sub-resource list
	idKind  string                 // validated when pathFor is set
	summary func(map[string]any) map[string]any
	flags   func(*cobra.Command)
	params  func(url.Values) error
}

func newV2ListCommand(globals shared.GlobalsFunc, opts v2ListOptions) *cobra.Command {
	var limit int
	var page string
	var full bool

	cmd := &cobra.Command{
		Use:   opts.use,
		Short: opts.short,
		Long:  opts.long,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := opts.path
			if opts.pathFor != nil {
				if err := validateExpectedStripeID(args[0], opts.idKind); err != nil {
					return err
				}
				path = opts.pathFor(args[0])
			}
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			if opts.params != nil {
				if err := opts.params(params); err != nil {
					return err
				}
			}
			if full {
				return shared.GetRawList(globals(), path, params)
			}
			return getSummarizedList(globals(), path, params, opts.summary)
		},
	}
	if opts.pathFor != nil {
		cmd.Args = cobra.ExactArgs(1)
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return")
	cmd.Flags().StringVar(&page, "page", "", "Page token from a previous @pagination next_page")
	cmd.Flags().BoolVar(&full, "full", false, "Return full Stripe objects instead of compact summaries")
	if opts.flags != nil {
		opts.flags(cmd)
	}
	return cmd
}
