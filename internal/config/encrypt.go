package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const keyFileName = "sentinel.key"

// ErrNoEncryptionKey indicates that no encryption key is available
var ErrNoEncryptionKey = errors.New("no encryption key found; run --wizard or create one manually")

// GenerateEncryptionKey creates a new 256-bit AES key and writes it to disk
func GenerateEncryptionKey(configDir string) ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	keyPath := filepath.Join(configDir, keyFileName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config dir: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(key)), 0600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	return key, nil
}

// LoadEncryptionKey reads the AES key from disk
func LoadEncryptionKey(configDir string) ([]byte, error) {
	keyPath := filepath.Join(configDir, keyFileName)
	encoded, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoEncryptionKey
		}
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	return base64.StdEncoding.DecodeString(string(encoded))
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a base64-encoded ciphertext
func Encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes a base64-encoded ciphertext and decrypts it using AES-256-GCM
func Decrypt(encoded string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
