# connect-readiness

Run `agent-stripe investigate connect-readiness` for the platform-wide question the per-account commands cannot answer: of my connected accounts, which ones cannot take payments or get paid?

## Why it costs more than a list

Neither account list endpoint carries the detail that decides this. Stripe's `/v1/accounts` list returns the Account objects but `/v2/core/accounts` supports no `include` at all, so capabilities and requirements come back `null` there. The sweep therefore retrieves each account individually, which is why `--limit` defaults to 10. Raise it deliberately.

## What it does

1. Lists accounts in whichever namespace the platform is on — `--namespace auto` tries Accounts v2 and falls back to Connect v1 on the documented interop errors.
2. Retrieves each account and applies the same health logic `account-health` uses, per namespace.
3. Emits a finding **only** for the accounts that have blockers, plus one summary finding naming how many of how many were inspected.

Severity on the summary is `warning` when any account is blocked, `info` when none are.

## Following up

The summary carries a `command` pointing at `investigate account-health <first blocked account>`. From there, the sub-resource commands say what a requirement is actually asking for: `accounts external-accounts`, `accounts capabilities`, `accounts persons list` / `accounts-v2 persons list`, and `accounts-v2 payout-methods`.
