package mockstripe

func products() []map[string]any {
	return []map[string]any{
		{
			"id":          "prod_mock_basic",
			"object":      "product",
			"active":      true,
			"name":        "Basic Plan",
			"description": longMockDescription(),
			"metadata": map[string]string{
				"internal_product_id": "prod_internal_basic",
			},
		},
		{
			"id":     "prod_mock_pro",
			"object": "product",
			"active": true,
			"name":   "Pro Plan",
			"metadata": map[string]string{
				"internal_product_id": "prod_internal_pro",
			},
		},
	}
}

func prices() []map[string]any {
	return []map[string]any{
		{
			"id":          "price_mock_basic",
			"object":      "price",
			"active":      true,
			"type":        "recurring",
			"currency":    "usd",
			"unit_amount": 4200,
			"product":     "prod_mock_basic",
			"recurring": map[string]string{
				"interval": "month",
			},
		},
		{
			"id":          "price_mock_pro",
			"object":      "price",
			"active":      true,
			"type":        "recurring",
			"currency":    "usd",
			"unit_amount": 9900,
			"product":     "prod_mock_pro",
			"recurring": map[string]string{
				"interval": "month",
			},
		},
	}
}

func longMockDescription() string {
	return "This product description is intentionally long so investigation output can demonstrate truncation controls. " +
		"Agents should see an explicit truncated_fields note and can rerun with --expand-field description or --full when this text matters. " +
		"Basic navigation IDs remain visible even when verbose descriptive fields are shortened for token efficiency."
}
