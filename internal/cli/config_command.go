package cli

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shhac/agent-stripe/internal/cli/shared"
	appconfig "github.com/shhac/agent-stripe/internal/config"
	agenterrors "github.com/shhac/agent-stripe/internal/errors"
	"github.com/shhac/agent-stripe/internal/output"
)

func registerConfig(root *cobra.Command) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and edit non-secret CLI configuration",
	}
	configCmd.AddCommand(newConfigPathCommand())
	configCmd.AddCommand(newConfigShowCommand())
	configCmd.AddCommand(newConfigGetCommand())
	configCmd.AddCommand(newConfigSetCommand())
	configCmd.AddCommand(newConfigUnsetCommand())
	root.AddCommand(configCmd)
}

func newConfigPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show config file locations",
		RunE: func(cmd *cobra.Command, args []string) error {
			shared.WriteItem(map[string]any{
				"config_dir":             appconfig.ConfigDir(),
				"config_path":            appconfig.ConfigPath(),
				"credentials_index_path": filepath.Join(appconfig.ConfigDir(), "credentials.json"),
			}, "")
			return nil
		},
	}
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show non-secret configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			shared.WriteItem(configView(appconfig.Read()), "")
			return nil
		},
	}
}

func newConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a non-secret config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeConfigKey(args[0])
			value, set, err := configValue(appconfig.Read(), key)
			if err != nil {
				output.WriteError(output.Stderr(), err)
				return nil
			}
			shared.WriteItem(map[string]any{
				"key":   key,
				"set":   set,
				"value": value,
			}, "")
			return nil
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a non-secret config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeConfigKey(args[0])
			value, err := strconv.Atoi(args[1])
			if err != nil || value < 0 {
				output.WriteError(output.Stderr(), agenterrors.New("config value must be a non-negative integer", agenterrors.FixableByAgent).
					WithHint("Supported keys: max_retries, timeout_ms"))
				return nil
			}
			if err := appconfig.SetDefaultValue(key, value); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByAgent).
					WithHint("Supported keys: max_retries, timeout_ms"))
				return nil
			}
			shared.WriteItem(map[string]any{
				"status": "set",
				"key":    key,
				"value":  value,
			}, "")
			return nil
		},
	}
}

func newConfigUnsetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Unset a non-secret config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := normalizeConfigKey(args[0])
			if err := appconfig.UnsetDefaultValue(key); err != nil {
				output.WriteError(output.Stderr(), agenterrors.Wrap(err, agenterrors.FixableByAgent).
					WithHint("Supported keys: max_retries, timeout_ms"))
				return nil
			}
			shared.WriteItem(map[string]any{
				"status": "unset",
				"key":    key,
			}, "")
			return nil
		},
	}
}

func configView(cfg *appconfig.Config) map[string]any {
	profiles := map[string]any{}
	for alias, profile := range cfg.Profiles {
		profiles[alias] = map[string]any{
			"context":     profile.Context,
			"api_version": profile.APIVersion,
			"credential":  "keychain",
		}
	}
	defaults := map[string]any{}
	if cfg.Defaults.MaxRetries != nil {
		defaults["max_retries"] = *cfg.Defaults.MaxRetries
	}
	if cfg.Defaults.TimeoutMS != nil {
		defaults["timeout_ms"] = *cfg.Defaults.TimeoutMS
	}
	return map[string]any{
		"config_path":     appconfig.ConfigPath(),
		"default_profile": cfg.DefaultProfile,
		"defaults":        defaults,
		"profiles":        profiles,
	}
}

func configValue(cfg *appconfig.Config, key string) (any, bool, error) {
	switch key {
	case "default_profile":
		return cfg.DefaultProfile, cfg.DefaultProfile != "", nil
	case "max_retries":
		if cfg.Defaults.MaxRetries == nil {
			return nil, false, nil
		}
		return *cfg.Defaults.MaxRetries, true, nil
	case "timeout_ms":
		if cfg.Defaults.TimeoutMS == nil {
			return nil, false, nil
		}
		return *cfg.Defaults.TimeoutMS, true, nil
	default:
		return nil, false, agenterrors.New("unknown config key "+key, agenterrors.FixableByAgent).
			WithHint("Supported keys: default_profile, max_retries, timeout_ms")
	}
}

func normalizeConfigKey(key string) string {
	return strings.ReplaceAll(strings.TrimSpace(key), "-", "_")
}
