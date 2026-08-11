# Accounts v2 (UA2) support

Last checked against Stripe docs: 2026-08-11.

## Why this is a separate design

Stripe's `/v2` namespace is not "more `/v1` endpoints". It is a different
transport with a different object model, and Connect accounts exist in both
namespaces at once. Treating a v2 account as if it were a v1 account produces
confidently wrong triage: a `v2.core.account` has no `charges_enabled` or
`payouts_enabled`, so a v1-shaped reader concludes every v2 account is dead
while ignoring the capability statuses and requirement entries that actually
explain the block.

agent-stripe therefore models the two namespaces explicitly rather than trying
to hide the split.

## Transport differences the client must honor

| Concern | `/v1` | `/v2` |
| --- | --- | --- |
| Auth header | `Authorization: Basic base64(key:)` | `Authorization: Bearer <key>` |
| `Stripe-Version` | optional | required on every request |
| Request body | form-encoded | JSON |
| Arrays in query strings | `expand[]=a` | indexed: `include[0]=a&include[1]=b` |
| Pagination request | `starting_after` / `ending_before` | `page` token |
| Pagination response | `has_more` | `next_page_url` / `previous_page_url` |
| Sparse fields | `expand[]` adds nested objects | `include[]` un-nulls top-level fields |
| Events | snapshot payloads | thin events only |
| List consistency | immediate | eventually consistent |

Both namespaces return errors nested under `error` with `type`, `code`, and
`message`, so error classification is shared.

`Stripe-Context` is valid on `/v2` requests and keeps its `--context` meaning.

### API versions

The v1 default (`2025-06-30.basil`) predates the Accounts v2 endpoints, and
sending a newer version to `/v1` would change v1 response shapes. So the client
carries **two** versions and picks by path prefix:

- v1 default: `2025-06-30.basil` (`--api-version`, profile `api_version`)
- v2 default: `2026-07-29.dahlia` (`--v2-api-version`, profile `v2_api_version`)

Some UA2 surface area is still preview-only. Pin a preview train when needed:
`--v2-api-version 2026-07-29.preview`.

## Object model

### Configurations replace account types

A `v2.core.account` has `applied_configurations`, any of `customer`,
`merchant`, `recipient`:

- `merchant` — can accept payments. Owns `card_payments` and
  `stripe_balance.payouts` (the v1 `payouts` capability).
- `customer` — can be charged by the platform; replaces a v1 `Customer` object
  for subscriptions and invoices. Owns `automatic_indirect_tax`.
- `recipient` — can receive funds. Owns `stripe_balance.stripe_transfers` (the
  v1 `transfers` capability) and payout-method capabilities such as
  `bank_accounts.local`.

`dashboard` (`full` / `express` / `none`) replaces the v1 dashboard-type
signal, and `closed` replaces relying on absence.

### Capabilities are nested, and nested unevenly

Capability leaves are `{status, status_details:[{code, resolution}]}` but they
sit at varying depths:

```text
configuration.merchant.capabilities.card_payments.status
configuration.merchant.capabilities.stripe_balance.payouts.status
configuration.recipient.capabilities.bank_accounts.local.status
```

Readers must walk to the leaf (a map containing a string `status`) rather than
assuming a fixed depth. agent-stripe flattens these to dotted capability names
(`merchant.stripe_balance.payouts`) for summaries and findings.

### Requirements are entries, not string arrays

v1 exposes `requirements.currently_due` as a list of field names. v2 exposes
`requirements.entries[]`, each with:

- `description` — machine-readable dotted field path
- `minimum_deadline.status` — `past_due` | `currently_due` | `eventually_due`
- `awaiting_action_from` — `user` (integrator must act) or `stripe`
- `errors[].{code, description}` — why collected info was rejected
- `impact.restricts_capabilities[].{capability, configuration, deadline.status}`
- `reference.{type, person, resource, inquiry}` — where to fix it

plus a rollup at `requirements.summary.minimum_deadline.{status, time}` and a
`requirements.collector` (`stripe` or `application`).

This is strictly richer than v1: a v2 account tells you which capability each
missing field will restrict, and whether you or Stripe is the blocker. Findings
should use it rather than degrading to a v1-shaped "requirements present".

### Sparse responses

v2 returns `null` for `configuration`, `identity`, `requirements`,
`future_requirements`, and `defaults` unless they are named in `include[]` —
`null` means "not requested", not "not set". This is a trap for an agent, so
`accounts-v2 get` requests every include by default and only narrows when
`--include` is given.

## Interop between the namespaces

- A **v2 account ID works on v1 endpoints**. The response is v1-shaped. This is
  why `investigate outgoing-payment`, `ledger`, transfers, and payouts keep
  working unchanged for UA2 accounts.
- A **v1-only account ID does not work on v2 endpoints**. Stripe returns 400
  with `v1_account_instead_of_v2_account` or
  `account_not_yet_compatible_with_v2`.
- A platform not enabled for UA2 gets `accounts_v2_access_blocked` or
  `non_connect_platform_accounts_v2_access_blocked`.

Both namespaces use the `acct_` prefix, so an ID alone cannot tell you which
model applies. Commands that must know probe v2 first and fall back to v1 on
exactly those error codes, then report which namespace answered.

## Command surface

```text
agent-stripe accounts-v2 get <acct_...>... [--include <field>]
agent-stripe accounts-v2 list [--applied-configuration merchant|customer|recipient] [--closed] [--page <token>]
agent-stripe accounts-v2 persons list <acct_...>
agent-stripe accounts-v2 persons get <acct_...> <person_...>
agent-stripe events-v2 list [--object-id <id>] [--type <v2.core...>] [--created-gte <rfc3339>]
agent-stripe events-v2 get <evt_...>

agent-stripe investigate account-health acct_... [--namespace auto|v1|v2]
agent-stripe investigate account-events acct_... [--limit N]
```

`accounts` stays v1-only, `accounts-v2` stays v2-only, and neither silently
retries against the other — an explicit command means an explicit namespace.
`investigate` commands are the place where auto-detection lives, because
triage should not require knowing the answer in advance.

### Namespace resolution in investigations

`investigate account-health` defaults to `--namespace auto`:

1. `GET /v2/core/accounts/{id}` with all includes.
2. On `v1_account_instead_of_v2_account`, `account_not_yet_compatible_with_v2`,
   `accounts_v2_access_blocked`,
   `non_connect_platform_accounts_v2_access_blocked`, or 404, retry against
   `GET /v1/accounts/{id}`.
3. Emit a finding naming the namespace that answered, so downstream commands
   are chosen correctly.

`--namespace v1` skips the probe for platforms that have no UA2 access;
`--namespace v2` fails loudly instead of falling back.

## Findings

### v2 account health

Blockers are derived from real v2 signals only:

- capability leaves whose status is not `active`, reported as
  `configuration.capability` with the `status_details` codes
- requirement entries bucketed by `minimum_deadline.status`, counting only
  `awaiting_action_from = user` as actionable
- `requirements.summary.minimum_deadline` as the deadline rollup
- `impact.restricts_capabilities` to explain which capability each missing
  field is holding back

Severity is `warning` when any capability is restricted/inactive or any
requirement is `past_due`/`currently_due` with `awaiting_action_from = user`;
otherwise `info`.

The v1 path keeps its existing `charges_enabled` / `payouts_enabled` finding.
Neither path ever reads the other's fields.

### Account events

`investigate account-events` lists `/v2/core/events?object_id=acct_...`, which
is the only way to see UA2 capability and requirement transitions — v1
`/v1/events` does not carry them. The finding groups the event types seen
(capability status updates, requirements updates, identity updates) so an agent
can answer "what changed, and when" without reading every thin event.

Thin events carry no snapshot, only `related_object.{id,type,url}`. Where a
v2 event is retrieved through `investigate webhook-event`, agent-stripe follows
`related_object.url` to fetch current state and emits it as its own entity
record, with a finding noting that the state is current rather than
point-in-time.

## Redaction

v2 identity data is more PII-dense than a v1 account: person records carry
names, `date_of_birth`, `id_numbers`, and addresses, and the account itself
carries `contact_email` / `contact_phone`.

The existing policy already masks emails, phones, and `name` under account
context. Three changes extend it:

- the account-context rule for name fields now covers `v2.core.account` and
  `v2.core.account_person` alongside `customer` and `account`, and applies to
  `given_name`, `surname`, and `legal_name` as well as `name`
- `contact_email` and `contact_phone` join the unconditional email/phone list
- `date_of_birth` is masked unconditionally — no triage question needs the
  value rather than its presence

Deliberately left visible: `display_name` (a business label, not a person),
country, entity type, relationship flags, and `id_numbers[].type` — Stripe
returns only the type, never the number. Addresses are treated as they already
are on v1 objects. `--expose` remains the explicit opt-in.

`persons list` summaries go further and omit the PII fields entirely rather
than emitting them masked, keeping relationship and verification structure.

## Coverage of the Connect + UA2 question set

The commands above answer "what is the state of this account" in both
namespaces. Three gaps were closed after the first pass, because triage kept
running out of road at the same three places:

- **What a requirement is asking for.** Requirements name what is missing;
  nothing showed what exists. `accounts persons list`, `accounts capabilities`,
  and `accounts external-accounts` cover the v1 sub-resources. External
  accounts matter for v2 accounts too: Accounts v2 has no external-accounts
  hash and Stripe directs platforms to the v1 endpoint regardless of namespace.
  The v1 Capability objects also carry their own requirements, which the
  Account object's name-to-status map does not.
- **Where a v2 recipient would actually be paid.** A recipient with an active
  payout capability and no payout method is a real and invisible state.
  `accounts-v2 payout-methods` reads `/v2/money_management/payout_methods`,
  which is addressed by `Stripe-Context` rather than a path segment — the
  command takes the account ID and scopes the request itself. That surface is
  preview-only, so it needs `--v2-api-version <preview train>`.
- **Which accounts need attention at all.** Per-account commands assume you
  already know which account to ask about. `investigate connect-readiness`
  sweeps and reports only the blocked ones. It retrieves each account because
  neither list endpoint carries capability or requirement detail.

Still deliberately absent: everything that writes. Account creation,
configuration updates, account links and onboarding flows, capability requests,
and outbound payment creation are all POSTs.

## How much of this is verified

The `/v1` surface is checked against Stripe's published OpenAPI spec by `make
apicheck`: every path exists and every query parameter is one the endpoint
declares.

Stripe does not publish the `/v2` namespace in those spec files, so nothing
mechanical validates the v2 requests. Their paths, parameter names, and the
default version train come from the API reference and are listed by `apicheck`
for review rather than checked. In particular these remain doc-derived until a
live call confirms them:

- `Authorization: Bearer` on `/v2` (the docs use it consistently; `/v1` Basic
  is unchanged and verified)
- the `2026-07-29.dahlia` default — a stale-but-valid version would pass any
  schema check anyway
- `created[gte]` / `created[lte]` on `/v2/core/events`
- `/v2/money_management/payout_methods` being addressed by `Stripe-Context`,
  and its preview-only status

## Out of scope

Writes. Creating accounts, updating configurations, creating account links, and
onboarding flows are POST endpoints and remain outside the read-first mandate.
The JSON request body encoding that v2 writes require is deliberately not
implemented.
