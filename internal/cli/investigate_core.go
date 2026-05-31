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

type investigationOutputOptions struct {
	full         bool
	expandFields []string
	maxString    int
}

func runInvestigation(flags *shared.GlobalFlags, outputOpts *investigationOutputOptions, fn func(context.Context, *api.Client, *evidenceCollector) ([]evidenceRecord, error)) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		evidenceOpts := outputOpts.evidenceOptions(flags)
		stream := newEvidenceStreamer(flags.Format, evidenceOpts)
		collector := newEvidenceCollector(stream)
		records, err := fn(ctx, client, collector)
		if err != nil {
			return err
		}
		if len(collector.records) > 0 {
			records = collector.records
		}
		if stream != nil {
			return nil
		}
		writeEvidence(records, flags.Format, evidenceOpts)
		return nil
	})
}

func runWithInvestigator(flags *shared.GlobalFlags, outputOpts *investigationOutputOptions, fn func(investigator) ([]evidenceRecord, error)) error {
	return runInvestigation(flags, outputOpts, func(ctx context.Context, client *api.Client, evidence *evidenceCollector) ([]evidenceRecord, error) {
		return fn(investigator{ctx: ctx, client: client, evidence: evidence})
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
	item, err := decodeObject(raw)
	if err != nil {
		return nil, err
	}
	i.emitEntity(item)
	return item, nil
}

func (i investigator) postForm(path string, params url.Values) (map[string]any, error) {
	raw, err := i.client.PostForm(i.ctx, path, params)
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
		i.emitEntity(item)
		items = append(items, item)
	}
	return items, nil
}

func (i investigator) emitEntity(item map[string]any) {
	if !isStripeEntity(item) {
		return
	}
	i.emitEvidence(entityRecord(mapString(item, "object"), item))
}

func (i investigator) appendEvidence(records []evidenceRecord, newRecords ...evidenceRecord) []evidenceRecord {
	if i.evidence == nil {
		return append(records, newRecords...)
	}
	return i.evidence.append(records, newRecords...)
}

func (i investigator) appendEvidenceAll(records []evidenceRecord, newRecords []evidenceRecord) []evidenceRecord {
	if i.evidence == nil {
		return append(records, newRecords...)
	}
	return i.evidence.appendAll(records, newRecords)
}

func (i investigator) emitEvidence(record evidenceRecord) {
	if i.evidence == nil {
		return
	}
	i.evidence.emit(record)
}

func decodeObject(raw json.RawMessage) (map[string]any, error) {
	var item map[string]any
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return item, nil
}
