package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/api"
	"github.com/shhac/agent-stripe/internal/cli/shared"
)

func newInvestigateConnectReadiness(globals shared.GlobalsFunc, outputOpts *evidenceOptions) *cobra.Command {
	var limit int
	var namespace string
	cmd := &cobra.Command{
		Use:   "connect-readiness",
		Short: "Find connected accounts that cannot take payments or get paid",
		Long: "Sweeps connected accounts and reports the ones with blockers. Neither account\n" +
			"list endpoint returns capability or requirement detail — Stripe's v2 list has no\n" +
			"include at all — so each account is retrieved individually; keep --limit modest.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateNamespace(namespace); err != nil {
				return err
			}
			return runWithInvestigator(globals(), outputOpts, func(inv investigator) error {
				return inv.connectReadiness(limit, namespace)
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum accounts to inspect")
	cmd.Flags().StringVar(&namespace, "namespace", namespaceAuto, "Account namespace to sweep: auto, v1, or v2")
	return cmd
}

// connectReadiness answers the platform-side question the per-account commands
// cannot: of my connected accounts, which ones need attention? It sweeps v2
// when the platform has it and falls back to v1 wholesale, because a platform
// is on one model or the other.
func (i investigator) connectReadiness(limit int, namespace string) error {
	if namespace == "" {
		namespace = namespaceAuto
	}
	if namespace != namespaceV1 {
		result, err := i.connectReadinessV2(limit)
		if err == nil {
			i.add(connectReadinessFinding(namespaceV2, result))
			return nil
		}
		if namespace == namespaceV2 || !isNotV2AccountError(err) {
			return err
		}
		i.add(evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  "Accounts v2 is not available for this platform (" + api.ErrorCode(err) + "); sweeping Connect v1 accounts instead.",
			Data:     map[string]any{"namespace": namespaceV1, "v2_error_code": api.ErrorCode(err)},
		})
	}
	result, err := i.connectReadinessV1(limit)
	if err != nil {
		return err
	}
	i.add(connectReadinessFinding(namespaceV1, result))
	return nil
}

// readinessSweep separates the three outcomes. Counting an account that could
// not be retrieved as "not blocked" let the sweep report a healthy platform
// when every assessment had failed.
type readinessSweep struct {
	inspected  int
	blocked    []string
	unassessed []string
}

func (i investigator) connectReadinessV2(limit int) (readinessSweep, error) {
	params := url.Values{}
	shared.AddLimit(params, limit)
	// The list is fetched without emitting: /v2/core/accounts supports no
	// include, so its records carry null configuration and requirements and
	// would shadow the detailed per-account records fetched below.
	accounts, err := i.fetchListV2("/v2/core/accounts", params)
	if err != nil {
		return readinessSweep{}, err
	}
	includes, err := v2AccountIncludeParams(nil)
	if err != nil {
		return readinessSweep{}, err
	}
	sweep := readinessSweep{inspected: len(accounts)}
	for _, listed := range accounts {
		id := mapString(listed, "id")
		account, err := i.get(v2AccountPath(id), includes)
		if err != nil {
			i.add(relatedWarning("account "+id, err))
			sweep.unassessed = append(sweep.unassessed, id)
			continue
		}
		finding := v2AccountHealthFinding(account, 0)
		if finding.Severity != "warning" {
			continue
		}
		sweep.blocked = append(sweep.blocked, id)
		i.add(finding)
	}
	return sweep, nil
}

func (i investigator) connectReadinessV1(limit int) (readinessSweep, error) {
	accounts, err := i.list("/v1/accounts", valuesWithLimit(limit))
	if err != nil {
		return readinessSweep{}, err
	}
	sweep := readinessSweep{inspected: len(accounts)}
	for _, account := range accounts {
		finding := accountHealthFinding(account)
		if finding.Severity != "warning" {
			continue
		}
		sweep.blocked = append(sweep.blocked, mapString(account, "id"))
		i.add(finding)
	}
	return sweep, nil
}

func connectReadinessFinding(namespace string, sweep readinessSweep) evidenceRecord {
	data := map[string]any{
		"namespace":     namespace,
		"inspected":     sweep.inspected,
		"blocked_count": len(sweep.blocked),
	}
	if len(sweep.blocked) > 0 {
		data["blocked"] = sweep.blocked
	}
	if len(sweep.unassessed) > 0 {
		data["unassessed"] = sweep.unassessed
		data["unassessed_count"] = len(sweep.unassessed)
	}

	summary := ""
	if len(sweep.blocked) == 0 {
		summary = fmt.Sprintf("No blockers found across %d inspected connected accounts (%s).", sweep.inspected, namespace)
	} else {
		summary = fmt.Sprintf("%d of %d inspected connected accounts have blockers (%s): %s. Use account-health for the detail on one.",
			len(sweep.blocked), sweep.inspected, namespace, joinAndTruncate(sweep.blocked, 10))
	}
	// An account that could not be retrieved was not assessed, so the sweep
	// cannot claim it is healthy — and cannot call the run clean.
	if len(sweep.unassessed) > 0 {
		summary += fmt.Sprintf(" %d could not be assessed and are not covered by that: %s.",
			len(sweep.unassessed), joinAndTruncate(sweep.unassessed, 10))
	}

	severity := "info"
	if len(sweep.blocked) > 0 || len(sweep.unassessed) > 0 {
		severity = "warning"
	}
	record := evidenceRecord{Type: "finding", Severity: severity, Summary: summary, Data: data}
	if len(sweep.blocked) > 0 {
		record.Command = "agent-stripe investigate account-health " + sweep.blocked[0]
	}
	return record
}
