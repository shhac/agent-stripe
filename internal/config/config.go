package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const DefaultAPIVersion = "2025-06-30.basil"

type Config struct {
	DefaultProfile string             `json:"default_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Profile struct {
	Context    string `json:"context,omitempty"`
	APIVersion string `json:"api_version,omitempty"`
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
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "agent-stripe")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agent-stripe")
}

func configPath() string {
	return filepath.Join(ConfigDir(), "config.json")
}

func Read() *Config {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cache != nil {
		return cache
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		cache = defaultConfig()
		return cache
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		cache = defaultConfig()
		return cache
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	cache = &cfg
	return cache
}

func Write(cfg *Config) error {
	cacheMu.Lock()
	cache = nil
	cacheMu.Unlock()

	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), append(data, '\n'), 0o644)
}

func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cache = nil
}

func defaultConfig() *Config {
	return &Config{
		Profiles: make(map[string]Profile),
	}
}

func StoreProfile(alias string, profile Profile) error {
	cfg := Read()
	if profile.APIVersion == "" {
		profile.APIVersion = DefaultAPIVersion
	}
	cfg.Profiles[alias] = profile
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = alias
	}
	return Write(cfg)
}

func RemoveProfile(alias string) error {
	cfg := Read()
	delete(cfg.Profiles, alias)
	if cfg.DefaultProfile == alias {
		cfg.DefaultProfile = ""
		for name := range cfg.Profiles {
			cfg.DefaultProfile = name
			break
		}
	}
	return Write(cfg)
}

func SetDefault(alias string) error {
	cfg := Read()
	if _, ok := cfg.Profiles[alias]; !ok {
		return nil
	}
	cfg.DefaultProfile = alias
	return Write(cfg)
}
