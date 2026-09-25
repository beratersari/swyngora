package optionsstore

import (
	"context"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process options sample ring (tests and fallback).
type Memory struct {
	mu      sync.Mutex
	rows    []domain.OptionsSample
	strikes []memStrikeBook
}

type memStrikeBook struct {
	asset  string
	at     time.Time
	points []domain.OptionsStrikeOIPoint
}

// InsertSample keeps one print per asset and 5-minute bucket.
func (m *Memory) InsertSample(_ context.Context, rec domain.OptionsSample) (bool, error) {
	if m == nil || rec.Asset == "" || rec.SampledAt.IsZero() {
		return false, nil
	}
	rec.Asset, _ = domain.ParseOptionsAsset(rec.Asset)
	rec.SampledAt = rec.SampledAt.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.rows {
		if m.rows[i].Asset == rec.Asset && m.rows[i].SampledAt.Equal(rec.SampledAt) {
			m.rows[i] = rec
			return false, nil
		}
	}
	m.rows = append(m.rows, rec)
	return true, nil
}

// ListSamples returns oldest-first prints in [from, to].
func (m *Memory) ListSamples(_ context.Context, asset string, from, to time.Time, limit int) ([]domain.OptionsSample, error) {
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > domain.OptionsHistoryListLimit {
		limit = domain.OptionsHistoryListLimit
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.OptionsSample, 0, len(m.rows))
	for _, r := range m.rows {
		if r.Asset != asset {
			continue
		}
		if !from.IsZero() && r.SampledAt.Before(from) {
			continue
		}
		if !to.IsZero() && r.SampledAt.After(to) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SampledAt.Before(out[j].SampledAt) })
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

// ReplaceStrikeBook stores one ladder per asset and sample time.
func (m *Memory) ReplaceStrikeBook(_ context.Context, asset string, at time.Time, points []domain.OptionsStrikeOIPoint) error {
	if m == nil || asset == "" || at.IsZero() {
		return nil
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return err
	}
	at = at.UTC()
	cp := append([]domain.OptionsStrikeOIPoint(nil), points...)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.strikes {
		if m.strikes[i].asset == asset && m.strikes[i].at.Equal(at) {
			m.strikes[i].points = cp
			return nil
		}
	}
	m.strikes = append(m.strikes, memStrikeBook{asset: asset, at: at, points: cp})
	return nil
}

// StrikeBookAt returns the latest ladder at or before at.
func (m *Memory) StrikeBookAt(_ context.Context, asset string, at time.Time) (domain.OptionsStrikeBook, bool, error) {
	var book domain.OptionsStrikeBook
	if m == nil || at.IsZero() {
		return book, false, nil
	}
	asset, err := domain.ParseOptionsAsset(asset)
	if err != nil {
		return book, false, err
	}
	at = at.UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	var hit *memStrikeBook
	for i := range m.strikes {
		row := &m.strikes[i]
		if row.asset != asset || row.at.After(at) {
			continue
		}
		if hit == nil || row.at.After(hit.at) {
			hit = row
		}
	}
	if hit == nil {
		return book, false, nil
	}
	book.SampledAt = hit.at
	book.Points = append([]domain.OptionsStrikeOIPoint(nil), hit.points...)
	return book, true, nil
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
		if r.SampledAt.Before(cutoff) {
			n++
			continue
		}
		kept = append(kept, r)
	}
	m.rows = kept
	keptBooks := m.strikes[:0]
	for _, r := range m.strikes {
		if r.at.Before(cutoff) {
			continue
		}
		keptBooks = append(keptBooks, r)
	}
	m.strikes = keptBooks
	return n, nil
}
