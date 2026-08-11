# webhook-event

Run `agent-stripe investigate webhook-event evt_...` when you need to explain what a Stripe event represents.

For a `/v1` event it retrieves the event and emits the embedded `data.object` as a separate entity when Stripe included it expanded, preserving IDs for follow-up commands.

It also accepts v2 core event IDs. Sandbox v2 IDs start `evt_test_`; in live mode the prefixes are indistinguishable, so a v1 miss is retried against `/v2/core/events`. A v2 event is thin: there is no snapshot in the payload, only `related_object`. The command follows that URL and emits the object it points at, and the finding states explicitly that the object is current state rather than state at event time.
