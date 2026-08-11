package cli

// The summaries whose commands are registered in resources.go. Every other
// resource's summary lives beside its own registration; these four have no
// file of their own to live in.

func customerListSummary(customer map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, customer, "id")
	copyString(summary, customer, "object")
	copyNumber(summary, customer, "created")
	copyBool(summary, customer, "delinquent")
	copyNumber(summary, customer, "balance")
	copyString(summary, customer, "currency")
	copySubset(summary, customer, "invoice_settings", "default_payment_method")
	return summary
}

func paymentMethodListSummary(pm map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, pm, "id")
	copyString(summary, pm, "object")
	copyString(summary, pm, "type")
	copyString(summary, pm, "customer")
	copyNumber(summary, pm, "created")
	if card, ok := pm["card"].(map[string]any); ok {
		summary["card"] = cardSummary(card)
	}
	copySubset(summary, pm, "us_bank_account", "bank_name", "last4", "account_type", "account_holder_type")
	return summary
}

func setupIntentListSummary(seti map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, seti, "id")
	copyString(summary, seti, "object")
	copyNumber(summary, seti, "created")
	copyString(summary, seti, "status")
	copyString(summary, seti, "customer")
	copyString(summary, seti, "payment_method")
	copyString(summary, seti, "usage")
	copyString(summary, seti, "cancellation_reason")
	return summary
}

func paymentLinkListSummary(link map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, link, "id")
	copyString(summary, link, "object")
	copyBool(summary, link, "active")
	copyString(summary, link, "url")
	copyNumber(summary, link, "application_fee_amount")
	copyNumber(summary, link, "application_fee_percent")
	addListDataCount(summary, link, "line_items")
	return summary
}

func cardSummary(card map[string]any) map[string]any {
	out := map[string]any{}
	copyString(out, card, "brand")
	copyString(out, card, "last4")
	copyNumber(out, card, "exp_month")
	copyNumber(out, card, "exp_year")
	copyString(out, card, "funding")
	return out
}
