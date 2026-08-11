package cli

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

// subscriptionRiskSpec is the only thing that differs between the two risk
// scans: which predicate decides a subscription is risky, and what to say when
// none are.
type subscriptionRiskSpec struct {
	use      string
	short    string
	daysHelp string
	empty    string
	risk     func(investigator, map[string]any) (string, error)
}

func newInvestigateCollectionRisk(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return newSubscriptionRiskCommand(globals, outputOpts, subscriptionRiskSpec{
		use:      "collection-risk",
		short:    "Find customers likely to need payment-method outreach before upcoming collection",
		daysHelp: "Upcoming renewal window in days",
		empty:    "No collection-risk subscriptions found in the inspected window.",
		risk:     investigator.collectionRisk,
	})
}

func newInvestigateSubscriptionCancelRisk(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	return newSubscriptionRiskCommand(globals, outputOpts, subscriptionRiskSpec{
		use:      "subscription-cancel-risk",
		short:    "Find subscriptions likely to cancel, end trial, or stop billing soon",
		daysHelp: "Upcoming window in days",
		empty:    "No subscription cancellation risks found in the inspected window.",
		risk: func(_ investigator, sub map[string]any) (string, error) {
			return subscriptionCancelRisk(sub), nil
		},
	})
}

func newSubscriptionRiskCommand(globals shared.GlobalsFunc, outputOpts *evidenceOptions, spec subscriptionRiskSpec) *cobra.Command {
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.subscriptionRiskScan(days, limit, spec)
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, spec.daysHelp)
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum subscriptions to inspect")
	return cmd
}

func (i investigator) subscriptionRiskScan(days, limit int, spec subscriptionRiskSpec) error {
	subs, err := i.list("/v1/subscriptions", upcomingSubscriptionParams(days, limit))
	if err != nil {
		return err
	}
	risky := 0
	for _, sub := range subs {
		risk, err := spec.risk(i, sub)
		if err != nil {
			// Saying "no payment method visible" because a lookup failed would
			// invent a risk; saying nothing would hide a real one.
			i.add(subscriptionRiskFinding("warning", fmt.Sprintf(
				"Could not assess subscription %s for customer %s: %s. Re-run before treating this subscription as healthy.",
				mapString(sub, "id"), mapString(sub, "customer"), err), sub))
			risky++
			continue
		}
		if risk == "" {
			continue
		}
		risky++
		i.add(entityRecord("subscription", sub), subscriptionRiskFinding("warning", risk, sub))
	}
	if risky == 0 {
		i.add(evidenceRecord{Type: "finding", Severity: "info", Summary: spec.empty})
	}
	return nil
}

func upcomingSubscriptionParams(days, limit int) url.Values {
	params := url.Values{"status": []string{"all"}}
	shared.AddLimit(params, limit)
	if days > 0 {
		params.Set("current_period_end[lte]", strconv.FormatInt(time.Now().Add(time.Duration(days)*24*time.Hour).Unix(), 10))
	}
	return params
}

func subscriptionRiskFinding(severity, summary string, sub map[string]any) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Severity: severity,
		Summary:  summary,
		Data: map[string]any{
			"customer":     mapString(sub, "customer"),
			"subscription": mapString(sub, "id"),
		},
	}
}

func (i investigator) collectionRisk(sub map[string]any) (string, error) {
	return i.collectionRiskAt(sub, time.Now())
}

// collectionRiskAt reports a lookup failure rather than absorbing it. Treating
// a failed customer fetch as "no payment method" fabricated a risk, and
// treating a failed invoice fetch as "no risk" hid a real one.
func (i investigator) collectionRiskAt(sub map[string]any, now time.Time) (string, error) {
	ctx := newCollectionRiskContext(sub)
	if risk := statusCollectionRisk(sub, ctx); risk != "" {
		return risk, nil
	}
	pmID, err := i.subscriptionDefaultPaymentMethodID(sub, ctx.customerID)
	if err != nil {
		return "", err
	}
	if pmID == "" {
		return missingPaymentMethodRisk(ctx), nil
	}
	expiring, err := i.paymentMethodExpiresSoon(pmID, now)
	if err != nil {
		return "", err
	}
	if expiring {
		return expiringPaymentMethodRisk(ctx, pmID), nil
	}
	return i.latestInvoiceCollectionRisk(sub, ctx)
}

func (i investigator) subscriptionDefaultPaymentMethodID(sub map[string]any, customerID string) (string, error) {
	if pmID := idFromValue(sub["default_payment_method"]); pmID != "" {
		return pmID, nil
	}
	if customerID == "" {
		return "", nil
	}
	customer, err := i.get("/v1/customers/"+url.PathEscape(customerID), url.Values{})
	if err != nil {
		return "", err
	}
	settings := mapAnyMap(customer, "invoice_settings")
	return idFromValue(settings["default_payment_method"]), nil
}

func (i investigator) paymentMethodExpiresSoon(paymentMethodID string, now time.Time) (bool, error) {
	pm, err := i.get("/v1/payment_methods/"+url.PathEscape(paymentMethodID), url.Values{})
	if err != nil {
		return false, err
	}
	return cardExpiresSoon(pm, now), nil
}

func (i investigator) latestInvoiceCollectionRisk(sub map[string]any, ctx collectionRiskContext) (string, error) {
	invoiceID := idFromValue(sub["latest_invoice"])
	if invoiceID == "" {
		return "", nil
	}
	invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
	if err != nil {
		return "", err
	}
	if !mapBool(invoice, "paid") && mapString(invoice, "status") == "open" {
		return openInvoiceRisk(ctx, invoiceID), nil
	}
	if idFromValue(invoice["payment_intent"]) == "" {
		return "", nil
	}
	pi, err := i.paymentIntentForInvoice(invoice)
	if err != nil {
		return "", err
	}
	if mapString(pi, "status") == "requires_action" {
		return latestInvoiceActionRisk(ctx), nil
	}
	return "", nil
}

type collectionRiskContext struct {
	customerID     string
	subscriptionID string
}

func newCollectionRiskContext(sub map[string]any) collectionRiskContext {
	return collectionRiskContext{
		customerID:     mapString(sub, "customer"),
		subscriptionID: mapString(sub, "id"),
	}
}

func statusCollectionRisk(sub map[string]any, ctx collectionRiskContext) string {
	status := mapString(sub, "status")
	if status == "past_due" || status == "unpaid" || status == "incomplete" {
		return fmt.Sprintf("Customer %s has subscription %s in %s status; outreach about payment details is recommended.", ctx.customerID, ctx.subscriptionID, status)
	}
	return ""
}

func missingPaymentMethodRisk(ctx collectionRiskContext) string {
	return fmt.Sprintf("Customer %s has subscription %s with no default payment method visible.", ctx.customerID, ctx.subscriptionID)
}

func expiringPaymentMethodRisk(ctx collectionRiskContext, pmID string) string {
	return fmt.Sprintf("Customer %s has subscription %s with card payment method %s expiring soon.", ctx.customerID, ctx.subscriptionID, pmID)
}

func openInvoiceRisk(ctx collectionRiskContext, invoiceID string) string {
	return fmt.Sprintf("Customer %s has an open unpaid invoice %s for subscription %s.", ctx.customerID, invoiceID, ctx.subscriptionID)
}

func latestInvoiceActionRisk(ctx collectionRiskContext) string {
	return fmt.Sprintf("Customer %s has subscription %s with latest invoice PaymentIntent requiring action.", ctx.customerID, ctx.subscriptionID)
}

func subscriptionCancelRisk(sub map[string]any) string {
	status := mapString(sub, "status")
	if status == "canceled" || status == "unpaid" {
		return fmt.Sprintf("Subscription %s for customer %s is %s.", mapString(sub, "id"), mapString(sub, "customer"), status)
	}
	if mapBool(sub, "cancel_at_period_end") {
		return fmt.Sprintf("Subscription %s for customer %s is set to cancel at period end %v.", mapString(sub, "id"), mapString(sub, "customer"), mapValue(sub, "current_period_end"))
	}
	if trialEnd, ok := mapInt64(sub, "trial_end"); ok && trialEnd > 0 {
		return fmt.Sprintf("Subscription %s for customer %s trial ends at %d.", mapString(sub, "id"), mapString(sub, "customer"), trialEnd)
	}
	return ""
}

func cardExpiresSoon(paymentMethod map[string]any, now time.Time) bool {
	card := mapAnyMap(paymentMethod, "card")
	year, yearOK := mapInt64(card, "exp_year")
	month, monthOK := mapInt64(card, "exp_month")
	if !yearOK || !monthOK || month < 1 || month > 12 {
		return false
	}
	expiry := time.Date(int(year), time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
	return expiry.Before(now.AddDate(0, 2, 0))
}
