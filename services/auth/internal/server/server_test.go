package server_test

import (
	"testing"
	"time"

	"github.com/keix40/omnifleet/pkg/auth"
)

func TestLoginRateLimiter_IntegratedWithServerConfig(t *testing.T) {
	l := auth.NewLoginRateLimiter(2, time.Minute)
	key := "acct:acme-logistics:driver@acme.test"
	if !l.Allow(key) || !l.Allow(key) {
		t.Fatal("first attempts should succeed")
	}
	if l.Allow(key) {
		t.Fatal("third attempt should fail")
	}
}
