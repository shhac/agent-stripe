package api

import (
	"encoding/json"
	"net/url"
	"strings"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

// V2ListResponse is the /v2 list envelope. It has no has_more and no cursor
// IDs: the next page is a token embedded in a URL Stripe returns.
type V2ListResponse struct {
	Data            []json.RawMessage `json:"data"`
	NextPageURL     string            `json:"next_page_url,omitempty"`
	PreviousPageURL string            `json:"previous_page_url,omitempty"`
}

func DecodeV2List(raw json.RawMessage) (*V2ListResponse, error) {
	var list V2ListResponse
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return &list, nil
}

func (l *V2ListResponse) HasMore() bool {
	return l != nil && l.NextPageURL != ""
}

// NextPageToken pulls the ?page= token out of next_page_url so callers can pass
// it straight back as --page, rather than having to parse Stripe's URL.
func (l *V2ListResponse) NextPageToken() string {
	if l == nil {
		return ""
	}
	return pageTokenFromURL(l.NextPageURL)
}

func pageTokenFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	query := raw
	if idx := strings.IndexByte(raw, '?'); idx >= 0 {
		query = raw[idx+1:]
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}
	return values.Get("page")
}
