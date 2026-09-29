package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// WSTicketStore issues single-use, short-lived tickets for WebSocket upgrades (avoids JWT in URLs/logs).
type WSTicketStore struct {
	mu      sync.Mutex
	tickets map[string]wsTicket
	ttl     time.Duration
}

type wsTicket struct {
	claims    *Claims
	expiresAt time.Time
	used      bool
}

func NewWSTicketStore(ttl time.Duration) *WSTicketStore {
	s := &WSTicketStore{
		tickets: make(map[string]wsTicket),
		ttl:     ttl,
	}
	go s.reapLoop()
	return s
}

func (s *WSTicketStore) Issue(claims *Claims) (string, time.Time, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	id := hex.EncodeToString(raw)
	exp := time.Now().Add(s.ttl)
	cpy := *claims
	s.mu.Lock()
	s.tickets[id] = wsTicket{claims: &cpy, expiresAt: exp}
	s.mu.Unlock()
	return id, exp, nil
}

func (s *WSTicketStore) Redeem(ticketID string) (*Claims, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tickets[ticketID]
	if !ok || t.used || time.Now().After(t.expiresAt) {
		return nil, false
	}
	t.used = true
	s.tickets[ticketID] = t
	return t.claims, true
}

func (s *WSTicketStore) reapLoop() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, t := range s.tickets {
			if t.used || now.After(t.expiresAt) {
				delete(s.tickets, id)
			}
		}
		s.mu.Unlock()
	}
}
