package domain

import (
	"sync"
	"time"
)

const takerBucket = time.Minute

// TakerMemoryRetain is how long the live 1-minute book keeps bars.
// Startup reloads this span from SQLite for the liquid names.
const TakerMemoryRetain = 4 * time.Hour

// TakerBook keeps rolling 1-minute buy/sell notional (USDT) per venue+symbol.
type TakerBook struct {
	mu       sync.Mutex
	byKey    map[string]map[int64]TakerBucket // symbol|ex → bucketMs → bucket
	started  map[string]time.Time
	restored map[string]struct{}
	now      func() time.Time
}

// NewTakerBook constructs an empty store.
func NewTakerBook() *TakerBook {
	return &TakerBook{
		byKey:    map[string]map[int64]TakerBucket{},
		started:  map[string]time.Time{},
		restored: map[string]struct{}{},
		now:      time.Now,
	}
}

func takerKey(ex Exchange, symbol string) string {
	return string(ex) + "|" + NormalizeLiquidationSymbol(symbol)
}

// Record adds one aggressive fill into its 1-minute bucket.
func (b *TakerBook) Record(p TakerPrint) {
	if b == nil || p.Notional <= 0 || p.Time.IsZero() {
		return
	}
	p.Symbol = NormalizeLiquidationSymbol(p.Symbol)
	if p.Symbol == "" || p.Exchange == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	cut := now.Add(-TakerMemoryRetain)
	k := takerKey(p.Exchange, p.Symbol)
	if _, ok := b.started[k]; !ok {
		b.started[k] = p.Time.UTC()
	}
	ms := TruncateToBucket(p.Time, takerBucket).UnixMilli()
	m := b.byKey[k]
	if m == nil {
		m = map[int64]TakerBucket{}
		b.byKey[k] = m
	}
	cur := m[ms]
	prior := cur.BuyNotional + cur.SellNotional
	cur.Exchange = p.Exchange
	cur.Symbol = p.Symbol
	cur.Start = time.UnixMilli(ms).UTC()
	// A minute that already has notional and was not classified stays unknown.
	// Mixing a new sized print into that total would make whale+retail disagree
	// with the bar. A fresh minute, or one already sized, takes this print.
	classify := prior <= 0 || cur.SizeKnown
	switch p.Side {
	case TakerSideBuy:
		cur.BuyNotional += p.Notional
		if classify {
			AddTakerPrintSize(&cur, TakerSideBuy, p.Notional)
			cur.SizeKnown = true
		}
	case TakerSideSell:
		cur.SellNotional += p.Notional
		if classify {
			AddTakerPrintSize(&cur, TakerSideSell, p.Notional)
			cur.SizeKnown = true
		}
	}
	m[ms] = cur
	for t := range m {
		if t < cut.UnixMilli() {
			delete(m, t)
		}
	}
}

// Restored reports whether startup already asked SQLite for this pair.
func (b *TakerBook) Restored(ex Exchange, symbol string) bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.restored[takerKey(ex, symbol)]
	return ok
}

// MarkRestored records that SQLite was checked, including when it had no rows.
func (b *TakerBook) MarkRestored(ex Exchange, symbol string) {
	if b == nil {
		return
	}
	symbol = NormalizeLiquidationSymbol(symbol)
	if symbol == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.restored == nil {
		b.restored = map[string]struct{}{}
	}
	b.restored[takerKey(ex, symbol)] = struct{}{}
}

// LoadBuckets restores persisted minutes (startup).
func (b *TakerBook) LoadBuckets(buckets []TakerBucket) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.restored == nil {
		b.restored = map[string]struct{}{}
	}
	for _, rec := range buckets {
		rec.Symbol = NormalizeLiquidationSymbol(rec.Symbol)
		if rec.Symbol == "" || rec.Start.IsZero() {
			continue
		}
		k := takerKey(rec.Exchange, rec.Symbol)
		m := b.byKey[k]
		if m == nil {
			m = map[int64]TakerBucket{}
			b.byKey[k] = m
		}
		ms := TruncateToBucket(rec.Start, takerBucket).UnixMilli()
		cur := m[ms]
		prior := cur.BuyNotional + cur.SellNotional
		cur.Exchange = rec.Exchange
		cur.Symbol = rec.Symbol
		cur.Start = time.UnixMilli(ms).UTC()
		cur.BuyNotional += rec.BuyNotional
		cur.SellNotional += rec.SellNotional
		switch {
		case prior <= 0:
			cur.WhaleBuy = rec.WhaleBuy
			cur.WhaleSell = rec.WhaleSell
			cur.RetailBuy = rec.RetailBuy
			cur.RetailSell = rec.RetailSell
			cur.SizeKnown = rec.SizeKnown
		case cur.SizeKnown && rec.SizeKnown:
			cur.WhaleBuy += rec.WhaleBuy
			cur.WhaleSell += rec.WhaleSell
			cur.RetailBuy += rec.RetailBuy
			cur.RetailSell += rec.RetailSell
		default:
			cur.WhaleBuy = 0
			cur.WhaleSell = 0
			cur.RetailBuy = 0
			cur.RetailSell = 0
			cur.SizeKnown = false
		}
		m[ms] = cur
		if t0, ok := b.started[k]; !ok || rec.Start.Before(t0) {
			b.started[k] = rec.Start.UTC()
		}
		b.restored[k] = struct{}{}
	}
}

// Snapshot windows for one venue+symbol.
func (b *TakerBook) Snapshot(ex Exchange, symbol string) TakerVenueFlow {
	symbol = NormalizeLiquidationSymbol(symbol)
	if b == nil {
		return TakerVenueFlow{Exchange: ex, Symbol: symbol, Windows: []TakerWindowFlow{}}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	k := takerKey(ex, symbol)
	m := b.byKey[k]
	buckets := make([]TakerBucket, 0, len(m))
	for _, rec := range m {
		buckets = append(buckets, rec)
	}
	return BuildTakerVenueFlow(ex, symbol, buckets, now, b.started[k])
}

// Buckets returns a copy of stored 1-minute bars for one venue+symbol.
func (b *TakerBook) Buckets(ex Exchange, symbol string) []TakerBucket {
	symbol = NormalizeLiquidationSymbol(symbol)
	if b == nil || symbol == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	m := b.byKey[takerKey(ex, symbol)]
	out := make([]TakerBucket, 0, len(m))
	for _, rec := range m {
		out = append(out, rec)
	}
	return out
}

// StartedAt is when we first saw flow for this pair (zero if never).
func (b *TakerBook) StartedAt(ex Exchange, symbol string) time.Time {
	if b == nil {
		return time.Time{}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.started[takerKey(ex, symbol)]
}
