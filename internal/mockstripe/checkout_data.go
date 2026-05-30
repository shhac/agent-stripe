package mockstripe

func checkoutSessions() []map[string]any {
	return []map[string]any{
		{
			"id":             "cs_mock_paid",
			"object":         "checkout.session",
			"customer":       "cus_mock_123",
			"payment_intent": "pi_mock_succeeded",
			"payment_link":   "plink_mock_basic",
			"mode":           "payment",
			"payment_status": "paid",
			"amount_total":   4200,
			"currency":       "usd",
			"metadata": map[string]string{
				"cart_id": "cart_123",
			},
		},
	}
}

func checkoutLineItems(sessionID string) []map[string]any {
	return []map[string]any{
		{
			"id":       "li_mock_" + sessionID,
			"object":   "item",
			"amount":   4200,
			"currency": "usd",
			"price":    prices()[0],
			"quantity": 1,
		},
	}
}

func paymentLinks() []map[string]any {
	return []map[string]any{
		{
			"id":     "plink_mock_basic",
			"object": "payment_link",
			"active": true,
			"url":    "https://buy.stripe.com/mock",
			"line_items": map[string]any{
				"object":   "list",
				"has_more": false,
				"data": []map[string]any{
					{
						"id":       "li_mock_payment_link",
						"object":   "item",
						"price":    "price_mock_basic",
						"quantity": 1,
					},
				},
			},
		},
	}
}
