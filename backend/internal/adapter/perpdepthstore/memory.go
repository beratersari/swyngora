package perpdepthstore

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process perp-depth sample ring for tests.
type Memory struct {
	mu       sync.Mutex
	rows     []domain.PerpDepthSample
	gaps     []domain.PerpDepthHistoryGap
	bands    []domain.PerpBookBandSample
	bookGaps []domain.PerpBookGap
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

// ListMinuteTotals sums bid+ask for each minute, oldest first.
func (m *Memory) ListMinuteTotals(_ context.Context, exchange string, rangePct float64, from, to time.Time) ([]domain.PerpDepthTotalPoint, error) {
	if m == nil {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	type acc struct{ total float64 }
	by := map[int64]acc{}
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
		ms := r.Time.UTC().UnixMilli()
		a := by[ms]
		a.total += r.Bid + r.Ask
		by[ms] = a
	}
	ms := make([]int64, 0, len(by))
	for k, a := range by {
		if a.total > 0 {
			ms = append(ms, k)
		}
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i] < ms[j] })
	out := make([]domain.PerpDepthTotalPoint, 0, len(ms))
	for _, k := range ms {
		out = append(out, domain.PerpDepthTotalPoint{Time: time.UnixMilli(k).UTC(), Total: by[k].total})
	}
	return out, nil
}

// InsertGap records a minute that has no total. A minute that already has samples is left alone.
func (m *Memory) InsertGap(_ context.Context, gap domain.PerpDepthHistoryGap) error {
	if m == nil || gap.Time.IsZero() || (len(gap.Missing) == 0 && len(gap.Stale) == 0) {
		return nil
	}
	if gap.Exchange == "" {
		gap.Exchange = "all"
	}
	gap.Time = domain.BucketPerpDepthSampleTime(gap.Time)
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.Exchange == gap.Exchange && r.Time.Equal(gap.Time) && almostRange(r.RangePct, gap.RangePct) {
			return nil
		}
	}
	for i := range m.gaps {
		if m.gaps[i].Exchange == gap.Exchange && m.gaps[i].Time.Equal(gap.Time) && almostRange(m.gaps[i].RangePct, gap.RangePct) {
			m.gaps[i] = gap
			return nil
		}
	}
	m.gaps = append(m.gaps, gap)
	return nil
}

// ListGaps returns gap minutes in [from, to], oldest first.
func (m *Memory) ListGaps(_ context.Context, exchange string, rangePct float64, from, to time.Time) ([]domain.PerpDepthHistoryGap, error) {
	if m == nil {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.PerpDepthHistoryGap, 0)
	for _, g := range m.gaps {
		if g.Exchange != exchange || !almostRange(g.RangePct, rangePct) {
			continue
		}
		if !from.IsZero() && g.Time.Before(from) {
			continue
		}
		if !to.IsZero() && g.Time.After(to) {
			continue
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}

// ClearGap drops a gap once that minute has a complete total.
func (m *Memory) ClearGap(_ context.Context, exchange string, rangePct float64, at time.Time) error {
	if m == nil || at.IsZero() {
		return nil
	}
	if exchange == "" {
		exchange = "all"
	}
	at = domain.BucketPerpDepthSampleTime(at)
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.gaps[:0]
	for _, g := range m.gaps {
		if g.Exchange == exchange && g.Time.Equal(at) && almostRange(g.RangePct, rangePct) {
			continue
		}
		kept = append(kept, g)
	}
	m.gaps = kept
	return nil
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
	next := make([]domain.PerpDepthHistoryGap, 0, len(m.gaps))
	for _, g := range m.gaps {
		if g.Time.Before(cutoff) {
			n++
			continue
		}
		next = append(next, g)
	}
	m.gaps = next
	bands := make([]domain.PerpBookBandSample, 0, len(m.bands))
	for _, r := range m.bands {
		if r.Time.Before(cutoff) {
			n++
			continue
		}
		bands = append(bands, r)
	}
	m.bands = bands
	bgaps := make([]domain.PerpBookGap, 0, len(m.bookGaps))
	for _, g := range m.bookGaps {
		if g.Time.Before(cutoff) {
			n++
			continue
		}
		bgaps = append(bgaps, g)
	}
	m.bookGaps = bgaps
	return n, nil
}

// ListBucketTotals returns the newest minute's summed bid+ask inside each step.
func (m *Memory) ListBucketTotals(_ context.Context, exchange string, rangePct float64, from, to time.Time, step time.Duration) ([]domain.PerpDepthTotalPoint, error) {
	mins, err := m.ListMinuteTotals(context.Background(), exchange, rangePct, from, to)
	if err != nil || len(mins) == 0 {
		return mins, err
	}
	if step < time.Minute {
		step = time.Minute
	}
	stepMs := step.Milliseconds()
	type pick struct {
		at    int64
		total float64
	}
	by := map[int64]pick{}
	for _, p := range mins {
		ms := p.Time.UTC().UnixMilli()
		bucket := ms / stepMs * stepMs
		prev, ok := by[bucket]
		if ok && prev.at > ms {
			continue
		}
		by[bucket] = pick{at: ms, total: p.Total}
	}
	keys := make([]int64, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]domain.PerpDepthTotalPoint, 0, len(keys))
	for _, k := range keys {
		out = append(out, domain.PerpDepthTotalPoint{Time: time.UnixMilli(k).UTC(), Total: by[k].total})
	}
	return out, nil
}

// InsertBookBands upserts one-minute perp band rows.
func (m *Memory) InsertBookBands(_ context.Context, rows []domain.PerpBookBandSample) (int, error) {
	if m == nil || len(rows) == 0 {
		return 0, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, rec := range rows {
		if rec.Symbol == "" || rec.Time.IsZero() || rec.Bid+rec.Ask <= 0 {
			continue
		}
		if rec.Exchange == "" {
			rec.Exchange = "all"
		}
		rec.Time = domain.BucketPerpDepthSampleTime(rec.Time)
		replaced := false
		for i := range m.bands {
			if m.bands[i].Exchange == rec.Exchange && m.bands[i].Symbol == rec.Symbol &&
				m.bands[i].Time.Equal(rec.Time) && almostRange(m.bands[i].RangePct, rec.RangePct) {
				m.bands[i] = rec
				replaced = true
				break
			}
		}
		if !replaced {
			m.bands = append(m.bands, rec)
		}
		n++
	}
	return n, nil
}

// ListBookBandBuckets returns the newest sample of each band inside each step.
func (m *Memory) ListBookBandBuckets(_ context.Context, exchange, symbol string, from, to time.Time, step time.Duration) ([]domain.PerpBookBandSample, error) {
	if m == nil || symbol == "" {
		return nil, nil
	}
	if exchange == "" {
		exchange = "all"
	}
	if step < time.Minute {
		step = time.Minute
	}
	stepMs := step.Milliseconds()
	m.mu.Lock()
	defer m.mu.Unlock()
	type pick struct {
		at  time.Time
		row domain.PerpBookBandSample
	}
	by := map[string]pick{}
	for _, r := range m.bands {
		if r.Exchange != exchange || r.Symbol != symbol {
			continue
		}
		if !from.IsZero() && r.Time.Before(from) {
			continue
		}
		if !to.IsZero() && r.Time.After(to) {
			continue
		}
		bucket := r.Time.UTC().UnixMilli() / stepMs * stepMs
		key := strconv.FormatInt(bucket, 10) + "|" + strconv.FormatFloat(r.RangePct, 'f', 2, 64)
		prev, ok := by[key]
		if ok && !r.Time.After(prev.at) {
			continue
		}
		row := r
		row.Time = time.UnixMilli(bucket).UTC()
		by[key] = pick{at: r.Time, row: row}
	}
	out := make([]domain.PerpBookBandSample, 0, len(by))
	for _, p := range by {
		out = append(out, p.row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Time.Equal(out[j].Time) {
			return out[i].RangePct < out[j].RangePct
		}
		return out[i].Time.Before(out[j].Time)
	})
	return out, nil
}

// InsertBookGap records why a coin's minute has no combined book.
func (m *Memory) InsertBookGap(_ context.Context, gap domain.PerpBookGap) error {
	if m == nil || gap.Time.IsZero() || gap.Symbol == "" || (len(gap.Missing) == 0 && len(gap.Stale) == 0) {
		return nil
	}
	gap.Time = domain.BucketPerpDepthSampleTime(gap.Time)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.bookGaps {
		if m.bookGaps[i].Symbol == gap.Symbol && m.bookGaps[i].Time.Equal(gap.Time) {
			m.bookGaps[i] = gap
			return nil
		}
	}
	m.bookGaps = append(m.bookGaps, gap)
	return nil
}

// ListBookGaps returns gap minutes for one coin, oldest first.
func (m *Memory) ListBookGaps(_ context.Context, symbol string, from, to time.Time) ([]domain.PerpBookGap, error) {
	if m == nil || symbol == "" {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.PerpBookGap, 0)
	for _, g := range m.bookGaps {
		if g.Symbol != symbol {
			continue
		}
		if !from.IsZero() && g.Time.Before(from) {
			continue
		}
		if !to.IsZero() && g.Time.After(to) {
			continue
		}
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}

// ClearBookGap drops a coin gap once that minute was stored.
func (m *Memory) ClearBookGap(_ context.Context, symbol string, at time.Time) error {
	if m == nil || symbol == "" || at.IsZero() {
		return nil
	}
	at = domain.BucketPerpDepthSampleTime(at)
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := make([]domain.PerpBookGap, 0, len(m.bookGaps))
	for _, g := range m.bookGaps {
		if g.Symbol == symbol && g.Time.Equal(at) {
			continue
		}
		kept = append(kept, g)
	}
	m.bookGaps = kept
	return nil
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
