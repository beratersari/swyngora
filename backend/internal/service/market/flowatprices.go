package market

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const flowAtPricesDisclaimer = "Aggressive futures prints binned by price (USDT notional). Each row is market buy vs sell, print count, and delta (buy − sell) at that price — not candle volume smeared across high–low. The live tape covers about the last hour on Binance and the newest ~1000 Bybit trades. Closed minutes are kept while this process runs (about 30 days) so 4h, 1d, 3d, 7d, and 30d — and Bybit further back — use that stored ladder. trades is aggregated print count, not individual orders. Informational only — not financial advice."

// GetFlowAtPrices returns buy/sell/count/delta at each traded price.
func (s *Service) GetFlowAtPrices(ctx context.Context, exchange, symbol, window string, start, end *time.Time, tickSize, minPrice, maxPrice float64) (*domain.FlowAtPricesReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	if tickSize < 0 || (tickSize > 0 && tickSize != tickSize) {
		return nil, fmt.Errorf("%w: tickSize must be > 0", domain.ErrInvalidArgument)
	}
	if minPrice < 0 || maxPrice < 0 || (minPrice > 0 && maxPrice > 0 && maxPrice < minPrice) {
		return nil, fmt.Errorf("%w: minPrice must be <= maxPrice", domain.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	from, to, windowID, err := domain.ResolveFlowAtPricesRange(window, start, end, now)
	if err != nil {
		return nil, err
	}
	s.noteFlowSymbol(symbol)
	want := configuredFutures(ex, func(v domain.Exchange) bool { return s.printPort(v) != nil })

	type fetched struct {
		ex     domain.Exchange
		prints []domain.TakerPrint
		cells  []domain.FlowAtPricesCell
		last   float64
	}
	got := make([]fetched, len(want))
	var wg sync.WaitGroup
	for i, v := range want {
		wg.Add(1)
		go func(i int, v domain.Exchange) {
			defer wg.Done()
			prints := s.fetchPrintRange(ctx, v, symbol, from, to)
			var cells []domain.FlowAtPricesCell
			if s.flowHist != nil {
				cells, _ = s.flowHist.ListFlowCells(ctx, v, symbol, from, to)
			}
			last := 0.0
			if tkr, err := s.GetTicker24h(ctx, string(v), symbol); err == nil && tkr != nil {
				last, _ = parseFloat(tkr.LastPrice)
			}
			got[i] = fetched{ex: v, prints: prints, cells: cells, last: last}
		}(i, v)
	}
	wg.Wait()

	var rangeLow, rangeHigh, last float64
	for _, f := range got {
		if f.last > 0 && last == 0 {
			last = f.last
		}
		for _, p := range f.prints {
			if p.Price <= 0 {
				continue
			}
			if minPrice > 0 && p.Price < minPrice {
				continue
			}
			if maxPrice > 0 && p.Price > maxPrice {
				continue
			}
			if rangeLow == 0 || p.Price < rangeLow {
				rangeLow = p.Price
			}
			if p.Price > rangeHigh {
				rangeHigh = p.Price
			}
			if last == 0 {
				last = p.Price
			}
		}
		for _, c := range f.cells {
			if c.Price <= 0 {
				continue
			}
			if minPrice > 0 && c.Price < minPrice {
				continue
			}
			if maxPrice > 0 && c.Price > maxPrice {
				continue
			}
			if rangeLow == 0 || c.Price < rangeLow {
				rangeLow = c.Price
			}
			if c.Price > rangeHigh {
				rangeHigh = c.Price
			}
		}
	}
	tick := tickSize
	if tick <= 0 {
		tick = domain.AutoFlowAtPricesTick(rangeLow, rangeHigh)
	} else {
		tick = domain.ClampFlowAtPricesTick(rangeLow, rangeHigh, tick)
	}

	out := &domain.FlowAtPricesReport{
		Symbol:   symbol,
		Exchange: ex,
		Window:   windowID,
		From:     from,
		To:       to,
		AsOf:     now,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		TickSize: tick,
		Venues:   make([]domain.FlowAtPricesVenue, 0, len(want)),
		Note:     flowAtPricesDisclaimer,
	}
	for _, f := range got {
		px := f.last
		if px <= 0 {
			px = last
		}
		out.Venues = append(out.Venues, flowVenue(f.ex, symbol, f.prints, f.cells, tick, px, minPrice, maxPrice, from, to))
	}
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombineFlowAtPricesVenues(symbol, out.Venues, tick, last, from, to)
	}
	out.Summary = domain.ExplainFlowAtPricesReport(*out)
	return out, nil
}
