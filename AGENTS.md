# agent-stripe

This repository contains a Go CLI for LLM-driven Stripe incident triage.

## Development

- Use `make test` for the full test suite.
- Use `make build` to build `agent-stripe`.
- Use `make build-mock` to build the local mock Stripe server.
- Use `make mock` to start `mockstripe` on `127.0.0.1:12111`.
- Use `make mock-dev ARGS="events list --type charge.failed"` to run the CLI against the mock server.
- Use `mockstripe --routes` or `GET /` on the mock server to inspect the supported mock API surface.
- Prefer `agent-stripe auth add <profile> --form` when guiding a user through credential setup; do not ask the user to paste API keys into chat.
- Prefer read-only Stripe commands unless a future design document explicitly approves mutation workflows.
- Do not print, log, or persist raw API keys outside the credential backend.
- Keep list outputs compact and NDJSON-friendly.

## Design Intent

See `design-docs/initial-design.md` for the first-pass command surface and Stripe-specific auth decisions.
See `design-docs/mock-stripe.md` for the local mock server contract.
