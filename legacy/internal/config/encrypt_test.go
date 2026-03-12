package config

import (
	"crypto/rand"
	"io"
	"testing"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := "super secret API key 12345"

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if encrypted == plaintext {
		t.Error("encrypted text should differ from plaintext")
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	encrypted, err := Encrypt("", key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != "" {
		t.Errorf("expected empty string, got %q", decrypted)
	}
}

func TestEncrypt_DifferentKeyLengths(t *testing.T) {
	tests := []struct {
		name    string
		keyLen  int
		wantErr bool
	}{
		{"16 byte key (AES-128)", 16, false},
		{"24 byte key (AES-192)", 24, false},
		{"32 byte key (AES-256)", 32, false},
		{"15 byte key (invalid)", 15, true},
		{"0 byte key (invalid)", 0, true},
		{"1 byte key (invalid)", 1, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, tc.keyLen)
			if tc.keyLen > 0 {
				if _, err := io.ReadFull(rand.Reader, key); err != nil {
					t.Fatalf("failed to generate key: %v", err)
				}
			}

			_, err := Encrypt("test plaintext", key)
			if tc.wantErr && err == nil {
				t.Error("expected error for invalid key length, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Try decrypting garbage data
	_, err := Decrypt("not-valid-base64!!!", key)
	if err == nil {
		t.Error("expected error for corrupted ciphertext, got nil")
	}

	// Try decrypting valid base64 but not valid ciphertext
	_, err = Decrypt("dGVzdGluZzEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ1Njc4OTA=", key)
	if err == nil {
		t.Error("expected error for invalid ciphertext, got nil")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	io.ReadFull(rand.Reader, key1)
	io.ReadFull(rand.Reader, key2)

	encrypted, err := Encrypt("secret", key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestDecrypt_TooShortCiphertext(t *testing.T) {
	key := make([]byte, 32)
	io.ReadFull(rand.Reader, key)

	// Base64 of just a few bytes — shorter than nonce
	_, err := Decrypt("YWI=", key)
	if err == nil {
		t.Error("expected error for ciphertext shorter than nonce")
	}
}

func TestGenerateEncryptionKey(t *testing.T) {
	tmpDir := t.TempDir()
	key, err := GenerateEncryptionKey(tmpDir)
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(key))
	}

	// Load the key back
	loaded, err := LoadEncryptionKey(tmpDir)
	if err != nil {
		t.Fatalf("LoadEncryptionKey failed: %v", err)
	}
	if len(loaded) != len(key) {
		t.Errorf("loaded key length %d != generated key length %d", len(loaded), len(key))
	}
	for i := range key {
		if key[i] != loaded[i] {
			t.Errorf("key mismatch at byte %d", i)
			break
		}
	}
}

func TestLoadEncryptionKey_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := LoadEncryptionKey(tmpDir)
	if err != ErrNoEncryptionKey {
		t.Errorf("expected ErrNoEncryptionKey, got %v", err)
	}
}
