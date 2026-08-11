# account-events

Run `agent-stripe investigate account-events acct_...` to answer "what changed on this connected account, and when".

Accounts v2 capability, requirement, and identity transitions are emitted only as v2 thin events on `/v2/core/events`. `agent-stripe events list` reads `/v1/events` and will never show them, so this is the only way to see that history.

## What it does

1. Retrieves the v2 account with every `include`, so current state is available alongside the events. If the ID turns out to be a Connect v1 account, it says so and continues.
2. Lists `/v2/core/events?object_id=acct_...` (30-day retention).
3. Emits a finding that counts the event types seen and the time range they cover.

## Interpreting it

Thin events carry no snapshot of the object — only `related_object` with an ID, type, and URL. The account emitted alongside them is **current** state, not state at event time. Say so when reporting; do not describe the current capability status as if it were what the event contained.

Severity is `warning` when a capability status or requirements update appears in the window, because those correspond to a change in what the account can do.

An empty result for a `acct_` ID usually means it is a Connect v1 account, which never emits v2 events. Check `agent-stripe events list --type account.updated` for v1 activity instead.

Flags: `--limit N` (default 20) and `--type <event-type>`, repeatable, for example `--type "v2.core.account[requirements].updated"`.
