package market

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const rsiHeatDisclaimer = "Informational only — not financial advice. Wilder RSI(14) on closed candles with a long seed so readings track TradingView-style charts. Stables are omitted."

// GetRSIHeatmap returns a ranked Wilder RSI scatter for top listed pairs.
// One interval. Stables are dropped. Results are cached for RSIHeatmapCacheTTL.
func (s *Service) GetRSIHeatmap(ctx context.Context, exchange, quote, intervalRaw, sortBy string, limit, period int) (*domain.RSIHeatmap, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	if quote == "" {
		quote = domain.DefaultQuoteAsset(ex)
	}
	if period == 0 {
		period = domain.DefaultRSIPeriod
	}
	if period < domain.MinIndicatorPeriod || period > domain.MaxIndicatorPeriod {
		return nil, fmt.Errorf("%w: rsiPeriod must be between %d and %d", domain.ErrInvalidArgument, domain.MinIndicatorPeriod, domain.MaxIndicatorPeriod)
	}
	interval, err := domain.ParseRSIHeatmapInterval(intervalRaw, ex)
	if err != nil {
		return nil, err
	}
	limit = domain.ClampRSIHeatmapLimit(limit)
	if sortBy == "" {
		sortBy = string(domain.SpotSortMarketCapCirculating)
	}
	if !domain.IsValidSpotSortField(sortBy) {
		return nil, fmt.Errorf("%w: sort must be one of %v", domain.ErrInvalidArgument, domain.SupportedSpotSortFields)
	}

	key := domain.RSIHeatmapCacheKey(ex, quote, sortBy, interval, limit, period)
	if s.rsiHeat != nil {
		if hit, ok := s.rsiHeat.Get(key); ok && hit != nil {
			cp := *hit
			return &cp, nil
		}
	}

	scanLimit := limit * 3
	if scanLimit < 80 {
		scanLimit = 80
	}
	if scanLimit > 500 {
		scanLimit = 500
	}
	spot, err := s.ListSpotMarkets(ctx, string(ex), domain.SpotListQuery{
		QuoteAsset: quote,
		Status:     "TRADING",
		SortBy:     domain.SpotSortField(sortBy),
		Order:      domain.SortDesc,
		Limit:      scanLimit,
	})
	if err != nil && sortBy == string(domain.SpotSortMarketCapCirculating) {
		sortBy = string(domain.SpotSortQuoteVolume)
		key = domain.RSIHeatmapCacheKey(ex, quote, sortBy, interval, limit, period)
		spot, err = s.ListSpotMarkets(ctx, string(ex), domain.SpotListQuery{
			QuoteAsset: quote,
			Status:     "TRADING",
			SortBy:     domain.SpotSortQuoteVolume,
			Order:      domain.SortDesc,
			Limit:      scanLimit,
		})
	}
	if err != nil {
		if s.rsiHeat != nil {
			if stale, ok := s.rsiHeat.GetStale(key); ok && stale != nil {
				cp := *stale
				cp.Stale = true
				return &cp, nil
			}
		}
		return nil, err
	}

	now := time.Now().UTC()
	out := &domain.RSIHeatmap{
		Exchange:   ex,
		Quote:      quote,
		Interval:   interval,
		Period:     period,
		Oversold:   domain.DefaultRSIOversold,
		Overbought: domain.DefaultRSIOverbought,
		SortBy:     sortBy,
		AsOf:       now,
		Items:      make([]domain.RSIHeatmapRow, 0, limit),
		Note:       rsiHeatDisclaimer,
	}

	for _, m := range spot.Items {
		if m.Symbol == "" {
			continue
		}
		base := strings.ToUpper(strings.TrimSpace(m.BaseAsset))
		if base == "" {
			base, _ = domain.SplitBaseQuote(ex, m.Symbol)
		}
		if domain.IsRSIHeatmapStableBase(base) {
			continue
		}
		out.Items = append(out.Items, domain.RSIHeatmapRow{
			Symbol:               m.Symbol,
			Base:                 base,
			LastPrice:            m.LastPrice,
			PriceChangePercent:   m.PriceChangePercent,
			QuoteVolume:          m.QuoteVolume,
			MarketCapCirculating: m.MarketCapCirculating,
		})
		if len(out.Items) >= limit {
			break
		}
	}

	fetchLimit := domain.RSIHeatmapCandleLimit
	if fetchLimit < period+50 {
		fetchLimit = period + 50
	}

	var wg sync.WaitGroup
	reqSem := make(chan struct{}, 8)
	for i := range out.Items {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				out.Items[i].Error = ctx.Err().Error()
				return
			case reqSem <- struct{}{}:
			}
			defer func() { <-reqSem }()

			select {
			case <-ctx.Done():
				out.Items[i].Error = ctx.Err().Error()
				return
			case batchUpstreamSem <- struct{}{}:
			}
			defer func() { <-batchUpstreamSem }()

			candles, err := s.GetCandles(ctx, string(ex), out.Items[i].Symbol, interval, fetchLimit, nil, nil)
			if err != nil {
				out.Items[i].Error = err.Error()
				return
			}
			closes, err := domain.ParseClosePrices(candles)
			if err != nil {
				out.Items[i].Error = err.Error()
				return
			}
			rsi := domain.LatestRSI(closes, period)
			out.Items[i].RSI = rsi
			out.Items[i].Zone = domain.RSIZoneFor(rsi, out.Oversold, out.Overbought)
		}()
	}
	wg.Wait()
	domain.SummarizeRSIHeatmap(out)

	if s.rsiHeat != nil {
		s.rsiHeat.Set(key, out)
	}
	cp := *out
	return &cp, nil
}
