package domain

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	OrderBookSourceREST      = "rest"
	OrderBookSourceWebSocket = "websocket"
)

// DepthLevel is one price/qty pair using the venue's price string as the key.
type DepthLevel struct {
	Price    string
	Quantity float64
}

// DepthDiff is one incremental depth event (Binance U/u semantics).
type DepthDiff struct {
	Symbol  string
	FirstID int64 // U — first update id in the event
	FinalID int64 // u — final update id in the event
	Bids    []DepthLevel
	Asks    []DepthLevel
	EventAt time.Time
}

// LocalDepthBook is a local copy of a spot book. It is only Synced after a
// snapshot plus a contiguous first diff. A gap or disconnect must Invalidate
// so callers never read a half-applied book.
type LocalDepthBook struct {
	mu           sync.RWMutex
	symbol       string
	bids         map[string]float64
	asks         map[string]float64
	lastID       int64
	haveSnapshot bool
	synced       bool
	updatedAt    time.Time
	source       string
}

// NewLocalDepthBook constructs an empty unsynced book.
func NewLocalDepthBook(symbol string) *LocalDepthBook {
	return &LocalDepthBook{
		symbol: symbol,
		source: OrderBookSourceWebSocket,
	}
}

// Symbol returns the book pair.
func (b *LocalDepthBook) Symbol() string {
	if b == nil {
		return ""
	}
	return b.symbol
}

// Synced is true only while the local copy matches the stream without a gap.
func (b *LocalDepthBook) Synced() bool {
	if b == nil {
		return false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.synced
}

// LastUpdateID is the last applied final update id (0 if none).
func (b *LocalDepthBook) LastUpdateID() int64 {
	if b == nil {
		return 0
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastID
}

// Invalidate drops all levels. Further Snapshot calls fail until a new snapshot+diff.
func (b *LocalDepthBook) Invalidate() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.invalidateLocked()
}

func (b *LocalDepthBook) invalidateLocked() {
	b.bids = nil
	b.asks = nil
	b.lastID = 0
	b.haveSnapshot = false
	b.synced = false
	b.updatedAt = time.Time{}
}

// ReplaceSnapshot loads a full venue snapshot and marks the book synced.
// Used when the stream itself sends a complete book (Coinbase, Bybit).
func (b *LocalDepthBook) ReplaceSnapshot(lastUpdateID int64, bids, asks []DepthLevel) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bids = make(map[string]float64, len(bids))
	b.asks = make(map[string]float64, len(asks))
	putLevels(b.bids, bids)
	putLevels(b.asks, asks)
	b.lastID = lastUpdateID
	b.haveSnapshot = true
	b.synced = true
	b.updatedAt = time.Now().UTC()
}

// ApplySequential applies a single-id delta (Bybit u). id must be lastID+1.
// id==1 after a live book means the venue restarted — invalidate for a full snapshot.
func (b *LocalDepthBook) ApplySequential(id int64, bids, asks []DepthLevel, at time.Time) error {
	if b == nil {
		return fmt.Errorf("%w: nil order book", ErrUpstream)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.haveSnapshot || !b.synced {
		return fmt.Errorf("%w: order book has no snapshot", ErrConflict)
	}
	if id == b.lastID {
		return nil
	}
	if id == 1 {
		b.invalidateLocked()
		return fmt.Errorf("%w: venue restarted order book (u=1)", ErrConflict)
	}
	if id != b.lastID+1 {
		b.invalidateLocked()
		return fmt.Errorf("%w: depth update gap (want u=%d got u=%d)", ErrConflict, b.lastID+1, id)
	}
	putLevels(b.bids, bids)
	putLevels(b.asks, asks)
	b.lastID = id
	if !at.IsZero() {
		b.updatedAt = at.UTC()
	} else {
		b.updatedAt = time.Now().UTC()
	}
	return nil
}

// ApplyUnsequenced applies Coinbase-style l2 changes. The book must already be
// synced from a snapshot; callers detect missed data via disconnect/heartbeat.
func (b *LocalDepthBook) ApplyUnsequenced(bids, asks []DepthLevel, at time.Time) error {
	if b == nil {
		return fmt.Errorf("%w: nil order book", ErrUpstream)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.haveSnapshot || !b.synced {
		return fmt.Errorf("%w: order book has no snapshot", ErrConflict)
	}
	putLevels(b.bids, bids)
	putLevels(b.asks, asks)
	if !at.IsZero() {
		b.updatedAt = at.UTC()
	} else {
		b.updatedAt = time.Now().UTC()
	}
	return nil
}

// BestBidAsk returns the top of book for checksums. ok is false if unsynced or empty.
func (b *LocalDepthBook) BestBidAsk() (bid, ask PriceLevel, ok bool) {
	if b == nil {
		return PriceLevel{}, PriceLevel{}, false
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if !b.synced {
		return PriceLevel{}, PriceLevel{}, false
	}
	bids := topLevels(b.bids, 1, true)
	asks := topLevels(b.asks, 1, false)
	if len(bids) == 0 || len(asks) == 0 {
		return PriceLevel{}, PriceLevel{}, false
	}
	return bids[0], asks[0], true
}

// LoadSnapshot replaces the book from a REST depth snapshot.
// The book is not Synced until the first valid stream event is applied.
func (b *LocalDepthBook) LoadSnapshot(lastUpdateID int64, bids, asks []DepthLevel) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bids = make(map[string]float64, len(bids))
	b.asks = make(map[string]float64, len(asks))
	putLevels(b.bids, bids)
	putLevels(b.asks, asks)
	b.lastID = lastUpdateID
	b.haveSnapshot = true
	b.synced = false
	b.updatedAt = time.Now().UTC()
}

// ApplyDiff applies one stream event. On a sequence gap it invalidates and returns ErrConflict.
// Stale events (u <= last snapshot/event id) are ignored.
func (b *LocalDepthBook) ApplyDiff(d DepthDiff) error {
	if b == nil {
		return fmt.Errorf("%w: nil order book", ErrUpstream)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.haveSnapshot {
		return fmt.Errorf("%w: order book has no snapshot", ErrConflict)
	}
	if d.FinalID <= b.lastID {
		return nil
	}
	if !b.synced {
		// First event after snapshot: U <= lastUpdateId+1 AND u >= lastUpdateId+1
		if d.FirstID > b.lastID+1 || d.FinalID < b.lastID+1 {
			b.invalidateLocked()
			return fmt.Errorf("%w: first depth event does not attach to snapshot", ErrConflict)
		}
		b.synced = true
	} else if d.FirstID != b.lastID+1 {
		b.invalidateLocked()
		return fmt.Errorf("%w: depth update gap (want U=%d got U=%d u=%d)", ErrConflict, b.lastID+1, d.FirstID, d.FinalID)
	}
	putLevels(b.bids, d.Bids)
	putLevels(b.asks, d.Asks)
	b.lastID = d.FinalID
	if !d.EventAt.IsZero() {
		b.updatedAt = d.EventAt.UTC()
	} else {
		b.updatedAt = time.Now().UTC()
	}
	return nil
}

// Snapshot copies the top `limit` bids/asks if the book is synced.
// Unsynced books return ErrConflict so callers cannot serve a broken copy.
func (b *LocalDepthBook) Snapshot(limit int) (*RawOrderBook, error) {
	if b == nil {
		return nil, fmt.Errorf("%w: nil order book", ErrUpstream)
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if !b.synced {
		return nil, fmt.Errorf("%w: order book not synced", ErrConflict)
	}
	limit = ClampOrderBookRawLimit(limit)
	out := &RawOrderBook{
		Symbol:    b.symbol,
		UpdateID:  b.lastID,
		FetchedAt: b.updatedAt,
		Bids:      topLevels(b.bids, limit, true),
		Asks:      topLevels(b.asks, limit, false),
		Live:      true,
		Source:    b.source,
	}
	if out.FetchedAt.IsZero() {
		out.FetchedAt = time.Now().UTC()
	}
	return out, nil
}

func putLevels(dst map[string]float64, levels []DepthLevel) {
	for _, lv := range levels {
		p := lv.Price
		if p == "" {
			continue
		}
		if lv.Quantity <= 0 {
			delete(dst, p)
			continue
		}
		dst[p] = lv.Quantity
	}
}

func topLevels(m map[string]float64, limit int, bids bool) []PriceLevel {
	type row struct {
		price float64
		qty   float64
	}
	rows := make([]row, 0, len(m))
	for ps, qty := range m {
		p, err := strconv.ParseFloat(ps, 64)
		if err != nil || p <= 0 || qty <= 0 {
			continue
		}
		rows = append(rows, row{price: p, qty: qty})
	}
	sort.Slice(rows, func(i, j int) bool {
		if bids {
			return rows[i].price > rows[j].price
		}
		return rows[i].price < rows[j].price
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]PriceLevel, 0, len(rows))
	for _, r := range rows {
		out = append(out, PriceLevel{Price: r.price, Quantity: r.qty})
	}
	return out
}
