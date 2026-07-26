package watchliststore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// DefaultMaxClients bounds distinct client IDs in the in-memory store.
const DefaultMaxClients = 10_000

// Memory is an in-process watchlist store (no auth; keyed by client id).
type Memory struct {
	mu         sync.RWMutex
	data       map[string]*domain.Watchlist
	maxClients int
}

// NewMemory constructs an empty store with DefaultMaxClients.
func NewMemory() *Memory {
	return NewMemoryWithMaxClients(DefaultMaxClients)
}

// NewMemoryWithMaxClients constructs a store with an explicit client cap (0 = unlimited; tests only).
func NewMemoryWithMaxClients(maxClients int) *Memory {
	return &Memory{data: map[string]*domain.Watchlist{}, maxClients: maxClients}
}

// Get returns a copy of the watchlist (empty items if unknown client).
func (m *Memory) Get(_ context.Context, clientID string) (*domain.Watchlist, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if wl, ok := m.data[clientID]; ok {
		return cloneWL(wl), nil
	}
	return &domain.Watchlist{
		ClientID: clientID,
		Items:    []domain.WatchlistItem{},
		Updated:  time.Now().UTC(),
	}, nil
}

// Set replaces the list.
func (m *Memory) Set(_ context.Context, clientID string, items []domain.WatchlistItem) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureClientLocked(clientID); err != nil {
		return nil, err
	}
	wl := &domain.Watchlist{
		ClientID: clientID,
		Items:    append([]domain.WatchlistItem(nil), items...),
		Updated:  time.Now().UTC(),
	}
	m.data[clientID] = wl
	return cloneWL(wl), nil
}

// Add upserts one item.
func (m *Memory) Add(ctx context.Context, clientID string, item domain.WatchlistItem) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureClientLocked(clientID); err != nil {
		return nil, err
	}
	wl := m.data[clientID]
	if wl == nil {
		wl = &domain.Watchlist{ClientID: clientID, Items: nil}
	}
	found := false
	items := append([]domain.WatchlistItem(nil), wl.Items...)
	for i, it := range items {
		if it.Exchange == item.Exchange && it.Symbol == item.Symbol {
			if item.AddedAt.IsZero() {
				item.AddedAt = it.AddedAt
			}
			items[i] = item
			found = true
			break
		}
	}
	if !found {
		if item.AddedAt.IsZero() {
			item.AddedAt = time.Now().UTC()
		}
		items = append(items, item)
	}
	out := &domain.Watchlist{ClientID: clientID, Items: items, Updated: time.Now().UTC()}
	m.data[clientID] = out
	return cloneWL(out), nil
}

// ensureClientLocked rejects new client IDs when at capacity. Caller holds write lock.
func (m *Memory) ensureClientLocked(clientID string) error {
	if _, ok := m.data[clientID]; ok {
		return nil
	}
	if m.maxClients > 0 && len(m.data) >= m.maxClients {
		return fmt.Errorf("%w: watchlist client capacity reached", domain.ErrInvalidArgument)
	}
	return nil
}

// Remove deletes one item.
func (m *Memory) Remove(_ context.Context, clientID string, exchange domain.Exchange, symbol string) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wl := m.data[clientID]
	if wl == nil {
		return &domain.Watchlist{ClientID: clientID, Items: []domain.WatchlistItem{}, Updated: time.Now().UTC()}, nil
	}
	next := make([]domain.WatchlistItem, 0, len(wl.Items))
	for _, it := range wl.Items {
		if it.Exchange == exchange && it.Symbol == symbol {
			continue
		}
		next = append(next, it)
	}
	out := &domain.Watchlist{ClientID: clientID, Items: next, Updated: time.Now().UTC()}
	m.data[clientID] = out
	return cloneWL(out), nil
}

func cloneWL(wl *domain.Watchlist) *domain.Watchlist {
	if wl == nil {
		return nil
	}
	cp := *wl
	cp.Items = append([]domain.WatchlistItem(nil), wl.Items...)
	return &cp
}
