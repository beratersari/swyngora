package domain

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultHeatmapWindow is how much resting-size history the UI asks for.
	DefaultHeatmapWindow = 10 * time.Minute
	MinHeatmapWindow     = 1 * time.Minute
	MaxHeatmapWindow     = 30 * time.Minute
	// HeatmapMinInterval collapses faster polls into one live column.
	HeatmapMinInterval = 1 * time.Second
	// Enough 1s samples to cover MaxHeatmapWindow (15m / 30m chips).
	HeatmapMaxColumns = int(MaxHeatmapWindow / HeatmapMinInterval)
	HeatmapMaxTapes   = 64
	// HeatmapRecordLevels is grouped rows per side stored on each column
	// (deeper than the 16-row ladder so the map shows more than the touch).
	HeatmapRecordLevels = 80
)

// HeatmapLevel is one price bucket of resting notional on a heatmap column.
type HeatmapLevel struct {
	Price    string
	Notional string
	IsWall   bool
}

// HeatmapColumn is one snapshot of the book (one vertical strip on the map).
type HeatmapColumn struct {
	Time time.Time
	Mid  string
	Bids []HeatmapLevel
	Asks []HeatmapLevel
}

// OrderBookHeatmap is recent resting bid/ask size over time for one pair.
type OrderBookHeatmap struct {
	Exchange      Exchange
	Symbol        string
	GroupSize     string
	WindowSeconds int
	SampleEveryMs int
	From          time.Time
	To            time.Time
	Columns       []HeatmapColumn
	Live          bool
}

// HeatmapTape keeps an in-memory ring of grouped book snapshots per venue pair.
type HeatmapTape struct {
	mu    sync.Mutex
	tapes map[string]*heatTape
}

type heatTape struct {
	exchange Exchange
	symbol   string
	group    string
	cols     []heatCol
}

type heatCol struct {
	t    time.Time
	mid  string
	bids []HeatmapLevel
	asks []HeatmapLevel
}

// NewHeatmapTape constructs an empty tape store.
func NewHeatmapTape() *HeatmapTape {
	return &HeatmapTape{tapes: map[string]*heatTape{}}
}

// ClampHeatmapWindowSeconds bounds the requested lookback (seconds).
func ClampHeatmapWindowSeconds(n int) int {
	if n <= 0 {
		return int(DefaultHeatmapWindow / time.Second)
	}
	min := int(MinHeatmapWindow / time.Second)
	max := int(MaxHeatmapWindow / time.Second)
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func heatmapKey(exchange, symbol string) string {
	return strings.ToLower(strings.TrimSpace(exchange)) + "|" + strings.ToUpper(strings.TrimSpace(symbol))
}

// Record appends (or replaces the last) column from a grouped book.
func (t *HeatmapTape) Record(now time.Time, book OrderBook) {
	if t == nil {
		return
	}
	now = now.UTC()
	ex := string(book.Exchange)
	symbol := strings.ToUpper(strings.TrimSpace(book.Symbol))
	if ex == "" || symbol == "" {
		return
	}
	col := heatCol{
		t:    now,
		mid:  book.LastPrice,
		bids: copyHeatLevels(book.Bids),
		asks: copyHeatLevels(book.Asks),
	}
	if col.t.IsZero() && !book.UpdatedAt.IsZero() {
		col.t = book.UpdatedAt.UTC()
	}
	key := heatmapKey(ex, symbol)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.tapes == nil {
		t.tapes = map[string]*heatTape{}
	}
	tp := t.tapes[key]
	if tp == nil {
		t.evictLocked(now)
		tp = &heatTape{exchange: book.Exchange, symbol: symbol}
		t.tapes[key] = tp
	}
	tp.group = book.GroupSize
	if n := len(tp.cols); n > 0 && now.Sub(tp.cols[n-1].t) < HeatmapMinInterval {
		tp.cols[n-1] = col
		return
	}
	tp.cols = append(tp.cols, col)
	if len(tp.cols) > HeatmapMaxColumns {
		tp.cols = append([]heatCol(nil), tp.cols[len(tp.cols)-HeatmapMaxColumns:]...)
	}
}

func (t *HeatmapTape) evictLocked(now time.Time) {
	if len(t.tapes) < HeatmapMaxTapes {
		return
	}
	var oldestKey string
	var oldest time.Time
	for k, tp := range t.tapes {
		last := now
		if n := len(tp.cols); n > 0 {
			last = tp.cols[n-1].t
		}
		if oldestKey == "" || last.Before(oldest) {
			oldestKey = k
			oldest = last
		}
	}
	if oldestKey != "" {
		delete(t.tapes, oldestKey)
	}
}

// View returns columns inside the lookback window (oldest first).
func (t *HeatmapTape) View(exchange, symbol string, window time.Duration) OrderBookHeatmap {
	windowSec := ClampHeatmapWindowSeconds(int(window / time.Second))
	if window <= 0 {
		window = time.Duration(windowSec) * time.Second
	}
	out := OrderBookHeatmap{
		Exchange:      ParseExchange(exchange),
		Symbol:        strings.ToUpper(strings.TrimSpace(symbol)),
		WindowSeconds: windowSec,
		SampleEveryMs: int(HeatmapMinInterval / time.Millisecond),
	}
	if t == nil {
		return out
	}
	key := heatmapKey(exchange, symbol)
	t.mu.Lock()
	defer t.mu.Unlock()
	tp := t.tapes[key]
	if tp == nil {
		return out
	}
	out.Exchange = tp.exchange
	out.Symbol = tp.symbol
	out.GroupSize = tp.group
	if len(tp.cols) == 0 {
		return out
	}
	cut := tp.cols[len(tp.cols)-1].t.Add(-window)
	cols := make([]HeatmapColumn, 0, len(tp.cols))
	for _, c := range tp.cols {
		if c.t.Before(cut) {
			continue
		}
		cols = append(cols, HeatmapColumn{
			Time: c.t,
			Mid:  c.mid,
			Bids: append([]HeatmapLevel(nil), c.bids...),
			Asks: append([]HeatmapLevel(nil), c.asks...),
		})
	}
	out.Columns = cols
	if len(cols) > 0 {
		out.From = cols[0].Time
		out.To = cols[len(cols)-1].Time
	}
	return out
}

func copyHeatLevels(in []OrderBookLevel) []HeatmapLevel {
	if len(in) == 0 {
		return nil
	}
	out := make([]HeatmapLevel, 0, len(in))
	for _, lv := range in {
		n, err := strconv.ParseFloat(lv.Notional, 64)
		if err != nil || n <= 0 {
			continue
		}
		out = append(out, HeatmapLevel{
			Price:    lv.Price,
			Notional: lv.Notional,
			IsWall:   lv.IsWall,
		})
	}
	return out
}
