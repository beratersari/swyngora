package domain

import (
	"strings"
	"time"
)

const (
	// BookFlashSample is how often the history watcher looks between minute snapshots.
	BookFlashSample = 5 * time.Second
	// bookFlashMatchFrac treats nearby prices as the same wall.
	bookFlashMatchFrac = 0.0008
)

// BookHistoryFlash is a wall that rested and left between minute snapshots.
type BookHistoryFlash struct {
	Exchange  Exchange
	Symbol    string
	Side      string
	Price     float64
	Quantity  float64
	Notional  float64
	SeenFrom  time.Time
	SeenUntil time.Time
	Duration  time.Duration
	Brief     bool
}

// BookFlashWatch remembers walls between minute snapshots and emits the ones
// that disappear before a snapshot can store them.
type BookFlashWatch struct {
	tracks []*bookFlashTrack
}

type bookFlashTrack struct {
	exchange Exchange
	symbol   string
	side     string
	price    float64
	qty      float64
	notional float64
	first    time.Time
	last     time.Time
	present  bool
	landed   bool
	emitted  bool
}

// BookWalls extracts large walls from a raw book (same rule as a stored sample).
func BookWalls(raw RawOrderBook) []BookHistoryWall {
	return bookWallsFromAnalysis(AnalyzeOrderBook(raw, DefaultOrderBookRangePct).Walls)
}

// Observe updates open walls. minute marks walls that the minute snapshot
// stored, so they are not also reported as missed. Walls that drop out
// before that return as flashes.
func (w *BookFlashWatch) Observe(ex Exchange, symbol string, now time.Time, walls []BookHistoryWall, minute bool) []BookHistoryFlash {
	if w == nil {
		return nil
	}
	now = now.UTC()
	symbol = NormalizeSymbol(ex, symbol)
	if symbol == "" || now.IsZero() {
		return nil
	}
	seen := make(map[*bookFlashTrack]struct{}, len(walls))
	for _, wall := range walls {
		if wall.Price <= 0 || wall.Notional <= 0 {
			continue
		}
		side := strings.ToLower(strings.TrimSpace(wall.Side))
		tr := w.match(ex, symbol, side, wall.Price)
		if tr == nil {
			tr = &bookFlashTrack{
				exchange: ex, symbol: symbol, side: side, price: wall.Price,
				first: now, present: true,
			}
			w.tracks = append(w.tracks, tr)
		}
		tr.last = now
		tr.present = true
		tr.qty = wall.Quantity
		if wall.Notional > tr.notional {
			tr.notional = wall.Notional
			tr.price = wall.Price
		}
		if minute {
			tr.landed = true
		}
		seen[tr] = struct{}{}
	}
	var out []BookHistoryFlash
	keep := w.tracks[:0]
	for _, tr := range w.tracks {
		if tr.exchange != ex || tr.symbol != symbol {
			keep = append(keep, tr)
			continue
		}
		if _, ok := seen[tr]; ok {
			keep = append(keep, tr)
			continue
		}
		tr.present = false
		if !tr.emitted && !tr.landed && !tr.first.IsZero() && tr.notional > 0 {
			until := now
			if until.Before(tr.last) {
				until = tr.last
			}
			out = append(out, BookHistoryFlash{
				Exchange: tr.exchange, Symbol: tr.symbol, Side: tr.side,
				Price: tr.price, Quantity: tr.qty, Notional: tr.notional,
				SeenFrom: tr.first, SeenUntil: until,
				Duration: until.Sub(tr.first), Brief: true,
			})
			tr.emitted = true
		}
		if tr.emitted || now.Sub(tr.last) > 2*time.Minute {
			continue
		}
		keep = append(keep, tr)
	}
	w.tracks = keep
	return out
}

func (w *BookFlashWatch) match(ex Exchange, symbol, side string, price float64) *bookFlashTrack {
	for _, tr := range w.tracks {
		if tr.emitted || tr.exchange != ex || tr.symbol != symbol || tr.side != side || tr.price <= 0 {
			continue
		}
		gap := flashAbs(tr.price-price) / tr.price
		if gap <= bookFlashMatchFrac {
			return tr
		}
	}
	return nil
}

func flashAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// AttachBookFlashes adds brief walls onto a minute snapshot when they lived
// in that minute and the ladder itself did not keep them.
func AttachBookFlashes(snap *BookHistorySnapshot, flashes []BookHistoryFlash) {
	if snap == nil || len(flashes) == 0 {
		return
	}
	start := snap.SampledAt.UTC()
	end := start.Add(DefaultBookHistoryBucket)
	for _, f := range flashes {
		if f.SeenUntil.Before(start) || !f.SeenFrom.Before(end) {
			continue
		}
		if bookFlashOnWalls(snap.Walls, f) {
			continue
		}
		snap.Flashes = append(snap.Flashes, f)
		snap.Walls = append(snap.Walls, BookHistoryWall{
			Side: f.Side, Price: f.Price, Quantity: f.Quantity, Notional: f.Notional,
			Brief: true, SeenSeconds: f.Duration.Seconds(),
		})
	}
}

func bookFlashOnWalls(walls []BookHistoryWall, f BookHistoryFlash) bool {
	for _, w := range walls {
		if w.Brief || !strings.EqualFold(w.Side, f.Side) || w.Price <= 0 {
			continue
		}
		if flashAbs(w.Price-f.Price)/w.Price <= bookFlashMatchFrac {
			return true
		}
	}
	return false
}
