package cli

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

type investigator struct {
	ctx    context.Context
	client *api.Client
}

type investigationOutputOptions struct {
	full         bool
	expandFields []string
	maxString    int
}

func runInvestigation(flags *shared.GlobalFlags, outputOpts *investigationOutputOptions, fn func(context.Context, *api.Client) ([]evidenceRecord, error)) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		records, err := fn(ctx, client)
		if err != nil {
			return err
		}
		writeEvidence(records, flags.Format, outputOpts.evidenceOptions(flags))
		return nil
	})
}

func runWithInvestigator(flags *shared.GlobalFlags, outputOpts *investigationOutputOptions, fn func(investigator) ([]evidenceRecord, error)) error {
	return runInvestigation(flags, outputOpts, func(ctx context.Context, client *api.Client) ([]evidenceRecord, error) {
		return fn(investigator{ctx: ctx, client: client})
	})
}

func (opts *investigationOutputOptions) evidenceOptions(flags *shared.GlobalFlags) evidenceOptions {
	redaction := shared.RedactionOptions(flags)
	if opts == nil {
		evidenceOpts := defaultEvidenceOptions()
		evidenceOpts.redaction = redaction
		return evidenceOpts
	}
	evidenceOpts := evidenceOptions{
		full:         opts.full,
		expandFields: opts.expandFields,
		maxString:    opts.maxString,
		redaction:    redaction,
	}
	if evidenceOpts.maxString <= 0 {
		evidenceOpts.maxString = defaultMaxString
	}
	return evidenceOpts
}

func (i investigator) get(path string, params url.Values) (map[string]any, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	return decodeObject(raw)
}

func (i investigator) postForm(path string, params url.Values) (map[string]any, error) {
	raw, err := i.client.PostForm(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	return decodeObject(raw)
}

func (i investigator) list(path string, params url.Values) ([]map[string]any, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	list, err := api.DecodeList(raw)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(list.Data))
	for _, rawItem := range list.Data {
		item, err := decodeObject(rawItem)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func decodeObject(raw json.RawMessage) (map[string]any, error) {
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return item, nil
}
