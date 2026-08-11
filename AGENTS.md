# agent-stripe

This repository contains a Go CLI for LLM-driven Stripe incident triage.

## Development

- Use `make test` for the full test suite.
- Use `make apicheck` to validate the requests the CLI emits against Stripe's published OpenAPI spec (paths and query parameter names). Add every new command to the table in `internal/apicheck` or its requests go unchecked. Stripe does not publish the `/v2` namespace in that spec, so v2 requests are listed rather than validated.
- Use `make build` to build `agent-stripe`.
- Use `make build-mock` to build the local mock Stripe server.
- Use `make mock` to start `mockstripe` on `127.0.0.1:12111`.
- Use `make mock-dev ARGS="events list --type charge.failed"` to run the CLI against the mock server.
- Use `mockstripe --routes` or `GET /` on the mock server to inspect the supported mock API surface.
- Prefer `agent-stripe auth add <profile> --form` when guiding a user through credential setup; do not ask the user to paste API keys into chat.
- Prefer `agent-stripe auth update <profile> --form` when guiding a user through replacing a stored key.
- Prefer read-only Stripe commands unless a future design document explicitly approves mutation workflows.
- Do not print, log, or persist raw API keys outside the credential backend.
- Keep list outputs compact and NDJSON-friendly.
- Investigations record evidence with `i.add(...)` and return `error`. The collector is the only accumulator; do not add a second one or thread a records slice.
- Never interpolate a Stripe object into a finding `summary` — summaries are not redacted. Name the fields you want.
- Follow related objects with `i.fetchRelated`/`i.followRef`/`i.listRelated` so a failed lookup is reported rather than read as an absence.
- The e2e suite runs in-process via `cli.RunForTest`; `mockstripe` can be told to fail a path (`X-Mock-Fault`, or `POST /_mock/faults?rules=...`) so degraded paths are testable.
- When adding/changing commands, update `agent-stripe usage`, any relevant domain usage (`subscriptions usage`, `invoices usage`, `payments usage`, `connect usage`, `accounts-v2 usage`, `events-v2 usage`), `skills/agent-stripe/SKILL.md`, skill references under `skills/agent-stripe/references/`, README examples when user-facing, and the investigation workflow design doc.
- Keep the two account namespaces separate. `accounts`/`/v1` and `accounts-v2`/`/v2` have different object shapes; a reader that mixes them reports blockers that do not exist. Namespace auto-detection belongs in `investigate`, not in resource commands.
- `/v2` requests are routed by path prefix in `internal/api`: Bearer auth, the `v2_api_version` train, indexed array query parameters, and `next_page_url` pagination. Add new `/v2` endpoints through those helpers rather than hand-rolling per command.

## Design Intent

See `design-docs/initial-design.md` for the first-pass command surface and Stripe-specific auth decisions.
See `design-docs/mock-stripe.md` for the local mock server contract.
See `design-docs/accounts-v2.md` for the Accounts v2 (UA2) namespace: transport differences, the v1/v2 split in commands, and namespace resolution in investigations.
