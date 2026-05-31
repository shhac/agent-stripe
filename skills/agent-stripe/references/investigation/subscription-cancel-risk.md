# subscription-cancel-risk

Run `agent-stripe investigate subscription-cancel-risk --days 30 [--limit N]` to find subscriptions likely to end soon.

It flags subscriptions set to cancel at period end, trial endings, and cancellation/period-end risks in the inspected window.
