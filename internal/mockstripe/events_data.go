package mockstripe

func events() []map[string]any {
	return []map[string]any{
		{
			"id":          "evt_mock_charge_failed",
			"object":      "event",
			"type":        "charge.failed",
			"api_version": "2025-06-30.basil",
			"created":     1760000000,
			"livemode":    false,
			"request": map[string]any{
				"id":              "req_mock_decline",
				"idempotency_key": "order_123",
			},
			"data": map[string]any{
				"object": charges()[1],
			},
		},
		{
			"id":          "evt_mock_payment_succeeded",
			"object":      "event",
			"type":        "payment_intent.succeeded",
			"api_version": "2025-06-30.basil",
			"created":     1760000300,
			"livemode":    false,
			"data": map[string]any{
				"object": paymentIntents()[0],
			},
		},
		{
			"id":          "evt_mock_invoice_failed",
			"object":      "event",
			"type":        "invoice.payment_failed",
			"api_version": "2025-06-30.basil",
			"created":     1760000500,
			"livemode":    false,
			"data": map[string]any{
				"object": invoices()[1],
			},
		},
	}
}
