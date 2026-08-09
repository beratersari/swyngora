package telegram

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	pendingTradeTTL   = 5 * time.Minute
	maxPendingTrades  = 64
	pendingIDLen      = 16
)

type pendingTrade struct {
	ID         string
	ChatID     int64
	UserID     int64
	ClientID    string
	PortfolioID string
	Side        domain.TradeSide
	Exchange   string
	Symbol     string
	Quantity   float64
	QuotePrice float64
	Notional   float64
	ExpiresAt  time.Time
}

type pendingStore struct {
	mu   sync.Mutex
	byID map[string]*pendingTrade
}

func newPendingStore() *pendingStore {
	return &pendingStore{byID: map[string]*pendingTrade{}}
}

func newPendingID() string {
	s := strings.ReplaceAll(uuid.NewString(), "-", "")
	if len(s) > pendingIDLen {
		s = s[:pendingIDLen]
	}
	return s
}

func (s *pendingStore) put(p *pendingTrade) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeLocked(time.Now())
	if len(s.byID) >= maxPendingTrades {
		// Drop oldest.
		var oldestID string
		var oldest time.Time
		for id, t := range s.byID {
			if oldestID == "" || t.ExpiresAt.Before(oldest) {
				oldestID = id
				oldest = t.ExpiresAt
			}
		}
		delete(s.byID, oldestID)
	}
	s.byID[p.ID] = p
}

func (s *pendingStore) get(id string) *pendingTrade {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.byID[id]
	if p == nil {
		return nil
	}
	if time.Now().After(p.ExpiresAt) {
		delete(s.byID, id)
		return nil
	}
	cp := *p
	return &cp
}

// take removes and returns the pending trade if still valid.
func (s *pendingStore) take(id string) *pendingTrade {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.byID[id]
	if p == nil {
		return nil
	}
	delete(s.byID, id)
	if time.Now().After(p.ExpiresAt) {
		return nil
	}
	return p
}

func (s *pendingStore) restore(p *pendingTrade) {
	if p == nil || time.Now().After(p.ExpiresAt) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[p.ID] = p
}

func (s *pendingStore) purgeLocked(now time.Time) {
	for id, p := range s.byID {
		if now.After(p.ExpiresAt) {
			delete(s.byID, id)
		}
	}
}
