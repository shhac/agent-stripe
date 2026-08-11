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
	ctx      context.Context
	client   *api.Client
	evidence *evidenceCollector
}

func runInvestigation(flags *shared.GlobalFlags, opts *evidenceOptions, fn func(context.Context, *api.Client, *evidenceCollector) error) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		collector := newEvidenceCollector(flags.Format, opts.withRedaction(flags))
		if err := fn(ctx, client, collector); err != nil {
			return err
		}
		collector.flush()
		return nil
	})
}

func runWithInvestigator(flags *shared.GlobalFlags, opts *evidenceOptions, fn func(investigator) error) error {
	return runInvestigation(flags, opts, func(ctx context.Context, client *api.Client, evidence *evidenceCollector) error {
		return fn(investigator{ctx: ctx, client: client, evidence: evidence})
	})
}

// postFormAs records the result under a caller-chosen label, for the cases
// where Stripe's own `object` field is not the name the investigation wants —
// an invoice fetched from create_preview is an "invoice_preview". Choosing the
// label at the fetch produces one record; labelling afterwards with an explicit
// add produced two, because the auto-emitted one was already kept under a
// different key. fetchList is the same idea for a collection.
func (i investigator) postFormAs(object, path string, params url.Values) (map[string]any, error) {
	raw, err := i.client.PostForm(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	item, err := decodeObject(raw)
	if err != nil {
		return nil, err
	}
	i.add(entityRecord(object, item))
	return item, nil
}

// fetchList retrieves a list without recording its items, for callers that
// label the records themselves.
func (i investigator) fetchList(path string, params url.Values) ([]map[string]any, error) {
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

func (i investigator) get(path string, params url.Values) (map[string]any, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	item, err := decodeObject(raw)
	if err != nil {
		return nil, err
	}
	i.emitEntity(item)
	return item, nil
}

func (i investigator) list(path string, params url.Values) ([]map[string]any, error) {
	items, _, err := i.listPage(path, params)
	return items, err
}

// listPage also reports whether Stripe had more results. A scan that states an
// absence — "no duplicates found", "no charge matched" — is only true of what
// it actually read, so the workflows making those claims need to know they saw
// a partial page.
func (i investigator) listPage(path string, params url.Values) ([]map[string]any, bool, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, false, err
	}
	list, err := api.DecodeList(raw)
	if err != nil {
		return nil, false, err
	}
	items, err := i.decodeListItems(list.Data)
	return items, list.HasMore, err
}

// listV2 is list for the /v2 list envelope, which has no has_more and no
// cursor IDs. Investigations read one page; deeper paging is the job of the
// resource commands with --page.
func (i investigator) listV2(path string, params url.Values) ([]map[string]any, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	list, err := api.DecodeV2List(raw)
	if err != nil {
		return nil, err
	}
	return i.decodeListItems(list.Data)
}

// fetchListV2 retrieves a /v2 list without recording its items. A sparse list
// record would otherwise shadow a fuller one fetched afterwards, because the
// collector dedups on object and ID and keeps the first it saw.
func (i investigator) fetchListV2(path string, params url.Values) ([]map[string]any, error) {
	raw, err := i.client.Get(i.ctx, path, params)
	if err != nil {
		return nil, err
	}
	list, err := api.DecodeV2List(raw)
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

func (i investigator) decodeListItems(data []json.RawMessage) ([]map[string]any, error) {
	items := make([]map[string]any, 0, len(data))
	for _, rawItem := range data {
		item, err := decodeObject(rawItem)
		if err != nil {
			return nil, err
		}
		i.emitEntity(item)
		items = append(items, item)
	}
	return items, nil
}

func (i investigator) emitEntity(item map[string]any) {
	if !isStripeEntity(item) {
		return
	}
	i.add(entityRecord(mapString(item, "object"), item))
}

// add records evidence. There is one accumulator — the collector — so a
// workflow adds records where it finds them rather than threading a slice
// through its call graph.
func (i investigator) add(records ...evidenceRecord) {
	i.evidence.add(records...)
}

func (i investigator) addList(object string, items []map[string]any) {
	i.evidence.addList(object, items)
}

// count reports how many records this investigation has produced so far, so a
// step can ask whether it found anything by comparing counts around itself.
func (i investigator) count() int {
	return i.evidence.count()
}

func (i investigator) since(mark int) []evidenceRecord {
	return i.evidence.since(mark)
}

func decodeObject(raw json.RawMessage) (map[string]any, error) {
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return item, nil
}
