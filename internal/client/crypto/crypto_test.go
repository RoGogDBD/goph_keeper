package crypto

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCryptoEmptyPassword(t *testing.T) {
	t.Parallel()
	if _, err := NewCrypto("", t.TempDir()); err == nil {
		t.Fatalf("expected error for empty password")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c, err := NewCrypto("secret", dir)
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}

	plain := []byte("hello")
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(dec, plain) {
		t.Fatalf("decrypt mismatch: got=%q want=%q", dec, plain)
	}

	saltPath := filepath.Join(dir, "salt")
	if _, err := os.Stat(saltPath); err != nil {
		t.Fatalf("salt file: %v", err)
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	t.Parallel()

	c, err := NewCrypto("secret", t.TempDir())
	if err != nil {
		t.Fatalf("NewCrypto: %v", err)
	}
	if _, err := c.Decrypt([]byte("short")); err == nil {
		t.Fatalf("expected error for invalid ciphertext")
	}
}
