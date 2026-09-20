package market

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const bookLiquidityDisclaimer = "Splits a resting-size change into traded (aggressive prints at that price) vs pulled (cancelled / no matching print) vs added. Spot books are 1-minute stored samples of the top grouped levels; prints are spot aggTrades (Binance) or recent public trades (Bybit / Coinbase). Not L3 order-by-order. Informational only — not financial advice."

// GetBookLiquidity attributes bid/ask size changes as traded, pulled, or added.
func (s *Service) GetBookLiquidity(ctx context.Context, exchange, symbol, window string, start, end *time.Time, minPrice, maxPrice float64) (*domain.BookLiquidityReport, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	if minPrice < 0 || maxPrice < 0 || (minPrice > 0 && maxPrice > 0 && maxPrice < minPrice) {
		return nil, fmt.Errorf("%w: minPrice must be <= maxPrice", domain.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	from, to, windowID, err := domain.ResolveBookLiquidityRange(window, start, end, now)
	if err != nil {
		return nil, err
	}
	want, err := bookLiquidityVenues(exchange)
	if err != nil {
		return nil, err
	}
	exLabel := "all"
	if len(want) == 1 {
		exLabel = string(want[0])
	}
	out := &domain.BookLiquidityReport{
		Symbol:   domain.NormalizeSymbol(domain.ExchangeBinance, symbol),
		Exchange: exLabel,
		Window:   windowID,
		From:     from,
		To:       to,
		AsOf:     now,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		Venues:   make([]domain.BookLiquidityVenue, 0, len(want)),
		Note:     bookLiquidityDisclaimer,
	}

	type fetched struct {
		ex     domain.Exchange
		sym    string
		snaps  []domain.BookHistorySnapshot
		prints []domain.TakerPrint
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			sym := normalizeSymbolForExchange(v, symbol)
			s.noteBook(v, sym)
			got[i] = fetched{
				ex:     v,
				sym:    sym,
				snaps:  s.bookLiquiditySnaps(ctx, v, sym, from, to),
				prints: s.fetchSpotPrintRange(ctx, v, sym, from, to),
			}
		}(i, v)
	}
	wg.Wait()

	for _, f := range got {
		out.Venues = append(out.Venues, domain.BuildBookLiquidityVenue(f.ex, f.sym, f.snaps, f.prints, minPrice, maxPrice, from, to))
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if len(want) > 1 {
		out.Combined = domain.CombineBookLiquidityVenues(out.Symbol, out.Venues, from, to)
	}
	out.Summary = domain.ExplainBookLiquidityReport(*out)
	return out, nil
}

func (s *Service) bookLiquiditySnaps(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) []domain.BookHistorySnapshot {
	var snaps []domain.BookHistorySnapshot
	if s != nil && s.bookHist != nil {
		rows, err := s.bookHist.ListLadders(ctx, domain.BookHistoryQuery{
			Exchange: string(ex), Symbol: symbol, From: from.Add(-time.Minute), To: to.Add(time.Minute),
			Limit: domain.MaxBookHistoryLimit,
		})
		if err == nil {
			snaps = rows
		}
	}
	if len(snaps) < 2 {
		if heat := s.heatmapSnapshots(ex, symbol, from, to); len(heat) > len(snaps) {
			snaps = heat
		}
	}
	if live := s.liveBookSnapshot(ctx, ex, symbol, to); live != nil {
		newest := time.Time{}
		for _, row := range snaps {
			if row.SampledAt.After(newest) {
				newest = row.SampledAt
			}
		}
		if len(snaps) < 2 || live.SampledAt.After(newest.Add(90*time.Second)) {
			snaps = append(snaps, *live)
		}
	}
	return snaps
}

func (s *Service) heatmapSnapshots(ex domain.Exchange, symbol string, from, to time.Time) []domain.BookHistorySnapshot {
	if s == nil || s.heat == nil {
		return nil
	}
	view := s.heat.View(string(ex), symbol, to.Sub(from)+time.Minute)
	if len(view.Columns) == 0 {
		return nil
	}
	tick, _ := strconv.ParseFloat(view.GroupSize, 64)
	out := make([]domain.BookHistorySnapshot, 0, len(view.Columns))
	for _, col := range view.Columns {
		t := col.Time.UTC()
		if t.Before(from.Add(-time.Minute)) || t.After(to.Add(time.Minute)) {
			continue
		}
		snap := domain.BookHistorySnapshot{
			Exchange:  ex,
			Symbol:    symbol,
			SampledAt: t,
			GroupSize: tick,
			Mid:       parseFloatOrZero(col.Mid),
			Bids:      heatLevelsToBook(col.Bids),
			Asks:      heatLevelsToBook(col.Asks),
			Complete:  true,
		}
		snap.BidNotional = domainBookSideNotional(snap.Bids)
		snap.AskNotional = domainBookSideNotional(snap.Asks)
		out = append(out, snap)
	}
	return out
}

func (s *Service) liveBookSnapshot(ctx context.Context, ex domain.Exchange, symbol string, at time.Time) *domain.BookHistorySnapshot {
	p, err := s.port(ex)
	if err != nil || p == nil {
		return nil
	}
	raw, err := p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: symbol, Limit: domain.MaxOrderBookRawLimit})
	if err != nil || raw == nil {
		return nil
	}
	snap := domain.CaptureBookHistory(ex, *raw, at)
	return &snap
}

func (s *Service) fetchSpotPrintRange(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) []domain.TakerPrint {
	p, err := s.port(ex)
	if err != nil || p == nil {
		return nil
	}
	sp, ok := p.(domain.SpotPrintPort)
	if !ok {
		return nil
	}
	got, err := sp.ListSpotPrints(ctx, symbol, from, to)
	if err != nil {
		return nil
	}
	return got
}

func bookLiquidityVenues(exchange string) ([]domain.Exchange, error) {
	raw := strings.ToLower(strings.TrimSpace(exchange))
	if raw == "" || raw == "all" {
		return []domain.Exchange{domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit}, nil
	}
	ex := domain.ParseExchange(raw)
	switch ex {
	case domain.ExchangeBinance, domain.ExchangeCoinbase, domain.ExchangeBybit:
		return []domain.Exchange{ex}, nil
	default:
		return nil, fmt.Errorf("%w: exchange must be binance, coinbase, bybit, or all", domain.ErrInvalidArgument)
	}
}

func heatLevelsToBook(in []domain.HeatmapLevel) []domain.BookHistoryLevel {
	out := make([]domain.BookHistoryLevel, 0, len(in))
	for _, lv := range in {
		p, e1 := strconv.ParseFloat(lv.Price, 64)
		n, e2 := strconv.ParseFloat(lv.Notional, 64)
		if e1 != nil || e2 != nil || p <= 0 || n <= 0 {
			continue
		}
		qty := n / p
		out = append(out, domain.BookHistoryLevel{Price: p, Quantity: qty, Notional: n, Wall: lv.IsWall})
	}
	return out
}

func parseFloatOrZero(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func domainBookSideNotional(levels []domain.BookHistoryLevel) float64 {
	var n float64
	for _, lv := range levels {
		n += lv.Notional
	}
	return n
}
