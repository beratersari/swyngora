package domain

import (
	"context"
	"time"
)

const minLiquidationGapFill = time.Second

// LiquidationHistoryQuery asks one venue for prints in [From, To].
// Symbols empty means venue-wide when the source supports it.
type LiquidationHistoryQuery struct {
	Exchange Exchange
	Symbols  []string
	From     time.Time
	To       time.Time
}

// LiquidationHistoryResult is one venue's answer for a gap window.
// CoveredFrom/CoveredTo are the range the source actually completed
// (empty when nothing can be trusted as complete, even if Events is set).
type LiquidationHistoryResult struct {
	Events      []LiquidationEvent
	CoveredFrom time.Time
	CoveredTo   time.Time
}

// LiquidationHistoryPort fetches missed liquidations after a disconnect.
// Implementations must not mix venues. Empty Events + a covered range means
// the source confirmed there were no prints in that window.
type LiquidationHistoryPort interface {
	ListLiquidationHistory(ctx context.Context, q LiquidationHistoryQuery) (LiquidationHistoryResult, error)
}

// LiquidationBackfillStats is what ApplyHistory changed on one venue.
type LiquidationBackfillStats struct {
	Added          int
	Filled         bool
	MissingSeconds int64
}

// HasCoveredRange reports a usable completed window.
func (r LiquidationHistoryResult) HasCoveredRange() bool {
	return !r.CoveredFrom.IsZero() && !r.CoveredTo.IsZero() && r.CoveredTo.After(r.CoveredFrom)
}

// WatchedSymbols returns coins we have started tracking on one venue.
func (b *LiquidationBook) WatchedSymbols(ex Exchange) []string {
	if b == nil || ex == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]string, 0, len(b.watchSince))
	suffix := "|" + string(ex)
	seen := map[string]struct{}{}
	for k := range b.watchSince {
		if !hasWatchSuffix(k, suffix) {
			continue
		}
		sym, _, ok := splitWatchKey(k)
		if !ok || sym == "" {
			continue
		}
		if _, dup := seen[sym]; dup {
			continue
		}
		seen[sym] = struct{}{}
		out = append(out, sym)
	}
	for sym := range b.bySym {
		if _, dup := seen[sym]; dup {
			continue
		}
		for _, e := range b.bySym[sym] {
			if e.Exchange == ex {
				seen[sym] = struct{}{}
				out = append(out, sym)
				break
			}
		}
	}
	return out
}

// ClosedGaps returns finished disconnect holes for one venue (oldest first).
func (b *LiquidationBook) ClosedGaps(ex Exchange) []LiquidationGap {
	if b == nil || ex == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.gaps[ex]
	out := make([]LiquidationGap, 0, len(list))
	for _, g := range list {
		if g.To.IsZero() || !g.To.After(g.From) {
			continue
		}
		out = append(out, stampGap(g))
	}
	return out
}

// ApplyHistory merges one venue's historical prints and, when the source
// completed a range, removes that time from the gap list and credits coverage.
// Events from another exchange are ignored. Overlaps with live prints are
// dropped by identity so the same liquidation is not counted twice.
func (b *LiquidationBook) ApplyHistory(ex Exchange, from, to time.Time, events []LiquidationEvent, coveredFrom, coveredTo time.Time) LiquidationBackfillStats {
	out := LiquidationBackfillStats{}
	if b == nil || ex == "" {
		return out
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	for _, e := range events {
		if e.Exchange != "" && e.Exchange != ex {
			continue
		}
		e.Exchange = ex
		if !from.IsZero() && e.Time.Before(from) {
			continue
		}
		if !to.IsZero() && e.Time.After(to) {
			continue
		}
		if b.recordLocked(e) {
			out.Added++
		}
	}
	if !coveredFrom.IsZero() && !coveredTo.IsZero() && coveredTo.After(coveredFrom) {
		before := gapMissingDur(b.gaps[ex], now)
		b.gaps[ex] = SubtractGapInterval(b.gaps[ex], coveredFrom, coveredTo)
		after := gapMissingDur(b.gaps[ex], now)
		if filled := before - after; filled > 0 {
			b.creditLiveLocked(ex, filled)
			out.Filled = after < before
		}
	}
	out.MissingSeconds = int64(gapMissingDur(b.gaps[ex], now) / time.Second)
	return out
}

func (b *LiquidationBook) creditLiveLocked(ex Exchange, d time.Duration) {
	if d <= 0 {
		return
	}
	if b.venueClock[ex] == nil {
		b.venueClock[ex] = &liveClock{}
	}
	b.venueClock[ex].add(d)
	suffix := "|" + string(ex)
	for k, c := range b.watchClock {
		if hasWatchSuffix(k, suffix) {
			c.add(d)
		}
	}
}

func hasWatchSuffix(k, suffix string) bool {
	return len(k) > len(suffix) && k[len(k)-len(suffix):] == suffix
}

// SubtractGapInterval removes a filled window from disconnect holes.
// Fully covered gaps disappear. Partial overlap is split so leftover
// pieces keep only the still-missing time. Open gaps are left alone.
func SubtractGapInterval(gaps []LiquidationGap, from, to time.Time) []LiquidationGap {
	if from.IsZero() || to.IsZero() || !to.After(from) {
		return gaps
	}
	from = from.UTC()
	to = to.UTC()
	out := make([]LiquidationGap, 0, len(gaps))
	for _, g := range gaps {
		if g.To.IsZero() {
			out = append(out, g)
			continue
		}
		if !g.To.After(from) || !to.After(g.From) {
			out = append(out, stampGap(g))
			continue
		}
		if from.After(g.From) {
			if left := newFilledGap(g.From, from); left.Seconds > 0 {
				out = append(out, left)
			}
		}
		if g.To.After(to) {
			if right := newFilledGap(to, g.To); right.Seconds > 0 {
				out = append(out, right)
			}
		}
	}
	return trimGaps(out)
}

func newFilledGap(from, to time.Time) LiquidationGap {
	return stampGap(LiquidationGap{From: from.UTC(), To: to.UTC()})
}

func stampGap(g LiquidationGap) LiquidationGap {
	if g.From.IsZero() {
		return g
	}
	end := g.To
	if end.IsZero() {
		g.Seconds = 0
		g.MissingSeconds = 0
		return g
	}
	if !end.After(g.From) {
		g.Seconds = 0
		g.MissingSeconds = 0
		return g
	}
	sec := int64(end.Sub(g.From) / time.Second)
	g.Seconds = sec
	g.MissingSeconds = sec
	return g
}

func gapMissingDur(gaps []LiquidationGap, now time.Time) time.Duration {
	var d time.Duration
	for _, g := range gaps {
		end := g.To
		if end.IsZero() {
			end = now
		}
		if end.After(g.From) {
			d += end.Sub(g.From)
		}
	}
	if d < 0 {
		return 0
	}
	return d
}

func sumGapMissing(gaps []LiquidationGap) int64 {
	var n int64
	for _, g := range gaps {
		if g.MissingSeconds > 0 {
			n += g.MissingSeconds
		} else {
			n += g.Seconds
		}
	}
	return n
}

// NormalizeHistoryQuery bounds a reconnect window.
func NormalizeHistoryQuery(q LiquidationHistoryQuery) (LiquidationHistoryQuery, bool) {
	q.From = q.From.UTC()
	q.To = q.To.UTC()
	if q.Exchange == "" || q.From.IsZero() || q.To.IsZero() || !q.To.After(q.From) {
		return q, false
	}
	if q.To.Sub(q.From) < minLiquidationGapFill {
		return q, false
	}
	if q.To.Sub(q.From) > liquidationRetain {
		q.From = q.To.Add(-liquidationRetain)
	}
	syms := make([]string, 0, len(q.Symbols))
	seen := map[string]struct{}{}
	for _, s := range q.Symbols {
		s = NormalizeLiquidationSymbol(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		syms = append(syms, s)
	}
	q.Symbols = syms
	return q, true
}
