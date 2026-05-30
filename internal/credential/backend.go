package credential

type keychainBackend interface {
	Store(name, apiKey string) error
	Get(name string) (string, error)
	Delete(name string)
}

var keychain keychainBackend = securityKeychain{}

func setKeychainBackendForTest(backend keychainBackend) func() {
	previous := keychain
	keychain = backend
	return func() { keychain = previous }
}
