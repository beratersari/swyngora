package deliststore

import (
	"sort"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process SpotDelistStore.
type Memory struct {
	mu   sync.RWMutex
	// byExchange[exchange][SYMBOL] = delist time
	byExchange map[domain.Exchange]map[string]time.Time
}

// NewMemory constructs an empty store.
func NewMemory() *Memory {
	return &Memory{
		byExchange: make(map[domain.Exchange]map[string]time.Time),
	}
}

// ReplaceAll implements domain.SpotDelistStore.
func (m *Memory) ReplaceAll(exchange domain.Exchange, entries []domain.SpotDelistEntry) {
	next := make(map[string]time.Time, len(entries))
	for _, e := range entries {
		sym := strings.ToUpper(strings.TrimSpace(e.Symbol))
		if sym == "" || e.DelistTime.IsZero() {
			continue
		}
		// Keep earliest delist time if duplicates.
		if prev, ok := next[sym]; ok && !prev.IsZero() && prev.Before(e.DelistTime) {
			continue
		}
		next[sym] = e.DelistTime.UTC()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.byExchange == nil {
		m.byExchange = make(map[domain.Exchange]map[string]time.Time)
	}
	m.byExchange[exchange] = next
}

// DelistTime implements domain.SpotDelistStore.
func (m *Memory) DelistTime(exchange domain.Exchange, symbol string) (time.Time, bool) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return time.Time{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	bySym, ok := m.byExchange[exchange]
	if !ok || bySym == nil {
		return time.Time{}, false
	}
	t, ok := bySym[sym]
	return t, ok
}

// List implements domain.SpotDelistStore.
func (m *Memory) List(exchange domain.Exchange) []domain.SpotDelistEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bySym, ok := m.byExchange[exchange]
	if !ok || len(bySym) == 0 {
		return nil
	}
	out := make([]domain.SpotDelistEntry, 0, len(bySym))
	for sym, t := range bySym {
		out = append(out, domain.SpotDelistEntry{
			Exchange:   exchange,
			Symbol:     sym,
			DelistTime: t,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DelistTime.Equal(out[j].DelistTime) {
			return out[i].Symbol < out[j].Symbol
		}
		return out[i].DelistTime.Before(out[j].DelistTime)
	})
	return out
}
