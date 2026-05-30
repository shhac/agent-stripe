# Stripe Docs Alignment

This file records checks against Stripe's public API docs so agent-stripe stays close to the API's real shape. Last checked: 2026-05-30.

## Expandable fields

Stripe marks related ID fields as expandable. When an expanded object is returned, the field that is normally a string ID can become an object. For list responses, expand paths move through `data`, such as `data.payment_method`.

agent-stripe keeps the parent field as the navigable ID in investigation output and emits expanded child objects as their own records. That preserves Stripe's object shape for normal resources while making expanded responses easier for an LLM to traverse.

Source: https://docs.stripe.com/api/expanding_objects

## Account context

`Stripe-Context` is Stripe's current header for requests scoped to related accounts. It supersedes `Stripe-Account` and also covers organization and v2 account contexts. agent-stripe therefore exposes `--context` and stores a default context per profile.

Some Connect API reference examples still show `Stripe-Account`. We should prefer `Stripe-Context` in user-facing docs unless a Stripe endpoint or user environment specifically requires the older header.

Sources:
- https://docs.stripe.com/context
- https://docs.stripe.com/api/connected-accounts

## Pagination

Stripe list endpoints use `limit`, `starting_after`, and `ending_before`. Search endpoints use `limit`, `page`, and `next_page`. Resource list commands expose both list cursors; search commands expose `--page`.

Sources:
- https://docs.stripe.com/api/pagination
- https://docs.stripe.com/search

## Search

Stripe search is not read-after-write consistent. Investigation commands should use direct retrieval or list endpoints when the user already has an ID or needs freshly-created data. Search is appropriate for flexible lookup, including invoice numbers and metadata, and generated clauses quote and escape values before sending them.

Source: https://docs.stripe.com/search

## Invoice previews and billing

Stripe's preview endpoint is `POST /v1/invoices/create_preview`. Passing `subscription` previews the upcoming invoice for an existing subscription; `preview_mode=recurring` can be used when a recurring estimate is desired.

For subscription item inspection, Stripe lists items at `GET /v1/subscription_items?subscription=<id>`. Invoice line items and Checkout Session line items are separate list endpoints, so agent-stripe exposes those as first-class commands.

Sources:
- https://docs.stripe.com/api/invoices/create_preview
- https://docs.stripe.com/api/subscription_items/list
- https://docs.stripe.com/api/checkout/sessions/line_items

## Connect reversals

Transfer reversals are nested under the transfer: `GET /v1/transfers/<transfer>/reversals/<reversal>`. A reversal ID alone is not enough to build that API path, so investigation commands that start from a reversal require or discover the transfer ID.

Source: https://docs.stripe.com/api/transfer_reversals/retrieve
