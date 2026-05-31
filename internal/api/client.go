package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

const defaultBaseURL = "https://api.stripe.com"

type Client struct {
	baseURL    string
	apiKey     string
	context    string
	apiVersion string
	maxRetries int
	http       *http.Client
	debug      bool
	redaction  output.RedactionOptions
}

type Options struct {
	APIKey     string
	Context    string
	APIVersion string
	BaseURL    string
	MaxRetries int
}

func NewClient(opts Options) *Client {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     opts.APIKey,
		context:    opts.Context,
		apiVersion: opts.APIVersion,
		maxRetries: nonNegative(opts.MaxRetries),
		http:       &http.Client{},
	}
}

func (c *Client) SetDebug(enabled bool) {
	c.debug = enabled
}

func (c *Client) SetDebugRedaction(opts output.RedactionOptions) {
	c.redaction = opts
}

func (c *Client) Get(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, buildPath(path, params), nil)
}

func (c *Client) PostForm(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, params)
}

func (c *Client) do(ctx context.Context, method, path string, form url.Values) (json.RawMessage, error) {
	for attempt := 0; ; attempt++ {
		resp, err := c.sendOnce(ctx, method, path, form)
		if err != nil {
			return nil, err
		}

		if shouldRetry(resp.status, attempt, c.maxRetries) {
			delay := retryDelay(resp.retryAfter, attempt)
			if c.debug {
				c.logRetry(method, resp.url, resp.status, resp.requestID, resp.rateLimitedReason, attempt+1, c.maxRetries, delay)
			}
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil, agenterrors.Wrap(err, agenterrors.FixableByRetry).WithHint("Retry wait interrupted; re-run the command")
			}
			continue
		}

		if resp.status >= 400 {
			return nil, classifyHTTPError(resp.status, resp.requestID, resp.rateLimitedReason, c.maxRetries, resp.body)
		}

		return json.RawMessage(resp.body), nil
	}
}

type responseEnvelope struct {
	status            int
	body              []byte
	url               string
	requestID         string
	rateLimitedReason string
	retryAfter        string
}

func (c *Client) sendOnce(ctx context.Context, method, path string, form url.Values) (*responseEnvelope, error) {
	req, err := c.buildRequest(ctx, method, path, form)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByRetry).WithHint("Network error: check connectivity and retry")
	}

	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return nil, agenterrors.Wrap(readErr, agenterrors.FixableByRetry)
	}

	if c.debug {
		c.logDebug(method, req.URL.String(), resp.StatusCode, resp.Header.Get("Request-Id"), body)
	}

	return &responseEnvelope{
		status:            resp.StatusCode,
		body:              body,
		url:               req.URL.String(),
		requestID:         resp.Header.Get("Request-Id"),
		rateLimitedReason: resp.Header.Get("Stripe-Rate-Limited-Reason"),
		retryAfter:        resp.Header.Get("Retry-After"),
	}, nil
}

func (c *Client) buildRequest(ctx context.Context, method, path string, form url.Values) (*http.Request, error) {
	var bodyReader io.Reader
	if form != nil {
		bodyReader = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}

	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.apiKey+":")))
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if c.context != "" {
		req.Header.Set("Stripe-Context", c.context)
	}
	if c.apiVersion != "" {
		req.Header.Set("Stripe-Version", c.apiVersion)
	}
	return req, nil
}

func buildPath(base string, params url.Values) string {
	if encoded := params.Encode(); encoded != "" {
		return base + "?" + encoded
	}
	return base
}
