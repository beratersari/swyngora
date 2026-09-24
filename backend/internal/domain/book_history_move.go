package domain

import (
	"strings"
	"time"
)

const (
	// BookMoveMinShare is the smallest resting-size change, as a fraction of that
	// side's visible notional, worth keeping between minute snapshots.
	BookMoveMinShare = 0.03
)

// BookHistoryMove is a resting-size change that was not a wall and did not
// survive onto the minute ladder the way it looked while it was live.
type BookHistoryMove struct {
	Exchange     Exchange
	Symbol       string
	Side         string
	Price        float64
	FromNotional float64
	ToNotional   float64
	PeakNotional float64
	Change       string
	SeenFrom     time.Time
	SeenUntil    time.Time
	Duration     time.Duration
}

// BookMoveSample is one grouped level on a 5-second read.
type BookMoveSample struct {
	Side     string
	Price    float64
	Notional float64
	Wall     bool
}

// BookMoveSamples lists grouped ladder levels for the between-minute watcher.
func BookMoveSamples(raw RawOrderBook) []BookMoveSample {
	grouped := GroupOrderBook(raw, 0, DefaultBookHistoryLevels)
	out := make([]BookMoveSample, 0, len(grouped.Bids)+len(grouped.Asks))
	for _, lv := range grouped.Bids {
		if s, ok := bookMoveSample("bid", lv); ok {
			out = append(out, s)
		}
	}
	for _, lv := range grouped.Asks {
		if s, ok := bookMoveSample("ask", lv); ok {
			out = append(out, s)
		}
	}
	return out
}

func bookMoveSample(side string, lv OrderBookLevel) (BookMoveSample, bool) {
	price := parseHistFloat(lv.Price)
	notional := parseHistFloat(lv.Notional)
	if price <= 0 || notional <= 0 {
		return BookMoveSample{}, false
	}
	return BookMoveSample{Side: side, Price: price, Notional: notional, Wall: lv.IsWall}, true
}

// BookMoveWatch remembers non-wall size between minute snapshots and emits
// changes the ladder does not keep.
type BookMoveWatch struct {
	tracks []*bookMoveTrack
}

type bookMoveTrack struct {
	exchange Exchange
	symbol   string
	side     string
	price    float64
	start    float64
	last     float64
	peak     float64
	trough   float64
	sideSeen float64
	first    time.Time
	lastAt   time.Time
	landed   bool
	wall     bool
	emitted  bool
}

// Observe updates open levels. minute is the snapshot that stores the ladder.
// Walls are ignored here; those stay on the flash path.
func (w *BookMoveWatch) Observe(ex Exchange, symbol string, now time.Time, levels []BookMoveSample, minute bool) []BookHistoryMove {
	if w == nil {
		return nil
	}
	now = now.UTC()
	symbol = NormalizeSymbol(ex, symbol)
	if symbol == "" || now.IsZero() {
		return nil
	}
	sideTotal := map[string]float64{}
	for _, lv := range levels {
		if lv.Notional > 0 {
			sideTotal[lv.Side] += lv.Notional
		}
	}
	seen := make(map[*bookMoveTrack]struct{}, len(levels))
	for _, lv := range levels {
		if lv.Price <= 0 || lv.Notional <= 0 {
			continue
		}
		side := strings.ToLower(strings.TrimSpace(lv.Side))
		tr := w.matchMove(ex, symbol, side, lv.Price)
		if lv.Wall {
			if tr != nil {
				tr.wall = true
				tr.lastAt = now
				tr.last = lv.Notional
				seen[tr] = struct{}{}
			}
			continue
		}
		total := sideTotal[side]
		if tr == nil {
			if total <= 0 || lv.Notional/total < BookMoveMinShare {
				continue
			}
			tr = &bookMoveTrack{
				exchange: ex, symbol: symbol, side: side, price: lv.Price,
				start: lv.Notional, last: lv.Notional, peak: lv.Notional, trough: lv.Notional,
				sideSeen: total, first: now, lastAt: now,
			}
			w.tracks = append(w.tracks, tr)
		}
		tr.lastAt = now
		tr.last = lv.Notional
		if total > tr.sideSeen {
			tr.sideSeen = total
		}
		if lv.Notional > tr.peak {
			tr.peak = lv.Notional
			tr.price = lv.Price
		}
		if lv.Notional < tr.trough {
			tr.trough = lv.Notional
		}
		if minute {
			tr.landed = true
		}
		seen[tr] = struct{}{}
	}
	var out []BookHistoryMove
	keep := w.tracks[:0]
	for _, tr := range w.tracks {
		if tr.exchange != ex || tr.symbol != symbol {
			keep = append(keep, tr)
			continue
		}
		if _, ok := seen[tr]; ok {
			if minute && !tr.emitted && !tr.wall {
				if mv, ok := tr.landedMove(now); ok {
					out = append(out, mv)
					tr.emitted = true
				}
			}
			keep = append(keep, tr)
			continue
		}
		if !tr.emitted && !tr.wall && !tr.landed {
			if mv, ok := tr.leftMove(now); ok {
				out = append(out, mv)
				tr.emitted = true
			}
		}
		if tr.emitted || now.Sub(tr.lastAt) > 2*time.Minute {
			continue
		}
		keep = append(keep, tr)
	}
	w.tracks = keep
	return out
}

func (tr *bookMoveTrack) meaningful(delta float64) bool {
	if tr == nil || tr.sideSeen <= 0 || delta <= 0 {
		return false
	}
	return delta/tr.sideSeen >= BookMoveMinShare
}

func (tr *bookMoveTrack) leftMove(now time.Time) (BookHistoryMove, bool) {
	if !tr.meaningful(tr.peak) {
		return BookHistoryMove{}, false
	}
	return tr.move(BookHistoryChangeRemoved, tr.start, 0, now), true
}

func (tr *bookMoveTrack) landedMove(now time.Time) (BookHistoryMove, bool) {
	up := tr.peak - tr.last
	down := tr.last - tr.trough
	switch {
	case tr.meaningful(up) && up >= down:
		return tr.move(BookHistoryChangeGained, tr.start, tr.last, now), true
	case tr.meaningful(down):
		return tr.move(BookHistoryChangeLost, tr.start, tr.last, now), true
	default:
		return BookHistoryMove{}, false
	}
}

func (tr *bookMoveTrack) move(change string, from, to float64, now time.Time) BookHistoryMove {
	until := now
	if until.Before(tr.lastAt) {
		until = tr.lastAt
	}
	return BookHistoryMove{
		Exchange: tr.exchange, Symbol: tr.symbol, Side: tr.side, Price: tr.price,
		FromNotional: from, ToNotional: to, PeakNotional: tr.peak, Change: change,
		SeenFrom: tr.first, SeenUntil: until, Duration: until.Sub(tr.first),
	}
}

func (w *BookMoveWatch) matchMove(ex Exchange, symbol, side string, price float64) *bookMoveTrack {
	for _, tr := range w.tracks {
		if tr.emitted || tr.exchange != ex || tr.symbol != symbol || tr.side != side || tr.price <= 0 {
			continue
		}
		if flashAbs(tr.price-price)/tr.price <= bookFlashMatchFrac {
			return tr
		}
	}
	return nil
}

// AttachBookMoves adds between-minute size changes onto the minute they happened.
func AttachBookMoves(snap *BookHistorySnapshot, moves []BookHistoryMove) {
	if snap == nil || len(moves) == 0 {
		return
	}
	start := snap.SampledAt.UTC()
	end := start.Add(DefaultBookHistoryBucket)
	for _, mv := range moves {
		if mv.SeenUntil.Before(start) || !mv.SeenFrom.Before(end) {
			continue
		}
		snap.Moves = append(snap.Moves, mv)
	}
}
