package shared

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/shhac/agent-stripe/internal/api"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

func WithClient(flags *GlobalFlags, fn func(context.Context, *api.Client) error) error {
	resolved, err := ResolveProfile(flags)
	if err != nil {
		return err
	}
	return WithResolvedClient(flags, resolved, fn)
}

func WithResolvedClient(flags *GlobalFlags, resolved *ResolvedProfile, fn func(context.Context, *api.Client) error) error {
	if flags.Debug {
		WriteDebug(map[string]any{
			"@debug":            "client",
			"profile":           resolved.Alias,
			"credential_source": resolved.CredentialSource,
			"context":           resolved.Profile.Context,
			"api_version":       resolved.Profile.APIVersion,
			"v2_api_version":    resolved.Profile.V2APIVersion,
			"base_url":          resolvedBaseURL(resolved.BaseURL),
			"timeout_ms":        flags.TimeoutMS,
			"max_retries":       flags.MaxRetries,
		})
	}
	ctx, cancel := ContextWithTimeout(context.Background(), flags.TimeoutMS)
	defer cancel()

	client := api.NewClient(api.Options{
		APIKey:       resolved.APIKey,
		Context:      resolved.Profile.Context,
		APIVersion:   resolved.Profile.APIVersion,
		V2APIVersion: resolved.Profile.V2APIVersion,
		BaseURL:      resolved.BaseURL,
		MaxRetries:   flags.MaxRetries,
	})
	client.SetDebug(flags.Debug)
	client.SetDebugRedaction(RedactionOptions(flags))
	return fn(ctx, client)
}

func GetRawItem(flags *GlobalFlags, path string, params url.Values) error {
	return WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.Get(ctx, path, params)
		if err != nil {
			return err
		}
		WriteRawItem(raw, flags.Format, RedactionOptions(flags))
		return nil
	})
}

// FetchItem retrieves path, decodes the JSON response, and applies the expose-
// aware redaction policy from flags. It returns the cleaned value so an
// EntityGet resolver can hand it back to the multi-get stream without writing
// it directly.
func FetchItem(ctx context.Context, client *api.Client, flags *GlobalFlags, path string, params url.Values) (any, error) {
	raw, err := client.Get(ctx, path, params)
	if err != nil {
		return nil, err
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return output.Redact(data, RedactionOptions(flags)), nil
}

// GetRawListInContext is GetRawList for an endpoint addressed by Stripe-Context
// rather than by a path segment.
func GetRawListInContext(flags *GlobalFlags, stripeContext, path string, params url.Values) error {
	return WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.WithContext(stripeContext).Get(ctx, path, params)
		if err != nil {
			return err
		}
		return WriteRawList(path, raw, flags.Format, RedactionOptions(flags))
	})
}

func GetRawList(flags *GlobalFlags, path string, params url.Values) error {
	return WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.Get(ctx, path, params)
		if err != nil {
			return err
		}
		return WriteRawList(path, raw, flags.Format, RedactionOptions(flags))
	})
}

func PostFormRawItem(flags *GlobalFlags, path string, params url.Values) error {
	return WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.PostForm(ctx, path, params)
		if err != nil {
			return err
		}
		WriteRawItem(raw, flags.Format, RedactionOptions(flags))
		return nil
	})
}

func WriteDebug(fields map[string]any) {
	enc := json.NewEncoder(output.Stderr())
	enc.SetEscapeHTML(false)
	_ = enc.Encode(fields)
}

func resolvedBaseURL(baseURL string) string {
	if baseURL == "" {
		return "https://api.stripe.com"
	}
	return baseURL
}
