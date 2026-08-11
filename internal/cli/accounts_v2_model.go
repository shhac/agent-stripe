package cli

import (
	"sort"
	"strings"
)

const (
	objectV2Account = "v2.core.account"
	objectV2Person  = "v2.core.account_person"
	objectV2Event   = "v2.core.event"
)

// v2AccountIncludes is Stripe's full `include` enum for a v2 account. Anything
// not requested comes back null regardless of its real value, so retrieval
// asks for everything unless the caller narrows it.
var v2AccountIncludes = []string{
	"configuration.customer",
	"configuration.merchant",
	"configuration.recipient",
	"defaults",
	"future_requirements",
	"identity",
	"requirements",
}

var v2Configurations = []string{"customer", "merchant", "recipient"}

// v2Capability is one capability leaf, flattened. Stripe nests capabilities at
// uneven depths (merchant.capabilities.card_payments vs
// merchant.capabilities.stripe_balance.payouts), so the name is the dotted path
// below `capabilities` and Configuration is the owning configuration.
type v2Capability struct {
	Configuration string
	Name          string
	Status        string
	StatusCodes   []string
}

func (c v2Capability) QualifiedName() string {
	return c.Configuration + "." + c.Name
}

func (c v2Capability) Active() bool {
	return c.Status == "active"
}

// v2AccountCapabilities flattens every capability leaf across the
// configurations present on the account, sorted for stable output.
func v2AccountCapabilities(account map[string]any) []v2Capability {
	configuration := mapAnyMap(account, "configuration")
	var capabilities []v2Capability
	for _, name := range v2Configurations {
		config := mapAnyMap(configuration, name)
		if config == nil {
			continue
		}
		capabilities = append(capabilities, collectV2Capabilities(name, "", mapAnyMap(config, "capabilities"))...)
	}
	sort.Slice(capabilities, func(a, b int) bool {
		return capabilities[a].QualifiedName() < capabilities[b].QualifiedName()
	})
	return capabilities
}

func collectV2Capabilities(configuration, prefix string, node map[string]any) []v2Capability {
	var capabilities []v2Capability
	for key, value := range node {
		child, ok := value.(map[string]any)
		if !ok {
			continue
		}
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		if status := mapString(child, "status"); status != "" {
			capabilities = append(capabilities, v2Capability{
				Configuration: configuration,
				Name:          name,
				Status:        status,
				StatusCodes:   v2StatusDetailCodes(child),
			})
			continue
		}
		capabilities = append(capabilities, collectV2Capabilities(configuration, name, child)...)
	}
	return capabilities
}

func v2StatusDetailCodes(capability map[string]any) []string {
	details, _ := capability["status_details"].([]any)
	codes := make([]string, 0, len(details))
	for _, detail := range details {
		if entry, ok := detail.(map[string]any); ok {
			if code := mapString(entry, "code"); code != "" {
				codes = append(codes, code)
			}
		}
	}
	return codes
}

// v2Requirement is one entry from requirements.entries, reduced to the fields
// that decide triage: how urgent it is, who has to act, why Stripe rejected
// what was collected, and which capabilities it holds back.
type v2Requirement struct {
	ID           string
	Description  string
	Deadline     string
	AwaitingFrom string
	ErrorCodes   []string
	Restricts    []string
	Reference    string
}

func (r v2Requirement) BlocksUser() bool {
	return r.AwaitingFrom == "user" && (r.Deadline == "past_due" || r.Deadline == "currently_due")
}

func v2AccountRequirements(account map[string]any, key string) []v2Requirement {
	entries, _ := mapValue(mapAnyMap(account, key), "entries").([]any)
	requirements := make([]v2Requirement, 0, len(entries))
	for _, raw := range entries {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		requirements = append(requirements, v2Requirement{
			ID:           mapString(entry, "id"),
			Description:  mapString(entry, "description"),
			Deadline:     mapString(mapAnyMap(entry, "minimum_deadline"), "status"),
			AwaitingFrom: mapString(entry, "awaiting_action_from"),
			ErrorCodes:   v2RequirementErrorCodes(entry),
			Restricts:    v2RequirementRestrictions(entry),
			Reference:    v2RequirementReference(entry),
		})
	}
	return requirements
}

func v2RequirementErrorCodes(entry map[string]any) []string {
	errors, _ := entry["errors"].([]any)
	codes := make([]string, 0, len(errors))
	for _, raw := range errors {
		if item, ok := raw.(map[string]any); ok {
			if code := mapString(item, "code"); code != "" {
				codes = append(codes, code)
			}
		}
	}
	return codes
}

func v2RequirementRestrictions(entry map[string]any) []string {
	restricts, _ := mapValue(mapAnyMap(entry, "impact"), "restricts_capabilities").([]any)
	names := make([]string, 0, len(restricts))
	for _, raw := range restricts {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		capability := mapString(item, "capability")
		if capability == "" {
			continue
		}
		if configuration := mapString(item, "configuration"); configuration != "" {
			capability = configuration + "." + capability
		}
		names = append(names, capability)
	}
	return names
}

func v2RequirementReference(entry map[string]any) string {
	reference := mapAnyMap(entry, "reference")
	if reference == nil {
		return ""
	}
	return firstNonEmpty(mapString(reference, "resource"), mapString(reference, "person"), mapString(reference, "inquiry"))
}

type v2CapabilityRollup struct {
	Counts     map[string]int
	Restricted []string
}

func summarizeV2Capabilities(capabilities []v2Capability) v2CapabilityRollup {
	rollup := v2CapabilityRollup{Counts: map[string]int{}}
	for _, capability := range capabilities {
		rollup.Counts[capability.Status]++
		if !capability.Active() {
			rollup.Restricted = append(rollup.Restricted, capability.QualifiedName())
		}
	}
	return rollup
}

// v2RequirementRollup keeps the all-entries counts and the awaiting-the-user
// counts apart. Mixing them reads as "you must supply 2 things" when one of
// them is a verification Stripe is still running and nobody can act on.
type v2RequirementRollup struct {
	Counts         map[string]int
	UserCounts     map[string]int
	AwaitingUser   int
	AwaitingStripe int
	Blocking       []string
}

func summarizeV2Requirements(requirements []v2Requirement) v2RequirementRollup {
	rollup := v2RequirementRollup{Counts: map[string]int{}, UserCounts: map[string]int{}}
	for _, requirement := range requirements {
		if requirement.Deadline != "" {
			rollup.Counts[requirement.Deadline]++
		}
		if requirement.AwaitingFrom == "user" {
			rollup.AwaitingUser++
			if requirement.Deadline != "" {
				rollup.UserCounts[requirement.Deadline]++
			}
		} else {
			rollup.AwaitingStripe++
		}
		if requirement.BlocksUser() {
			rollup.Blocking = append(rollup.Blocking, requirement.Description)
		}
	}
	return rollup
}

func v2RequirementSummary(account map[string]any, key string) map[string]any {
	summary := mapAnyMap(mapAnyMap(account, key), "summary")
	deadline := mapAnyMap(summary, "minimum_deadline")
	if deadline == nil {
		return nil
	}
	out := map[string]any{}
	copyString(out, deadline, "status")
	copyString(out, deadline, "time")
	if len(out) == 0 {
		return nil
	}
	return out
}

func v2AppliedConfigurations(account map[string]any) []string {
	values, _ := account["applied_configurations"].([]any)
	result := make([]string, 0, len(values))
	for _, value := range values {
		if name, ok := value.(string); ok {
			result = append(result, name)
		}
	}
	return result
}

func joinAndTruncate(values []string, max int) string {
	if len(values) <= max {
		return strings.Join(values, ", ")
	}
	return strings.Join(values[:max], ", ") + ", …"
}

// Serialization of the flattened model into finding data. It lives with the
// types it serializes rather than with the one investigation that first needed
// it.

func v2CapabilityData(capabilities []v2Capability) []map[string]any {
	data := make([]map[string]any, 0, len(capabilities))
	for _, capability := range capabilities {
		entry := map[string]any{
			"capability":    capability.QualifiedName(),
			"configuration": capability.Configuration,
			"status":        capability.Status,
		}
		if len(capability.StatusCodes) > 0 {
			entry["status_details"] = capability.StatusCodes
		}
		data = append(data, entry)
	}
	return data
}

func v2RequirementData(requirements []v2Requirement) []map[string]any {
	data := make([]map[string]any, 0, len(requirements))
	for _, requirement := range requirements {
		entry := map[string]any{
			"description":          requirement.Description,
			"minimum_deadline":     requirement.Deadline,
			"awaiting_action_from": requirement.AwaitingFrom,
		}
		if requirement.ID != "" {
			entry["id"] = requirement.ID
		}
		if len(requirement.ErrorCodes) > 0 {
			entry["error_codes"] = requirement.ErrorCodes
		}
		if len(requirement.Restricts) > 0 {
			entry["restricts_capabilities"] = requirement.Restricts
		}
		if requirement.Reference != "" {
			entry["reference"] = requirement.Reference
		}
		data = append(data, entry)
	}
	return data
}
