package domain

import (
	"sort"
	"time"
)

const (
	// LiquidationFeedGapLookback is how far back we expose disconnects.
	LiquidationFeedGapLookback = 6 * time.Hour
	// liquidationFeedStall is how long a "live" socket may go silent before
	// we treat the venue as missing rather than quietly healthy.
	liquidationFeedStall = 2 * time.Minute
	maxLiquidationGaps   = 64
)

// LiquidationGap is a stretch when that venue's websocket was not live
// and history has not (yet) filled the hole. To is zero while the gap is
// still open. Seconds / MissingSeconds are the remaining unfilled duration
// (a fully filled hole is removed, not listed with 0).
type LiquidationGap struct {
	From           time.Time
	To             time.Time
	Seconds        int64
	MissingSeconds int64
}

// LiquidationVenueHealth is one venue's last print, last socket message, and gaps.
type LiquidationVenueHealth struct {
	Exchange        string
	Live            bool
	LastEventAt     time.Time
	LastSeenAt      time.Time
	CoverageSeconds int64
	MissingSeconds  int64
	Gaps            []LiquidationGap
}

// LiquidationFeed is per-venue health for a snapshot or overview.
// Missing lists venues that are down, stalled, or never started.
type LiquidationFeed struct {
	Venues  []LiquidationVenueHealth
	Missing []string
}

// NoteSeen records that the venue websocket delivered any payload (including
// a heartbeat or an empty liquidation batch). Does not start coverage.
func (b *LiquidationBook) NoteSeen(ex Exchange) {
	if b == nil || ex == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	if b.lastSeen == nil {
		b.lastSeen = map[Exchange]time.Time{}
	}
	b.lastSeen[ex] = now
}

// Feed reports last print, last socket message, and recent gaps.
// exchange is binance, bybit, or all.
func (b *LiquidationBook) Feed(exchange string) LiquidationFeed {
	out := LiquidationFeed{
		Venues:  []LiquidationVenueHealth{},
		Missing: []string{},
	}
	if b == nil {
		return out
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	return b.feedLocked(exchange, now)
}

func (b *LiquidationBook) feedLocked(exchange string, now time.Time) LiquidationFeed {
	out := LiquidationFeed{
		Venues:  []LiquidationVenueHealth{},
		Missing: []string{},
	}
	want := liquidationVenues()
	if exchange != "" && exchange != "all" {
		want = []Exchange{Exchange(exchange)}
	}
	since := now.Add(-LiquidationFeedGapLookback)
	for _, ex := range want {
		h := b.venueHealthLocked(ex, now, since)
		out.Venues = append(out.Venues, h)
		if !h.Live {
			out.Missing = append(out.Missing, string(ex))
		}
	}
	return out
}

func (b *LiquidationBook) venueHealthLocked(ex Exchange, now, since time.Time) LiquidationVenueHealth {
	live := b.live[ex] && !b.stalledLocked(ex, now)
	gaps := clipGaps(b.gaps[ex], since, now)
	h := LiquidationVenueHealth{
		Exchange:        string(ex),
		Live:            live,
		LastEventAt:     b.lastEvent[ex],
		LastSeenAt:      b.lastSeen[ex],
		CoverageSeconds: int64(b.venueClock[ex].elapsed(now).Seconds()),
		Gaps:            gaps,
		MissingSeconds:  sumGapMissing(gaps),
	}
	return h
}

func (b *LiquidationBook) stalledLocked(ex Exchange, now time.Time) bool {
	if !b.live[ex] {
		return false
	}
	seen := b.lastSeen[ex]
	if seen.IsZero() {
		// Live was set but no payload yet — allow a short handshake.
		return false
	}
	return now.Sub(seen) > liquidationFeedStall
}

func (b *LiquidationBook) effectivelyLiveLocked(ex Exchange, now time.Time) bool {
	return b.live[ex] && !b.stalledLocked(ex, now)
}

func (b *LiquidationBook) recordGapLocked(ex Exchange, from, to time.Time) {
	if ex == "" || from.IsZero() {
		return
	}
	from = from.UTC()
	if !to.IsZero() {
		to = to.UTC()
		if !to.After(from) {
			return
		}
	}
	if b.gaps == nil {
		b.gaps = map[Exchange][]LiquidationGap{}
	}
	list := b.gaps[ex]
	if n := len(list); n > 0 && list[n-1].To.IsZero() {
		if to.IsZero() {
			return
		}
		list[n-1].To = to
		list[n-1] = stampGap(list[n-1])
		b.gaps[ex] = trimGaps(list)
		return
	}
	g := stampGap(LiquidationGap{From: from, To: to})
	b.gaps[ex] = trimGaps(append(list, g))
}

func (b *LiquidationBook) closeOpenGapLocked(ex Exchange, at time.Time) {
	if b.gaps == nil {
		return
	}
	list := b.gaps[ex]
	n := len(list)
	if n == 0 || !list[n-1].To.IsZero() {
		return
	}
	if at.IsZero() || !at.After(list[n-1].From) {
		b.gaps[ex] = list[:n-1]
		return
	}
	list[n-1].To = at.UTC()
	list[n-1] = stampGap(list[n-1])
	b.gaps[ex] = list
}

// RestoreFeed applies durable last-print / last-seen / gap clocks.
// lastSaved in the past becomes a downtime gap so a restart is not treated
// as live coverage.
func (b *LiquidationBook) RestoreFeed(c LiquidationCoverage, now time.Time) {
	if b == nil || c.Exchange == "" {
		return
	}
	if now.IsZero() {
		now = b.now().UTC()
	} else {
		now = now.UTC()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.lastEvent == nil {
		b.lastEvent = map[Exchange]time.Time{}
	}
	if b.lastSeen == nil {
		b.lastSeen = map[Exchange]time.Time{}
	}
	if !c.LastEvent.IsZero() {
		if t, ok := b.lastEvent[c.Exchange]; !ok || c.LastEvent.After(t) {
			b.lastEvent[c.Exchange] = c.LastEvent.UTC()
		}
	}
	if !c.LastSeen.IsZero() {
		if t, ok := b.lastSeen[c.Exchange]; !ok || c.LastSeen.After(t) {
			b.lastSeen[c.Exchange] = c.LastSeen.UTC()
		}
	}
	if len(c.Gaps) > 0 {
		if b.gaps == nil {
			b.gaps = map[Exchange][]LiquidationGap{}
		}
		if c.Symbol == "" {
			b.gaps[c.Exchange] = trimGaps(mergeGaps(b.gaps[c.Exchange], c.Gaps))
		}
	}
	if c.Symbol == "" && !c.LastSaved.IsZero() && now.Sub(c.LastSaved) > liquidationFeedStall {
		b.recordGapLocked(c.Exchange, c.LastSaved, now)
	}
}

func clipGaps(in []LiquidationGap, since, now time.Time) []LiquidationGap {
	if len(in) == 0 {
		return []LiquidationGap{}
	}
	out := make([]LiquidationGap, 0, len(in))
	for _, g := range in {
		to := g.To
		if to.IsZero() {
			to = now
		}
		if to.Before(since) {
			continue
		}
		from := g.From
		if from.Before(since) {
			from = since
		}
		row := LiquidationGap{From: from, To: g.To}
		end := to
		if !g.To.IsZero() {
			end = g.To
		} else {
			end = now
		}
		if end.After(from) {
			row.Seconds = int64(end.Sub(from).Seconds())
			row.MissingSeconds = row.Seconds
		}
		out = append(out, row)
	}
	return out
}

func trimGaps(in []LiquidationGap) []LiquidationGap {
	if len(in) <= maxLiquidationGaps {
		return in
	}
	return in[len(in)-maxLiquidationGaps:]
}

func mergeGaps(a, b []LiquidationGap) []LiquidationGap {
	out := append([]LiquidationGap{}, a...)
	out = append(out, b...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].From.Before(out[j].From)
	})
	return trimGaps(out)
}

func liquidationVenues() []Exchange {
	return []Exchange{ExchangeBinance, ExchangeBybit}
}
