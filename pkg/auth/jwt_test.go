package auth_test

import (
	"testing"
	"time"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestJWT_RoundTrip(t *testing.T) {
	issuer := auth.NewTokenIssuer("test-secret", time.Hour, "test")
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
