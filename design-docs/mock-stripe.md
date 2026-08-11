# mockstripe design

## Purpose

`mockstripe` is a small local HTTP server that returns Stripe-shaped JSON for agent-stripe e2e tests. It exists so command behavior can be tested safely without using live Stripe credentials, sandbox state, rate limits, or network access.

## Scope

The mock should cover the read-first command surface and the safe read-like POSTs used by investigation workflows:

- `GET /v1/account`
- `GET /v1/balance`
- `GET /v1/customers`, `GET /v1/customers/:id`, `GET /v1/customers/search`
- `GET /v1/events`, `GET /v1/events/:id`
- `GET /v1/products`, `GET /v1/products/:id`, `GET /v1/products/search`
- `GET /v1/prices`, `GET /v1/prices/:id`, `GET /v1/prices/search`
- `GET /v1/payment_intents`, `GET /v1/payment_intents/:id`, `GET /v1/payment_intents/search`
- `GET /v1/setup_intents`, `GET /v1/setup_intents/:id`
- `GET /v1/payment_methods`, `GET /v1/payment_methods/:id`
- `GET /v1/charges`, `GET /v1/charges/:id`, `GET /v1/charges/search`
- `GET /v1/disputes`, `GET /v1/disputes/:id`
- `GET /v1/refunds`, `GET /v1/refunds/:id`
- `GET /v1/subscriptions`, `GET /v1/subscriptions/:id`, `GET /v1/subscriptions/search`
- `GET /v1/subscription_items`, `GET /v1/subscription_items/:id`
- `GET /v1/invoices`
- `GET /v1/invoices/:id`
- `GET /v1/invoices/:id/lines`
- `GET /v1/invoices/search`
- `POST /v1/invoices/create_preview`
- `GET /v1/checkout/sessions`, `GET /v1/checkout/sessions/:id`, `GET /v1/checkout/sessions/:id/line_items`
- `GET /v1/transfers`, `GET /v1/transfers/:id`, `GET /v1/transfers/:id/reversals/:reversal_id`
- `GET /v1/payouts`, `GET /v1/payouts/:id`
- `GET /v1/balance_transactions`, `GET /v1/balance_transactions/:id`
- `GET /v1/application_fees`, `GET /v1/application_fees/:id`
- `GET /v1/payment_links`, `GET /v1/payment_links/:id`
- `GET /v1/radar/early_fraud_warnings`, `GET /v1/radar/early_fraud_warnings/:id`
- `GET /v1/accounts`
- `GET /v1/accounts/:id`

It also covers the `/v2` (UA2) surface the CLI wraps:

- `GET /v2/core/accounts`, `GET /v2/core/accounts/:id`
- `GET /v2/core/accounts/:id/persons`, `GET /v2/core/accounts/:id/persons/:person_id`
- `GET /v2/core/events`, `GET /v2/core/events/:id`

The v2 routes reproduce the parts of the v2 transport that command code has to get right, not just the JSON bodies:

- `Authorization: Bearer <key>` is accepted on `/v2` (and required — a Basic-only client is rejected), while `/v1` keeps Basic.
- Indexed array parameters (`include[0]`, `applied_configurations[0]`, `types[0]`) are parsed; the non-indexed form is also accepted so a hand-written request still works.
- Sparse responses: `configuration`, `identity`, `requirements`, `future_requirements`, and `defaults` are `null` unless named in `include`.
- Pagination returns `next_page_url` with a `page` token and no `has_more`.
- A v1-only fixture account returns 400 `v1_account_instead_of_v2_account` from `/v2/core/accounts/:id`, so namespace fallback is exercised end to end.

The mock intentionally rejects missing credentials and unsupported methods. Most Stripe-shaped endpoints are GET-only; `POST /v1/invoices/create_preview` is allowed because it is the API shape Stripe uses for previewing a future invoice and is central to subscription investigations.

## CLI wiring

The real CLI defaults to `https://api.stripe.com`. Tests can override that with:

```bash
AGENT_STRIPE_BASE_URL=http://127.0.0.1:12111
```

There is also a hidden `--base-url` flag for subprocess e2e tests. It should stay hidden from normal user help so LLM-facing guidance continues to emphasize real Stripe profile setup rather than internal test plumbing.

For development:

```bash
make mock
make mock-dev ARGS="events list --type charge.failed"
mockstripe --routes
```

The mock server exposes a route map at `GET /` without authentication. Stripe-shaped `/v1/...` endpoints still require a mock Basic API key so tests exercise auth wiring.

## Fixtures

Fixtures should be intentionally small but incident-shaped:

- a succeeded PaymentIntent and charge
- a failed PaymentIntent and declined charge
- a `charge.failed` event with request metadata
- a `payment_intent.succeeded` event
- a dispute needing response
- a connected account missing an external account
- v2 accounts covering the states that change triage: a fully active merchant/customer account, a merchant account with `card_payments` restricted and past-due requirement entries that name the capabilities they restrict, and a recipient-only account with an eventually-due requirement
- v2 account persons including a representative with an outstanding verification error
- v2 core events for capability status, requirements, and identity updates on those accounts
- subscription renewal, past-due, cancellation, missing payment method, and expiring-card cases
- invoice line items, subscription items, prices, and products with metadata
- refunds, transfer reversals, payouts, balance transactions, application fees, payment links, Checkout Sessions, SetupIntents, and early fraud warnings
- fake sensitive fields such as client secrets, customer contact fields, receipt/invoice URLs, card fingerprints, and token-like metadata so redaction can be tested safely

Prefer adding focused fixture fields that support a real triage workflow over copying full Stripe objects.
