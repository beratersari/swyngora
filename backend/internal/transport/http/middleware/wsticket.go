package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// WSTicketTTL is how long a minted WebSocket ticket remains valid.
const WSTicketTTL = 60 * time.Second

type wsTicket struct {
	id  AuthIdentity
	exp time.Time
}

// WSTicketIssuer mints one-time, short-lived tickets so browsers never put
// the long-lived API secret on the WebSocket URL.
type WSTicketIssuer struct {
	mu   sync.Mutex
	byID map[string]wsTicket
}

// NewWSTicketIssuer constructs an empty in-memory issuer.
func NewWSTicketIssuer() *WSTicketIssuer {
	return &WSTicketIssuer{byID: make(map[string]wsTicket)}
}

// Issue stores a one-time ticket bound to id. Token is 32 random bytes hex.
func (s *WSTicketIssuer) Issue(id AuthIdentity) (token string, exp time.Time, err error) {
	if s == nil {
		return "", time.Time{}, fmt.Errorf("%w: ticket issuer not configured", domain.ErrUpstream)
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", time.Time{}, fmt.Errorf("%w: ticket entropy", domain.ErrUpstream)
	}
	token = hex.EncodeToString(raw[:])
	exp = time.Now().UTC().Add(WSTicketTTL)
	s.mu.Lock()
	s.evictLocked(time.Now().UTC())
	s.byID[token] = wsTicket{id: id, exp: exp}
	s.mu.Unlock()
	return token, exp, nil
}

// Consume validates and deletes a ticket. Expired or unknown → ErrUnauthorized.
func (s *WSTicketIssuer) Consume(token string) (*AuthIdentity, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: ticket issuer not configured", domain.ErrUpstream)
	}
	token = trimTicket(token)
	if token == "" {
		return nil, fmt.Errorf("%w: ticket is required", domain.ErrForbidden)
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	got, ok := s.byID[token]
	if !ok {
		return nil, fmt.Errorf("%w: invalid or used ticket", domain.ErrForbidden)
	}
	delete(s.byID, token)
	if now.After(got.exp) {
		return nil, fmt.Errorf("%w: ticket expired", domain.ErrForbidden)
	}
	id := got.id
	return &id, nil
}

func (s *WSTicketIssuer) evictLocked(now time.Time) {
	for k, v := range s.byID {
		if now.After(v.exp) {
			delete(s.byID, k)
		}
	}
}

func trimTicket(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}
