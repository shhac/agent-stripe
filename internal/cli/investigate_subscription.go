package cli

import (
	"net/url"
	"strings"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

func (i investigator) findSubscriptions(subscription, customer, metadata string, limit int) ([]map[string]any, error) {
	switch {
	case subscription != "":
		if err := validateExpectedStripeID(subscription, "subscription"); err != nil {
			return nil, err
		}
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
		params := url.Values{"query": []string{stripeSearchMetadataEquals(key, value)}}
		shared.AddLimit(params, limit)
		return i.list("/v1/subscriptions/search", params)
	default:
		params := url.Values{}
		shared.AddLimit(params, limit)
		shared.AddString(params, "customer", customer)
		return i.list("/v1/subscriptions", params)
	}
}
