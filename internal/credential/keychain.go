package credential

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const keychainService = "app.paulie.agent-stripe"

type securityKeychain struct{}

func (securityKeychain) Store(name, apiKey string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("keychain not available on %s", runtime.GOOS)
	}

	_ = exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", name).Run()

	return exec.Command("security", "add-generic-password",
		"-s", keychainService, "-a", name, "-w", apiKey,
		"-U",
	).Run()
}

func (securityKeychain) Get(name string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("keychain not available on %s", runtime.GOOS)
	}

	out, err := exec.Command("security", "find-generic-password",
		"-s", keychainService, "-a", name, "-w",
	).Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (securityKeychain) Delete(name string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	return exec.Command("security", "delete-generic-password", "-s", keychainService, "-a", name).Run()
}
