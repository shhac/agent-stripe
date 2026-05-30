package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

type listFlag struct {
	name  string
	param string
	help  string
}

type resourceOptions struct {
	use           string
	aliases       []string
	short         string
	path          string
	idName        string
	getShort      string
	listShort     string
	searchShort   string
	searchHint    string
	usageText     string
	searchable    bool
	listFlags     []listFlag
	expandGet     bool
	expandList    bool
	extraCommands []func(shared.GlobalsFunc) *cobra.Command
}

func registerResource(root *cobra.Command, globals shared.GlobalsFunc, opts resourceOptions) {
	resource := &cobra.Command{
		Use:     opts.use,
		Aliases: opts.aliases,
		Short:   opts.short,
	}
	resource.AddCommand(newResourceGetCommand(globals, opts))
	resource.AddCommand(newResourceListCommand(globals, opts))
	if opts.searchable {
		resource.AddCommand(newResourceSearchCommand(globals, opts))
	}
	if opts.usageText != "" {
		resource.AddCommand(newUsageCommand(opts.usageText))
	}
	for _, newCommand := range opts.extraCommands {
		resource.AddCommand(newCommand(globals))
	}
	root.AddCommand(resource)
}

func newResourceGetCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	var expand []string
	cmd := &cobra.Command{
		Use:   "get <" + opts.idName + ">",
		Short: resourceText(opts.getShort, "Retrieve a "+opts.idName),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			shared.AddExpand(params, expand)
			return shared.GetRawItem(globals(), opts.path+"/"+url.PathEscape(args[0]), params)
		},
	}
	if opts.expandGet {
		cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	}
	return cmd
}

func newResourceListCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	var limit int
	var createdGTE string
	var createdLTE string
	var startingAfter string
	var endingBefore string
	var expand []string
	values := make(map[string]*string, len(opts.listFlags))
	cmd := &cobra.Command{
		Use:   "list",
		Short: resourceText(opts.listShort, "List "+opts.use),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddCreatedRange(params, createdGTE, createdLTE)
			shared.AddString(params, "starting_after", startingAfter)
			shared.AddString(params, "ending_before", endingBefore)
			shared.AddExpand(params, expand)
			for _, flag := range opts.listFlags {
				shared.AddString(params, flag.param, *values[flag.name])
			}
			return shared.GetRawList(globals(), opts.path, params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	cmd.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	cmd.Flags().StringVar(&startingAfter, "starting-after", "", "Stripe cursor")
	cmd.Flags().StringVar(&endingBefore, "ending-before", "", "Stripe cursor")
	markCursorFlagsMutuallyExclusive(cmd)
	if opts.expandList {
		cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	}
	for _, flag := range opts.listFlags {
		var value string
		values[flag.name] = &value
		cmd.Flags().StringVar(&value, flag.name, "", flag.help)
	}
	return cmd
}

func newResourceSearchCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	var limit int
	var query string
	var page string
	cmd := &cobra.Command{
		Use:   "search",
		Short: resourceText(opts.searchShort, "Search "+opts.use+" with Stripe Search Query Language"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("query", query, opts.searchHint) {
				return nil
			}
			params := url.Values{"query": []string{query}}
			shared.AddLimit(params, limit)
			shared.AddString(params, "page", page)
			return shared.GetRawList(globals(), opts.path+"/search", params)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Stripe search query")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&page, "page", "", "Search pagination cursor")
	return cmd
}

func resourceText(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
