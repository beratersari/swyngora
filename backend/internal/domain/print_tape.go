package domain

import (
	"fmt"
	"sync"
	"time"
)

const (
	// PrintTapeRetain is how long individual prints stay in memory for
	// print-size, order-count, whales, and flow-at-prices.
	PrintTapeRetain = time.Hour
	// printTapeMax is the per-symbol cap. Older prints fall out first.
	printTapeMax = 8000
	// printDedupKeep bounds the duplicate set so a REST seed and the
	// websocket cannot add the same fill twice.
	printDedupKeep = 12000
)

// PrintTape is a bounded per-venue print log. Minute notionals stay on TakerBook.
type PrintTape struct {
	mu   sync.Mutex
	buf  map[string][]TakerPrint
	seen map[string]map[string]int64
	now  func() time.Time
}

// NewPrintTape constructs an empty tape.
func NewPrintTape() *PrintTape {
	return &PrintTape{
		buf:  map[string][]TakerPrint{},
		seen: map[string]map[string]int64{},
		now:  time.Now,
	}
}

// Record stores one fill. It returns false when the print is a duplicate,
// empty, or older than the minute-book retention (the book would drop it too).
func (t *PrintTape) Record(p TakerPrint) bool {
	if t == nil || p.Notional <= 0 || p.Time.IsZero() || p.Exchange == "" {
		return false
	}
	p.Symbol = NormalizeLiquidationSymbol(p.Symbol)
	if p.Symbol == "" {
		return false
	}
	now := t.now().UTC()
	if p.Time.Before(now.Add(-takerRetain)) {
		return false
	}
	key := printKey(p)
	t.mu.Lock()
	defer t.mu.Unlock()
	sk := takerKey(p.Exchange, p.Symbol)
	seen := t.seen[sk]
	if seen == nil {
		seen = map[string]int64{}
		t.seen[sk] = seen
	}
	if _, ok := seen[key]; ok {
		return false
	}
	seen[key] = p.Time.UTC().UnixMilli()
	if len(seen) > printDedupKeep {
		cut := now.Add(-takerRetain).UnixMilli()
		for k, ms := range seen {
			if ms < cut {
				delete(seen, k)
			}
		}
	}
	if p.Time.Before(now.Add(-PrintTapeRetain)) {
		return true
	}
	buf := append(t.buf[sk], p)
	min := now.Add(-PrintTapeRetain).UnixMilli()
	dst := buf[:0]
	for _, row := range buf {
		if row.Time.UnixMilli() >= min {
			dst = append(dst, row)
		}
	}
	if len(dst) > printTapeMax {
		dst = append([]TakerPrint(nil), dst[len(dst)-printTapeMax:]...)
	}
	t.buf[sk] = dst
	return true
}

// Prints returns a copy of the in-memory fills, oldest first.
func (t *PrintTape) Prints(ex Exchange, symbol string) []TakerPrint {
	if t == nil {
		return nil
	}
	symbol = NormalizeLiquidationSymbol(symbol)
	t.mu.Lock()
	defer t.mu.Unlock()
	src := t.buf[takerKey(ex, symbol)]
	if len(src) == 0 {
		return nil
	}
	out := append([]TakerPrint(nil), src...)
	return out
}

// Covers reports whether the tape reaches back to from (a one-second slack).
func (t *PrintTape) Covers(ex Exchange, symbol string, from time.Time) bool {
	ps := t.Prints(ex, symbol)
	if len(ps) == 0 || from.IsZero() {
		return false
	}
	return !ps[0].Time.After(from.Add(time.Second))
}

func printKey(p TakerPrint) string {
	return fmt.Sprintf("%d|%s|%.8f|%.8f", p.Time.UTC().UnixMilli(), p.Side, p.Price, p.Quantity)
}

// RecordMeasuredPrint writes one aggressive fill to the minute book and the print tape.
// Duplicates already on the tape are ignored so a REST seed cannot double a websocket fill.
func RecordMeasuredPrint(book *TakerBook, tape *PrintTape, p TakerPrint) {
	if p.Notional <= 0 || p.Time.IsZero() {
		return
	}
	if tape != nil && !tape.Record(p) {
		return
	}
	if book != nil {
		book.Record(p)
	}
}
