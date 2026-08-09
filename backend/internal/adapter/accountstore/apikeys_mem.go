package accountstore

import (
	"context"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func cloneKey(k *domain.APIKey) *domain.APIKey {
	if k == nil {
		return nil
	}
	cp := *k
	if k.LastUsedAt != nil {
		t := k.LastUsedAt.UTC()
		cp.LastUsedAt = &t
	}
	if k.RevokedAt != nil {
		t := k.RevokedAt.UTC()
		cp.RevokedAt = &t
	}
	return &cp
}

func (m *Memory) ensureKeys() {
	if m.keys == nil {
		m.keys = map[string]*domain.APIKey{}
	}
}

// CreateAPIKey stores a key in memory.
func (m *Memory) CreateAPIKey(_ context.Context, k domain.APIKey) (*domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now().UTC()
	}
	cp := k
	m.keys[k.ID] = &cp
	return cloneKey(&cp), nil
}

// GetAPIKeyByHash finds by hash.
func (m *Memory) GetAPIKeyByHash(_ context.Context, hash string) (*domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	for _, k := range m.keys {
		if k.Hash == hash {
			return cloneKey(k), nil
		}
	}
	return nil, domain.ErrNotFound
}

// GetAPIKey finds by id+owner.
func (m *Memory) GetAPIKey(_ context.Context, clientID, id string) (*domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	k, ok := m.keys[id]
	if !ok || k.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	return cloneKey(k), nil
}

// ListAPIKeys lists by client.
func (m *Memory) ListAPIKeys(_ context.Context, clientID string) ([]domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	var out []domain.APIKey
	for _, k := range m.keys {
		if k.ClientID == clientID {
			out = append(out, *cloneKey(k))
		}
	}
	return out, nil
}

// CountActiveAPIKeys counts non-revoked.
func (m *Memory) CountActiveAPIKeys(_ context.Context, clientID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	n := 0
	for _, k := range m.keys {
		if k.ClientID == clientID && !k.IsRevoked() {
			n++
		}
	}
	return n, nil
}

// RevokeAPIKey marks revoked.
func (m *Memory) RevokeAPIKey(_ context.Context, clientID, id string, at time.Time) (*domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	k, ok := m.keys[id]
	if !ok || k.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	if k.RevokedAt == nil {
		t := at.UTC()
		if t.IsZero() {
			t = time.Now().UTC()
		}
		k.RevokedAt = &t
	}
	return cloneKey(k), nil
}

// TouchAPIKeyLastUsed updates last used.
func (m *Memory) TouchAPIKeyLastUsed(_ context.Context, id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	k, ok := m.keys[id]
	if !ok || k.IsRevoked() {
		return domain.ErrNotFound
	}
	t := at.UTC()
	k.LastUsedAt = &t
	return nil
}

// DeleteAPIKeysByClient drops all keys for a client.
func (m *Memory) DeleteAPIKeysByClient(_ context.Context, clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureKeys()
	for id, k := range m.keys {
		if k.ClientID == clientID {
			delete(m.keys, id)
		}
	}
	return nil
}
