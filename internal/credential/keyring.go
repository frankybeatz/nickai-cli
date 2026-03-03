package credential

import (
	"sync"

	"github.com/zalando/go-keyring"
)

const keyringService = "nickai-cli"

// keyringAvailable caches whether the OS keyring is usable.
// It is determined once on first access so we don't repeatedly
// probe a broken backend on headless / CI environments.
var (
	keyringOnce      sync.Once
	keyringSupported bool
)

// probeKeyring writes and deletes a throwaway entry to decide
// whether the system keyring actually works (e.g. macOS Keychain,
// Linux Secret Service, Windows Credential Manager).
func probeKeyring() bool {
	const probeUser = "__nickai_probe__"
	const probeVal = "1"
	if err := keyring.Set(keyringService, probeUser, probeVal); err != nil {
		return false
	}
	// Clean up the probe entry; ignore errors.
	_ = keyring.Delete(keyringService, probeUser)
	return true
}

// KeyringAvailable returns true if the OS keyring can be used.
// The result is cached after the first call.
func KeyringAvailable() bool {
	keyringOnce.Do(func() {
		keyringSupported = probeKeyring()
	})
	return keyringSupported
}

// KeyringGet retrieves a secret from the OS keyring.
// Returns ("", false) if the keyring is unavailable or the key is not found.
func KeyringGet(key string) (string, bool) {
	if !KeyringAvailable() {
		return "", false
	}
	val, err := keyring.Get(keyringService, key)
	if err != nil {
		return "", false
	}
	return val, true
}

// KeyringSet stores a secret in the OS keyring.
// Returns false if the keyring is unavailable.
func KeyringSet(key, value string) bool {
	if !KeyringAvailable() {
		return false
	}
	return keyring.Set(keyringService, key, value) == nil
}

// KeyringDelete removes a secret from the OS keyring.
// Returns false if the keyring is unavailable.
func KeyringDelete(key string) bool {
	if !KeyringAvailable() {
		return false
	}
	return keyring.Delete(keyringService, key) == nil
}
