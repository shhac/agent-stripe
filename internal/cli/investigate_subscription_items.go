package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateSubscriptionItems(globals shared.GlobalsFunc, outputOpts *investigationOutputOptions) *cobra.Command {
	var subscription string
	cmd := &cobra.Command{
		Use:   "subscription-items",
		Short: "Show subscription items, prices, products, and product metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !shared.RequireFlag("subscription", subscription, "Provide a Subscription ID such as sub_...") {
				return nil
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) ([]evidenceRecord, error) {
				return inv.subscriptionItemsEvidence(subscription)
			})
		},
	}
	cmd.Flags().StringVar(&subscription, "subscription", "", "Subscription ID")
	return cmd
}

func (i investigator) subscriptionItemsEvidence(subscriptionID string) ([]evidenceRecord, error) {
	bundle, err := i.subscriptionItemsBundle(subscriptionID)
	if err != nil {
		return nil, err
	}
	records := i.appendEvidenceAll(nil, bundle.records)
	records = i.appendEvidence(records, subscriptionItemsFinding(subscriptionID, len(bundle.items)))
	return records, nil
}

type subscriptionItemsBundle struct {
	sub     map[string]any
	items   []map[string]any
	records []evidenceRecord
}

func (i investigator) subscriptionItemsBundle(subscriptionID string) (*subscriptionItemsBundle, error) {
	if err := validateExpectedStripeID(subscriptionID, "subscription"); err != nil {
		return nil, err
	}
	records := []evidenceRecord{}
	sub, err := i.get("/v1/subscriptions/"+url.PathEscape(subscriptionID), url.Values{})
	if err != nil {
		return nil, err
	}
	records = i.appendEvidence(records, entityRecord("subscription", sub))

	items, err := i.list("/v1/subscription_items", url.Values{"subscription": []string{subscriptionID}, "limit": []string{"100"}})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		records = i.appendEvidence(records, entityRecord("subscription_item", item))
		price := mapAnyMap(item, "price")
		if productID := idFromValue(price["product"]); productID != "" {
			if product, productErr := i.get("/v1/products/"+url.PathEscape(productID), url.Values{}); productErr == nil {
				records = i.appendEvidence(records, entityRecord("product", product))
			}
		}
	}
	return &subscriptionItemsBundle{sub: sub, items: items, records: records}, nil
}

func subscriptionItemsFinding(subscriptionID string, itemCount int) evidenceRecord {
	return evidenceRecord{
		Type:     "finding",
		Severity: "info",
		Summary:  fmt.Sprintf("Subscription %s has %d visible item(s). Use price/product metadata for internal product mapping.", subscriptionID, itemCount),
		Data: map[string]any{
			"subscription": subscriptionID,
			"item_count":   itemCount,
		},
	}
}
