package cli

import (
	"context"
	"net/url"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

// getSummarizedV2List is getSummarizedList for the /v2 list envelope: no
// has_more, and the next page is a token lifted out of next_page_url.
func getSummarizedV2List(flags *shared.GlobalFlags, path string, params url.Values, summarize func(map[string]any) map[string]any) error {
	return shared.WithClient(flags, func(ctx context.Context, client *api.Client) error {
		raw, err := client.Get(ctx, path, params)
		if err != nil {
			return err
		}
		list, err := api.DecodeV2List(raw)
		if err != nil {
			return err
		}
		return writeSummarizedList(flags, list.Data, shared.V2Pagination(list), summarize)
	})
}

func v2AccountListSummary(account map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, account, "id")
	copyString(summary, account, "object")
	copyString(summary, account, "display_name")
	copyString(summary, account, "created")
	copyString(summary, account, "dashboard")
	copyBool(summary, account, "closed")
	copyBool(summary, account, "livemode")
	if configurations := v2AppliedConfigurations(account); len(configurations) > 0 {
		summary["applied_configurations"] = configurations
	}
	addV2CapabilitySummary(summary, account)
	addV2RequirementSummary(summary, account, "requirements")
	addV2RequirementSummary(summary, account, "future_requirements")
	return summary
}

func addV2CapabilitySummary(summary, account map[string]any) {
	capabilities := v2AccountCapabilities(account)
	if len(capabilities) == 0 {
		return
	}
	rollup := summarizeV2Capabilities(capabilities)
	out := map[string]any{}
	for status, count := range rollup.Counts {
		out[status+"_count"] = count
	}
	if len(rollup.Restricted) > 0 {
		out["not_active"] = rollup.Restricted
	}
	summary["capabilities"] = out
}

func addV2RequirementSummary(summary, account map[string]any, key string) {
	out := map[string]any{}
	if deadline := v2RequirementSummary(account, key); deadline != nil {
		out["minimum_deadline"] = deadline
	}
	requirements := v2AccountRequirements(account, key)
	rollup := summarizeV2Requirements(requirements)
	for status, count := range rollup.Counts {
		out[status+"_count"] = count
	}
	if rollup.AwaitingUser > 0 {
		out["awaiting_user_count"] = rollup.AwaitingUser
	}
	if collector := mapString(mapAnyMap(account, key), "collector"); collector != "" {
		out["collector"] = collector
	}
	if len(out) > 0 {
		summary[key] = out
	}
}

// v2PersonListSummary keeps relationship and verification-relevant structure
// while leaving the PII-dense body (addresses, DOB, ID numbers) to
// 'persons get', where the redaction policy still applies.
func v2PersonListSummary(person map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, person, "id")
	copyString(summary, person, "object")
	copyString(summary, person, "account")
	copyString(summary, person, "created")
	copyString(summary, person, "updated")
	if relationship := mapAnyMap(person, "relationship"); relationship != nil {
		out := map[string]any{}
		copyBool(out, relationship, "owner")
		copyBool(out, relationship, "representative")
		copyBool(out, relationship, "director")
		copyBool(out, relationship, "executive")
		copyString(out, relationship, "percent_ownership")
		copyString(out, relationship, "title")
		if len(out) > 0 {
			summary["relationship"] = out
		}
	}
	if idNumbers, ok := person["id_numbers"].([]any); ok && len(idNumbers) > 0 {
		types := make([]string, 0, len(idNumbers))
		for _, raw := range idNumbers {
			if entry, ok := raw.(map[string]any); ok {
				if kind := mapString(entry, "type"); kind != "" {
					types = append(types, kind)
				}
			}
		}
		if len(types) > 0 {
			summary["id_number_types"] = types
		}
	}
	return summary
}

func v2EventListSummary(event map[string]any) map[string]any {
	summary := map[string]any{}
	copyString(summary, event, "id")
	copyString(summary, event, "object")
	copyString(summary, event, "type")
	copyString(summary, event, "created")
	copyBool(summary, event, "livemode")
	if related := mapAnyMap(event, "related_object"); related != nil {
		out := map[string]any{}
		copyString(out, related, "id")
		copyString(out, related, "type")
		copyString(out, related, "url")
		if len(out) > 0 {
			summary["related_object"] = out
		}
	}
	return summary
}
