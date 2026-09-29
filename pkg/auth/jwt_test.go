package auth_test

import (
	"testing"
	"time"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestJWT_RoundTrip(t *testing.T) {
	issuer := auth.NewTokenIssuer("01234567890123456789012345678901", time.Hour, "test", "test-api")
	token, _, err := issuer.Issue("user-1", "tenant-1", "a@test.com", auth.RoleDriver)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := issuer.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TenantID != "tenant-1" || claims.Role != auth.RoleDriver {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestJWT_RejectsWrongAudience(t *testing.T) {
	issuer := auth.NewTokenIssuer("01234567890123456789012345678901", time.Hour, "test", "expected-aud")
	token, _, err := issuer.Issue("user-1", "tenant-1", "a@test.com", auth.RoleDriver)
	if err != nil {
		t.Fatal(err)
	}
	other := auth.NewTokenIssuer("01234567890123456789012345678901", time.Hour, "test", "other-aud")
	if _, err := other.Parse(token); err == nil {
		t.Fatal("expected audience mismatch error")
	}
}
