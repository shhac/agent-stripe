package config

import (
	"path/filepath"
	"testing"
)

func TestStoreProfileDefaultsAPIVersionAndDefault(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(func() { SetConfigDir("") })

	if err := StoreProfile("sandbox", Profile{Context: "acct_123"}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}

	cfg := Read()
	if cfg.DefaultProfile != "sandbox" {
		t.Fatalf("DefaultProfile = %q, want sandbox", cfg.DefaultProfile)
	}
	if got := cfg.Profiles["sandbox"].APIVersion; got != DefaultAPIVersion {
		t.Fatalf("APIVersion = %q, want %q", got, DefaultAPIVersion)
	}
	if got := ConfigDir(); got != filepath.Clean(dir) {
		t.Fatalf("ConfigDir() = %q, want %q", got, dir)
	}
}

func TestRemoveProfilePromotesAnotherDefault(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(func() { SetConfigDir("") })

	if err := StoreProfile("one", Profile{}); err != nil {
		t.Fatalf("StoreProfile(one) error = %v", err)
	}
	if err := StoreProfile("two", Profile{}); err != nil {
		t.Fatalf("StoreProfile(two) error = %v", err)
	}
	if err := RemoveProfile("one"); err != nil {
		t.Fatalf("RemoveProfile() error = %v", err)
	}

	cfg := Read()
	if cfg.DefaultProfile == "one" {
		t.Fatalf("DefaultProfile still points at removed profile")
	}
	if cfg.DefaultProfile == "" {
		t.Fatalf("DefaultProfile should promote another profile")
	}
}

func TestUpdateProfileAndDefaults(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(func() { SetConfigDir("") })

	if err := StoreProfile("prod", Profile{Context: "acct_old"}); err != nil {
		t.Fatalf("StoreProfile() error = %v", err)
	}
	if err := UpdateProfile("prod", func(profile Profile) Profile {
		profile.Context = "acct_new"
		profile.APIVersion = "2025-06-30.basil"
		return profile
	}); err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if err := SetDefaultValue("max_retries", 4); err != nil {
		t.Fatalf("SetDefaultValue(max_retries) error = %v", err)
	}
	if err := SetDefaultValue("timeout_ms", 1500); err != nil {
		t.Fatalf("SetDefaultValue(timeout_ms) error = %v", err)
	}

	cfg := Read()
	profile := cfg.Profiles["prod"]
	if profile.Context != "acct_new" || profile.APIVersion != "2025-06-30.basil" {
		t.Fatalf("profile = %+v", profile)
	}
	if cfg.Defaults.MaxRetries == nil || *cfg.Defaults.MaxRetries != 4 {
		t.Fatalf("MaxRetries = %#v, want 4", cfg.Defaults.MaxRetries)
	}
	if cfg.Defaults.TimeoutMS == nil || *cfg.Defaults.TimeoutMS != 1500 {
		t.Fatalf("TimeoutMS = %#v, want 1500", cfg.Defaults.TimeoutMS)
	}

	if err := UnsetDefaultValue("max_retries"); err != nil {
		t.Fatalf("UnsetDefaultValue(max_retries) error = %v", err)
	}
	if got := Read().Defaults.MaxRetries; got != nil {
		t.Fatalf("MaxRetries = %#v, want nil", *got)
	}
}

func TestConfigMissingProfileErrors(t *testing.T) {
	dir := t.TempDir()
	SetConfigDir(dir)
	t.Cleanup(func() { SetConfigDir("") })

	if err := SetDefault("missing"); err == nil {
		t.Fatalf("SetDefault() should fail for a missing profile")
	}
	if err := UpdateProfile("missing", func(profile Profile) Profile { return profile }); err == nil {
		t.Fatalf("UpdateProfile() should fail for a missing profile")
	}
}
