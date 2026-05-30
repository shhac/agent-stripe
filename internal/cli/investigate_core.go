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

func runInvestigation(flags *shared.GlobalFlags, fn func(context.Context, *api.Client) ([]evidenceRecord, error)) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		records, err := fn(ctx, client)
		if err != nil {
			return err
		}
		writeEvidence(records, flags.Format)
		return nil
	})
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
