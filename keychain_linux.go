//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	secretToolService = "rickspanish"
	secretToolAttr    = "application"
	secretToolLabel   = "rickspanish-encryption-key"
)

// getOrCreateEncryptionKey retrieves the encryption key using the best available
// method on Linux:
//  1. GNOME Keyring via secret-tool (if available)
//  2. RICKSPANISH_ENCRYPTION_KEY environment variable (fallback)
func getOrCreateEncryptionKey() ([]byte, error) {
	// Try secret-tool first
	if secretToolAvailable() {
		hexKey, err := secretToolGet()
		if err != nil {
			// Not found — generate and store
			hexKey, err = generateKey()
			if err != nil {
				return nil, err
			}
			if err := secretToolSet(hexKey); err != nil {
				return nil, fmt.Errorf("storing key via secret-tool: %w", err)
			}
			fmt.Println("New encryption key generated and stored in GNOME Keyring.")
		}
		return hexToKey(hexKey)
	}

	// Fallback: environment variable
	hexKey := os.Getenv("RICKSPANISH_ENCRYPTION_KEY")
	if hexKey == "" {
		return nil, fmt.Errorf(
			"no keyring available and RICKSPANISH_ENCRYPTION_KEY is not set.\n" +
				"Install libsecret-tools (apt install libsecret-tools) for keyring support,\n" +
				"or export RICKSPANISH_ENCRYPTION_KEY=<your-64-char-hex-key>.\n" +
				"Generate a key with: openssl rand -hex 32",
		)
	}
	return hexToKey(hexKey)
}

func secretToolAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

func secretToolGet() (string, error) {
	out, err := exec.Command(
		"secret-tool", "lookup",
		secretToolAttr, secretToolService,
	).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func secretToolSet(hexKey string) error {
	cmd := exec.Command(
		"secret-tool", "store",
		"--label", secretToolLabel,
		secretToolAttr, secretToolService,
	)
	cmd.Stdin = strings.NewReader(hexKey)
	return cmd.Run()
}
