package httpapi

import (
	"os"
	"strings"
)

const defaultCORSOrigins = "http://localhost:3000,http://127.0.0.1:3000"

// CORSAllowedOriginsFromEnv returns exact-match browser origins (comma-separated).
// Wildcards are not supported when AllowCredentials is true.
func CORSAllowedOriginsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		raw = defaultCORSOrigins
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		o := strings.TrimSpace(part)
		if o == "" {
			continue
		}
		out = append(out, o)
	}
	if len(out) == 0 {
		return []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	}
	return out
}

func originAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return false
	}
	for _, a := range allowed {
		if origin == a {
			return true
		}
	}
	return false
}
