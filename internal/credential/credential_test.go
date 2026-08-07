package credential

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/shhac/agent-stripe/internal/config"
)

// fakeKeychain stands in for the OS keychain, which is safe for concurrent
// use — so this must be too, or the concurrency tests below report a race in
// the double rather than exercising the code under test.
type fakeKeychain struct {
	mu        sync.Mutex
	values    map[string]string
	deleted   []string
	err       error
	deleteErr error
}

func (f *fakeKeychain) Store(name, apiKey string) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
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
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.values[name]
	if !ok {
		return "", errors.New("not found")
	}
	return value, nil
}

func (f *fakeKeychain) Delete(name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, name)
	delete(f.values, name)
	return nil
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

func TestStoreFallsBackToFileWhenKeychainUnavailable(t *testing.T) {
	dir := withCredentialTestDir(t, &fakeKeychain{err: errors.New("keychain unavailable")})

	storage, err := Store("prod", "sk_test_secret")
	if err != nil {
		t.Fatalf("Store() error = %v", err)
	}
	if storage != "file" {
		t.Fatalf("storage = %q, want file", storage)
	}
	data, err := os.ReadFile(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "sk_test_secret") {
		t.Fatalf("credentials index should hold the raw key under keychain failure: %s", data)
	}
	if strings.Contains(string(data), keychainSentinel) {
		t.Fatalf("credentials index should not hold the sentinel under keychain failure: %s", data)
	}

	got, err := Get("prod")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "sk_test_secret" {
		t.Fatalf("Get() = %q, want sk_test_secret", got)
	}
}

func TestRemoveReturnsDeleteErrorWithoutRemovingIndex(t *testing.T) {
	fake := &fakeKeychain{deleteErr: errors.New("keychain delete failed")}
	dir := withCredentialTestDir(t, fake)
	if err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte(`{"prod":{"keychain_managed":true}}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := Remove("prod"); err == nil || !strings.Contains(err.Error(), "keychain delete failed") {
		t.Fatalf("Remove() error = %v, want keychain delete failed", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "prod") {
		t.Fatalf("credentials index removed prod after delete failure: %s", data)
	}
}

func TestGetReturnsFileManagedSecret(t *testing.T) {
	dir := withCredentialTestDir(t, &fakeKeychain{})
	err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte(`{"prod":{"api_key":"sk_file_secret","keychain_managed":false}}`), 0o600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := Get("prod")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "sk_file_secret" {
		t.Fatalf("Get() = %q, want sk_file_secret", got)
	}
}

// TestStore_Headless_FileFallback exercises the credential-WRITE path
// non-interactively. The per-CLI keychain opt-out (derived by lib-agent-cli from
// the "app.paulie.agent-stripe" service) makes the keychain report unavailable,
// so Store deterministically keeps the raw key in the 0600 index file on every
// platform — including darwin, where it would otherwise reach the `security` GUI
// prompt. Before the file fallback existed, Store simply failed under the opt-out
// (and on any non-macOS host).
func TestStore_Headless_FileFallback(t *testing.T) {
	t.Setenv("AGENT_STRIPE_NO_KEYCHAIN", "1")
	dir := t.TempDir()
	config.SetConfigDir(dir)
	t.Cleanup(func() { config.SetConfigDir("") })

	storage, err := Store("headless", "sk_test_headless")
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if storage != "file" {
		t.Fatalf("storage=%q, want \"file\" (keychain opt-out should force the file path)", storage)
	}

	path := filepath.Join(dir, "credentials.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("index not written: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("index mode=%o, want 0600", mode)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "sk_test_headless") {
		t.Errorf("file should contain the raw key under opt-out; got %s", data)
	}
	if strings.Contains(string(data), keychainSentinel) {
		t.Errorf("file should NOT contain the keychain sentinel under opt-out; got %s", data)
	}

	got, err := Get("headless")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sk_test_headless" {
		t.Errorf("Get=%q, want sk_test_headless", got)
	}

	if err := Remove("headless"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := Get("headless"); err == nil {
		t.Error("expected NotFound after Remove")
	}
}

// Concurrent stores must not lose each other's entries.
//
// This is the failure that matters most for this index: the keychain write
// has already succeeded by the time the index is written, so an entry lost
// to a racing writer leaves a live Stripe key in the OS keychain that
// nothing references — invisible to `auth list` and unreachable by `auth
// remove`, which looks the name up in the index first. Before this went
// through creds.Store.Update, twenty concurrent writers left a single
// surviving entry.
func TestConcurrentStoresDoNotLoseEntries(t *testing.T) {
	withCredentialTestDir(t, &fakeKeychain{})

	const writers = 20
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := Store(fmt.Sprintf("profile-%02d", i), fmt.Sprintf("sk_test_%02d", i)); err != nil {
				t.Errorf("Store: %v", err)
			}
		}(i)
	}
	wg.Wait()

	for i := range writers {
		name := fmt.Sprintf("profile-%02d", i)
		got, err := Get(name)
		if err != nil {
			t.Errorf("%s was lost from the index — its keychain secret is now orphaned: %v", name, err)
			continue
		}
		if want := fmt.Sprintf("sk_test_%02d", i); got != want {
			t.Errorf("%s round-tripped as %q, want %q", name, got, want)
		}
	}
}
