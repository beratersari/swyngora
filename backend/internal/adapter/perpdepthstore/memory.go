package perpdepthstore

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process perp-depth sample ring for tests.
type Memory struct {
	mu   sync.Mutex
	rows []domain.PerpDepthSample
}

// InsertSamples upserts one-minute buckets.
func (m *Memory) InsertSamples(_ context.Context, rows []domain.PerpDepthSample) (int, error) {
	if m == nil || len(rows) == 0 {
		return 0, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, rec := range rows {
		if rec.Symbol == "" || rec.Time.IsZero() {
			continue
		}
		if rec.Exchange == "" {
			rec.Exchange = "all"
		}
		rec.Time = domain.BucketPerpDepthSampleTime(rec.Time)
		replaced := false
		for i := range m.rows {
			if m.rows[i].Exchange == rec.Exchange && m.rows[i].Symbol == rec.Symbol &&
				m.rows[i].Time.Equal(rec.Time) && almostRange(m.rows[i].RangePct, rec.RangePct) {
				m.rows[i] = rec
				replaced = true
				break
			}
		}
		if !replaced {
			m.rows = append(m.rows, rec)
			n++
		}
	}
	return n, nil
}

// ListSamples returns oldest-first prints in [from, to].
func (m *Memory) ListSamples(_ context.Context, exchange string, rangePct float64, from, to time.Time) ([]domain.PerpDepthSample, error) {
	if m == nil {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.PerpDepthSample, 0, len(m.rows))
	for _, r := range m.rows {
		if r.Exchange != exchange || !almostRange(r.RangePct, rangePct) {
			continue
		}
		if !from.IsZero() && r.Time.Before(from) {
			continue
		}
		if !to.IsZero() && r.Time.After(to) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	if len(out) > domain.PerpDepthHistoryListLimit {
		out = out[len(out)-domain.PerpDepthHistoryListLimit:]
	}
	return out, nil
}

// PurgeOlderThan drops samples before cutoff.
func (m *Memory) PurgeOlderThan(_ context.Context, cutoff time.Time) (int, error) {
	if m == nil || cutoff.IsZero() {
		return 0, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.rows[:0]
	n := 0
	for _, r := range m.rows {
		if r.Time.Before(cutoff) {
			n++
			continue
		}
		kept = append(kept, r)
	}
	m.rows = kept
	return n, nil
}

func almostRange(a, b float64) bool {
	if a <= 0 {
		a = domain.DefaultPerpDepthRangePct
	}
	if b <= 0 {
		b = domain.DefaultPerpDepthRangePct
	}
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-6
}
