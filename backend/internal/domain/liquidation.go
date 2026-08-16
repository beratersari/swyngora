package domain

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	LiquidationSideLong  = "long"
	LiquidationSideShort = "short"

	LiquidationWindow5m  = "5m"
	LiquidationWindow1h  = "1h"
	LiquidationWindow4h  = "4h"
	LiquidationWindow24h = "24h"

	maxLiquidationEventsPerSymbol = 20000
	liquidationRetain             = 24 * time.Hour
)

// LiquidationWindows is the Coinglass-style lookback set.
var LiquidationWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{LiquidationWindow5m, 5 * time.Minute},
	{LiquidationWindow1h, time.Hour},
	{LiquidationWindow4h, 4 * time.Hour},
	{LiquidationWindow24h, 24 * time.Hour},
}

// LiquidationEvent is one forced close seen on a futures feed.
type LiquidationEvent struct {
	Exchange Exchange
	Symbol   string
	Side     string // long | short — the position that was liquidated
	Price    float64
	Quantity float64
	Notional float64
	Time     time.Time
}

// LiquidationHit is the largest event in a window.
type LiquidationHit struct {
	Exchange string
	Side     string
	Price    string
	Quantity string
	Notional string
	Time     time.Time
}

// LiquidationWindowTotals is long/short value, count, and the biggest hit.
type LiquidationWindowTotals struct {
	Window          string
	LongNotional    string
	ShortNotional   string
	TotalNotional   string
	Count           int
	Biggest         *LiquidationHit
	CoverageSeconds int64
	Complete        bool // false until this coin+venue has been tracked for the full window
}

// LiquidationSnapshot is per-coin totals for every lookback window.
type LiquidationSnapshot struct {
	Symbol          string
	Exchange        string // binance | bybit | all
	CollectingSince time.Time
	Live            bool
	VenueCount      int
	Windows         []LiquidationWindowTotals
}

// NormalizeLiquidationSymbol maps spot-style ids (BTC-USD) to USDT linear (BTCUSDT).
func NormalizeLiquidationSymbol(raw string) string {
	s := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(raw), "-", ""))
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, "USDT") || strings.HasSuffix(s, "USDC") {
		return s
	}
	if strings.HasSuffix(s, "USD") {
		return strings.TrimSuffix(s, "USD") + "USDT"
	}
	return s
}

// LiquidationSideFromBinanceOrder maps the liquidation *order* side to the
// position that was closed. A SELL force-order dumps a long; a BUY covers a short.
func LiquidationSideFromBinanceOrder(orderSide string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(orderSide)) {
	case "SELL":
		return LiquidationSideLong, nil
	case "BUY":
		return LiquidationSideShort, nil
	default:
		return "", fmt.Errorf("%w: liquidation order side must be BUY or SELL", ErrInvalidArgument)
	}
}

// LiquidationSideFromBybit maps Bybit allLiquidation S: Buy = long liquidated.
func LiquidationSideFromBybit(positionSide string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(positionSide)) {
	case "buy":
		return LiquidationSideLong, nil
	case "sell":
		return LiquidationSideShort, nil
	default:
		return "", fmt.Errorf("%w: bybit liquidation side must be Buy or Sell", ErrInvalidArgument)
	}
}

// ParseLiquidationExchange accepts binance, bybit, or all (empty = all).
func ParseLiquidationExchange(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" || s == "all" || s == "combined" {
		return "all", nil
	}
	if s == string(ExchangeBinance) || s == string(ExchangeBybit) {
		return s, nil
	}
	return "", fmt.Errorf("%w: exchange must be binance, bybit, or all", ErrInvalidArgument)
}

// SummarizeLiquidations folds events since cut into one window.
// coverage is how long the websocket was actually live for this coin+venue.
func SummarizeLiquidations(events []LiquidationEvent, windowID string, cut time.Time, coverage time.Duration) LiquidationWindowTotals {
	var dur time.Duration
	for _, w := range LiquidationWindows {
		if w.ID == windowID {
			dur = w.Dur
			break
		}
	}
	out := LiquidationWindowTotals{Window: windowID}
	if coverage < 0 {
		coverage = 0
	}
	if dur > 0 {
		if coverage >= dur {
			out.Complete = true
			out.CoverageSeconds = int64(dur.Seconds())
		} else {
			out.CoverageSeconds = int64(coverage.Seconds())
		}
	}
	var longN, shortN float64
	var biggest LiquidationEvent
	var hasBiggest bool
	for _, e := range events {
		if e.Time.Before(cut) {
			continue
		}
		out.Count++
		switch e.Side {
		case LiquidationSideLong:
			longN += e.Notional
		case LiquidationSideShort:
			shortN += e.Notional
		}
		if !hasBiggest || e.Notional > biggest.Notional {
			biggest = e
			hasBiggest = true
		}
	}
	out.LongNotional = formatQty(longN)
	out.ShortNotional = formatQty(shortN)
	out.TotalNotional = formatQty(longN + shortN)
	if hasBiggest {
		hit := LiquidationHit{
			Exchange: string(biggest.Exchange),
			Side:     biggest.Side,
			Price:    formatFixed(biggest.Price, decimalsForStep(biggest.Price/10000)+1),
			Quantity: formatQty(biggest.Quantity),
			Notional: formatQty(biggest.Notional),
			Time:     biggest.Time.UTC(),
		}
		out.Biggest = &hit
	}
	return out
}

// liveClock counts only time a websocket was actually connected.
type liveClock struct {
	accumulated  time.Duration
	sessionStart time.Time // zero while disconnected
}

func (c *liveClock) set(now time.Time, live bool) {
	if c == nil {
		return
	}
	if live {
		if c.sessionStart.IsZero() {
			c.sessionStart = now
		}
		return
	}
	if !c.sessionStart.IsZero() {
		c.accumulated += now.Sub(c.sessionStart)
		c.sessionStart = time.Time{}
	}
}

func (c *liveClock) elapsed(now time.Time) time.Duration {
	if c == nil {
		return 0
	}
	d := c.accumulated
	if !c.sessionStart.IsZero() {
		d += now.Sub(c.sessionStart)
	}
	if d < 0 {
		return 0
	}
	return d
}

// LiquidationBook keeps a rolling 24h of futures liquidations in memory.
type LiquidationBook struct {
	mu         sync.Mutex
	bySym      map[string][]LiquidationEvent
	watchSince map[string]time.Time // symbol|exchange → first watch
	venueSince map[Exchange]time.Time
	venueClock map[Exchange]*liveClock
	watchClock map[string]*liveClock
	now        func() time.Time
	live       map[Exchange]bool
}

func watchKey(symbol string, ex Exchange) string {
	return symbol + "|" + string(ex)
}

// NewLiquidationBook constructs an empty rolling store.
func NewLiquidationBook() *LiquidationBook {
	return &LiquidationBook{
		bySym:      map[string][]LiquidationEvent{},
		watchSince: map[string]time.Time{},
		venueSince: map[Exchange]time.Time{},
		venueClock: map[Exchange]*liveClock{},
		watchClock: map[string]*liveClock{},
		now:        time.Now,
		live:       map[Exchange]bool{},
	}
}

// SetLive marks a venue stream as connected or dropped.
// The first time a venue comes live, that becomes its all-symbol start
// (Binance USD-M forceOrder covers every coin from that moment).
func (b *LiquidationBook) SetLive(ex Exchange, live bool) {
	if b == nil {
		return
	}
	b.mu.Lock()
	now := b.now().UTC()
	b.live[ex] = live
	if b.venueClock[ex] == nil {
		b.venueClock[ex] = &liveClock{}
	}
	b.venueClock[ex].set(now, live)
	if live {
		if _, ok := b.venueSince[ex]; !ok {
			b.venueSince[ex] = now
		}
	}
	suffix := "|" + string(ex)
	for k, c := range b.watchClock {
		if strings.HasSuffix(k, suffix) {
			c.set(now, live)
		}
	}
	b.mu.Unlock()
}

// MarkWatch records when we started listening to one coin on one venue.
// The first call wins so reconnects do not reset coverage.
func (b *LiquidationBook) MarkWatch(ex Exchange, symbol string) {
	if b == nil {
		return
	}
	symbol = NormalizeLiquidationSymbol(symbol)
	if symbol == "" || ex == "" {
		return
	}
	b.mu.Lock()
	b.markWatchLocked(ex, symbol, b.now().UTC())
	b.mu.Unlock()
}

func (b *LiquidationBook) markWatchLocked(ex Exchange, symbol string, at time.Time) {
	k := watchKey(symbol, ex)
	if _, ok := b.watchSince[k]; ok {
		return
	}
	b.watchSince[k] = at
	c := &liveClock{}
	if b.live[ex] {
		c.sessionStart = at
	}
	b.watchClock[k] = c
}

// Record appends one event and drops anything older than 24h.
func (b *LiquidationBook) Record(e LiquidationEvent) {
	if b == nil || e.Symbol == "" || e.Notional <= 0 || e.Time.IsZero() {
		return
	}
	e.Symbol = NormalizeLiquidationSymbol(e.Symbol)
	e.Time = e.Time.UTC()
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	// Events do not start a clock. Coverage is only live-socket time from
	// SetLive / MarkWatch. A first print must not reset Binance venue coverage.
	cut := now.Add(-liquidationRetain)
	list := append(b.bySym[e.Symbol], e)
	if len(list) > 1 && list[0].Time.Before(cut) {
		kept := list[:0]
		for _, ev := range list {
			if !ev.Time.Before(cut) {
				kept = append(kept, ev)
			}
		}
		list = kept
	}
	if len(list) > maxLiquidationEventsPerSymbol {
		list = list[len(list)-maxLiquidationEventsPerSymbol:]
	}
	b.bySym[e.Symbol] = list
}

func (b *LiquidationBook) startOfLocked(symbol string, ex Exchange) time.Time {
	// Binance USD-M all-market stream covers every symbol from venue connect.
	if ex == ExchangeBinance {
		return b.venueSince[ex]
	}
	if t, ok := b.watchSince[watchKey(symbol, ex)]; ok {
		return t
	}
	return time.Time{}
}

func (b *LiquidationBook) trackingStartLocked(symbol, exchange string) time.Time {
	if exchange != "all" {
		return b.startOfLocked(symbol, Exchange(exchange))
	}
	var latest time.Time
	any := false
	for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit} {
		t := b.startOfLocked(symbol, ex)
		if t.IsZero() {
			continue
		}
		if !any || t.After(latest) {
			latest = t
			any = true
		}
	}
	return latest
}

func (b *LiquidationBook) coverageLocked(symbol, exchange string, now time.Time) time.Duration {
	if exchange != "all" {
		return b.coverageOneLocked(symbol, Exchange(exchange), now)
	}
	var minCov time.Duration
	any := false
	for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit} {
		if b.startOfLocked(symbol, ex).IsZero() {
			continue
		}
		c := b.coverageOneLocked(symbol, ex, now)
		if !any || c < minCov {
			minCov = c
			any = true
		}
	}
	return minCov
}

func (b *LiquidationBook) coverageOneLocked(symbol string, ex Exchange, now time.Time) time.Duration {
	// Binance USD-M is all-market: count venue-live time, not first print.
	if ex == ExchangeBinance {
		return b.venueClock[ex].elapsed(now)
	}
	if c := b.watchClock[watchKey(symbol, ex)]; c != nil {
		return c.elapsed(now)
	}
	return 0
}

// Snapshot totals for every lookback window. exchange is binance, bybit, or all.
func (b *LiquidationBook) Snapshot(exchange, symbol string) *LiquidationSnapshot {
	symbol = NormalizeLiquidationSymbol(symbol)
	out := &LiquidationSnapshot{
		Symbol:          symbol,
		Exchange:        exchange,
		Windows:         []LiquidationWindowTotals{},
		CollectingSince: time.Time{},
	}
	if b == nil {
		return out
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	started := b.trackingStartLocked(symbol, exchange)
	out.CollectingSince = started
	liveCount := 0
	if exchange == "all" {
		for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit} {
			if b.live[ex] {
				liveCount++
			}
		}
	} else if b.live[Exchange(exchange)] {
		liveCount = 1
	}
	out.Live = liveCount > 0
	out.VenueCount = liveCount
	list := b.bySym[symbol]
	filtered := make([]LiquidationEvent, 0, len(list))
	for _, e := range list {
		if exchange != "all" && string(e.Exchange) != exchange {
			continue
		}
		filtered = append(filtered, e)
	}
	cov := b.coverageLocked(symbol, exchange, now)
	for _, w := range LiquidationWindows {
		out.Windows = append(out.Windows, SummarizeLiquidations(filtered, w.ID, now.Add(-w.Dur), cov))
	}
	return out
}

// Events returns copies of stored prints for one coin (newest last) since cutoff.
func (b *LiquidationBook) Events(exchange, symbol string, since time.Time) []LiquidationEvent {
	symbol = NormalizeLiquidationSymbol(symbol)
	if b == nil || symbol == "" {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	list := b.bySym[symbol]
	out := make([]LiquidationEvent, 0, len(list))
	for _, e := range list {
		if exchange != "" && exchange != "all" && string(e.Exchange) != exchange {
			continue
		}
		if !since.IsZero() && e.Time.Before(since) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// RecentLarge returns liquidations since cutoff that meet a notional floor.
func (b *LiquidationBook) RecentLarge(since time.Time, minNotional float64) []LiquidationEvent {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]LiquidationEvent, 0, 32)
	for _, list := range b.bySym {
		for _, e := range list {
			if e.Notional < minNotional {
				continue
			}
			if !since.IsZero() && e.Time.Before(since) {
				continue
			}
			out = append(out, e)
		}
	}
	return out
}
