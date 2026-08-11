package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/auth"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	"github.com/shhac/agent-stripe/internal/config"
	"github.com/shhac/agent-stripe/internal/credential"
	"github.com/shhac/agent-stripe/internal/output"
	libcli "github.com/shhac/lib-agent-cli/cli"
	agentmcp "github.com/shhac/lib-agent-mcp"
)

func newRootCmd(version string) *cobra.Command {
	globals := &shared.GlobalFlags{}
	globalsFunc := func() *shared.GlobalFlags {
		return globals
	}

	root := libcli.NewRoot(libcli.Options{
		Use:           "agent-stripe",
		Short:         "Stripe incident triage CLI for AI agents",
		Version:       version,
		Globals:       &globals.Globals,
		DefaultFormat: output.FormatNDJSON,
		Redacts:       true,
		UnknownHint:   "run 'agent-stripe usage' to see the available domains",
	})

	// ConfigDefaults needs the root's persistent flag set to honor explicit
	// flags over persisted config; wrap the libcli pre-run (which validates
	// --format) so the defaults pass runs first, before validation.
	root.PersistentPreRunE = wrapConfigDefaults(root, globals, root.PersistentPreRunE)

	pf := root.PersistentFlags()
	pf.StringVarP(&globals.Profile, "profile", "p", "", "Stripe profile alias (or AGENT_STRIPE_PROFILE)")
	pf.StringVar(&globals.Context, "context", "", "Stripe-Context value for organization or related-account requests")
	pf.StringVar(&globals.APIKey, "api-key", "", "API key override; never printed or persisted")
	pf.StringVar(&globals.BaseURL, "base-url", "", "Stripe API base URL override for tests")
	pf.IntVar(&globals.MaxRetries, "max-retries", 2, "Maximum automatic retries for transient Stripe 429 responses")
	pf.StringVar(&globals.APIVersion, "api-version", "", "Stripe API version header override for /v1 requests")
	pf.StringVar(&globals.V2APIVersion, "v2-api-version", "", "Stripe API version header override for /v2 requests (Accounts v2, v2 core events)")
	_ = pf.MarkHidden("base-url")

	registerUsageCommand(root)
	registerConfig(root, globalsFunc)
	registerPaymentsDomain(root)
	registerConnectDomain(root)
	auth.Register(root, globalsFunc)
	registerBalance(root, globalsFunc)
	registerCheckoutSessions(root, globalsFunc)
	registerCustomers(root, globalsFunc)
	registerEvents(root, globalsFunc)
	registerProducts(root, globalsFunc)
	registerPrices(root, globalsFunc)
	registerInvoices(root, globalsFunc)
	registerPaymentIntents(root, globalsFunc)
	registerSetupIntents(root, globalsFunc)
	registerCharges(root, globalsFunc)
	registerDisputes(root, globalsFunc)
	registerPaymentMethods(root, globalsFunc)
	registerRefunds(root, globalsFunc)
	registerSubscriptions(root, globalsFunc)
	registerTransfers(root, globalsFunc)
	registerPayouts(root, globalsFunc)
	registerBalanceTransactions(root, globalsFunc)
	registerApplicationFees(root, globalsFunc)
	registerPaymentLinks(root, globalsFunc)
	registerEarlyFraudWarnings(root, globalsFunc)
	registerAccounts(root, globalsFunc)
	registerAccountsV2(root, globalsFunc)
	registerEventsV2(root, globalsFunc)
	registerInvestigate(root, globalsFunc)
	registerRawAPI(root, globalsFunc)

	installGroupUnknownHandlers(root)

	// Expose the whole command tree as an MCP server (added last, so it reflects
	// the complete tree). --color/--expose are output-shaping, irrelevant to a
	// tool call, so hide them from the generated schemas.
	//
	// Every top-level group is agent-facing except the ones below: credential,
	// config and reference commands aren't agent tasks. Stated as a deny-list
	// because the allow-list had to be edited for every new resource and
	// exposeGroups skips an unmatched name silently, so a rename dropped a tool
	// from the MCP surface with no signal. Note `mcp` itself is registered after
	// this call, so it is not a candidate either way.
	exposeAllGroupsExcept(root, "auth", "config", "usage", "completion", "help")

	root.AddCommand(agentmcp.Command(root, agentmcp.WithHiddenFlags("color", "expose"), agentmcp.WithOAuthKeyringService(credential.MCPKeychainService())))

	return root
}

// installGroupUnknownHandlers gives every command group (a parent that only
// holds subcommands, with no action of its own) the same structured
// unknown-subcommand behavior libcli installs on the root: an unknown
// subcommand returns a fixable_by:agent error listing the valid ones, instead
// of cobra's usage text. Groups already carrying a Run/RunE are left alone.
func installGroupUnknownHandlers(root *cobra.Command) {
	for _, group := range root.Commands() {
		if !group.HasSubCommands() || group.Run != nil || group.RunE != nil {
			continue
		}
		hint := "run '" + root.Name() + " " + group.Name() + " --help' to see its subcommands"
		libcli.HandleUnknownCommand(group, hint)
	}
}

// wrapConfigDefaults runs the config-defaults pass (explicit flag > config >
// built-in default) ahead of the libcli PersistentPreRunE that validates
// --format.
func wrapConfigDefaults(root *cobra.Command, globals *shared.GlobalFlags, next func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		applyConfiguredDefaults(root, globals)
		if next != nil {
			return next(cmd, args)
		}
		return nil
	}
}

func applyConfiguredDefaults(root *cobra.Command, globals *shared.GlobalFlags) {
	cfg := config.Read()
	flags := root.PersistentFlags()
	if cfg.Defaults.TimeoutMS != nil && !flags.Changed("timeout") {
		globals.TimeoutMS = *cfg.Defaults.TimeoutMS
	}
	if cfg.Defaults.MaxRetries != nil && !flags.Changed("max-retries") {
		globals.MaxRetries = *cfg.Defaults.MaxRetries
	}
}

// Execute builds the root command and runs it via the shared sink, which
// renders any bubbled error as the family's structured JSON on stderr exactly
// once and exits non-zero.
func Execute(version string) {
	libcli.Run(newRootCmd(version))
}

// LeafCommandsForTest lists every runnable leaf command as a space-joined path
// ("investigate account-health"). apicheck uses it to prove its command table
// covers the real tree, rather than relying on a comment asking contributors to
// keep the two in sync.
func LeafCommandsForTest(version string) []string {
	var walk func(cmd *cobra.Command, prefix []string) []string
	walk = func(cmd *cobra.Command, prefix []string) []string {
		var out []string
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			path := append(append([]string{}, prefix...), child.Name())
			if len(child.Commands()) > 0 {
				out = append(out, walk(child, path)...)
				continue
			}
			if child.RunE == nil && child.Run == nil {
				continue
			}
			out = append(out, strings.Join(path, " "))
		}
		return out
	}
	return walk(newRootCmd(version), nil)
}

// RunForTest executes one command in-process and returns the exit code the
// binary would have used. The e2e suite drove the CLI with `go run`, which put
// every assertion in a subprocess: none of it counted toward coverage, -race
// never applied, and each case paid a recompile. Output goes through
// output.SetWritersForTest, so callers capture stdout and stderr as usual.
func RunForTest(version string, args []string) int {
	root := newRootCmd(version)
	root.SetArgs(args)
	root.SetOut(output.Stdout())
	root.SetErr(output.Stderr())
	if err := root.Execute(); err != nil {
		output.WriteError(output.Stderr(), err)
		return 1
	}
	return 0
}

// exposeAllGroupsExcept opts every top-level command into the MCP tool surface
// apart from the named ones, so a newly registered group is exposed by default
// rather than needing to be remembered.
func exposeAllGroupsExcept(root *cobra.Command, names ...string) {
	skip := make(map[string]bool, len(names))
	for _, n := range names {
		skip[n] = true
	}
	for _, c := range root.Commands() {
		if skip[c.Name()] || c.Hidden {
			continue
		}
		agentmcp.Expose(c)
	}
}
