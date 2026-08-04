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
// Enforces domain.MaxWatchlistItems under the write lock.
type Memory struct {
	mu         sync.RWMutex
	data       map[string]*domain.Watchlist
	shares     map[string]domain.WatchlistShare // owner|grantee
	audit      []domain.WatchlistAuditEvent
	maxClients int
	maxItems   int
}

// NewMemory constructs an empty store with DefaultMaxClients.
func NewMemory() *Memory {
	return NewMemoryWithMaxClients(DefaultMaxClients)
}

// NewMemoryWithMaxClients constructs a store with an explicit client cap (0 = unlimited; tests only).
func NewMemoryWithMaxClients(maxClients int) *Memory {
	return &Memory{
		data:       map[string]*domain.Watchlist{},
		shares:     map[string]domain.WatchlistShare{},
		audit:      nil,
		maxClients: maxClients,
		maxItems:   domain.MaxWatchlistItems,
	}
}

func shareKey(owner, grantee string) string { return owner + "|" + grantee }

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
		Version:  0,
	}, nil
}

func (m *Memory) currentVersionLocked(clientID string) int64 {
	if wl, ok := m.data[clientID]; ok {
		return wl.Version
	}
	return 0
}

func (m *Memory) checkVersionLocked(clientID string, expectedVersion int64) (*domain.Watchlist, error) {
	if expectedVersion < 0 {
		return nil, nil
	}
	cur := m.currentVersionLocked(clientID)
	if cur == expectedVersion {
		return nil, nil
	}
	if wl, ok := m.data[clientID]; ok {
		return nil, &domain.WatchlistVersionMismatch{Current: cloneWL(wl)}
	}
	return nil, &domain.WatchlistVersionMismatch{Current: &domain.Watchlist{
		ClientID: clientID, Items: []domain.WatchlistItem{}, Updated: time.Now().UTC(), Version: 0,
	}}
}

// Set replaces the list. Rejects when len(items) > maxItems.
func (m *Memory) Set(_ context.Context, clientID string, items []domain.WatchlistItem, expectedVersion int64) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureClientLocked(clientID); err != nil {
		return nil, err
	}
	if _, err := m.checkVersionLocked(clientID, expectedVersion); err != nil {
		return nil, err
	}
	if m.maxItems > 0 && len(items) > m.maxItems {
		return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, m.maxItems)
	}
	ver := m.currentVersionLocked(clientID) + 1
	wl := &domain.Watchlist{
		ClientID: clientID,
		Items:    append([]domain.WatchlistItem(nil), items...),
		Updated:  time.Now().UTC(),
		Version:  ver,
	}
	m.data[clientID] = wl
	return cloneWL(wl), nil
}

// Add upserts one item. Enforces maxItems under the same lock (no TOCTOU).
func (m *Memory) Add(ctx context.Context, clientID string, item domain.WatchlistItem, expectedVersion int64) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.ensureClientLocked(clientID); err != nil {
		return nil, err
	}
	if _, err := m.checkVersionLocked(clientID, expectedVersion); err != nil {
		return nil, err
	}
	wl := m.data[clientID]
	if wl == nil {
		wl = &domain.Watchlist{ClientID: clientID, Items: nil, Version: 0}
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
		if m.maxItems > 0 && len(items) >= m.maxItems {
			return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, m.maxItems)
		}
		if item.AddedAt.IsZero() {
			item.AddedAt = time.Now().UTC()
		}
		items = append(items, item)
	}
	out := &domain.Watchlist{
		ClientID: clientID, Items: items, Updated: time.Now().UTC(), Version: wl.Version + 1,
	}
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
func (m *Memory) Remove(_ context.Context, clientID string, exchange domain.Exchange, symbol string, expectedVersion int64) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.checkVersionLocked(clientID, expectedVersion); err != nil {
		return nil, err
	}
	wl := m.data[clientID]
	if wl == nil {
		if expectedVersion >= 0 && expectedVersion != 0 {
			return nil, &domain.WatchlistVersionMismatch{Current: &domain.Watchlist{
				ClientID: clientID, Items: []domain.WatchlistItem{}, Updated: time.Now().UTC(), Version: 0,
			}}
		}
		return &domain.Watchlist{ClientID: clientID, Items: []domain.WatchlistItem{}, Updated: time.Now().UTC(), Version: 0}, nil
	}
	next := make([]domain.WatchlistItem, 0, len(wl.Items))
	for _, it := range wl.Items {
		if it.Exchange == exchange && it.Symbol == symbol {
			continue
		}
		next = append(next, it)
	}
	out := &domain.Watchlist{
		ClientID: clientID, Items: next, Updated: time.Now().UTC(), Version: wl.Version + 1,
	}
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

// CreateShare inserts a share; fails if already shared.
func (m *Memory) CreateShare(_ context.Context, share domain.WatchlistShare) (*domain.WatchlistShare, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := shareKey(share.OwnerClientID, share.GranteeClientID)
	if _, ok := m.shares[k]; ok {
		return nil, fmt.Errorf("%w: watchlist already shared with this user", domain.ErrInvalidArgument)
	}
	if share.CreatedAt.IsZero() {
		share.CreatedAt = time.Now().UTC()
	}
	if share.UpdatedAt.IsZero() {
		share.UpdatedAt = share.CreatedAt
	}
	m.shares[k] = share
	cp := share
	return &cp, nil
}

// UpdateShareRole updates an existing share role.
func (m *Memory) UpdateShareRole(_ context.Context, ownerClientID, granteeClientID string, role domain.WatchlistShareRole, at time.Time) (*domain.WatchlistShare, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := shareKey(ownerClientID, granteeClientID)
	sh, ok := m.shares[k]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	sh.Role = role
	sh.UpdatedAt = at.UTC()
	m.shares[k] = sh
	cp := sh
	return &cp, nil
}

// GetShare returns one share.
func (m *Memory) GetShare(_ context.Context, ownerClientID, granteeClientID string) (*domain.WatchlistShare, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sh, ok := m.shares[shareKey(ownerClientID, granteeClientID)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := sh
	return &cp, nil
}

// ListSharesByOwner lists shares for owner.
func (m *Memory) ListSharesByOwner(_ context.Context, ownerClientID string) ([]domain.WatchlistShare, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.WatchlistShare, 0)
	for _, sh := range m.shares {
		if sh.OwnerClientID == ownerClientID {
			out = append(out, sh)
		}
	}
	return out, nil
}

// ListSharesForGrantee lists shares received.
func (m *Memory) ListSharesForGrantee(_ context.Context, granteeClientID string) ([]domain.WatchlistShare, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.WatchlistShare, 0)
	for _, sh := range m.shares {
		if sh.GranteeClientID == granteeClientID {
			out = append(out, sh)
		}
	}
	return out, nil
}

// DeleteShare revokes access.
func (m *Memory) DeleteShare(_ context.Context, ownerClientID, granteeClientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := shareKey(ownerClientID, granteeClientID)
	if _, ok := m.shares[k]; !ok {
		return domain.ErrNotFound
	}
	delete(m.shares, k)
	return nil
}

// CountSharesByOwner counts shares.
func (m *Memory) CountSharesByOwner(_ context.Context, ownerClientID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, sh := range m.shares {
		if sh.OwnerClientID == ownerClientID {
			n++
		}
	}
	return n, nil
}

// AppendAudit records an event.
func (m *Memory) AppendAudit(_ context.Context, ev domain.WatchlistAuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	m.audit = append(m.audit, ev)
	return nil
}

// ListAudit returns newest-first events for owner.
func (m *Memory) ListAudit(_ context.Context, ownerClientID string, limit, offset int) ([]domain.WatchlistAuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	// filter reverse
	tmp := make([]domain.WatchlistAuditEvent, 0)
	for i := len(m.audit) - 1; i >= 0; i-- {
		if m.audit[i].OwnerClientID == ownerClientID {
			tmp = append(tmp, m.audit[i])
		}
	}
	if offset >= len(tmp) {
		return []domain.WatchlistAuditEvent{}, nil
	}
	end := offset + limit
	if end > len(tmp) {
		end = len(tmp)
	}
	return append([]domain.WatchlistAuditEvent(nil), tmp[offset:end]...), nil
}
