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
		blocked, total, err := i.connectReadinessV2(limit)
		if err == nil {
			i.add(connectReadinessFinding(namespaceV2, blocked, total))
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
	blocked, total, err := i.connectReadinessV1(limit)
	if err != nil {
		return err
	}
	i.add(connectReadinessFinding(namespaceV1, blocked, total))
	return nil
}

func (i investigator) connectReadinessV2(limit int) ([]string, int, error) {
	params := url.Values{}
	shared.AddLimit(params, limit)
	accounts, err := i.listV2("/v2/core/accounts", params)
	if err != nil {
		return nil, 0, err
	}
	includes, err := v2AccountIncludeParams(nil)
	if err != nil {
		return nil, 0, err
	}
	blocked := []string{}
	for _, listed := range accounts {
		id := mapString(listed, "id")
		account, err := i.get(v2AccountPath(id), includes)
		if err != nil {
			i.add(relatedWarning("account "+id, err))
			continue
		}
		finding := v2AccountHealthFinding(account, 0)
		if finding.Severity != "warning" {
			continue
		}
		blocked = append(blocked, id)
		i.add(finding)
	}
	return blocked, len(accounts), nil
}

func (i investigator) connectReadinessV1(limit int) ([]string, int, error) {
	accounts, err := i.list("/v1/accounts", valuesWithLimit(limit))
	if err != nil {
		return nil, 0, err
	}
	blocked := []string{}
	for _, account := range accounts {
		finding := accountHealthFinding(account)
		if finding.Severity != "warning" {
			continue
		}
		blocked = append(blocked, mapString(account, "id"))
		i.add(finding)
	}
	return blocked, len(accounts), nil
}

func connectReadinessFinding(namespace string, blocked []string, total int) evidenceRecord {
	if len(blocked) == 0 {
		return evidenceRecord{
			Type:     "finding",
			Severity: "info",
			Summary:  fmt.Sprintf("All %d inspected connected accounts are unblocked (%s).", total, namespace),
			Data:     map[string]any{"namespace": namespace, "inspected": total, "blocked_count": 0},
		}
	}
	return evidenceRecord{
		Type:     "finding",
		Severity: "warning",
		Summary: fmt.Sprintf("%d of %d inspected connected accounts have blockers (%s): %s. Use account-health for the detail on one.",
			len(blocked), total, namespace, joinAndTruncate(blocked, 10)),
		Command: "agent-stripe investigate account-health " + blocked[0],
		Data: map[string]any{
			"namespace":     namespace,
			"inspected":     total,
			"blocked_count": len(blocked),
			"blocked":       blocked,
		},
	}
}
