package auth

import (
	"sync"
	"time"
)

// LoginRateLimiter enforces per-key request limits (IP or account identifier).
type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (l *LoginRateLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	history := l.attempts[key]
	var kept []time.Time
	for _, ts := range history {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= l.limit {
		l.attempts[key] = kept
		return false
	}
	kept = append(kept, now)
	l.attempts[key] = kept
	return true
}
