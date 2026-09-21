package feargreedstore

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process fear-and-greed history (tests).
type Memory struct {
	mu   sync.Mutex
	rows []domain.FearGreedSample
}

// NewMemory returns an empty store.
func NewMemory() *Memory { return &Memory{} }

// SaveFearGreed upserts one bucket.
func (m *Memory) SaveFearGreed(_ context.Context, sample domain.FearGreedSample) error {
	if m == nil || sample.Time.IsZero() {
		return nil
	}
	sample.Time = sample.Time.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.rows {
		if m.rows[i].Time.Equal(sample.Time) {
			m.rows[i] = sample
			return nil
		}
	}
	m.rows = append(m.rows, sample)
	return nil
}

// ListFearGreed returns samples at or after since, oldest first.
func (m *Memory) ListFearGreed(_ context.Context, since time.Time) ([]domain.FearGreedSample, error) {
	if m == nil {
		return nil, nil
	}
	since = since.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []domain.FearGreedSample{}
	for _, row := range m.rows {
		if !row.Time.Before(since) {
			out = append(out, row)
		}
	}
	return out, nil
}

// PurgeFearGreedBefore drops samples older than the cutoff.
func (m *Memory) PurgeFearGreedBefore(_ context.Context, before time.Time) (int, error) {
	if m == nil {
		return 0, nil
	}
	before = before.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.rows[:0]
	dropped := 0
	for _, row := range m.rows {
		if row.Time.Before(before) {
			dropped++
			continue
		}
		kept = append(kept, row)
	}
	m.rows = kept
	return dropped, nil
}
