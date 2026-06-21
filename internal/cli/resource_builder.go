package cli

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
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
	idKind        string
	getShort      string
	listShort     string
	searchShort   string
	searchHint    string
	usageText     string
	searchable    bool
	listFlags     []listFlag
	expandGet     bool
	expandList    bool
	listSummary   func(map[string]any) map[string]any
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
		Use:   "get <" + opts.idName + ">...",
		Short: resourceText(opts.getShort, "Retrieve a "+opts.idName),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := globals()
			params := url.Values{}
			shared.AddExpand(params, expand)
			return shared.GetEntities(flags, args, func(ctx context.Context, client *api.Client, id string) (any, error) {
				if err := validateExpectedStripeID(id, opts.idKind); err != nil {
					return nil, err
				}
				return shared.FetchItem(ctx, client, flags, opts.path+"/"+url.PathEscape(id), params)
			})
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
	var cursor cursorFlags
	var expand []string
	var full bool
	values := make(map[string]*string, len(opts.listFlags))
	cmd := &cobra.Command{
		Use:   "list",
		Short: resourceText(opts.listShort, "List "+opts.use),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			shared.AddLimit(params, limit)
			shared.AddCreatedRange(params, createdGTE, createdLTE)
			cursor.AddTo(params)
			shared.AddExpand(params, expand)
			for _, flag := range opts.listFlags {
				shared.AddString(params, flag.param, *values[flag.name])
			}
			if opts.listSummary != nil && len(expand) > 0 && !full {
				return agenterrors.New("--expand requires --full on compact list commands", agenterrors.FixableByAgent).
					WithHint("Re-run with --full, or use get <id> for focused expanded fields")
			}
			if opts.listSummary != nil && !full {
				return getSummarizedList(globals(), opts.path, params, opts.listSummary)
			}
			return shared.GetRawList(globals(), opts.path, params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cmd.Flags().StringVar(&createdGTE, "created-gte", "", "Created at or after Unix timestamp")
	cmd.Flags().StringVar(&createdLTE, "created-lte", "", "Created at or before Unix timestamp")
	cursor.AddFlags(cmd)
	if opts.expandList {
		cmd.Flags().StringArrayVar(&expand, "expand", nil, "Expand response property; repeatable")
	}
	if opts.listSummary != nil {
		cmd.Flags().BoolVar(&full, "full", false, "Return full Stripe objects instead of compact summaries")
	}
	for _, flag := range opts.listFlags {
		var value string
		values[flag.name] = &value
		cmd.Flags().StringVar(&value, flag.name, "", flag.help)
	}
	return cmd
}

func getSummarizedList(flags *shared.GlobalFlags, path string, params url.Values, summarize func(map[string]any) map[string]any) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.Get(ctx, path, params)
		if err != nil {
			return err
		}
		list, err := api.DecodeList(raw)
		if err != nil {
			return err
		}
		if output.ResolveFormat(flags.Format, output.FormatNDJSON) == output.FormatNDJSON {
			w := output.NewNDJSONWriter(output.Stdout())
			for _, rawItem := range list.Data {
				item, err := summarizedListItem(rawItem, summarize)
				if err != nil {
					return err
				}
				_ = w.WriteItem(item)
			}
			if pagination := listPagination(list); pagination != nil {
				_ = w.WritePagination(pagination)
			}
			return nil
		}
		items := make([]any, 0, len(list.Data))
		for _, rawItem := range list.Data {
			item, err := summarizedListItem(rawItem, summarize)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		shared.WritePaginatedList(items, listPagination(list), flags.Format)
		return nil
	})
}

func summarizedListItem(raw json.RawMessage, summarize func(map[string]any) map[string]any) (map[string]any, error) {
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return summarize(item), nil
}

func listPagination(list *api.ListResponse) *output.Pagination {
	if !list.HasMore && list.NextPage == "" {
		return nil
	}
	return &output.Pagination{
		HasMore:  list.HasMore,
		NextPage: list.NextPage,
	}
}

func newResourceSearchCommand(globals shared.GlobalsFunc, opts resourceOptions) *cobra.Command {
	var limit int
	var query string
	var page string
	cmd := &cobra.Command{
		Use:   "search",
		Short: resourceText(opts.searchShort, "Search "+opts.use+" with Stripe Search Query Language"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shared.RequireFlag("query", query, opts.searchHint); err != nil {
				return err
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
