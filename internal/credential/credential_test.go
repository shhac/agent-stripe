package credential

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shhac/agent-stripe/internal/config"
)

type fakeKeychain struct {
	values  map[string]string
	deleted []string
	err     error
}

func (f *fakeKeychain) Store(name, apiKey string) error {
	if f.err != nil {
		return f.err
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[name] = apiKey
	return nil
}

func (f *fakeKeychain) Get(name string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	value, ok := f.values[name]
	if !ok {
		return "", errors.New("not found")
	}
	return value, nil
}

func (f *fakeKeychain) Delete(name string) {
	f.deleted = append(f.deleted, name)
	delete(f.values, name)
}

func withCredentialTestDir(t *testing.T, backend keychainBackend) string {
	t.Helper()
	dir := t.TempDir()
	config.SetConfigDir(dir)
	t.Cleanup(func() { config.SetConfigDir("") })
	restore := setKeychainBackendForTest(backend)
	t.Cleanup(restore)
	return dir
}

func TestStoreGetListRemove(t *testing.T) {
	fake := &fakeKeychain{}
	dir := withCredentialTestDir(t, fake)

	storage, err := Store("prod", "sk_test_secret")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if storage != "keychain" {
		t.Fatalf("storage = %q", storage)
	}

	indexPath := filepath.Join(dir, "credentials.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(data), "sk_test_secret") {
		t.Fatalf("credentials index leaked API key: %s", data)
	}
	info, err := os.Stat(indexPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %v, want 0600", got)
	}

	got, err := Get("prod")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "sk_test_secret" {
		t.Fatalf("Get() = %q", got)
	}

	names, err := List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(names) != 1 || names[0] != "prod" {
		t.Fatalf("List() = %v", names)
	}

	if err := Remove("prod"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if len(fake.deleted) != 1 || fake.deleted[0] != "prod" {
		t.Fatalf("deleted = %v", fake.deleted)
	}
	if _, err := Get("prod"); err == nil {
		t.Fatalf("Get() after remove should fail")
	}
}

func TestStoreReturnsKeychainErrorWithoutWritingIndex(t *testing.T) {
	dir := withCredentialTestDir(t, &fakeKeychain{err: errors.New("keychain unavailable")})

	if _, err := Store("prod", "sk_test_secret"); err == nil {
		t.Fatalf("Store() should fail")
	}
	if _, err := os.Stat(filepath.Join(dir, "credentials.json")); !os.IsNotExist(err) {
		t.Fatalf("credentials index should not exist, stat err = %v", err)
	}
}

func TestGetRejectsNonKeychainManagedEntry(t *testing.T) {
	dir := withCredentialTestDir(t, &fakeKeychain{})
	err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte(`{"prod":{"keychain_managed":false}}`), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := Get("prod"); err == nil || !strings.Contains(err.Error(), "not keychain managed") {
		t.Fatalf("Get() error = %v, want not keychain managed", err)
	}
}
