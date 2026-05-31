package mockstripe

func subscriptions() []map[string]any {
	return []map[string]any{
		{
			"id":                   "sub_mock_active",
			"object":               "subscription",
			"customer":             "cus_mock_123",
			"status":               "active",
			"collection_method":    "charge_automatically",
			"latest_invoice":       "in_mock_paid",
			"current_period_start": 1760000000,
			"current_period_end":   1762592000,
			"cancel_at_period_end": false,
			"items": map[string]any{
				"object":   "list",
				"has_more": false,
				"data":     []map[string]any{subscriptionItems()[0]},
			},
			"metadata": map[string]string{
				"tenant_id": "acme",
			},
		},
		{
			"id":                   "sub_mock_past_due",
			"object":               "subscription",
			"customer":             "cus_mock_456",
			"status":               "past_due",
			"collection_method":    "charge_automatically",
			"latest_invoice":       "in_mock_open_failed",
			"current_period_start": 1760000000,
			"current_period_end":   1762592000,
			"cancel_at_period_end": false,
			"metadata": map[string]string{
				"tenant_id": "delinquent",
			},
		},
		{
			"id":                   "sub_mock_canceling",
			"object":               "subscription",
			"customer":             "cus_mock_123",
			"status":               "active",
			"collection_method":    "charge_automatically",
			"latest_invoice":       "in_mock_paid",
			"current_period_start": 1760000000,
			"current_period_end":   1762592000,
			"cancel_at_period_end": true,
			"metadata": map[string]string{
				"tenant_id": "acme",
			},
		},
		{
			"id":                     "sub_mock_expiring_card",
			"object":                 "subscription",
			"customer":               "cus_mock_expiring",
			"status":                 "active",
			"collection_method":      "charge_automatically",
			"default_payment_method": "pm_mock_expiring",
			"latest_invoice":         "in_mock_requires_action",
			"current_period_start":   1760000000,
			"current_period_end":     1762592000,
			"cancel_at_period_end":   false,
			"metadata": map[string]string{
				"tenant_id": "expiring",
			},
		},
		{
			"id":                   "sub_mock_missing_pm",
			"object":               "subscription",
			"customer":             "cus_mock_missing_pm",
			"status":               "active",
			"collection_method":    "charge_automatically",
			"latest_invoice":       "in_mock_missing_pm",
			"current_period_start": 1760000000,
			"current_period_end":   1762592000,
			"cancel_at_period_end": false,
			"metadata": map[string]string{
				"tenant_id": "missing_pm",
			},
		},
	}
}

func subscriptionItems() []map[string]any {
	return []map[string]any{
		{
			"id":           "si_mock_basic",
			"object":       "subscription_item",
			"subscription": "sub_mock_active",
			"quantity":     1,
			"price": map[string]any{
				"id":          "price_mock_basic",
				"object":      "price",
				"currency":    "usd",
				"unit_amount": 4200,
				"recurring": map[string]string{
					"interval": "month",
				},
				"product": "prod_mock_basic",
			},
		},
		{
			"id":           "si_mock_pro",
			"object":       "subscription_item",
			"subscription": "sub_mock_past_due",
			"quantity":     3,
			"price": map[string]any{
				"id":          "price_mock_pro",
				"object":      "price",
				"currency":    "usd",
				"unit_amount": 9900,
				"recurring": map[string]string{
					"interval": "month",
				},
				"product": "prod_mock_pro",
			},
		},
	}
}

func invoices() []map[string]any {
	return []map[string]any{
		{
			"id":                 "in_mock_paid",
			"object":             "invoice",
			"created":            1760000150,
			"subscription":       "sub_mock_active",
			"customer":           "cus_mock_123",
			"status":             "paid",
			"paid":               true,
			"amount_due":         4200,
			"amount_paid":        4200,
			"currency":           "usd",
			"number":             "MOCK-0001",
			"payment_intent":     "pi_mock_succeeded",
			"attempt_count":      1,
			"hosted_invoice_url": "https://invoices.stripe.example.test/fake/in_mock_paid",
			"invoice_pdf":        "https://invoices.stripe.example.test/fake/in_mock_paid.pdf",
		},
		{
			"id":                   "in_mock_open_failed",
			"object":               "invoice",
			"created":              1760000550,
			"subscription":         "sub_mock_past_due",
			"customer":             "cus_mock_456",
			"status":               "open",
			"paid":                 false,
			"amount_due":           29700,
			"amount_paid":          0,
			"currency":             "usd",
			"number":               "MOCK-0002",
			"payment_intent":       "pi_mock_failed",
			"attempt_count":        3,
			"next_payment_attempt": 1760100000,
			"hosted_invoice_url":   "https://invoices.stripe.example.test/fake/in_mock_open_failed",
			"invoice_pdf":          "https://invoices.stripe.example.test/fake/in_mock_open_failed.pdf",
		},
		{
			"id":                   "in_mock_requires_action",
			"object":               "invoice",
			"created":              1760000650,
			"subscription":         "sub_mock_expiring_card",
			"customer":             "cus_mock_expiring",
			"status":               "open",
			"paid":                 false,
			"amount_due":           5500,
			"amount_paid":          0,
			"currency":             "usd",
			"number":               "MOCK-0003",
			"payment_intent":       "pi_mock_requires_action",
			"attempt_count":        1,
			"next_payment_attempt": 1760200000,
			"hosted_invoice_url":   "https://invoices.stripe.example.test/fake/in_mock_requires_action",
			"invoice_pdf":          "https://invoices.stripe.example.test/fake/in_mock_requires_action.pdf",
		},
		{
			"id":                 "in_mock_missing_pm",
			"object":             "invoice",
			"created":            1760000750,
			"subscription":       "sub_mock_missing_pm",
			"customer":           "cus_mock_missing_pm",
			"status":             "draft",
			"paid":               false,
			"amount_due":         1200,
			"amount_paid":        0,
			"currency":           "usd",
			"number":             "MOCK-0004",
			"payment_intent":     "",
			"attempt_count":      0,
			"hosted_invoice_url": "https://invoices.stripe.example.test/fake/in_mock_missing_pm",
			"invoice_pdf":        "https://invoices.stripe.example.test/fake/in_mock_missing_pm.pdf",
		},
	}
}

func invoiceLines(invoiceID string) []map[string]any {
	return []map[string]any{
		{
			"id":       "il_mock_" + invoiceID,
			"object":   "line_item",
			"invoice":  invoiceID,
			"amount":   4200,
			"currency": "usd",
			"metadata": map[string]string{
				"internal_product_id": "prod_internal_basic",
			},
		},
	}
}

func customers() []map[string]any {
	return []map[string]any{
		{
			"id":     "cus_mock_123",
			"object": "customer",
			"email":  "buyer@example.com",
			"name":   "Mock Buyer",
			"phone":  "+15550101001",
			"invoice_settings": map[string]any{
				"default_payment_method": "pm_mock_visa",
			},
			"metadata": map[string]string{
				"tenant_id": "acme",
			},
		},
		{
			"id":     "cus_mock_456",
			"object": "customer",
			"email":  "delinquent@example.com",
			"name":   "Mock Delinquent",
			"phone":  "+15550101002",
			"invoice_settings": map[string]any{
				"default_payment_method": "pm_mock_declined",
			},
			"metadata": map[string]string{
				"tenant_id": "delinquent",
			},
		},
		{
			"id":     "cus_mock_expiring",
			"object": "customer",
			"email":  "expiring@example.com",
			"name":   "Mock Expiring",
			"phone":  "+15550101003",
			"invoice_settings": map[string]any{
				"default_payment_method": "pm_mock_expiring",
			},
			"metadata": map[string]string{
				"tenant_id": "expiring",
			},
		},
		{
			"id":     "cus_mock_missing_pm",
			"object": "customer",
			"email":  "missingpm@example.com",
			"name":   "Mock Missing PM",
			"phone":  "+15550101004",
			"invoice_settings": map[string]any{
				"default_payment_method": "",
			},
			"metadata": map[string]string{
				"tenant_id": "missing_pm",
			},
		},
	}
}

func paymentMethods() []map[string]any {
	return []map[string]any{
		{
			"id":       "pm_mock_visa",
			"object":   "payment_method",
			"type":     "card",
			"customer": "cus_mock_123",
			"card": map[string]any{
				"brand":       "visa",
				"last4":       "4242",
				"exp_month":   12,
				"exp_year":    2030,
				"fingerprint": "fp_mock_visa",
			},
		},
		{
			"id":       "pm_mock_declined",
			"object":   "payment_method",
			"type":     "card",
			"customer": "cus_mock_456",
			"card": map[string]any{
				"brand":       "visa",
				"last4":       "0002",
				"exp_month":   1,
				"exp_year":    2025,
				"fingerprint": "fp_mock_declined",
			},
		},
		{
			"id":       "pm_mock_expiring",
			"object":   "payment_method",
			"type":     "card",
			"customer": "cus_mock_expiring",
			"card": map[string]any{
				"brand":       "visa",
				"last4":       "0341",
				"exp_month":   1,
				"exp_year":    2026,
				"fingerprint": "fp_mock_expiring",
			},
		},
	}
}
