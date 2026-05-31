package mockstripe

func events() []map[string]any {
	return []map[string]any{
		{
			"id":               "evt_mock_charge_failed",
			"object":           "event",
			"type":             "charge.failed",
			"api_version":      "2025-06-30.basil",
			"created":          1760000000,
			"livemode":         false,
			"pending_webhooks": 1,
			"request": map[string]any{
				"id":              "req_mock_decline",
				"idempotency_key": "order_123",
			},
			"data": map[string]any{
				"object": charges()[1],
			},
		},
		{
			"id":               "evt_mock_payment_succeeded",
			"object":           "event",
			"type":             "payment_intent.succeeded",
			"api_version":      "2025-06-30.basil",
			"created":          1760000300,
			"livemode":         false,
			"pending_webhooks": 0,
			"data": map[string]any{
				"object": paymentIntents()[0],
			},
		},
		{
			"id":               "evt_mock_invoice_failed",
			"object":           "event",
			"type":             "invoice.payment_failed",
			"api_version":      "2025-06-30.basil",
			"created":          1760000500,
			"livemode":         false,
			"pending_webhooks": 1,
			"data": map[string]any{
				"object": invoices()[1],
			},
		},
	}
}

func webhookEndpoints() []map[string]any {
	return []map[string]any{
		{
			"id":             "we_mock_primary",
			"object":         "webhook_endpoint",
			"status":         "enabled",
			"url":            "https://example.test/stripe/webhook",
			"enabled_events": []string{"charge.failed", "payment_intent.succeeded", "invoice.payment_failed"},
		},
		{
			"id":             "we_mock_disabled",
			"object":         "webhook_endpoint",
			"status":         "disabled",
			"url":            "https://example.test/stripe/old-webhook",
			"enabled_events": []string{"*"},
		},
	}
}
