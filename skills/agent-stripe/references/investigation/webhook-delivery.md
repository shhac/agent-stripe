# webhook-delivery

Run `agent-stripe investigate webhook-delivery evt_... [--endpoint we_...]` when a webhook may not have arrived. Run `agent-stripe investigate webhook-delivery we_...` to inspect endpoint config directly.

It uses event `pending_webhooks`, request metadata, endpoint status, and enabled events. It does not claim per-endpoint delivery attempt details unless Stripe returns them.
