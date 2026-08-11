package cli

import (
	"fmt"
	"net/url"
)

var invoiceTotalInvestigation = investigationSpec{
	use:   "invoice-total <invoice-id>",
	short: "Explain how an invoice total was reached: lines, discounts, tax, and what was paid",
	long: "Answers \"this total isn't what we expected\". Reconciles the line items against\n" +
		"subtotal, discounts, tax, total, and amount paid, and reports the first step where\n" +
		"the arithmetic stops agreeing rather than only printing the fields.",
	run: investigator.invoiceTotal,
}

func (i investigator) invoiceTotal(invoiceID string) error {
	if err := validateExpectedStripeID(invoiceID, "invoice"); err != nil {
		return err
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return err
	}
	i.add(entityRecord("invoice", invoice))

	lines := i.listRelated("invoice lines", "/v1/invoices/"+url.PathEscape(invoiceID)+"/lines", url.Values{"limit": []string{"100"}})
	i.addList("line_item", lines)
	if customerID := idFromValue(invoice["customer"]); customerID != "" {
		i.fetchRelated("customer", customerID)
	}
	i.add(invoiceTotalFinding(invoice, lines))
	return nil
}

// invoiceTotalFinding walks the same arithmetic Stripe does, and names the step
// that disagrees. Printing the fields alone leaves the reader to do this by
// hand, which is the part they got wrong before asking.
func invoiceTotalFinding(invoice map[string]any, lines []map[string]any) evidenceRecord {
	currency := mapString(invoice, "currency")
	subtotal, _ := mapInt64(invoice, "subtotal")
	total, _ := mapInt64(invoice, "total")
	amountDue, _ := mapInt64(invoice, "amount_due")
	amountPaid, _ := mapInt64(invoice, "amount_paid")
	tax, hasTax := mapInt64(invoice, "tax")
	discount := invoiceDiscountAmount(invoice)

	lineSum := int64(0)
	taxedLines := 0
	for _, line := range lines {
		amount, _ := mapInt64(line, "amount")
		lineSum += amount
		if amounts, ok := line["tax_amounts"].([]any); ok && len(amounts) > 0 {
			taxedLines++
		}
	}

	data := map[string]any{
		"invoice":     mapString(invoice, "id"),
		"currency":    currency,
		"line_count":  len(lines),
		"line_sum":    lineSum,
		"subtotal":    subtotal,
		"total":       total,
		"amount_due":  amountDue,
		"amount_paid": amountPaid,
		"taxed_lines": taxedLines,
	}
	if hasTax {
		data["tax"] = tax
	}
	if discount != 0 {
		data["discount_amount"] = discount
	}

	mismatches := []string{}
	if len(lines) > 0 && lineSum != subtotal {
		mismatches = append(mismatches, fmt.Sprintf("line items sum to %d but subtotal is %d", lineSum, subtotal))
	}
	if expected := subtotal - discount + tax; expected != total {
		mismatches = append(mismatches, fmt.Sprintf("subtotal %d minus discounts %d plus tax %d is %d, but total is %d",
			subtotal, discount, tax, expected, total))
	}
	if amountPaid != 0 && amountPaid != total {
		mismatches = append(mismatches, fmt.Sprintf("amount paid %d differs from total %d", amountPaid, total))
	}

	summary := fmt.Sprintf("Invoice %s: %d line item(s) summing to %d, subtotal %d, discounts %d, tax %d, total %d, paid %d (%s).",
		mapString(invoice, "id"), len(lines), lineSum, subtotal, discount, tax, total, amountPaid, currency)
	severity := "info"
	if len(mismatches) > 0 {
		severity = "warning"
		summary += " Does not reconcile: " + joinAndTruncate(mismatches, 3) + "."
		data["mismatches"] = mismatches
	} else {
		summary += " The arithmetic reconciles."
	}
	if taxedLines == 0 && tax != 0 {
		summary += " Tax is charged at invoice level with no per-line tax amounts."
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Command:  "agent-stripe invoices line-items " + mapString(invoice, "id"),
		Data:     data,
	}
}

// invoiceDiscountAmount sums the applied discounts, which Stripe reports as
// total_discount_amounts rather than a single field.
func invoiceDiscountAmount(invoice map[string]any) int64 {
	amounts, ok := invoice["total_discount_amounts"].([]any)
	if !ok {
		return 0
	}
	total := int64(0)
	for _, raw := range amounts {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		amount, _ := mapInt64(entry, "amount")
		total += amount
	}
	return total
}
