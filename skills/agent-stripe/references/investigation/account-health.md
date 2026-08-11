# account-health

Run `agent-stripe investigate account-health acct_...` when a connected account cannot take payments, cannot be paid out, or is stuck in onboarding.

## Namespace

Stripe has two connected-account models and both use the `acct_` prefix, so the ID cannot tell you which applies. `--namespace auto` (the default) reads `/v2/core/accounts` first, with every `include` requested, and falls back to `/v1/accounts` when Stripe reports the ID is not a v2 account (`v1_account_instead_of_v2_account`, `account_not_yet_compatible_with_v2`, `accounts_v2_access_blocked`, `non_connect_platform_accounts_v2_access_blocked`, or 404). A finding names the namespace that answered, and the finding `data.namespace` field carries it too.

Use `--namespace v1` to skip the v2 probe on a platform that has no UA2 access, and `--namespace v2` to fail loudly instead of falling back. Errors that are not about the namespace — auth, permissions, rate limits — always surface rather than triggering a fallback.

## What it reports

Accounts v2:

- applied configurations (`customer`, `merchant`, `recipient`), `dashboard`, and whether the account is closed
- every capability leaf flattened to `configuration.capability` with its status and `status_details` codes, for example `merchant.card_payments = restricted (requirements_past_due)`
- requirement entries with `minimum_deadline.status`, `awaiting_action_from`, error codes, the capabilities each entry restricts, and any referenced person
- the `requirements.summary.minimum_deadline` rollup and the person records on the account
- recent transfers to the account

Connect v1:

- `charges_enabled`, `payouts_enabled`, requirements, capabilities, future requirements, and recent transfers

## Interpreting it

A v2 account has no `charges_enabled` or `payouts_enabled` — do not report them, and do not treat their absence as "disabled". The blocker is the capability whose status is not `active`, and the requirement entries that restrict it.

`awaiting_action_from` decides who is blocked: `user` means the platform or account holder must supply something; `stripe` means Stripe is still reviewing and there is nothing to collect. The finding counts them separately.

Severity is `warning` when any capability is not active, any user-owned requirement is past due or currently due, or the account is closed. Otherwise it is `info`.

Follow with `agent-stripe investigate account-events acct_...` to see when the state changed.
