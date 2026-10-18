package portfolio

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// stressReplayChunkBars stays under Coinbase's 300-bar cap and Binance's 1000.
const stressReplayChunkBars = 240

// StressPeriod reprices the open book by each coin's close-to-close change
// between from and to. Nothing is saved. A coin with no close at either end
// stays unchanged and is named on the report.
func (s *Service) StressPeriod(ctx context.Context, clientID, portfolioID, ownerID, fromRaw, toRaw string) (*domain.StressReport, error) {
	now := time.Now().UTC()
	from, to, interval, err := domain.ParseStressWindow(fromRaw, toRaw, now)
	if err != nil {
		return nil, err
	}
	if s == nil || s.candles == nil {
		return nil, errPriceHistoryUnset
	}
	view, err := s.View(ctx, clientID, portfolioID, ownerID)
	if err != nil {
		return nil, err
	}
	held := groupReplayHoldings(*view)
	if len(held) == 0 {
		return nil, errNoOpenPosition
	}
	series := make([]domain.ReplaySeries, 0, len(held))
	for _, row := range held {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		bars, ferr := s.loadReplayBars(ctx, row.exchange, row.symbol, string(interval), from, to)
		item := domain.ReplaySeries{
			Asset: row.asset, Exchange: string(row.exchange), Symbol: row.symbol, Weight: row.weight,
		}
		if ferr != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			item.Failed = true
		} else {
			item.Bars = bars
		}
		series = append(series, item)
	}
	plan := domain.BuildReplayPlan(series, from, to, interval, now)
	if len(plan.Moves) == 0 {
		if plan.Series > 0 && plan.FetchFailures == plan.Series {
			return nil, errPriceHistoryRead
		}
		return nil, errNoReplayClose
	}
	rep, err := domain.StressPortfolio(*view, plan.Moves)
	if err != nil {
		return nil, err
	}
	rep.Replay = &plan.Replay
	rep.Note = domain.PortfolioReplayNote
	return &rep, nil
}

type replayHolding struct {
	asset    string
	exchange domain.Exchange
	symbol   string
	weight   float64
}

func groupReplayHoldings(v domain.PortfolioView) []replayHolding {
	type key struct {
		ex  string
		sym string
	}
	var order []key
	acc := map[key]*replayHolding{}
	add := func(ex domain.Exchange, symbol string, weight float64) {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" || weight <= 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
			return
		}
		if ex == "" {
			ex = domain.ExchangeBinance
		}
		k := key{string(ex), symbol}
		if row, ok := acc[k]; ok {
			row.weight += weight
			return
		}
		acc[k] = &replayHolding{asset: symbol, exchange: ex, symbol: symbol, weight: weight}
		order = append(order, k)
	}
	for _, p := range v.Positions {
		if p.Quantity <= 0 {
			continue
		}
		w := p.MarketValue
		if w <= 0 && p.MarkPrice > 0 {
			w = p.Quantity * p.MarkPrice
		}
		if w <= 0 && p.AvgCost > 0 {
			w = p.Quantity * p.AvgCost
		}
		if w <= 0 {
			w = p.Quantity
		}
		add(p.Exchange, p.Symbol, w)
	}
	for _, p := range v.MarginPositions {
		if p.Status != "" && p.Status != domain.MarginPositionOpen {
			continue
		}
		if p.Quantity <= 0 {
			continue
		}
		mark := p.MarkPrice
		if mark <= 0 {
			mark = p.EntryPrice
		}
		w := p.Quantity * mark
		if w <= 0 {
			w = p.Quantity
		}
		add(p.Exchange, p.Symbol, w)
	}
	out := make([]replayHolding, 0, len(order))
	for _, k := range order {
		out = append(out, *acc[k])
	}
	return out
}

// loadReplayBars reads the closes that bracket [from, to].
// Equities ignore a caller range and return the venue's latest window, so the
// ask is one unscoped read. Crypto is walked in short chunks so a long window
// is not cut off at one venue's page size.
func (s *Service) loadReplayBars(ctx context.Context, ex domain.Exchange, symbol, interval string, from, to time.Time) ([]domain.Candle, error) {
	if domain.IsEquityExchange(ex) {
		return s.candles.GetCandles(ctx, string(ex), symbol, interval, 1000, nil, nil)
	}
	step := domain.IntervalDuration(domain.CandleInterval(interval))
	if step <= 0 {
		return nil, fmt.Errorf("%w: interval %s", domain.ErrInvalidArgument, interval)
	}
	cursor := from.Add(-step)
	limitEnd := to
	seen := map[int64]struct{}{}
	var merged []domain.Candle
	for n := 0; cursor.Before(limitEnd) && n < 40; n++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		chunkEnd := cursor.Add(step * stressReplayChunkBars)
		if chunkEnd.After(limitEnd) {
			chunkEnd = limitEnd
		}
		if !chunkEnd.After(cursor) {
			break
		}
		reqStart, reqEnd := cursor, chunkEnd
		bars, err := s.candles.GetCandles(ctx, string(ex), symbol, interval, stressReplayChunkBars, &reqStart, &reqEnd)
		if err != nil {
			return nil, err
		}
		for _, b := range bars {
			key := b.OpenTime.UTC().UnixMilli()
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, b)
		}
		cursor = chunkEnd
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].OpenTime.Before(merged[j].OpenTime)
	})
	return merged, nil
}
