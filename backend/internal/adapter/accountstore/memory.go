package accountstore

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process account store for tests.
type Memory struct {
	mu   sync.Mutex
	byID map[string]*domain.Account
	keys map[string]*domain.APIKey
}

// NewMemory constructs an empty store.
func NewMemory() *Memory {
	return &Memory{byID: map[string]*domain.Account{}, keys: map[string]*domain.APIKey{}}
}

func cloneAcc(a *domain.Account) *domain.Account {
	if a == nil {
		return nil
	}
	cp := *a
	if a.ClosedAt != nil {
		t := a.ClosedAt.UTC()
		cp.ClosedAt = &t
	}
	if a.PurgeAt != nil {
		t := a.PurgeAt.UTC()
		cp.PurgeAt = &t
	}
	if a.ReopenedAt != nil {
		t := a.ReopenedAt.UTC()
		cp.ReopenedAt = &t
	}
	return &cp
}

func (m *Memory) Get(_ context.Context, clientID string) (*domain.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[clientID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneAcc(a), nil
}

func (m *Memory) UpsertActive(_ context.Context, clientID string, at time.Time) (*domain.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.byID[clientID]; ok {
		if a.Status == domain.AccountClosed {
			return cloneAcc(a), nil
		}
		a.UpdatedAt = at.UTC()
		return cloneAcc(a), nil
	}
	a := &domain.Account{
		ClientID: clientID, Status: domain.AccountActive,
		CreatedAt: at.UTC(), UpdatedAt: at.UTC(),
	}
	m.byID[clientID] = a
	return cloneAcc(a), nil
}

func (m *Memory) Close(_ context.Context, clientID string, closedAt, purgeAt time.Time) (*domain.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[clientID]
	if !ok {
		a = &domain.Account{ClientID: clientID, CreatedAt: closedAt.UTC()}
		m.byID[clientID] = a
	}
	c, p := closedAt.UTC(), purgeAt.UTC()
	a.Status = domain.AccountClosed
	a.ClosedAt = &c
	a.PurgeAt = &p
	a.ReopenedAt = nil
	a.UpdatedAt = c
	return cloneAcc(a), nil
}

func (m *Memory) Reopen(_ context.Context, clientID string, at time.Time) (*domain.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[clientID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	t := at.UTC()
	a.Status = domain.AccountActive
	a.ClosedAt = nil
	a.PurgeAt = nil
	a.ReopenedAt = &t
	a.UpdatedAt = t
	return cloneAcc(a), nil
}

func (m *Memory) ListDueForPurge(_ context.Context, now time.Time, limit int) ([]domain.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Account
	for _, a := range m.byID {
		if a.Status == domain.AccountClosed && a.PurgeAt != nil && !a.PurgeAt.After(now) {
			out = append(out, *cloneAcc(a))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *Memory) MarkPurged(_ context.Context, clientID string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.byID[clientID]
	if !ok {
		return domain.ErrNotFound
	}
	a.Status = domain.AccountPurged
	a.UpdatedAt = at.UTC()
	return nil
}

func (m *Memory) Delete(_ context.Context, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[clientID]; !ok {
		return domain.ErrNotFound
	}
	delete(m.byID, clientID)
	return nil
}
