package auth_test

import (
	"testing"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestLoadJWTSettings_FailsWithoutSecretInProduction(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("OMNIFLEET_DEV_MODE", "")
	t.Setenv("E2E_COMPOSE", "")
	_, err := auth.LoadJWTSettingsFromEnv()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET unset outside dev mode")
	}
}

func TestLoadJWTSettings_RejectsShortSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "too-short")
	t.Setenv("OMNIFLEET_DEV_MODE", "")
	_, err := auth.LoadJWTSettingsFromEnv()
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestLoadJWTSettings_DevModeFallback(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("OMNIFLEET_DEV_MODE", "1")
	cfg, err := auth.LoadJWTSettingsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Secret) < auth.MinJWTSecretLength {
		t.Fatalf("dev fallback secret too short: %d", len(cfg.Secret))
	}
	if cfg.Issuer != "omnifleet" || cfg.Audience != "omnifleet-api" {
		t.Fatalf("unexpected issuer/audience: %+v", cfg)
	}
}

func TestLoadJWTSettings_CustomIssuerAudience(t *testing.T) {
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("JWT_ISSUER", "issuer-x")
	t.Setenv("JWT_AUDIENCE", "aud-y")
	cfg, err := auth.LoadJWTSettingsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Issuer != "issuer-x" || cfg.Audience != "aud-y" {
		t.Fatalf("unexpected: %+v", cfg)
	}
}
