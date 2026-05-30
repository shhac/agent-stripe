package cli

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func newInvestigateSubscriptionRenewal(globals shared.GlobalsFunc) *cobra.Command {
	var subscription string
	var metadata string
	var customer string
	var limit int
	cmd := &cobra.Command{
		Use:   "subscription-renewal",
		Short: "Summarize last and next payment for subscriptions found by ID, customer, or metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				subs, err := inv.findSubscriptions(subscription, customer, metadata, limit)
				if err != nil {
					return nil, err
				}
				if len(subs) == 0 {
					return []evidenceRecord{{Type: "finding", Severity: "warning", Summary: "No subscriptions matched the supplied filters."}}, nil
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					records = append(records, entityRecord("subscription", sub))
					records = append(records, inv.subscriptionPaymentSummary(sub)...)
				}
				return records, nil
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	cmd.Flags().StringVar(&metadata, "metadata", "", "Metadata equality filter as key=value")
	cmd.Flags().StringVar(&customer, "customer", "", "Customer ID")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum matching subscriptions to inspect")
	return cmd
}

func newInvestigateCollectionRisk(globals shared.GlobalsFunc) *cobra.Command {
	var days int
	var limit int
	cmd := &cobra.Command{
		Use:   "collection-risk",
		Short: "Find customers likely to need payment-method outreach before upcoming collection",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvestigation(globals(), func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
				inv := investigator{ctx: ctx, client: client}
				params := url.Values{"status": []string{"all"}}
				api.AddLimit(params, limit)
				if days > 0 {
					params.Set("current_period_end[lte]", strconv.FormatInt(time.Now().Add(time.Duration(days)*24*time.Hour).Unix(), 10))
				}
				subs, err := inv.list("/v1/subscriptions", params)
				if err != nil {
					return nil, err
				}
				records := []evidenceRecord{}
				for _, sub := range subs {
					risk := inv.collectionRisk(sub)
					if risk == "" {
						continue
					}
					records = append(records, entityRecord("subscription", sub))
					records = append(records, evidenceRecord{
						Type:     "finding",
						Severity: "warning",
						Summary:  risk,
						Data: map[string]any{
							"customer":     mapString(sub, "customer"),
							"subscription": mapString(sub, "id"),
						},
					})
				}
				if len(records) == 0 {
					records = append(records, evidenceRecord{Type: "finding", Severity: "info", Summary: "No collection-risk subscriptions found in the inspected window."})
				}
				return records, nil
			})
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "Upcoming renewal window in days")
	cmd.Flags().IntVar(&limit, "limit", 25, "Maximum subscriptions to inspect")
	return cmd
}

func (i investigator) findSubscriptions(subscription, customer, metadata string, limit int) ([]map[string]any, error) {
	switch {
	case subscription != "":
		sub, err := i.get("/v1/subscriptions/"+url.PathEscape(subscription), url.Values{})
		if err != nil {
			return nil, err
		}
		return []map[string]any{sub}, nil
	case metadata != "":
		key, value, ok := strings.Cut(metadata, "=")
		if !ok || key == "" || value == "" {
			return nil, agenterrors.New("--metadata must be key=value", agenterrors.FixableByAgent).
				WithHint("Example: --metadata tenant_id=acme")
		}
		params := url.Values{"query": []string{"metadata['" + key + "']:'" + value + "'"}}
		api.AddLimit(params, limit)
		return i.list("/v1/subscriptions/search", params)
	default:
		params := url.Values{}
		api.AddLimit(params, limit)
		shared.AddString(params, "customer", customer)
		return i.list("/v1/subscriptions", params)
	}
}

func (i investigator) subscriptionPaymentSummary(sub map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	if latestInvoiceID := idFromValue(sub["latest_invoice"]); latestInvoiceID != "" {
		invoiceRecords, err := i.invoicePayment(latestInvoiceID)
		if err == nil {
			records = append(records, invoiceRecords...)
		} else {
			records = append(records, evidenceRecord{Type: "finding", Severity: "warning", Summary: "Could not retrieve latest invoice " + latestInvoiceID + ": " + err.Error()})
		}
	}
	nextAmount := "unknown"
	if preview, err := i.postForm("/v1/invoices/create_preview", url.Values{"subscription": []string{mapString(sub, "id")}}); err == nil {
		records = append(records, entityRecord("invoice_preview", preview))
		nextAmount = formatAmount(preview)
	}
	records = append(records, evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary: fmt.Sprintf("Subscription %s last invoice is %s; next renewal is at %v and preview amount is %s.",
			mapString(sub, "id"), idFromValue(sub["latest_invoice"]), mapValue(sub, "current_period_end"), nextAmount),
	})
	return records
}

func (i investigator) collectionRisk(sub map[string]any) string {
	status := mapString(sub, "status")
	if status == "past_due" || status == "unpaid" || status == "incomplete" {
		return fmt.Sprintf("Customer %s has subscription %s in %s status; outreach about payment details is recommended.", mapString(sub, "customer"), mapString(sub, "id"), status)
	}
	if invoiceID := idFromValue(sub["latest_invoice"]); invoiceID != "" {
		invoice, err := i.get("/v1/invoices/"+url.PathEscape(invoiceID), url.Values{})
		if err == nil && !mapBool(invoice, "paid") && mapString(invoice, "status") == "open" {
			return fmt.Sprintf("Customer %s has an open unpaid invoice %s for subscription %s.", mapString(sub, "customer"), invoiceID, mapString(sub, "id"))
		}
	}
	return ""
}
