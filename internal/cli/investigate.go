package cli

import (
	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func registerInvestigate(root *cobra.Command, globals shared.GlobalsFunc) {
	outputOpts := &evidenceOptions{maxString: defaultMaxString}
	investigate := &cobra.Command{
		Use:     "investigate",
		Aliases: []string{"invest"},
		Short:   "Opinionated Stripe incident investigations",
	}
	for _, newCommand := range investigationCommands {
		investigate.AddCommand(newCommand(globals, outputOpts))
	}
	investigate.AddCommand(newUsageCommand(investigationUsageText))
	investigate.PersistentFlags().BoolVar(&outputOpts.full, "full", false, "Do not truncate investigation entity fields")
	investigate.PersistentFlags().StringArrayVar(&outputOpts.expandFields, "expand-field", nil, "Do not truncate a field path in investigation output; repeatable")
	investigate.PersistentFlags().IntVar(&outputOpts.maxString, "max-string", defaultMaxString, "Maximum string length in investigation entity fields before truncation")
	root.AddCommand(investigate)
}

type investigationCommandFactory func(shared.GlobalsFunc, *evidenceOptions) *cobra.Command

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
	newInvestigateAccountEvents,
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
  Resolve identifies likely object type and next commands. For an acct_ ID it also names which account
  namespace the ID belongs to (Connect v1 or Accounts v2), which the prefix alone cannot tell you.
  Customer context gathers customer, payment methods, subscriptions, invoices, payment intents, charges,
  disputes, and refunds.

CUSTOMER CARD LAST4
  agent-stripe investigate customer-card-payment --customer cus_... --last4 4242 [--limit 25]
  Use when a customer gave only card last4. Last4 is not unique, so --customer is required.

WEBHOOKS AND DISPUTES
  agent-stripe investigate webhook-event evt_...
  agent-stripe investigate webhook-delivery evt_... [--endpoint we_...]
  agent-stripe investigate webhook-delivery we_...
  agent-stripe investigate dispute-response dp_...
  agent-stripe investigate dispute-impact dp_...|ch_...|cus_...
  Webhook-event fetches the event and underlying object; it accepts v2 core event IDs too, following the
  thin event's related_object to current state. Dispute-response summarizes due date, reason, status,
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

CONNECTED ACCOUNTS (CONNECT v1 AND ACCOUNTS v2 / UA2)
  agent-stripe investigate account-health acct_... [--namespace auto|v1|v2]
  agent-stripe investigate account-events acct_... [--limit 20]
  Stripe has two connected-account models sharing the acct_ prefix. account-health defaults to
  --namespace auto: it reads /v2/core/accounts first and falls back to /v1/accounts when Stripe says the
  ID is not a v2 account, then names the namespace that answered in a finding.
  v2 findings report per-configuration capability status (merchant.card_payments,
  merchant.stripe_balance.payouts, recipient.bank_accounts.local) and requirement entries with
  awaiting_action_from plus the capabilities each entry restricts. A v2 account has no
  charges_enabled/payouts_enabled, and those are never reported for one.
  account-events reads /v2/core/events, the only place Accounts v2 capability and requirement changes
  appear. Those events are thin, so any object shown with them is current state, not point-in-time.

CONNECT MONEY MOVEMENT
  agent-stripe investigate outgoing-payment tr_...
  agent-stripe investigate outgoing-payment po_...
  agent-stripe investigate outgoing-payment acct_...
  agent-stripe investigate ledger ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...
  agent-stripe investigate refund re_...|ch_...|pi_...
  agent-stripe investigate payout-failure po_...
  agent-stripe investigate refund-recovery re_...
  agent-stripe investigate refund-recovery trr_... --transfer tr_...
  Use ledger for reconciliation, refund for customer-visible refund state, and refund-recovery for refund
  funding or transfer reversal recovery. These endpoints are /v1 for both account namespaces, and a v2
  account ID is accepted by them, so they need no namespace choice.

RISK AND FRAUD
  agent-stripe investigate fraud-review issfr_...|ch_...|pi_...
  agent-stripe investigate dispute-impact dp_...|ch_...|cus_...
  Use fraud-review for Radar early fraud warnings, charge risk outcome, disputes, and refunds.
`
