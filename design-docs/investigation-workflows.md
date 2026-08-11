# Investigation Workflows

`agent-stripe investigate` exists for cases where an LLM should not need to manually walk the Stripe object graph. Commands emit NDJSON evidence records:

- `entity`: a Stripe object retrieved during the investigation.
- `finding`: a concise conclusion, warning, or next step.

The resource commands remain available for direct exploration, but investigation commands should encode common incident paths.

Investigation output preserves Stripe-shaped `data` as much as possible. When Stripe returns an expanded nested object in a field that can also be an ID string, the parent field is replaced by the nested object's ID and the nested object is emitted as its own `entity` record. Long strings are truncated by default, with `truncated_fields` pointing at `--expand-field <path>` or `--full`. Sensitive Stripe fields are redacted by default with `"[REDACTED]"` leaf values and a top-level `@redacted` path list; `--expose <field-or-path>` is required to reveal them.

## Evidence Pipeline

There is one accumulator: the collector. A workflow calls `i.add(...)` where it finds a record and returns `error`; it does not thread a `[]evidenceRecord` slice through its call graph. Records are normalized and deduped exactly once, inside `collector.add`, and are then either streamed as they arrive (NDJSON) or buffered for a single envelope (`--format json|yaml`). Both paths therefore emit the same records in the same order.

This replaced a threaded slice that was really the collector's own list handed back to callers, alongside a second normalize-and-dedup path used only by the buffered format. The two disagreed: `--format json` emitted a duplicate entity and dropped one that NDJSON showed.

**One object, one record.** The fetch helpers (`get`, `list`, `listV2`) record what they fetch, under Stripe's own `object` name. A caller must not then add the same object again — the collector keys on object *and* ID, so a differently-labelled second add survives as a duplicate rather than being deduped. When an investigation wants a different name (`invoice_preview` for a create_preview result, `line_item` for a Checkout `item`), choose it at the fetch with `postFormAs`/`fetchList`, not afterwards.

Where a workflow needs to know whether a step produced anything, it compares `i.count()` around the step, or tests the domain collection it just fetched — not the length of a records slice, which was always the whole investigation's output.

Related lookups go through `i.fetchRelated` / `i.followRef` / `i.listRelated`, which resolve the API path from the ID prefix and record a warning when the fetch fails. A failed lookup must never read as an absence: "no disputes" and "the dispute call failed" are different answers, and only one of them is safe to act on.

Finding summaries are free text and are **not** passed through the redaction policy. Never interpolate a Stripe object or sub-object into a summary — name the scalar fields you want. `last_payment_error` nests a full PaymentMethod, so `%v` on it leaks billing name, email, phone, and the card fingerprint.

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
- `webhook-delivery`: event `pending_webhooks` plus endpoint config.
- `dispute-response`: summarize dispute response status and related payment.
- `dispute-impact`: dispute exposure from dispute, charge, or customer.
- `invoice-payment`: invoice to PaymentIntent to latest Charge.
- `invoice-collection`: invoice retry/collection state from invoice, customer, or subscription.
- `invoice-metadata`: invoice to PaymentIntent metadata.
- `subscription-renewal`: latest and next invoice/payment.
- `subscription-items`: subscription items, prices, products, and product metadata.
- `subscription-amount-change`: latest invoice, invoice lines, preview, and current item subtotal.
- `entitlement`: product/price metadata across subscriptions, invoices, and Checkout.
- `collection-risk`: payment-method outreach candidates.
- `subscription-cancel-risk`: cancellations and trial/period end risk.
- `incoming-payment`: failed/successful customer payment explanation.
- `checkout-session`: Checkout Session, line items, and resulting payment/subscription.
- `payment-method-readiness`: saved payment method attachment/readiness.
- `setup`: SetupIntent/payment method setup status.
- `timeline`: chronological customer context from recent Stripe objects.
- `outgoing-payment`: transfers, payouts, and connected account readiness.
- `account-health`: connected account requirements and blockers.
- `ledger`: balance transaction and reconciliation evidence.
- `refund` and `refund-recovery`: refund state and transfer reversal recovery.
- `payout-failure`: payout failure plus ledger movement.
- `fraud-review`: Radar early fraud warnings, charge outcome, disputes, and refunds.

## Command Shape Decisions

Keep commands separate when the user's question implies a different starting object, vocabulary, or answer shape. Combine internally when the graph traversal is the same.

- `refund` is the broad public command for refund state from `re_`, `ch_`, or `pi_`; pre-major releases do not keep old alias names.
- `account-health` owns connected account requirement/capability checks. `outgoing-payment acct_...` delegates to it, while `outgoing-payment tr_...|po_...` remains money-movement focused.
- `dispute-response` stays narrow for evidence deadline/response state. `dispute-impact` is broader and can start from a dispute, charge, or customer.
- `subscription-items` stays direct and compact. `entitlement` is the broader "what internal product should they have?" workflow across subscription, invoice, and Checkout evidence.
- `webhook-event` answers "what event was this?" while `webhook-delivery` answers "did our webhook receive it?"

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

## Invoice Collection

Command:

```bash
agent-stripe investigate invoice-collection in_...
agent-stripe investigate invoice-collection cus_...
agent-stripe investigate invoice-collection sub_...
```

Path:

```text
Invoice(s) -> PaymentIntent -> latest Charge
```

This answers open/past-due invoice questions with status, paid flag, amount due, attempt count, next payment attempt, and hosted invoice URL.

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
Use `entitlement` when the user asks whether the customer has the right internal product/plan across subscription items, invoice lines, or Checkout line items.

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

## Checkout, Setup, And Timeline

Commands:

```bash
agent-stripe investigate checkout-session cs_...
agent-stripe investigate payment-method-readiness cus_...|pm_...
agent-stripe investigate setup seti_...|pm_...|cus_...
agent-stripe investigate timeline cus_...
```

Checkout follows the session to line items, prices/products, PaymentIntent, Charge, Subscription, Invoice, Customer, and Payment Link when present. Payment method readiness checks visible saved cards and related SetupIntents. Setup focuses on SetupIntent status and saved payment method usability. Timeline reuses customer context and emits ordered finding records for timestamped customer objects.

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
agent-stripe investigate account-health acct_...
agent-stripe investigate outgoing-payment tr_...
agent-stripe investigate outgoing-payment po_...
agent-stripe investigate outgoing-payment acct_...
agent-stripe investigate ledger ch_...|pi_...|re_...|tr_...|po_...|txn_...|fee_...
agent-stripe investigate refund re_...|ch_...|pi_...
agent-stripe investigate refund-recovery re_...
agent-stripe investigate refund-recovery trr_... --transfer tr_...
```

These distinguish Transfers, Payouts, connected Accounts, Refunds, and Transfer Reversals. For transfer reversals, Stripe nests the reversal under its parent transfer, so the command requires `--transfer` when given only a `trr_` ID.
Use `ledger` when finance/support needs amount, fee, net, and balance transaction evidence.

Money-movement objects (transfers, payouts, balance transactions, application fees) live in `/v1` for both account namespaces, and a v2 account ID is accepted by those v1 endpoints, so these workflows are unchanged by UA2.

## Connected Account Health And UA2

Commands:

```bash
agent-stripe investigate account-health acct_...
agent-stripe investigate account-health acct_... --namespace v1
agent-stripe investigate account-health acct_... --namespace v2
agent-stripe investigate account-events acct_... [--limit 20]
```

`account-health` answers "why can't this account take payments or get paid". The `acct_` prefix does not say which account model applies, so `--namespace auto` (the default) retrieves the v2 account with every `include`, and falls back to `/v1/accounts` on the documented interop errors (`v1_account_instead_of_v2_account`, `account_not_yet_compatible_with_v2`, `accounts_v2_access_blocked`, `non_connect_platform_accounts_v2_access_blocked`) or a 404. A finding names the namespace that answered so follow-up commands target the right one.

```text
acct_ -> v2 account (+ includes) -> capability leaves + requirement entries -> persons -> recent transfers
      \-> v1 account (fallback)  -> charges_enabled/payouts_enabled + requirements arrays -> recent transfers
```

The two shapes are read by separate code paths. The v2 finding never reads `charges_enabled`/`payouts_enabled` (a v2 account has neither), and reports restricted capabilities as `configuration.capability` with their `status_details` codes, requirement entries bucketed by `minimum_deadline.status`, and which capabilities each outstanding requirement restricts. The v1 finding is unchanged.

`account-events` covers "what changed on this account, and when". UA2 capability and requirement transitions are emitted only as v2 thin events, so this reads `/v2/core/events?object_id=acct_...` rather than `/v1/events`. Thin events carry no snapshot, so the finding summarizes the event types and times seen and points at the account state fetched alongside it, which is current rather than point-in-time.

## Webhooks, Disputes, And Fraud

Commands:

```bash
agent-stripe investigate webhook-event evt_...
agent-stripe investigate webhook-delivery evt_... [--endpoint we_...]
agent-stripe investigate webhook-delivery we_...
agent-stripe investigate dispute-response dp_...
agent-stripe investigate dispute-impact dp_...|ch_...|cus_...
agent-stripe investigate fraud-review issfr_...|ch_...|pi_...
```

Webhook delivery uses event `pending_webhooks` and webhook endpoint configuration. It does not claim per-endpoint delivery-attempt logs when Stripe has not returned them. Fraud review follows Early Fraud Warning or Charge/PaymentIntent to charge outcome, disputes, and refunds.

## Customer Billing Questions

```bash
agent-stripe investigate duplicate-charge --customer cus_... [--last4 4242] [--window-hours 24]
agent-stripe investigate statement-descriptor --descriptor "FUREVER" [--customer cus_...]
agent-stripe investigate action-required [--customer cus_...]
agent-stripe investigate refund-settlement re_...|ch_...
agent-stripe investigate invoice-total in_...
```

Each of these answers a question support actually receives, and each does the reasoning the reader would otherwise do by hand:

- **duplicate-charge** groups by amount, currency, and card last4, then clusters by time. Grouping alone would call two legitimate monthly charges duplicates; the window is what makes the claim mean something. Two different cards for the same amount is a customer retrying, not a double charge, so the card is part of the key.
- **statement-descriptor** scans client-side because Stripe's charge search does not index `statement_descriptor`. Bank text is truncated and prefixed, so matching is substring in both directions. The scan only sees the page it fetched — the finding says so when it misses, rather than implying the charge does not exist.
- **action-required** finds payments waiting on the customer rather than failed. The completion URLs are redacted by policy, so the finding reports that a hosted page exists and how to reveal it instead of leaking it into a summary.
- **refund-settlement** separates Stripe sending the money from the bank posting it, which is the distinction behind "we refunded them and nothing arrived", and surfaces the acquirer reference the customer's bank can trace.
- **invoice-total** walks the same arithmetic Stripe does and names the first step that disagrees. Printing the fields alone leaves the reader to do the sum, which is the part they got wrong before asking.

Connect refund liability is folded into `refund-recovery` rather than given its own command: it needs the same refund, transfer, and reversal lookups, and splitting them would have meant two commands answering 80% of the same question. It reports whether the transfer was reversed (the connected account absorbed it) and whether the application fee was refunded (the platform gave back its cut).

Two entries from the original list are already covered: subscription quantity/price drift by `subscription-amount-change` and `subscription-items`, and Payment Link issues by `checkout-session` plus `payment-links get`.

## Improvements To Prioritize

The next likely common triage scenarios:

Potential output improvements:

- `--summary-only` for high-level finding records without entity payloads.
- `--include-object <type>` / `--exclude-object <type>` for large investigations.
- Redaction policy presets for stricter customer PII contexts, building on default `@redacted` and `--expose`.
- `--limit-related N` to cap fan-out per related collection.
