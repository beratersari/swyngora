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
	mu sync.RWMutex
	// byExchange[exchange][SYMBOL] = full entry (halt + announcement)
	byExchange map[domain.Exchange]map[string]domain.SpotDelistEntry
}

// NewMemory constructs an empty store.
func NewMemory() *Memory {
	return &Memory{
		byExchange: make(map[domain.Exchange]map[string]domain.SpotDelistEntry),
	}
}

// ReplaceAll implements domain.SpotDelistStore.
func (m *Memory) ReplaceAll(exchange domain.Exchange, entries []domain.SpotDelistEntry) {
	next := make(map[string]domain.SpotDelistEntry, len(entries))
	for _, e := range entries {
		sym := strings.ToUpper(strings.TrimSpace(e.Symbol))
		if sym == "" || e.DelistTime.IsZero() {
			continue
		}
		cp := e
		cp.Symbol = sym
		cp.Exchange = exchange
		cp.DelistTime = e.DelistTime.UTC()
		if !e.AnnouncedAt.IsZero() {
			cp.AnnouncedAt = e.AnnouncedAt.UTC()
		}
		if prev, ok := next[sym]; ok {
			// Keep earliest halt; fill announcement from the other row if missing.
			if !prev.DelistTime.After(cp.DelistTime) {
				if prev.AnnouncedAt.IsZero() && !cp.AnnouncedAt.IsZero() {
					prev.AnnouncedAt = cp.AnnouncedAt
					next[sym] = prev
				}
				continue
			}
			if cp.AnnouncedAt.IsZero() {
				cp.AnnouncedAt = prev.AnnouncedAt
			}
		}
		next[sym] = cp
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.byExchange == nil {
		m.byExchange = make(map[domain.Exchange]map[string]domain.SpotDelistEntry)
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
	e, ok := bySym[sym]
	if !ok {
		return time.Time{}, false
	}
	return e.DelistTime, true
}

// Get implements domain.SpotDelistStore.
func (m *Memory) Get(exchange domain.Exchange, symbol string) (domain.SpotDelistEntry, bool) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	if sym == "" {
		return domain.SpotDelistEntry{}, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	bySym, ok := m.byExchange[exchange]
	if !ok || bySym == nil {
		return domain.SpotDelistEntry{}, false
	}
	e, ok := bySym[sym]
	return e, ok
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
	for _, e := range bySym {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DelistTime.Equal(out[j].DelistTime) {
			return out[i].Symbol < out[j].Symbol
		}
		return out[i].DelistTime.Before(out[j].DelistTime)
	})
	return out
}
