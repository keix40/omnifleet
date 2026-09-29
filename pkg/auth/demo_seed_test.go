package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashSeedDemoPassword_Default(t *testing.T) {
	hash, err := HashSeedDemoPassword("")
	if err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(DefaultSeedDemoPassword)); err != nil {
		t.Fatalf("hash does not match default password: %v", err)
	}
}

func TestHashSeedDemoPassword_Custom(t *testing.T) {
	hash, err := HashSeedDemoPassword("custom-secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("custom-secret")); err != nil {
		t.Fatal(err)
	}
}
