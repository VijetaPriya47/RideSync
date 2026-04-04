package repo

import "golang.org/x/crypto/bcrypt"

func hashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, pw string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
}

// VerifyPassword compares a bcrypt hash with a plaintext password.
func VerifyPassword(hash, pw string) error {
	return checkPassword(hash, pw)
}
