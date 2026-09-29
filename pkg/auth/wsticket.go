package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

// SingleUseStore records one-time ticket identifiers (shared across gateway replicas).
type SingleUseStore interface {
	RedeemOnce(jti string, expiresAt time.Time) (bool, error)
}

// MemorySingleUseStore is an in-process store for tests and single-replica dev.
type MemorySingleUseStore struct {
	mu    sync.Mutex
	used  map[string]struct{}
}

func NewMemorySingleUseStore() *MemorySingleUseStore {
	return &MemorySingleUseStore{used: make(map[string]struct{})}
}

func (m *MemorySingleUseStore) RedeemOnce(jti string, expiresAt time.Time) (bool, error) {
	if time.Now().After(expiresAt) {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.used[jti]; ok {
		return false, nil
	}
	m.used[jti] = struct{}{}
	return true, nil
}

type signedTicketPayload struct {
	JTI      string `json:"jti"`
	UserID   string `json:"uid"`
	TenantID string `json:"tid"`
	Email    string `json:"email"`
	Role     Role   `json:"role"`
	Exp      int64  `json:"exp"`
}

// WSTicketStore issues HMAC-signed, single-use WebSocket tickets.
type WSTicketStore struct {
	macKey []byte
	ttl    time.Duration
	store  SingleUseStore
}

func NewWSTicketStore(macKey []byte, ttl time.Duration, store SingleUseStore) *WSTicketStore {
	if store == nil {
		store = NewMemorySingleUseStore()
	}
	return &WSTicketStore{macKey: macKey, ttl: ttl, store: store}
}

func (s *WSTicketStore) Issue(claims *Claims) (string, time.Time, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	jti := hex.EncodeToString(raw)
	exp := time.Now().Add(s.ttl)
	payload := signedTicketPayload{
		JTI:      jti,
		UserID:   claims.UserID,
		TenantID: claims.TenantID,
		Email:    claims.Email,
		Role:     claims.Role,
		Exp:      exp.Unix(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", time.Time{}, err
	}
	bodyB64 := base64.RawURLEncoding.EncodeToString(body)
	sig := signTicket(s.macKey, bodyB64)
	return bodyB64 + "." + sig, exp, nil
}

func (s *WSTicketStore) Redeem(ticket string) (*Claims, bool) {
	bodyB64, sig, ok := strings.Cut(ticket, ".")
	if !ok || sig == "" {
		return nil, false
	}
	if !hmac.Equal([]byte(sig), []byte(signTicket(s.macKey, bodyB64))) {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(bodyB64)
	if err != nil {
		return nil, false
	}
	var payload signedTicketPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false
	}
	exp := time.Unix(payload.Exp, 0)
	redeemed, err := s.store.RedeemOnce(payload.JTI, exp)
	if err != nil || !redeemed {
		return nil, false
	}
	return &Claims{
		UserID:   payload.UserID,
		TenantID: payload.TenantID,
		Email:    payload.Email,
		Role:     payload.Role,
	}, true
}

func signTicket(macKey []byte, bodyB64 string) string {
	mac := hmac.New(sha256.New, macKey)
	_, _ = mac.Write([]byte(bodyB64))
	return hex.EncodeToString(mac.Sum(nil))
}

var ErrTicketStore = errors.New("ticket store error")
