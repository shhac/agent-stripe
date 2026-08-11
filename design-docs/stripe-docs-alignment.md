# Stripe Docs Alignment

This file records checks against Stripe's public API docs so agent-stripe stays close to the API's real shape. Last checked: 2026-08-11.

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

## API v2 namespace

The `/v2` namespace is a different transport, not just more endpoints. Requests and responses are JSON, `Authorization: Bearer` is the documented auth header, `Stripe-Version` is required on every request, arrays in query strings must use indexed bracket notation (`include[0]=...`), and `expand` is not supported — sparse fields are opted into with `include`.

agent-stripe routes on the path prefix: `/v2/...` requests get Bearer auth and the v2 API version; `/v1/...` requests keep Basic auth and the v1 API version. Because the namespaces move on different version trains, the profile stores `api_version` and `v2_api_version` separately.

Sources:
- https://docs.stripe.com/api-v2-overview
- https://docs.stripe.com/api-includable-response-values

## Pagination

Stripe v1 list endpoints use `limit`, `starting_after`, and `ending_before`. Search endpoints use `limit`, `page`, and `next_page`. Resource list commands expose both list cursors; search commands expose `--page`.

v2 list endpoints paginate differently: request a `page` token and read `next_page_url` / `previous_page_url` from the response, with no `has_more`. agent-stripe decodes those into the same `@pagination` record by extracting the `page` token from `next_page_url`, so `--page <token>` works the same way an agent already expects, and keeps the raw URL alongside it.

v2 lists are eventually consistent by default, where v1 top-level lists are immediately consistent. Investigations should not treat a v2 list as read-after-write proof.

Sources:
- https://docs.stripe.com/api/pagination
- https://docs.stripe.com/search
- https://docs.stripe.com/api-v2-overview#list-pagination

## Search

Stripe search is not read-after-write consistent. Investigation commands should use direct retrieval or list endpoints when the user already has an ID or needs freshly-created data. Search is appropriate for flexible lookup, including invoice numbers and metadata, and generated clauses quote and escape values before sending them.

Stripe's search docs currently list supported search resources that match our wrapped search commands: Charges, Customers, Invoices, PaymentIntents, Prices, Products, and Subscriptions. Search has its own rate limit of 20 read operations per second, so broad investigation workflows should avoid using search as a fan-out primitive.

Source: https://docs.stripe.com/search

## Rate limiting

Stripe returns HTTP 429 for rate limits and object lock timeouts. Rate-limited responses include `Stripe-Rate-Limited-Reason` when the limiter caused the response; 429s without that header can be lock timeouts. Stripe recommends exponential backoff with randomness.

agent-stripe retries 429s a small bounded number of times, defaulting to 2. After that, it stops and emits a structured retryable error. Investigation commands should still narrow list filters, avoid unnecessary expansions, and prefer IDs or direct retrieval when available.

Source: https://docs.stripe.com/rate-limits

## Invoice previews and billing

Stripe's preview endpoint is `POST /v1/invoices/create_preview`. Passing `subscription` previews the upcoming invoice for an existing subscription; `preview_mode=recurring` can be used when a recurring estimate is desired.

For subscription item inspection, Stripe lists items at `GET /v1/subscription_items?subscription=<id>`. Invoice line items and Checkout Session line items are separate list endpoints, so agent-stripe exposes those as first-class commands.

Sources:
- https://docs.stripe.com/api/invoices/create_preview
- https://docs.stripe.com/api/subscription_items/list
- https://docs.stripe.com/api/checkout/sessions/line_items

## Accounts v1 and Accounts v2

Stripe now has two connected-account models sharing the `acct_` prefix. A v2 account ID can be passed to v1 Accounts endpoints and comes back v1-shaped; a v1-only account ID passed to a v2 endpoint fails with `v1_account_instead_of_v2_account` or `account_not_yet_compatible_with_v2`. Platforms without UA2 enabled get `accounts_v2_access_blocked` or `non_connect_platform_accounts_v2_access_blocked`.

A `v2.core.account` has no `charges_enabled`, `payouts_enabled`, `details_submitted`, `type`, or `capabilities` map. Enablement lives in nested per-configuration capability leaves, and outstanding information lives in `requirements.entries[]` rather than `requirements.currently_due[]`. Any reader that mixes the two shapes reports false blockers, so agent-stripe keeps the summaries and findings for each shape strictly separate and dispatches on `object`.

Sources:
- https://docs.stripe.com/connect/accounts-v2
- https://docs.stripe.com/api/v2/core/accounts/object
- https://docs.stripe.com/api/v2/core/accounts/retrieve

## v2 core events

`/v2/core/events` is a separate event stream from `/v1/events`, retains 30 days, and only emits thin events: no snapshot payload, just `related_object.{id,type,url}` plus `type`, `created`, and `reason`. UA2 capability and requirement transitions (`v2.core.account[configuration.merchant].capability_status_updated`, `v2.core.account[requirements].updated`) appear only here.

Because a thin event carries no snapshot, following `related_object.url` returns *current* state, not the state at event time. Investigation output labels that explicitly rather than implying a point-in-time snapshot.

Sources:
- https://docs.stripe.com/api/v2/core/events/list
- https://docs.stripe.com/api/v2/core/accounts/event-types

## Connect reversals

Transfer reversals are nested under the transfer: `GET /v1/transfers/<transfer>/reversals/<reversal>`. A reversal ID alone is not enough to build that API path, so investigation commands that start from a reversal require or discover the transfer ID.

Source: https://docs.stripe.com/api/transfer_reversals/retrieve

## ID prefix guidance

Stripe's v1 resource APIs are generally organized around resource-specific IDs. agent-stripe uses known Stripe ID prefixes as a local navigation aid, not as an authorization or existence check:

- Direct resource `get` commands preflight known wrong prefixes and return an `@unresolved` record on stdout (exit 0) with a command suggestion — treated as an item-level miss, not a command-level error.
- Investigation commands can accept multiple related prefixes when the object graph is recoverable.
- Unknown values still pass through to Stripe or search-oriented commands so invoice numbers and future Stripe ID shapes are not rejected prematurely.
- Nested resources remain special: transfer reversal IDs require a parent transfer ID, and line item listing commands validate only the parent resource ID they can route with.
