package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword hashes the password for storage.
func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

// ComparePassword compares a hash with a plaintext password.
func ComparePassword(hash []byte, password string) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(password))
}
