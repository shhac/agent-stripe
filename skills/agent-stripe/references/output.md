# agent-stripe Output Reference

## Defaults

- List commands default to NDJSON/jsonl: one JSON object per line.
- Single-object `get` commands default to NDJSON (one line); pass `--format json` for the pretty object.
- Investigation commands default to NDJSON evidence records.
- Errors are JSON on stderr with `error`, `fixable_by`, and usually `hint`.

## Get Contract (single + multi)

`get <id>...` takes one or more ids and returns one result per id, in input order. Default output is NDJSON: one line per id — the record, or `{"@unresolved":{"id","reason","fixable_by","hint"?}}` for an id that couldn't be resolved (e.g. not found / bad id). `--format json|yaml` collapses to one `{"data":[…], "@unresolved":[…]}` envelope. A single `get <id>` is just the one-element case (NDJSON one line by default; was pretty JSON before — pass `--format json` for the object). Item-level misses stay on stdout and exit 0; only a command-level failure (auth, network) goes to stderr with exit 1 and empty stdout.

A wrong ID prefix on a `get` (e.g. `invoices get pi_...`) yields an `@unresolved` record on stdout (exit 0) instead of a stderr error. Redaction (`@redacted` / `[REDACTED]`) is unchanged and applies inside resolved records.

Commands excluded from multi-get (take no id arg, so multi does not apply): `balance get` and `accounts self` (no id; default to NDJSON like all other gets — pass `--format json` for the object), invoice/checkout `line-items`, `invoice preview`. Raw passthroughs (`api get`, `get --full` raw dumps) output pretty JSON rather than NDJSON. `config get <key>...` accepts one or more keys and returns one NDJSON line per key; misses produce `{"@unresolved":{"id","reason"}}` entries (exit 0).

Some list commands return compact summaries by default because their Stripe objects can carry bulky nested payloads or sensitive person/payment details. Use `--full` on that list command for full redacted objects, or use `get <id>` for one focused object. On compact list commands, `--expand` requires `--full`.

## Null On /v2 Surfaces

`/v2` responses are sparse: `configuration`, `identity`, `requirements`, `future_requirements`, and `defaults` on a v2 account are `null` unless requested with `include`. `null` therefore means "not requested" as often as it means "not set".

`accounts-v2 get` requests every include by default, so a `null` there is genuine. `accounts-v2 list` cannot request any (Stripe's list endpoint has no `include`), so those fields are always `null` in list output — use `get` before concluding anything about capabilities or requirements.

## Evidence Records

Investigation commands emit `entity` and `finding` records:

```json
{"type":"entity","object":"invoice","id":"in_...","data":{}}
{"type":"finding","severity":"warning","summary":"...","data":{}}
```

When Stripe returns an expanded nested object in a field that can also be an ID string, the parent field is replaced by the nested object's ID and the nested object is emitted as a separate `entity` record. This keeps navigation IDs visible and preserves Stripe-shaped fields.

## Redaction

Sensitive Stripe fields are redacted by default in direct resource output, investigation evidence, and debug response bodies.

Redacted string values are replaced in place with `"[REDACTED]"` so scalar Stripe fields stay scalar:

```json
{
  "id": "pi_...",
  "object": "payment_intent",
  "client_secret": "[REDACTED]",
  "metadata": {
    "order_id": "order_123",
    "api_token": "[REDACTED]"
  }
}
```

The containing top-level object gets a single `@redacted` path list:

```json
{
  "@redacted": [
    {
      "path": "client_secret",
      "reason": "sensitive_field",
      "expose_hint": "--expose client_secret"
    },
    {
      "path": "metadata.api_token",
      "reason": "sensitive_field",
      "expose_hint": "--expose metadata.api_token"
    }
  ]
}
```

Use `--expose <path,key>` only when the user explicitly needs a hidden field. It accepts comma-separated values and can be repeated:

```bash
agent-stripe payment-intents get pi_... --expose client_secret
agent-stripe payment-intents get pi_... --expose metadata.internal_order_token,receipt_url
agent-stripe --expose hosted_invoice_url invoices get in_...
```

Stored profile API keys are never exposed by `--expose`.

Common default-redacted fields include client secrets, token/secret/password-like keys, customer email/name/phone, receipt and invoice URLs, card fingerprints, IINs, authorization codes, network transaction IDs, and request-log URLs. Card `last4`, brand, funding, expiration month/year, navigable Stripe IDs, and ordinary metadata remain visible unless a metadata key looks secret-like.

Accounts v2 identity data is treated the same way: `contact_email`, `contact_phone`, `date_of_birth`, and person name fields (`given_name`, `surname`, `legal_name`, `name`) are redacted on `v2.core.account` and `v2.core.account_person` objects. A v2 account's `display_name` is a business label, not a person, so it stays visible — as do country, entity type, relationship flags (owner, representative, percent ownership, title), and `id_numbers[].type` (Stripe never returns the values).

## Truncation

Investigation commands truncate long strings by default. Truncated entity records include `truncated_fields` with the path, byte counts, and an expansion hint.

```bash
agent-stripe investigate --max-string 200 resolve prod_...
agent-stripe investigate --expand-field description resolve prod_...
agent-stripe investigate --full resolve prod_...
```

Truncation controls do not override redaction. Use `--expose` for redacted fields.

## Pagination

The pagination trailer uses the same `@pagination` key in every format: a trailing record in NDJSON, and a top-level key beside `data` in the `--format json|yaml` envelope. List NDJSON output includes it when Stripe reports more results:

```json
{"id":"pi_...","object":"payment_intent"}
{"@pagination":{"has_more":true,"next_page":"..."}}
```

Use `--starting-after` / `--ending-before` for `/v1` list endpoints, and `--page` for search endpoints.

`/v2` lists (`accounts-v2`, `accounts-v2 persons`, `events-v2`) have no cursor IDs and no `has_more` of their own. Stripe returns a `next_page_url`; the CLI lifts the token out of it and reports it the same way, so `@pagination.next_page` is the value to pass back as `--page <token>`. v2 lists are also eventually consistent by default, unlike v1 top-level lists — do not treat one as read-after-write proof.

## Errors

Error JSON uses `fixable_by`:

- `agent` - adjust the command, ID, query, or flags.
- `human` - needs a user action, permission change, profile setup, or account context decision.
- `retry` - transient condition. Retrying later or narrowing the query may help.

Wrong known ID prefixes on a `get` command are handled as item-level misses: `invoices get pi_...` returns an `@unresolved` record on stdout (exit 0) with `fixable_by:"agent"` and a hint pointing at `payment-intents get`, `investigate incoming-payment`, or `investigate resolve`. Only command-level failures (auth, network) go to stderr with exit 1.

Stripe 429s retry automatically with bounded exponential backoff and jitter. Use `--max-retries 0` for one-shot behavior. After retries are exhausted, the error is `fixable_by:"retry"` and includes Stripe's rate-limit reason when present.

## Debug

`--debug` writes structured records to stderr:

- client setup: profile alias, credential source label, context, API version, base URL, timeout, max retries.
- HTTP responses: method, URL, status, request ID, and redacted body.
- retry records: attempt, max retries, status, delay, request ID, and rate-limit reason when present.

Debug output must not include raw API keys.
