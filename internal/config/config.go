package config

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/shhac/lib-agent-cli/creds"
	"github.com/shhac/lib-agent-cli/xdg"
)

const DefaultAPIVersion = "2025-06-30.basil"

type Config struct {
	DefaultProfile string             `json:"default_profile,omitempty"`
	Defaults       Defaults           `json:"defaults,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Defaults struct {
	TimeoutMS  *int `json:"timeout_ms,omitempty"`
	MaxRetries *int `json:"max_retries,omitempty"`
}

type Profile struct {
	Context        string `json:"context,omitempty"`
	APIVersion     string `json:"api_version,omitempty"`
	CredentialType string `json:"credential_type,omitempty"`
}

var (
	cache       *Config
	cacheMu     sync.Mutex
	overrideDir string
)

func SetConfigDir(dir string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	overrideDir = dir
	cache = nil
}

func ConfigDir() string {
	if overrideDir != "" {
		return overrideDir
	}
	return xdg.ConfigDir("agent-stripe")
}

func configPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func ConfigPath() string {
	return configPath()
}

// store is the shared config file: 0600 writes into a 0700 parent, atomic
// replacement, and Update for a locked read-modify-write. This used to be
// hand-rolled here with os.ReadFile/os.WriteFile — no lock, not atomic — so
// two concurrent writers (e.g. `auth add` racing `config set`) could each
// build their write from a stale snapshot and silently erase each other's
// change, including a just-added profile.
func store() creds.Store {
	return creds.Store{Path: configPath()}
}

func Read() *Config {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cache != nil {
		return cache
	}
	cache = loadFresh()
	return cache
}

// loadFresh reads config.json straight off disk, bypassing the cache. update
// uses this (via store().Update) rather than Read so a locked read-modify-
// write mutates the file's actual current state, not a value that may have
// been cached before a concurrent writer's change landed.
func loadFresh() *Config {
	var cfg Config
	if err := store().Load(&cfg); err != nil {
		return defaultConfig()
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	return &cfg
}

func invalidateCache() {
	cacheMu.Lock()
	cache = nil
	cacheMu.Unlock()
}

// Write overwrites config.json with cfg wholesale and invalidates the cache.
//
// This is NOT a read-modify-write: it has no read step of its own, so a
// caller that built cfg from a stale Read() can still clobber a concurrent
// writer. Every read-modify-write in this package goes through update
// instead, which holds one lock across load, mutate, and save. Write still
// takes the same lock around its own save so it cannot interleave with a
// concurrent update's write.
func Write(cfg *Config) error {
	if err := store().WithLock(func() error {
		return store().Save(cfg)
	}); err != nil {
		return err
	}
	invalidateCache()
	return nil
}

// update runs mutate against config.json's current on-disk state under the
// store's exclusive lock, then invalidates the cache so the next Read
// reflects the write. Concurrent callers serialize instead of each building
// their change from a snapshot taken before the other's write landed.
func update(mutate func(cfg *Config) error) error {
	var cfg Config
	err := store().Update(&cfg, func() error {
		if cfg.Profiles == nil {
			cfg.Profiles = make(map[string]Profile)
		}
		return mutate(&cfg)
	})
	if err != nil {
		return err
	}
	invalidateCache()
	return nil
}

func defaultConfig() *Config {
	return &Config{
		Profiles: make(map[string]Profile),
	}
}

func StoreProfile(alias string, profile Profile) error {
	if profile.APIVersion == "" {
		profile.APIVersion = DefaultAPIVersion
	}
	return update(func(cfg *Config) error {
		cfg.Profiles[alias] = profile
		if cfg.DefaultProfile == "" {
			cfg.DefaultProfile = alias
		}
		return nil
	})
}

func RemoveProfile(alias string) error {
	return update(func(cfg *Config) error {
		delete(cfg.Profiles, alias)
		if cfg.DefaultProfile == alias {
			cfg.DefaultProfile = ""
			for name := range cfg.Profiles {
				cfg.DefaultProfile = name
				break
			}
		}
		return nil
	})
}

func SetDefault(alias string) error {
	return update(func(cfg *Config) error {
		if _, ok := cfg.Profiles[alias]; !ok {
			return fmt.Errorf("profile %q is not configured", alias)
		}
		cfg.DefaultProfile = alias
		return nil
	})
}

func UpdateProfile(alias string, apply func(Profile) Profile) error {
	return update(func(cfg *Config) error {
		profile, ok := cfg.Profiles[alias]
		if !ok {
			return fmt.Errorf("profile %q is not configured", alias)
		}
		profile = apply(profile)
		if profile.APIVersion == "" {
			profile.APIVersion = DefaultAPIVersion
		}
		cfg.Profiles[alias] = profile
		return nil
	})
}

func SetDefaultValue(key string, value int) error {
	return update(func(cfg *Config) error {
		switch key {
		case "timeout_ms":
			cfg.Defaults.TimeoutMS = intPtr(value)
		case "max_retries":
			cfg.Defaults.MaxRetries = intPtr(value)
		default:
			return fmt.Errorf("unknown config key %q", key)
		}
		return nil
	})
}

func UnsetDefaultValue(key string) error {
	return update(func(cfg *Config) error {
		switch key {
		case "timeout_ms":
			cfg.Defaults.TimeoutMS = nil
		case "max_retries":
			cfg.Defaults.MaxRetries = nil
		default:
			return fmt.Errorf("unknown config key %q", key)
		}
		return nil
	})
}

func intPtr(value int) *int {
	return &value
}
