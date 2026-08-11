package cli

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerInvoices(root *cobra.Command, globals shared.GlobalsFunc) {
	registerResource(root, globals, resourceOptions{
		use:         "invoices",
		short:       "Invoice lookup, line items, and payment bridge investigation",
		path:        "/v1/invoices",
		idName:      "invoice-id",
		idKind:      "invoice",
		searchable:  true,
		searchHint:  "Use a Stripe search query, for example number:'ABC-0001' or metadata['order_id']:'123'",
		usageText:   invoicesUsageText,
		expandGet:   true,
		expandList:  true,
		listSummary: invoiceListSummary,
		listFlags: []listFlag{
			{name: "customer", param: "customer", help: "Customer ID"},
			{name: "subscription", param: "subscription", help: "Subscription ID"},
			{name: "status", param: "status", help: "Invoice status: draft, open, paid, uncollectible, void"},
		},
		extraCommands: []func(shared.GlobalsFunc) *cobra.Command{
			newInvoiceLineItemsCommand,
			func(globals shared.GlobalsFunc) *cobra.Command {
				return newInvoicePreviewCommand(globals, "/v1/invoices/create_preview", []listFlag{
					{name: "customer", param: "customer", help: "Customer ID"},
					{name: "subscription", param: "subscription", help: "Subscription ID"},
					{name: "preview-mode", param: "preview_mode", help: "Preview mode: next or recurring"},
				})
			},
		},
	})
}

func newInvoiceLineItemsCommand(globals shared.GlobalsFunc) *cobra.Command {
	var limit int
	var cursor cursorFlags
	cmd := &cobra.Command{
		Use:   "line-items <invoice-id>",
		Short: "List line items on an invoice",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateExpectedStripeID(args[0], "invoice"); err != nil {
				return err
			}
			params := url.Values{}
			shared.AddLimit(params, limit)
			cursor.AddTo(params)
			return shared.GetRawList(globals(), "/v1/invoices/"+url.PathEscape(args[0])+"/lines", params)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum results to return (1-100)")
	cursor.AddFlags(cmd)
	return cmd
}

func newInvoicePreviewCommand(globals shared.GlobalsFunc, path string, flags []listFlag) *cobra.Command {
	values := make(map[string]*string, len(flags))
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Create a preview invoice for a customer or subscription",
		RunE: func(cmd *cobra.Command, args []string) error {
			params := url.Values{}
			for _, flag := range flags {
				shared.AddString(params, flag.param, *values[flag.name])
			}
			return shared.PostFormRawItem(globals(), path, params)
		},
	}
	for _, flag := range flags {
		var value string
		values[flag.name] = &value
		cmd.Flags().StringVar(&value, flag.name, "", flag.help)
	}
	return cmd
}

const invoicesUsageText = `invoices — invoice payment and metadata triage

COMMON STARTS
  agent-stripe invoices get in_... --expand payment_intent
  agent-stripe invoices line-items in_...
  agent-stripe invoices list --customer cus_... --status open
  agent-stripe invoices search --query "number:'ABC-0001'"
  agent-stripe investigate invoice-payment in_...
  agent-stripe investigate invoice-metadata in_...
  agent-stripe investigate invoice-metadata --number ABC-0001

WHEN A CUSTOMER SENDS AN INVOICE COPY
  1. Resolve the invoice number if needed:
     agent-stripe investigate resolve ABC-0001
  2. Find payment details:
     agent-stripe investigate invoice-payment in_...
  3. Find internal product metadata on the PaymentIntent:
     agent-stripe investigate invoice-metadata in_...

OUTPUT NOTES
  Invoice, PaymentIntent, Charge, and line-item IDs are preserved in compact output.
  Invoice lists are compact by default; use invoices list --full or invoices get in_... when you need raw expanded fields.
  Sensitive URLs and customer contact fields are redacted by default; use --expose only when needed.
`
