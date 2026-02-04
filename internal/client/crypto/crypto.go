package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

const (
	saltFileName = "salt"
	saltSize     = 16
	keySize      = 32
	nonceSize    = 12
)

// ErrEmptyMasterPassword indicates missing master password.
var ErrEmptyMasterPassword = errors.New("master password is required")

// Crypto handles payload encryption/decryption on the client side.
// Crypto encrypts and decrypts payloads using a master password.
type Crypto struct {
	key []byte
}

// NewCrypto initializes a Crypto instance.
func NewCrypto(masterPassword, dataDir string) (*Crypto, error) {
	if masterPassword == "" {
		return nil, ErrEmptyMasterPassword
	}

	salt, err := loadOrCreateSalt(dataDir)
	if err != nil {
		return nil, err
	}

	key, err := scrypt.Key([]byte(masterPassword), salt, 32768, 8, 1, keySize)
	if err != nil {
		return nil, err
	}

	return &Crypto{key: key}, nil
}

// Encrypt encrypts plaintext bytes.
func (c *Crypto) Encrypt(plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plain, nil)
	out := append(nonce, ciphertext...)
	return out, nil
}

// Decrypt decrypts ciphertext bytes.
func (c *Crypto) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < nonceSize {
		return nil, errors.New("invalid ciphertext")
	}

	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plain, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}

	return plain, nil
}

func loadOrCreateSalt(dataDir string) ([]byte, error) {
	path := filepath.Join(dataDir, saltFileName)
	if b, err := os.ReadFile(path); err == nil {
		if len(b) == saltSize {
			return b, nil
		}
		return nil, errors.New("invalid salt file")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, salt, 0o600); err != nil {
		return nil, err
	}

	return salt, nil
}
