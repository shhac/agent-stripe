# webhook-event

Run `agent-stripe investigate webhook-event evt_...` when you need to explain what a Stripe event represents.

It retrieves the event and emits the embedded `data.object` as a separate entity when Stripe included it expanded, preserving IDs for follow-up commands.
