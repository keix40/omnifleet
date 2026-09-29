package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const MinJWTSecretLength = 32

// DevFallbackJWTSecret is only used when OMNIFLEET_DEV_MODE=1 or E2E_COMPOSE=1 and JWT_SECRET is unset.
const DevFallbackJWTSecret = "dev-only-change-me-use-32-chars-min!!"

// JWTSettings holds validated JWT signing and verification parameters.
type JWTSettings struct {
	Secret   []byte
	Issuer   string
	Audience string
}

// DevModeEnabled returns true for local docker-compose and E2E runs.
// It does not change CORS or HTTP routes — only allows a dev JWT fallback when JWT_SECRET is unset.
func DevModeEnabled() bool {
	return os.Getenv("OMNIFLEET_DEV_MODE") == "1" || os.Getenv("E2E_COMPOSE") == "1"
}

// LoadJWTSettingsFromEnv loads JWT configuration and fails closed outside dev mode.
func LoadJWTSettingsFromEnv() (JWTSettings, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		if DevModeEnabled() {
			secret = DevFallbackJWTSecret
		} else {
			return JWTSettings{}, errors.New("JWT_SECRET must be set")
		}
	}
	if len(secret) < MinJWTSecretLength {
		return JWTSettings{}, fmt.Errorf("JWT_SECRET must be at least %d bytes", MinJWTSecretLength)
	}
	issuer := strings.TrimSpace(os.Getenv("JWT_ISSUER"))
	if issuer == "" {
		issuer = "omnifleet"
	}
	audience := strings.TrimSpace(os.Getenv("JWT_AUDIENCE"))
	if audience == "" {
		audience = "omnifleet-api"
	}
	return JWTSettings{
		Secret:   []byte(secret),
		Issuer:   issuer,
		Audience: audience,
	}, nil
}
