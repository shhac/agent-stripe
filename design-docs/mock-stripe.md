# mockstripe design

## Purpose

`mockstripe` is a small local HTTP server that returns Stripe-shaped JSON for agent-stripe e2e tests. It exists so command behavior can be tested safely without using live Stripe credentials, sandbox state, rate limits, or network access.

## Scope

The mock should cover the read-first command surface:

- `GET /v1/account`
- `GET /v1/balance`
- `GET /v1/events`
- `GET /v1/events/:id`
- `GET /v1/payment_intents`
- `GET /v1/payment_intents/:id`
- `GET /v1/payment_intents/search`
- `GET /v1/charges`
- `GET /v1/charges/:id`
- `GET /v1/charges/search`
- `GET /v1/disputes`
- `GET /v1/disputes/:id`
- `GET /v1/subscriptions`
- `GET /v1/subscriptions/:id`
- `GET /v1/subscriptions/search`
- `GET /v1/subscription_items`
- `GET /v1/invoices`
- `GET /v1/accounts`
- `GET /v1/accounts/:id`

The mock intentionally rejects missing credentials and non-GET methods. This keeps tests honest about authentication wiring while preserving the CLI's read-only safety posture.

## CLI wiring

The real CLI defaults to `https://api.stripe.com`. Tests can override that with:

```bash
AGENT_STRIPE_BASE_URL=http://127.0.0.1:12111
```

There is also a hidden `--base-url` flag for subprocess e2e tests. It should stay hidden from normal user help so LLM-facing guidance continues to emphasize real Stripe profile setup rather than internal test plumbing.

## Fixtures

Fixtures should be intentionally small but incident-shaped:

- a succeeded PaymentIntent and charge
- a failed PaymentIntent and declined charge
- a `charge.failed` event with request metadata
- a `payment_intent.succeeded` event
- a dispute needing response
- a connected account missing an external account

Prefer adding focused fixture fields that support a real triage workflow over copying full Stripe objects.
