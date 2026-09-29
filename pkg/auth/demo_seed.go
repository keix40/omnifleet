package auth

import "golang.org/x/crypto/bcrypt"

const DefaultSeedDemoPassword = "demo-password-change-me"

// HashSeedDemoPassword returns a bcrypt hash for demo tenant seed users.
func HashSeedDemoPassword(password string) (string, error) {
	if password == "" {
		password = DefaultSeedDemoPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
