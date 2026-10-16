package portfolio

import (
	"context"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	// moveQuoteTTL is how long a minute close (or a confirmed miss) is reused.
	// Portfolio screens poll faster than this, so a cold read does not refetch
	// the same minute on every tick.
	moveQuoteTTL = 45 * time.Second
	// moveQuoteMax bounds the in-process minute cache. A full cache drops the
	// soonest expiry. Expired keys are removed on read so they cannot evict a fresh minute.
	moveQuoteMax = 2048
	// moveChunkSpan is the longest open-to-open gap packed into one candle request.
	// 1h and 4h (three hours apart) share a request. 24h is far enough to need its own.
	// The span stays under Coinbase's 300-bar cap after the one-minute end pad.
	// A 1000-bar span would drop Coinbase's older minutes and look like a missing close.
	moveChunkSpan = 278 * time.Minute
	// moveEquityLimit is the latest one-minute window asked from an equity venue.
	// Public equity charts only keep about a day of 1m bars, and that adapter
	// ignores start and end. A minute that is not in the result stays missing.
	moveEquityLimit = 1000
)

// CandleSource loads OHLCV bars. market.Service satisfies it.
type CandleSource interface {
	GetCandles(ctx context.Context, exchange, symbol, interval string, limit int, start, end *time.Time) ([]domain.Candle, error)
}

type moveAnchor struct {
	exchange domain.Exchange
	symbol   string
	price    float64
	at       time.Time
}

type moveNeed struct {
	exchange domain.Exchange
	symbol   string
	minute   time.Time
}

type moveQuote struct {
	price   float64
	found   bool
	expires time.Time
}

type moveQuoteCache struct {
	mu    sync.Mutex
	items map[string]moveQuote
}

func newMoveQuoteCache() *moveQuoteCache {
	return &moveQuoteCache{items: map[string]moveQuote{}}
}

func (c *moveQuoteCache) get(key string, now time.Time) (moveQuote, bool) {
	if c == nil {
		return moveQuote{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	q, ok := c.items[key]
	if !ok || !now.Before(q.expires) {
		if ok {
			delete(c.items, key)
		}
		return moveQuote{}, false
	}
	return q, true
}

func (c *moveQuoteCache) set(key string, q moveQuote) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.items[key]
	if !exists && len(c.items) >= moveQuoteMax {
		var drop string
		var soon time.Time
		for k, v := range c.items {
			if drop == "" || v.expires.Before(soon) {
				drop = k
				soon = v.expires
			}
		}
		if drop != "" {
			delete(c.items, drop)
		}
	}
	c.items[key] = q
}

func (s *Service) nowForMoves() time.Time {
	if s != nil && s.moveClock != nil {
		return s.moveClock().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) stampTrade(ctx context.Context, tr *domain.Trade) *domain.Trade {
	if tr == nil {
		return nil
	}
	one := []domain.Trade{*tr}
	s.attachTradeMoves(ctx, one)
	out := one[0]
	return &out
}

func (s *Service) attachTradeMoves(ctx context.Context, trades []domain.Trade) {
	if len(trades) == 0 {
		return
	}
	anchors := make([]moveAnchor, len(trades))
	for i := range trades {
		anchors[i] = moveAnchor{
			exchange: trades[i].Exchange,
			symbol:   trades[i].Symbol,
			price:    trades[i].Price,
			at:       trades[i].CreatedAt,
		}
	}
	moved := s.fillMoves(ctx, anchors)
	for i := range trades {
		if i < len(moved) {
			trades[i].Moves = moved[i]
		}
	}
}

// openLotAnchor is the newest open lot. An empty anchor makes every window missing,
// which is what a position with no lot gets. Average cost is not a fill.
func openLotAnchor(pos domain.Position, lots []domain.TaxLot) moveAnchor {
	lot, ok := domain.LatestOpenLot(lots)
	if !ok || lot.Price <= 0 || lot.OpenedAt.IsZero() {
		return moveAnchor{}
	}
	ex := lot.Exchange
	if ex == "" {
		ex = pos.Exchange
	}
	sym := lot.Symbol
	if sym == "" {
		sym = pos.Symbol
	}
	return moveAnchor{exchange: ex, symbol: sym, price: lot.Price, at: lot.OpenedAt}
}

func (s *Service) fillMoves(ctx context.Context, anchors []moveAnchor) [][]domain.FillMove {
	now := time.Now().UTC()
	if s != nil {
		now = s.nowForMoves()
	}
	need := map[string]moveNeed{}
	for _, a := range anchors {
		if a.price <= 0 || a.at.IsZero() || a.symbol == "" {
			continue
		}
		for _, w := range domain.FillMoveWindows() {
			horizon := domain.FillMoveHorizon(a.at, w.Span)
			if !domain.FillMoveDue(horizon, now) {
				continue
			}
			minute := domain.FillMoveBarOpen(horizon)
			need[moveCacheKey(a.exchange, a.symbol, minute)] = moveNeed{
				exchange: a.exchange, symbol: a.symbol, minute: minute,
			}
		}
	}
	quotes := s.resolveMoveQuotes(ctx, need, now)
	out := make([][]domain.FillMove, len(anchors))
	for i, a := range anchors {
		anchor := a
		out[i] = domain.FillMoves(anchor.price, anchor.at, now, func(horizon time.Time) (float64, bool) {
			minute := domain.FillMoveBarOpen(horizon)
			q, ok := quotes[moveCacheKey(anchor.exchange, anchor.symbol, minute)]
			if !ok || !q.found || q.price <= 0 {
				return 0, false
			}
			return q.price, true
		})
	}
	return out
}

func (s *Service) resolveMoveQuotes(ctx context.Context, need map[string]moveNeed, now time.Time) map[string]moveQuote {
	out := make(map[string]moveQuote, len(need))
	pending := map[string]moveNeed{}
	for key, n := range need {
		if s != nil {
			if q, ok := s.moveQuotes.get(key, now); ok {
				out[key] = q
				continue
			}
		}
		pending[key] = n
	}
	if len(pending) == 0 {
		return out
	}
	if s == nil || s.candles == nil || ctx.Err() != nil {
		for key := range pending {
			out[key] = moveQuote{found: false}
		}
		return out
	}

	type group struct {
		exchange string
		symbol   string
		equity   bool
		minutes  []time.Time
		keys     []string
	}
	groups := map[string]*group{}
	for key, n := range pending {
		id := string(n.exchange) + "|" + n.symbol
		g := groups[id]
		if g == nil {
			g = &group{
				exchange: string(n.exchange),
				symbol:   n.symbol,
				equity:   domain.IsEquityExchange(n.exchange),
			}
			groups[id] = g
		}
		g.minutes = append(g.minutes, n.minute)
		g.keys = append(g.keys, key)
	}
	for _, g := range groups {
		fetched, cacheMisses := s.fetchMoveMinutes(ctx, g.exchange, g.symbol, g.equity, g.minutes, now)
		for i, key := range g.keys {
			minute := g.minutes[i].UTC().Truncate(time.Minute)
			q, ok := fetched[minute.Unix()]
			if !ok {
				q = moveQuote{found: false}
			}
			if cacheMisses && ctx.Err() == nil {
				stored := q
				stored.expires = now.Add(moveQuoteTTL)
				s.moveQuotes.set(key, stored)
			} else if ok && q.found && ctx.Err() == nil {
				stored := q
				stored.expires = now.Add(moveQuoteTTL)
				s.moveQuotes.set(key, stored)
			}
			out[key] = q
		}
	}
	return out
}

// fetchMoveMinutes returns closes keyed by unix minute.
// cacheMisses is false when the lookup was cancelled, so a cancel is not stored as a miss.
func (s *Service) fetchMoveMinutes(ctx context.Context, exchange, symbol string, equity bool, minutes []time.Time, now time.Time) (map[int64]moveQuote, bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	if equity {
		bars, err := s.candles.GetCandles(ctx, exchange, symbol, string(domain.Interval1m), moveEquityLimit, nil, nil)
		if err != nil || ctx.Err() != nil {
			return map[int64]moveQuote{}, ctx.Err() == nil && err != nil
		}
		return indexMoveBars(bars, now), true
	}
	acc := map[int64]moveQuote{}
	cacheMisses := true
	for _, chunk := range chunkMoveMinutes(minutes) {
		if ctx.Err() != nil {
			return acc, false
		}
		start := chunk[0]
		last := chunk[len(chunk)-1]
		end := last.Add(time.Minute)
		limit := int(last.Sub(start)/time.Minute) + 2
		if limit < 1 {
			limit = 1
		}
		if limit > 300 {
			limit = 300
		}
		bars, err := s.candles.GetCandles(ctx, exchange, symbol, string(domain.Interval1m), limit, &start, &end)
		if err != nil || ctx.Err() != nil {
			if ctx.Err() != nil {
				cacheMisses = false
			}
			continue
		}
		for k, q := range indexMoveBars(bars, now) {
			acc[k] = q
		}
	}
	return acc, cacheMisses
}

func indexMoveBars(bars []domain.Candle, now time.Time) map[int64]moveQuote {
	out := map[int64]moveQuote{}
	for _, bar := range bars {
		if !oneMinuteBar(bar) {
			continue
		}
		open := bar.OpenTime.UTC().Truncate(time.Minute)
		if !domain.FillMoveDue(open, now) {
			continue
		}
		px, ok := parseMoveClose(bar.Close)
		if !ok {
			continue
		}
		if _, exists := out[open.Unix()]; exists {
			continue
		}
		out[open.Unix()] = moveQuote{price: px, found: true}
	}
	return out
}

func oneMinuteBar(bar domain.Candle) bool {
	if bar.OpenTime.IsZero() {
		return false
	}
	if bar.CloseTime.IsZero() {
		return true
	}
	span := bar.CloseTime.Sub(bar.OpenTime)
	return span >= 0 && span <= 2*time.Minute
}

func parseMoveClose(raw string) (float64, bool) {
	px, err := strconv.ParseFloat(raw, 64)
	if err != nil || px <= 0 || math.IsNaN(px) || math.IsInf(px, 0) {
		return 0, false
	}
	return px, true
}

func chunkMoveMinutes(minutes []time.Time) [][]time.Time {
	if len(minutes) == 0 {
		return nil
	}
	uniq := make([]time.Time, 0, len(minutes))
	seen := map[int64]struct{}{}
	for _, m := range minutes {
		m = m.UTC().Truncate(time.Minute)
		if m.IsZero() {
			continue
		}
		if _, ok := seen[m.Unix()]; ok {
			continue
		}
		seen[m.Unix()] = struct{}{}
		uniq = append(uniq, m)
	}
	if len(uniq) == 0 {
		return nil
	}
	sort.Slice(uniq, func(i, j int) bool { return uniq[i].Before(uniq[j]) })
	out := [][]time.Time{{uniq[0]}}
	start := uniq[0]
	for _, m := range uniq[1:] {
		if m.Sub(start) > moveChunkSpan {
			out = append(out, []time.Time{m})
			start = m
			continue
		}
		out[len(out)-1] = append(out[len(out)-1], m)
	}
	return out
}

func moveCacheKey(ex domain.Exchange, symbol string, minute time.Time) string {
	return string(ex) + "|" + symbol + "|" + strconv.FormatInt(minute.UTC().Truncate(time.Minute).Unix(), 10)
}
