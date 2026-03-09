//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

const (
	keychainService = "rickspanish"
	keychainAccount = "encryption-key"
)

// getOrCreateEncryptionKey retrieves the encryption key from macOS Keychain,
// creating and storing a new one if it doesn't exist yet.
func getOrCreateEncryptionKey() ([]byte, error) {
	hexKey, err := keychainGet()
	if err != nil {
		// Key not found — generate and store a new one
		hexKey, err = generateKey()
		if err != nil {
			return nil, err
		}
		if err := keychainSet(hexKey); err != nil {
			return nil, fmt.Errorf("storing key in Keychain: %w", err)
		}
		fmt.Println("New encryption key generated and stored in macOS Keychain.")
	}
	return hexToKey(hexKey)
}

func keychainGet() (string, error) {
	out, err := exec.Command(
		"security", "find-generic-password",
		"-s", keychainService,
		"-a", keychainAccount,
		"-w",
	).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func keychainSet(hexKey string) error {
	// Try to add; if it already exists, update it.
	err := exec.Command(
		"security", "add-generic-password",
		"-s", keychainService,
		"-a", keychainAccount,
		"-w", hexKey,
	).Run()
	if err != nil {
		// Attempt to update in case the item already exists
		return exec.Command(
			"security", "add-generic-password",
			"-U",
			"-s", keychainService,
			"-a", keychainAccount,
			"-w", hexKey,
		).Run()
	}
	return nil
}
