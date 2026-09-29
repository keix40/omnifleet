package auth_test

import (
	"testing"
	"time"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestLoginRateLimiter_BlocksAfterLimit(t *testing.T) {
	l := auth.NewLoginRateLimiter(3, time.Minute)
	key := "192.168.0.1"
	for i := 0; i < 3; i++ {
		if !l.Allow(key) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if l.Allow(key) {
		t.Fatal("fourth attempt should be blocked")
	}
}

func TestLoginRateLimiter_IndependentKeys(t *testing.T) {
	l := auth.NewLoginRateLimiter(1, time.Minute)
	if !l.Allow("a") || !l.Allow("b") {
		t.Fatal("distinct keys should not share limits")
	}
}
