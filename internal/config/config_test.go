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
