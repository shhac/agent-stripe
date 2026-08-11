package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/output"
)

func newUsageCommand(text string) *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "LLM-optimized reference card",
		Run: func(cmd *cobra.Command, args []string) {
			// Through the output writer, not os.Stdout, so usage text is
			// redirectable like every other command's output.
			_, _ = fmt.Fprint(output.Stdout(), text)
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
  PaymentIntent, Charge, and PaymentMethod lists are compact by default; use list --full or get <id> for full objects.
  Card last4 stays visible for triage; fingerprints, receipts, tokens, and client secrets are redacted by default.
`

const connectUsageText = `connect — connected-account and money movement triage

TWO ACCOUNT NAMESPACES
  Stripe has two connected-account models and both use the acct_ prefix:
    Connect v1  /v1/accounts       -> 'accounts'      charges_enabled, payouts_enabled, requirements arrays
    Accounts v2 /v2/core/accounts  -> 'accounts-v2'   configurations, nested capabilities, requirement entries
  If you do not know which one an ID is, do not guess:
    agent-stripe investigate resolve acct_...            names the namespace
    agent-stripe investigate account-health acct_...     probes v2, falls back to v1, and says which answered
  A v2 account ID also works on v1 money-movement endpoints (transfers, payouts, balance transactions,
  application fees), so those commands are namespace-independent. A v1-only ID is rejected by /v2.

COMMON STARTS
  agent-stripe accounts self
  agent-stripe accounts get acct_...
  agent-stripe accounts-v2 get acct_...
  agent-stripe accounts-v2 list --applied-configuration merchant
  agent-stripe accounts-v2 persons list acct_...
  agent-stripe events-v2 list --object-id acct_...
  agent-stripe investigate account-events acct_...
  agent-stripe transfers list --destination acct_...
  agent-stripe payouts get po_...
  agent-stripe balance-transactions get txn_...
  agent-stripe application-fees list --charge ch_...
  agent-stripe refunds list --payment-intent pi_...
  agent-stripe investigate outgoing-payment tr_...
  agent-stripe investigate outgoing-payment po_...
  agent-stripe investigate outgoing-payment acct_...
  agent-stripe investigate account-health acct_... [--namespace auto|v1|v2]
  agent-stripe investigate ledger ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...
  agent-stripe investigate refund re_...|ch_...|pi_...
  agent-stripe investigate payout-failure po_...
  agent-stripe investigate refund-recovery trr_... --transfer tr_...

WHY IS THIS ACCOUNT BLOCKED?
  v1: charges_enabled / payouts_enabled plus requirements.currently_due field names.
  v2: no such flags. Each capability has its own status per configuration
      (merchant.card_payments, merchant.stripe_balance.payouts, recipient.bank_accounts.local),
      and each requirement entry says who must act (awaiting_action_from) and which capabilities
      it restricts. account-health reports whichever model applies and never mixes them.

WHAT CHANGED RECENTLY?
  v1 activity: agent-stripe events list --type account.updated
  v2 activity: agent-stripe investigate account-events acct_...
  Accounts v2 capability and requirement changes exist ONLY as v2 thin events, so /v1/events
  will never show them. Thin events carry no snapshot; state fetched for them is current state.

CONTEXT
  Use --context for organization keys or related-account requests. It applies to /v1 and /v2 alike.
  Connected-account failures often involve requirements, external accounts, balance transactions, transfers, reversals, and payouts.

OUTPUT NOTES
  Findings summarize failed/canceled/reversed movement and include failure_code, failure_message, failure_reason, or failure balance transaction when present.
  accounts list and accounts-v2 list are compact by default; use --full or get <acct_id> for full objects.
  accounts-v2 get requests every include by default, because Stripe returns null for fields you did not ask for.
  v2 lists paginate by token: read @pagination.next_page and pass it back as --page.
  Connected account IDs and ledger IDs stay visible; person names, dates of birth, and contact fields are redacted unless exposed.
`
