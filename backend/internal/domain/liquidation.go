package domain

import (
	"fmt"
	"sort"
	"strconv"
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
	LiquidationWindow12h = "12h"
	LiquidationWindow24h = "24h"

	maxLiquidationEventsPerSymbol   = 20000
	liquidationRetain               = 24 * time.Hour
	defaultLiquidationOverviewLimit = 50
	maxLiquidationOverviewLimit     = 100
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

// LiquidationOverviewWindows is the CoinGlass-style market desk set (no 5m).
var LiquidationOverviewWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{LiquidationWindow1h, time.Hour},
	{LiquidationWindow4h, 4 * time.Hour},
	{LiquidationWindow12h, 12 * time.Hour},
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
	Feed            LiquidationFeed
}

// LiquidationCoinTile is one coin's totals for the selected overview window.
type LiquidationCoinTile struct {
	Symbol        string
	Base          string
	LongNotional  string
	ShortNotional string
	TotalNotional string
	Count         int
	Biggest       *LiquidationHit
}

// LiquidationOverview is market-wide cards plus ranked coins for a treemap.
type LiquidationOverview struct {
	Exchange        string // binance | bybit | all
	CoinWindow      string
	CollectingSince time.Time
	Live            bool
	VenueCount      int
	Windows         []LiquidationWindowTotals
	Coins           []LiquidationCoinTile
	Feed            LiquidationFeed
}

// LiquidationCoverage is durable live-socket time for one venue or one pair.
// Empty Symbol is venue-wide (Binance USD-M all-market).
type LiquidationCoverage struct {
	Exchange   Exchange
	Symbol     string
	FirstWatch time.Time
	Live       time.Duration
	LastEvent  time.Time
	LastSeen   time.Time
	LastSaved  time.Time
	Gaps       []LiquidationGap
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

// LiquidationBaseAsset strips a linear quote suffix (BTCUSDT → BTC).
func LiquidationBaseAsset(symbol string) string {
	s := NormalizeLiquidationSymbol(symbol)
	for _, q := range []string{"USDT", "USDC"} {
		if strings.HasSuffix(s, q) && len(s) > len(q) {
			return strings.TrimSuffix(s, q)
		}
	}
	return s
}

// ParseLiquidationOverviewWindow accepts 1h, 4h, 12h, or 24h (empty = 24h).
func ParseLiquidationOverviewWindow(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return LiquidationWindow24h, nil
	}
	switch s {
	case LiquidationWindow1h, LiquidationWindow4h, LiquidationWindow12h, LiquidationWindow24h:
		return s, nil
	default:
		return "", fmt.Errorf("%w: window must be 1h, 4h, 12h, or 24h", ErrInvalidArgument)
	}
}

// ClampLiquidationOverviewLimit bounds the treemap coin count.
func ClampLiquidationOverviewLimit(n int) int {
	if n <= 0 {
		return defaultLiquidationOverviewLimit
	}
	if n > maxLiquidationOverviewLimit {
		return maxLiquidationOverviewLimit
	}
	return n
}

// LiquidationWindowDuration is the lookback for a window id (per-coin or overview).
func LiquidationWindowDuration(id string) time.Duration {
	for _, w := range LiquidationWindows {
		if w.ID == id {
			return w.Dur
		}
	}
	for _, w := range LiquidationOverviewWindows {
		if w.ID == id {
			return w.Dur
		}
	}
	return 0
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
	dur := LiquidationWindowDuration(windowID)
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
	lastEvent  map[Exchange]time.Time
	lastSeen   map[Exchange]time.Time
	gaps       map[Exchange][]LiquidationGap
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
		lastEvent:  map[Exchange]time.Time{},
		lastSeen:   map[Exchange]time.Time{},
		gaps:       map[Exchange][]LiquidationGap{},
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
	was := b.live[ex]
	b.live[ex] = live
	if b.venueClock[ex] == nil {
		b.venueClock[ex] = &liveClock{}
	}
	b.venueClock[ex].set(now, live)
	if live {
		if _, ok := b.venueSince[ex]; !ok {
			b.venueSince[ex] = now
		}
		if b.lastSeen == nil {
			b.lastSeen = map[Exchange]time.Time{}
		}
		b.lastSeen[ex] = now
		b.closeOpenGapLocked(ex, now)
	} else if was {
		b.recordGapLocked(ex, now, time.Time{})
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
	if b.lastEvent == nil {
		b.lastEvent = map[Exchange]time.Time{}
	}
	if t, ok := b.lastEvent[e.Exchange]; !ok || e.Time.After(t) {
		b.lastEvent[e.Exchange] = e.Time
	}
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
	// Combined never borrows the other venue's start. Both must be tracking.
	var latest time.Time
	for _, ex := range liquidationVenues() {
		t := b.startOfLocked(symbol, ex)
		if t.IsZero() {
			return time.Time{}
		}
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}
	return latest
}

func (b *LiquidationBook) coverageLocked(symbol, exchange string, now time.Time) time.Duration {
	if exchange != "all" {
		return b.coverageOneLocked(symbol, Exchange(exchange), now)
	}
	// Combined uses the shorter live clock. A venue that never started is 0,
	// not "use the other exchange instead".
	var minCov time.Duration
	for i, ex := range liquidationVenues() {
		if b.startOfLocked(symbol, ex).IsZero() {
			return 0
		}
		c := b.coverageOneLocked(symbol, ex, now)
		if i == 0 || c < minCov {
			minCov = c
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
	want := liquidationVenues()
	if exchange != "all" {
		want = []Exchange{Exchange(exchange)}
	}
	for _, ex := range want {
		if b.effectivelyLiveLocked(ex, now) {
			liveCount++
		}
	}
	out.VenueCount = liveCount
	out.Live = liveCount == len(want)
	out.Feed = b.feedLocked(exchange, now)
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
	return b.EventsSince(exchange, symbol, since)
}

// EventsSince returns prints since cutoff. Empty / "all" symbol includes every coin.
func (b *LiquidationBook) EventsSince(exchange, symbol string, since time.Time) []LiquidationEvent {
	if b == nil {
		return nil
	}
	symbol = NormalizeLiquidationSymbol(symbol)
	if symbol == "ALL" {
		symbol = ""
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]LiquidationEvent, 0, 64)
	for sym, list := range b.bySym {
		if symbol != "" && sym != symbol {
			continue
		}
		for _, e := range list {
			if exchange != "" && exchange != "all" && string(e.Exchange) != exchange {
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

func (b *LiquidationBook) marketStartLocked(exchange string) time.Time {
	if exchange != "all" {
		return b.venueSince[Exchange(exchange)]
	}
	var latest time.Time
	for _, ex := range liquidationVenues() {
		t := b.venueSince[ex]
		if t.IsZero() {
			return time.Time{}
		}
		if latest.IsZero() || t.After(latest) {
			latest = t
		}
	}
	return latest
}

func (b *LiquidationBook) marketCoverageLocked(exchange string, now time.Time) time.Duration {
	if exchange != "all" {
		return b.venueClock[Exchange(exchange)].elapsed(now)
	}
	var minCov time.Duration
	for i, ex := range liquidationVenues() {
		if b.venueSince[ex].IsZero() {
			return 0
		}
		c := b.venueClock[ex].elapsed(now)
		if i == 0 || c < minCov {
			minCov = c
		}
	}
	return minCov
}

// Overview sums every tracked coin for the desk windows and ranks coins
// in coinWindow for a treemap (largest total notional first).
func (b *LiquidationBook) Overview(exchange, coinWindow string, limit int) *LiquidationOverview {
	limit = ClampLiquidationOverviewLimit(limit)
	if coinWindow == "" {
		coinWindow = LiquidationWindow24h
	}
	out := &LiquidationOverview{
		Exchange:   exchange,
		CoinWindow: coinWindow,
		Windows:    []LiquidationWindowTotals{},
		Coins:      []LiquidationCoinTile{},
	}
	if b == nil {
		return out
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now().UTC()
	out.CollectingSince = b.marketStartLocked(exchange)
	liveCount := 0
	want := liquidationVenues()
	if exchange != "all" {
		want = []Exchange{Exchange(exchange)}
	}
	for _, ex := range want {
		if b.effectivelyLiveLocked(ex, now) {
			liveCount++
		}
	}
	out.VenueCount = liveCount
	out.Live = liveCount == len(want)
	out.Feed = b.feedLocked(exchange, now)

	bySym := make(map[string][]LiquidationEvent, len(b.bySym))
	all := make([]LiquidationEvent, 0, 256)
	for sym, list := range b.bySym {
		for _, e := range list {
			if exchange != "all" && string(e.Exchange) != exchange {
				continue
			}
			bySym[sym] = append(bySym[sym], e)
			all = append(all, e)
		}
	}

	cov := b.marketCoverageLocked(exchange, now)
	for _, w := range LiquidationOverviewWindows {
		out.Windows = append(out.Windows, SummarizeLiquidations(all, w.ID, now.Add(-w.Dur), cov))
	}

	coinDur := LiquidationWindowDuration(coinWindow)
	if coinDur <= 0 {
		coinDur = 24 * time.Hour
	}
	cut := now.Add(-coinDur)
	type ranked struct {
		tile  LiquidationCoinTile
		total float64
	}
	rows := make([]ranked, 0, len(bySym))
	for sym, list := range bySym {
		tot := SummarizeLiquidations(list, coinWindow, cut, cov)
		if tot.Count == 0 {
			continue
		}
		total, _ := strconv.ParseFloat(tot.TotalNotional, 64)
		rows = append(rows, ranked{
			tile: LiquidationCoinTile{
				Symbol:        sym,
				Base:          LiquidationBaseAsset(sym),
				LongNotional:  tot.LongNotional,
				ShortNotional: tot.ShortNotional,
				TotalNotional: tot.TotalNotional,
				Count:         tot.Count,
				Biggest:       tot.Biggest,
			},
			total: total,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].total != rows[j].total {
			return rows[i].total > rows[j].total
		}
		return rows[i].tile.Symbol < rows[j].tile.Symbol
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out.Coins = make([]LiquidationCoinTile, 0, len(rows))
	for _, r := range rows {
		out.Coins = append(out.Coins, r.tile)
	}
	return out
}

func clampLiveCoverage(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > liquidationRetain {
		return liquidationRetain
	}
	return d
}

// RestoreTracking seeds first-watch and accumulated live time from durable
// history so 1h/4h/12h/24h windows stay usable after a process restart.
// Empty symbol is venue-wide. Does not start a live session.
func (b *LiquidationBook) RestoreTracking(ex Exchange, symbol string, first time.Time, live time.Duration) {
	if b == nil || ex == "" || first.IsZero() {
		return
	}
	first = first.UTC()
	live = clampLiveCoverage(live)
	symbol = NormalizeLiquidationSymbol(symbol)
	b.mu.Lock()
	defer b.mu.Unlock()
	if symbol == "" {
		if t, ok := b.venueSince[ex]; !ok || first.Before(t) {
			b.venueSince[ex] = first
		}
		if b.venueClock[ex] == nil {
			b.venueClock[ex] = &liveClock{}
		}
		if b.venueClock[ex].accumulated < live {
			b.venueClock[ex].accumulated = live
		}
		return
	}
	k := watchKey(symbol, ex)
	if t, ok := b.watchSince[k]; !ok || first.Before(t) {
		b.watchSince[k] = first
	}
	if b.watchClock[k] == nil {
		b.watchClock[k] = &liveClock{}
	}
	if b.watchClock[k].accumulated < live {
		b.watchClock[k].accumulated = live
	}
}

// CoverageSnapshot is venue and per-pair live time for durable save.
func (b *LiquidationBook) CoverageSnapshot(now time.Time) []LiquidationCoverage {
	if b == nil {
		return nil
	}
	if now.IsZero() {
		now = b.now().UTC()
	} else {
		now = now.UTC()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]LiquidationCoverage, 0, len(b.venueSince)+len(b.watchSince))
	for ex, first := range b.venueSince {
		out = append(out, LiquidationCoverage{
			Exchange:   ex,
			FirstWatch: first,
			Live:       clampLiveCoverage(b.venueClock[ex].elapsed(now)),
			LastEvent:  b.lastEvent[ex],
			LastSeen:   b.lastSeen[ex],
			LastSaved:  now,
			Gaps:       append([]LiquidationGap(nil), b.gaps[ex]...),
		})
	}
	for k, first := range b.watchSince {
		sym, ex, ok := splitWatchKey(k)
		if !ok {
			continue
		}
		out = append(out, LiquidationCoverage{
			Exchange:   ex,
			Symbol:     sym,
			FirstWatch: first,
			Live:       clampLiveCoverage(b.watchClock[k].elapsed(now)),
		})
	}
	return out
}

func splitWatchKey(k string) (symbol string, ex Exchange, ok bool) {
	i := strings.LastIndexByte(k, '|')
	if i <= 0 || i == len(k)-1 {
		return "", "", false
	}
	return k[:i], Exchange(k[i+1:]), true
}

// CoverageFromEvents rebuilds first-watch + live span from stored prints
// when no coverage rows exist yet (upgrade path).
func CoverageFromEvents(events []LiquidationEvent, now time.Time) []LiquidationCoverage {
	type span struct{ first, last time.Time }
	venues := map[Exchange]span{}
	pairs := map[string]span{}
	for _, e := range events {
		if e.Exchange == "" || e.Time.IsZero() {
			continue
		}
		t := e.Time.UTC()
		vs := venues[e.Exchange]
		if vs.first.IsZero() || t.Before(vs.first) {
			vs.first = t
		}
		if vs.last.IsZero() || t.After(vs.last) {
			vs.last = t
		}
		venues[e.Exchange] = vs
		sym := NormalizeLiquidationSymbol(e.Symbol)
		if sym == "" {
			continue
		}
		k := watchKey(sym, e.Exchange)
		ps := pairs[k]
		if ps.first.IsZero() || t.Before(ps.first) {
			ps.first = t
		}
		if ps.last.IsZero() || t.After(ps.last) {
			ps.last = t
		}
		pairs[k] = ps
	}
	out := make([]LiquidationCoverage, 0, len(venues)+len(pairs))
	spanLive := func(s span) time.Duration {
		live := s.last.Sub(s.first)
		if !now.IsZero() && now.Sub(s.last) >= 0 && now.Sub(s.last) < time.Minute {
			live = now.Sub(s.first)
		}
		return clampLiveCoverage(live)
	}
	for ex, s := range venues {
		out = append(out, LiquidationCoverage{
			Exchange:   ex,
			FirstWatch: s.first,
			Live:       spanLive(s),
			LastEvent:  s.last,
		})
	}
	for k, s := range pairs {
		sym, ex, ok := splitWatchKey(k)
		if !ok {
			continue
		}
		out = append(out, LiquidationCoverage{
			Exchange:   ex,
			Symbol:     sym,
			FirstWatch: s.first,
			Live:       spanLive(s),
		})
	}
	return out
}
