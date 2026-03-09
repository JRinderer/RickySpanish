package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// generateKey creates a new random 32-byte AES-256 key and returns it as a hex string.
func generateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("generating random key: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// hexToKey decodes a hex-encoded key and returns the raw 32-byte key.
// If the hex string is not exactly 32 bytes (64 hex chars), it derives a
// 32-byte key via SHA-256 so that human-chosen passwords still work.
func hexToKey(hexKey string) ([]byte, error) {
	raw, err := hex.DecodeString(hexKey)
	if err != nil {
		// Treat as a passphrase — derive a key via SHA-256
		h := sha256.Sum256([]byte(hexKey))
		return h[:], nil
	}
	if len(raw) == 32 {
		return raw, nil
	}
	// Wrong length hex — derive via SHA-256
	h := sha256.Sum256(raw)
	return h[:], nil
}

// encrypt encrypts plaintext using AES-256-GCM.
// The output format is: [12-byte nonce][ciphertext+tag].
func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data produced by encrypt.
func decrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting data (wrong key?): %w", err)
	}
	return plaintext, nil
}
