package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

const defaultBaseURL = "https://api.stripe.com"

var retryBaseDelay = 250 * time.Millisecond

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

type ListResponse struct {
	Object   string            `json:"object"`
	URL      string            `json:"url,omitempty"`
	HasMore  bool              `json:"has_more"`
	Data     []json.RawMessage `json:"data"`
	NextPage string            `json:"next_page,omitempty"`
}

func DecodeList(raw json.RawMessage) (*ListResponse, error) {
	var list ListResponse
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return &list, nil
}

func extractErrorMessage(status int, body []byte) (string, string, string, string) {
	var parsed struct {
		Error struct {
			Type          string `json:"type"`
			Code          string `json:"code"`
			DeclineCode   string `json:"decline_code"`
			Message       string `json:"message"`
			Param         string `json:"param"`
			DocURL        string `json:"doc_url"`
			RequestLogURL string `json:"request_log_url"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		if len(body) > 0 && len(body) <= 200 {
			return fmt.Sprintf("HTTP %d: %s", status, string(body)), "", "", ""
		}
		return fmt.Sprintf("HTTP %d", status), "", "", ""
	}

	msg := parsed.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", status)
	}
	if parsed.Error.Param != "" {
		msg = msg + " (param: " + parsed.Error.Param + ")"
	}
	code := parsed.Error.Code
	if code == "" {
		code = parsed.Error.DeclineCode
	}
	return msg, parsed.Error.Type, code, parsed.Error.RequestLogURL
}

func classifyHTTPError(status int, requestID, rateLimitedReason string, maxRetries int, body []byte) *agenterrors.APIError {
	msg, errType, code, requestLogURL := extractErrorMessage(status, body)
	if requestID != "" {
		msg = msg + " (request_id: " + requestID + ")"
	}

	hintParts := []string{}
	if code != "" {
		hintParts = append(hintParts, "Stripe code: "+code)
	}
	if requestLogURL != "" {
		hintParts = append(hintParts, "request log: "+requestLogURL)
	}
	if rateLimitedReason != "" {
		hintParts = append(hintParts, "rate limit reason: "+rateLimitedReason)
	}

	switch {
	case status == 401:
		return withHint(agenterrors.New("Authentication failed: "+msg, agenterrors.FixableByHuman),
			append(hintParts, "Check the stored profile with 'agent-stripe auth check' or re-add it with a valid restricted/secret key")...)
	case status == 403:
		return withHint(agenterrors.New("Permission denied: "+msg, agenterrors.FixableByHuman),
			append(hintParts, "The key may need read permissions for this Stripe resource, or the profile may need --context for an organization key")...)
	case status == 404:
		return withHint(agenterrors.New("Not found: "+msg, agenterrors.FixableByAgent),
			append(hintParts, "Check the ID, live/sandbox mode, and Stripe context")...)
	case status == 429:
		return withHint(agenterrors.New("Rate limited: "+msg, agenterrors.FixableByRetry),
			append(hintParts, retryExhaustedHint(maxRetries))...)
	case status >= 500:
		return withHint(agenterrors.New("Stripe API error: "+msg, agenterrors.FixableByRetry),
			append(hintParts, "Stripe returned a server error; retry later")...)
	default:
		fixable := agenterrors.FixableByAgent
		if errType == "authentication_error" || errType == "permission_error" {
			fixable = agenterrors.FixableByHuman
		}
		return withHint(agenterrors.New(msg, fixable), hintParts...)
	}
}

func shouldRetry(status, attempt, maxRetries int) bool {
	return status == http.StatusTooManyRequests && attempt < maxRetries
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if parsed := retryAfterDelay(retryAfter); parsed > 0 {
		return parsed
	}
	base := retryBaseDelay * time.Duration(1<<attempt)
	if base <= 0 {
		return 0
	}
	return base + randomJitter(base/2)
}

func retryAfterDelay(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		if delay := time.Until(when); delay > 0 {
			return delay
		}
	}
	return 0
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryExhaustedHint(maxRetries int) string {
	if maxRetries <= 0 {
		return "Wait and retry with a smaller --limit or narrower time range"
	}
	return fmt.Sprintf("Retried %d time(s); wait and retry with a smaller --limit, narrower time range, or fewer expansions", maxRetries)
}

func randomJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max / 2
	}
	return time.Duration(n.Int64())
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func withHint(err *agenterrors.APIError, hints ...string) *agenterrors.APIError {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint != "" {
			parts = append(parts, hint)
		}
	}
	if len(parts) > 0 {
		err = err.WithHint(strings.Join(parts, "; "))
	}
	return err
}

func (c *Client) logDebug(method, requestURL string, status int, requestID string, body []byte) {
	entry := map[string]any{
		"@debug": "http",
		"method": method,
		"url":    requestURL,
		"status": status,
	}
	if requestID != "" {
		entry["request_id"] = requestID
	}
	var parsed any
	if json.Unmarshal(body, &parsed) == nil {
		entry["body"] = output.Redact(parsed, c.redaction)
	} else {
		entry["body_raw"] = string(body)
	}
	enc := json.NewEncoder(os.Stderr)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
}

func (c *Client) logRetry(method, requestURL string, status int, requestID, rateLimitedReason string, attempt, maxRetries int, delay time.Duration) {
	entry := map[string]any{
		"@debug":      "retry",
		"method":      method,
		"url":         requestURL,
		"status":      status,
		"attempt":     attempt,
		"max_retries": maxRetries,
		"delay_ms":    delay.Milliseconds(),
	}
	if requestID != "" {
		entry["request_id"] = requestID
	}
	if rateLimitedReason != "" {
		entry["rate_limited_reason"] = rateLimitedReason
	}
	enc := json.NewEncoder(os.Stderr)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(entry)
}
