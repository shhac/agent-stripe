# Investigation Workflows

`agent-stripe investigate` exists for cases where an LLM should not need to manually walk the Stripe object graph. Commands emit NDJSON evidence records:

- `entity`: a Stripe object retrieved during the investigation.
- `finding`: a concise conclusion, warning, or next step.

The resource commands remain available for direct exploration, but investigation commands should encode common incident paths.

Investigation output preserves Stripe-shaped `data` as much as possible. When Stripe returns an expanded nested object in a field that can also be an ID string, the parent field is replaced by the nested object's ID and the nested object is emitted as its own `entity` record. Long strings are truncated by default, with `truncated_fields` pointing at `--expand-field <path>` or `--full`. Sensitive Stripe fields are redacted by default with `"[REDACTED]"` leaf values and a top-level `@redacted` path list; `--expose <field-or-path>` is required to reveal them.

## Adding A Workflow

Each investigation class should be mostly independent:

1. Add a focused file in `internal/cli/investigate_<domain>.go`.
2. Add a `newInvestigate<Name>(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command` factory.
3. Register the factory in `investigationCommands` in `internal/cli/investigate.go`.
4. Keep Stripe traversal methods near the workflow that owns them unless they are clearly shared.
5. Emit raw Stripe objects as `entity` records and conclusions as `finding` records.
6. Add mock fixtures/routes in `internal/mockstripe`.
7. Add a mock-backed e2e test that proves the user-facing scenario works.
8. Update `agent-stripe investigate usage`, top-level `usage`, this design doc, and the skill.
9. Add or update ID-prefix validation when the workflow accepts only specific object types. Use `validateAllowedStripeID` when related prefixes are recoverable, and `validateExpectedStripeID` when the input must be one object type.

Good workflows accept incident-language inputs (`in_...`, `pi_...`, `last4`, metadata, customer, account) and return enough evidence for an LLM to answer without issuing five follow-up raw API calls.

## ID Handling

Direct `get` commands reject known wrong Stripe ID prefixes before making a request. For example, `invoices get pi_...` returns a JSON error that points at `payment-intents get`, `investigate incoming-payment`, and `investigate resolve`.

Investigation commands should be relationship-aware:

- Accept alternate IDs when the workflow can recover by traversing related objects, such as `incoming-payment` accepting `in_`, `ch_`, and `pi_`.
- Reject unrelated known prefixes with `fixable_by=agent` and a suggested command.
- Let unknown strings pass through when a workflow intentionally supports invoice numbers, search terms, or future Stripe ID shapes.
- Keep navigable IDs visible in output even when an expanded object is emitted separately.

Current workflow families:

- `resolve`: identify a Stripe ID or invoice number and suggest next command.
- `customer-context`: gather recent objects around a customer.
- `customer-card-payment`: bounded last4 lookup for a customer.
- `webhook-event`: fetch event and underlying object.
- `dispute-response`: summarize dispute response status and related payment.
- `invoice-payment`: invoice to PaymentIntent to latest Charge.
- `invoice-metadata`: invoice to PaymentIntent metadata.
- `subscription-renewal`: latest and next invoice/payment.
- `subscription-items`: subscription items, prices, products, and product metadata.
- `subscription-amount-change`: latest invoice, invoice lines, preview, and current item subtotal.
- `collection-risk`: payment-method outreach candidates.
- `subscription-cancel-risk`: cancellations and trial/period end risk.
- `incoming-payment`: failed/successful customer payment explanation.
- `outgoing-payment`: transfers, payouts, and connected account readiness.
- `refund-status` and `refund-recovery`: refund and transfer reversal state.
- `payout-failure`: payout failure plus ledger movement.

## Customer Card Last4

Command:

```bash
agent-stripe investigate customer-card-payment --customer cus_... --last4 4242
```

Card last4 is not globally unique, so this command requires a customer. It lists recent charges for that customer and filters locally on `payment_method_details.card.last4`.

## Invoice Payment

Command:

```bash
agent-stripe investigate invoice-payment in_...
```

Path:

```text
Invoice -> PaymentIntent -> latest Charge
```

The finding includes invoice amount paid and card last4 when a card charge is present.

## Subscription Renewal

Command:

```bash
agent-stripe investigate subscription-renewal --subscription sub_...
agent-stripe investigate subscription-renewal --customer cus_...
agent-stripe investigate subscription-renewal --metadata tenant_id=acme
agent-stripe investigate subscription-items --subscription sub_...
agent-stripe investigate subscription-amount-change --subscription sub_...
```

Path:

```text
Subscription -> latest Invoice -> PaymentIntent -> latest Charge
Subscription -> invoice preview
Subscription -> subscription items -> Price -> Product metadata
Subscription -> latest Invoice lines and preview amount
```

This is aimed at support questions such as "when did they last pay, how much, when will they next pay, and how much?"
Use `subscription-items` when the important question is "which internal product IDs or prices are attached?"
Use `subscription-amount-change` when the important question is "why is this invoice amount different?"

## Invoice Metadata

Command:

```bash
agent-stripe investigate invoice-metadata in_...
agent-stripe investigate invoice-metadata --number ABC-0001
```

Path:

```text
Invoice -> PaymentIntent.metadata
```

This handles customer-provided invoice copies where internal product IDs are stored on the PaymentIntent.

## Collection Risk

Command:

```bash
agent-stripe investigate collection-risk --days 30
```

This scans upcoming subscription renewals and flags subscriptions that are already `past_due`, `unpaid`, `incomplete`, have no visible default payment method, have a default card expiring soon, require customer action, or have an open unpaid latest invoice. Future versions should add deeper checks for invoice retry policy.

## Failed Customer Payment

Command:

```bash
agent-stripe investigate incoming-payment pi_...
agent-stripe investigate incoming-payment ch_...
agent-stripe investigate incoming-payment in_...
```

Path:

```text
PaymentIntent or Charge or Invoice -> related charge -> disputes/refunds -> failure fields
```

The finding summarizes charge failure messages, PaymentIntent `last_payment_error`, and related refunds/disputes.

## Connect Money Movement

Commands:

```bash
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate outgoing-payment po_...
agent-stripe investigate outgoing-payment acct_...
agent-stripe investigate refund-recovery re_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

These distinguish Transfers, Payouts, connected Accounts, Refunds, and Transfer Reversals. For transfer reversals, Stripe nests the reversal under its parent transfer, so the command requires `--transfer` when given only a `trr_` ID.

## Improvements To Prioritize

The next likely common triage scenarios:

- Checkout conversion: `Checkout Session -> PaymentIntent/Subscription -> line items -> product/price metadata`.
- Duplicate charge: customer, amount, card last4, and time window to identify repeated PaymentIntents/Charges.
- Unknown statement descriptor: descriptor to Charge/PaymentIntent/Customer candidates.
- Refund missing from bank: Refund -> BalanceTransaction -> Charge -> bank/card network status.
- Failed setup for future billing: SetupIntent -> PaymentMethod -> mandate/last setup error.
- SCA required: PaymentIntent or Invoice requiring customer action, with hosted invoice/payment links when available.
- Product entitlement mismatch: invoice/checkout line items -> Price -> Product metadata/internal IDs.
- Account onboarding blocker: connected Account requirements and recent failed payouts/transfers.
- Balance discrepancy: BalanceTransaction ledger around a charge/refund/transfer/payout.
- Fraud triage: Early Fraud Warning -> Charge -> Customer -> Refund/Dispute state.
- Webhook delivery confusion: Event type, request ID/idempotency key, underlying object, and likely handler action.
- Subscription quantity/price drift: Subscription items -> Price/Product metadata over time.
- Tax or total mismatch: Invoice line items, discounts, tax amounts, customer tax IDs, and final amount paid.
- Payment Link issue: Payment Link -> Checkout Sessions -> line items -> resulting PaymentIntent/Subscription.
- Connect refund liability: Refund with reverse transfer/application fee refund and connected account balance state.

Potential output improvements:

- `--summary-only` for high-level finding records without entity payloads.
- `--include-object <type>` / `--exclude-object <type>` for large investigations.
- Redaction policy presets for stricter customer PII contexts, building on default `@redacted` and `--expose`.
- `--limit-related N` to cap fan-out per related collection.
