package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUsageCommand(text string) *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "LLM-optimized reference card",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(text)
		},
	}
}

func registerPaymentsDomain(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "payments",
		Short: "Payment triage reference",
	}
	cmd.AddCommand(newUsageCommand(paymentsUsageText))
	root.AddCommand(cmd)
}

func registerConnectDomain(root *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect money movement triage reference",
	}
	cmd.AddCommand(newUsageCommand(connectUsageText))
	root.AddCommand(cmd)
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
  Use --expand payment_intent on direct invoice reads when you need raw expanded fields.
`

const subscriptionsUsageText = `subscriptions — renewal, collection, and item triage

COMMON STARTS
  agent-stripe subscriptions get sub_... --expand latest_invoice --expand latest_invoice.payment_intent
  agent-stripe subscriptions list --customer cus_... --status active|past_due|unpaid|all
  agent-stripe subscriptions invoices sub_... --status open
  agent-stripe subscriptions items sub_... --expand data.price.product
  agent-stripe investigate subscription-renewal --subscription sub_...
  agent-stripe investigate subscription-renewal --customer cus_...
  agent-stripe investigate subscription-renewal --metadata tenant_id=acme
  agent-stripe investigate subscription-items --subscription sub_...
  agent-stripe investigate subscription-amount-change --subscription sub_...
  agent-stripe investigate collection-risk --days 30
  agent-stripe investigate subscription-cancel-risk --days 30

QUESTIONS THIS ANSWERS
  Last paid invoice, latest PaymentIntent/Charge, next renewal time, next preview amount.
  Which Price/Product/metadata drove a subscription charge.
  Whether the customer needs outreach for missing, expiring, declined, or action-required payment details.

OUTPUT NOTES
  Investigation output emits subscription, invoice, payment, item, price, and product evidence records.
  Use --full or --expand-field for verbose item/product metadata.
`

const paymentsUsageText = `payments — PaymentIntent, Charge, card, and failure triage

COMMON STARTS
  agent-stripe payment-intents get pi_... --expand latest_charge
  agent-stripe payment-intents search --query "metadata['order_id']:'123'"
  agent-stripe charges get ch_... --expand payment_intent --expand balance_transaction
  agent-stripe charges search --query "payment_method_details.card.last4:'4242'"
  agent-stripe payment-methods list --customer cus_... --type card
  agent-stripe investigate incoming-payment pi_...
  agent-stripe investigate incoming-payment ch_...
  agent-stripe investigate customer-card-payment --customer cus_... --last4 4242

PAYMENT FAILURE FLOW
  Start with incoming-payment when the user gives a PaymentIntent, Charge, or Invoice ID.
  It emits the PaymentIntent/Charge/Invoice plus related disputes and refunds when available.
  Findings summarize status, decline details, failure messages, and action-required states.

CARD LAST4 FLOW
  Last4 is not unique, so include --customer.
  Use customer-card-payment for the most recent matching charge.
`

const connectUsageText = `connect — connected-account and money movement triage

COMMON STARTS
  agent-stripe accounts self
  agent-stripe accounts get acct_...
  agent-stripe transfers list --destination acct_...
  agent-stripe payouts get po_...
  agent-stripe balance-transactions get txn_...
  agent-stripe application-fees list --charge ch_...
  agent-stripe refunds list --payment-intent pi_...
  agent-stripe investigate outgoing-payment tr_...
  agent-stripe investigate outgoing-payment po_...
  agent-stripe investigate outgoing-payment acct_...
  agent-stripe investigate refund-status re_...
  agent-stripe investigate payout-failure po_...
  agent-stripe investigate refund-recovery trr_... --transfer tr_...

CONTEXT
  Use --context for organization keys or related-account requests.
  Connected-account failures often involve requirements, external accounts, balance transactions, transfers, reversals, and payouts.

OUTPUT NOTES
  Findings summarize failed/canceled/reversed movement and include failure_code, failure_message, failure_reason, or failure balance transaction when present.
`
