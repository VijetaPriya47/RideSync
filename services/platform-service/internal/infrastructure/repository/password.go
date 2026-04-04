package repository

import "golang.org/x/crypto/bcrypt"

func hashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// ComparePasswordHash checks a bcrypt hash against a plaintext password.
func ComparePasswordHash(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
