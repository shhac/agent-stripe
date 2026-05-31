package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerInvestigate(root *cobra.Command, globals shared.GlobalsFunc) {
	outputOpts := &investigationOutputOptions{maxString: 800}
	investigate := &cobra.Command{
		Use:     "investigate",
		Aliases: []string{"invest"},
		Short:   "Opinionated Stripe incident investigations",
	}
	for _, newCommand := range investigationCommands {
		investigate.AddCommand(newCommand(globals, outputOpts))
	}
	investigate.AddCommand(&cobra.Command{
		Use:   "usage",
		Short: "LLM-optimized investigation reference",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(investigationUsageText)
		},
	})
	investigate.PersistentFlags().BoolVar(&outputOpts.full, "full", false, "Do not truncate investigation entity fields")
	investigate.PersistentFlags().StringArrayVar(&outputOpts.expandFields, "expand-field", nil, "Do not truncate a field path in investigation output; repeatable")
	investigate.PersistentFlags().IntVar(&outputOpts.maxString, "max-string", 800, "Maximum string length in investigation entity fields before truncation")
	root.AddCommand(investigate)
}

type investigationCommandFactory func(shared.GlobalsFunc, *investigationOutputOptions) *cobra.Command

var investigationCommands = []investigationCommandFactory{
	newInvestigateResolve,
	newInvestigateCustomerContext,
	newInvestigateCustomerCardPayment,
	newInvestigateWebhookEvent,
	newInvestigateWebhookDelivery,
	newInvestigateDisputeResponse,
	newInvestigateDisputeImpact,
	newInvestigateInvoicePayment,
	newInvestigateInvoiceCollection,
	newInvestigateSubscriptionRenewal,
	newInvestigateSubscriptionCancelRisk,
	newInvestigateInvoiceMetadata,
	newInvestigateCollectionRisk,
	newInvestigateSubscriptionItems,
	newInvestigateSubscriptionAmountChange,
	newInvestigateEntitlement,
	newInvestigateIncomingPayment,
	newInvestigateCheckoutSession,
	newInvestigatePaymentMethodReadiness,
	newInvestigateSetup,
	newInvestigateTimeline,
	newInvestigateOutgoingPayment,
	newInvestigateAccountHealth,
	newInvestigateLedger,
	newInvestigateFraudReview,
	newInvestigateRefund,
	newInvestigatePayoutFailure,
	newInvestigateRefundRecovery,
}

const investigationUsageText = `agent-stripe investigate — scenario workflows for Stripe incident triage

OUTPUT
  NDJSON by default. Records are:
    {"type":"entity","object":"invoice|payment_intent|charge|...","id":"...","data":{...}}
    {"type":"finding","severity":"info|warning","summary":"...","data":{...}}
  Expanded nested Stripe objects are emitted as their own entity records and replaced by ID in the parent.
  Use --full or --expand-field <path> for long fields that were truncated.
  Sensitive Stripe fields are replaced by "[REDACTED]" and indexed in @redacted unless exposed with --expose <path,key>.

RESOLUTION AND CONTEXT
  agent-stripe investigate resolve <stripe-id-or-invoice-number>
  agent-stripe investigate customer-context --customer cus_... [--limit 5]
  Resolve identifies likely object type and next commands. Customer context gathers customer, payment methods,
  subscriptions, invoices, payment intents, charges, disputes, and refunds.

CUSTOMER CARD LAST4
  agent-stripe investigate customer-card-payment --customer cus_... --last4 4242 [--limit 25]
  Use when a customer gave only card last4. Last4 is not unique, so --customer is required.

WEBHOOKS AND DISPUTES
  agent-stripe investigate webhook-event evt_...
  agent-stripe investigate webhook-delivery evt_... [--endpoint we_...]
  agent-stripe investigate webhook-delivery we_...
  agent-stripe investigate dispute-response dp_...
  agent-stripe investigate dispute-impact dp_...|ch_...|cus_...
  Webhook-event fetches the event and underlying object. Dispute-response summarizes due date, reason, status,
  related charge, customer, and PaymentIntent. Use webhook-delivery for pending_webhooks and endpoint config.
  Use dispute-impact when the question is revenue exposure or customer/account impact.

INVOICE QUESTIONS
  agent-stripe investigate invoice-payment in_...
  agent-stripe investigate invoice-collection in_...|cus_...|sub_...
  agent-stripe investigate invoice-metadata in_...
  agent-stripe investigate invoice-metadata --number ABC-0001
  Walks Invoice -> PaymentIntent -> latest Charge. Use invoice-metadata when internal IDs live on PaymentIntent metadata.
  Use invoice-collection for open/past-due invoices, attempt count, next retry, and hosted invoice URL.

SUBSCRIPTION QUESTIONS
  agent-stripe investigate subscription-renewal --subscription sub_...
  agent-stripe investigate subscription-renewal --customer cus_...
  agent-stripe investigate subscription-renewal --metadata tenant_id=acme
  agent-stripe investigate subscription-items --subscription sub_...
  agent-stripe investigate subscription-amount-change --subscription sub_...
  agent-stripe investigate entitlement --subscription sub_...|--customer cus_...|--metadata key=value|--invoice in_...|--checkout-session cs_...
  agent-stripe investigate collection-risk --days 30 [--limit 25]
  agent-stripe investigate subscription-cancel-risk --days 30 [--limit 25]
  Uses latest invoice, invoice preview, items, prices, products, and payment methods to explain renewals and outreach candidates.
  Use entitlement for internal product/price metadata across subscriptions, invoices, and Checkout.

FAILED CUSTOMER PAYMENTS
  agent-stripe investigate incoming-payment pi_...
  agent-stripe investigate incoming-payment ch_...
  agent-stripe investigate incoming-payment in_...
  agent-stripe investigate checkout-session cs_...
  agent-stripe investigate payment-method-readiness cus_...|pm_...
  agent-stripe investigate setup seti_...|pm_...|cus_...
  agent-stripe investigate timeline cus_...
  Pulls PaymentIntent/Charge/Invoice evidence plus related disputes and refunds.
  Checkout-session follows Checkout -> line items -> resulting payment/subscription. Timeline creates ordered customer context.

CONNECT MONEY MOVEMENT
  agent-stripe investigate account-health acct_...
  agent-stripe investigate outgoing-payment tr_...
  agent-stripe investigate outgoing-payment po_...
  agent-stripe investigate outgoing-payment acct_...
  agent-stripe investigate ledger ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...
  agent-stripe investigate refund re_...|ch_...|pi_...
  agent-stripe investigate payout-failure po_...
  agent-stripe investigate refund-recovery re_...
  agent-stripe investigate refund-recovery trr_... --transfer tr_...
  Use account-health for connected account blockers, ledger for reconciliation, refund for customer-visible refund state,
  and refund-recovery for refund funding or transfer reversal recovery.

RISK AND FRAUD
  agent-stripe investigate fraud-review issfr_...|ch_...|pi_...
  agent-stripe investigate dispute-impact dp_...|ch_...|cus_...
  Use fraud-review for Radar early fraud warnings, charge risk outcome, disputes, and refunds.
`
