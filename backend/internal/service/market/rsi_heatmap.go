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

// rsiHeatBeforeBuild lets tests fill the cache after a miss and before singleflight.
var rsiHeatBeforeBuild func()

// GetRSIHeatmap returns a ranked Wilder RSI scatter for top listed pairs.
// One interval. Stables are dropped. Fresh results are cached for RSIHeatmapCacheTTL;
// an expired map is returned immediately (stale) while a refresh runs.
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

	key := domain.RSIHeatmapCacheKey(ex, quote, sortBy, interval, period)
	if s.rsiHeat != nil {
		if hit, ok := s.rsiHeat.Get(key); ok && hit != nil && len(hit.Items) >= limit {
			return domain.ClipRSIHeatmap(hit, limit), nil
		}
		if stale, ok := s.rsiHeat.GetStale(key); ok && stale != nil && len(stale.Items) >= limit {
			go s.refreshRSIHeatmap(key, string(ex), quote, interval, sortBy, limit, period)
			out := domain.ClipRSIHeatmap(stale, limit)
			out.Stale = true
			return out, nil
		}
	}

	if rsiHeatBeforeBuild != nil {
		rsiHeatBeforeBuild()
	}
	v, err, _ := s.rsiHeatSF.Do(fmt.Sprintf("%s|%d", key, limit), func() (any, error) {
		if s.rsiHeat != nil {
			if hit, ok := s.rsiHeat.Get(key); ok && hit != nil && len(hit.Items) >= limit {
				return hit, nil
			}
		}
		return s.buildRSIHeatmap(ctx, ex, quote, interval, sortBy, limit, period)
	})
	if err != nil {
		if s.rsiHeat != nil {
			if stale, ok := s.rsiHeat.GetStale(key); ok && stale != nil && len(stale.Items) >= limit {
				out := domain.ClipRSIHeatmap(stale, limit)
				out.Stale = true
				return out, nil
			}
		}
		return nil, err
	}
	got, _ := v.(*domain.RSIHeatmap)
	return domain.ClipRSIHeatmap(got, limit), nil
}

func (s *Service) refreshRSIHeatmap(key, exchange, quote, interval, sortBy string, limit, period int) {
	ctx, cancel := context.WithTimeout(context.Background(), domain.RSIHeatmapBuildTimeout)
	defer cancel()
	_, _, _ = s.rsiHeatSF.Do(fmt.Sprintf("%s|%d", key, limit), func() (any, error) {
		if s.rsiHeat != nil {
			if hit, ok := s.rsiHeat.Get(key); ok && hit != nil && len(hit.Items) >= limit {
				return hit, nil
			}
		}
		ex, err := s.ResolveExchange(exchange)
		if err != nil {
			return nil, err
		}
		return s.buildRSIHeatmap(ctx, ex, quote, interval, sortBy, limit, period)
	})
}

func (s *Service) buildRSIHeatmap(ctx context.Context, ex domain.Exchange, quote, interval, sortBy string, limit, period int) (*domain.RSIHeatmap, error) {
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
		spot, err = s.ListSpotMarkets(ctx, string(ex), domain.SpotListQuery{
			QuoteAsset: quote,
			Status:     "TRADING",
			SortBy:     domain.SpotSortQuoteVolume,
			Order:      domain.SortDesc,
			Limit:      scanLimit,
		})
	}
	if err != nil {
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

	fetchLimit := domain.RSIHeatmapFetchLimit(period)
	var wg sync.WaitGroup
	for i := range out.Items {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
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
			closes, err := domain.ParseClosePrices(domain.ClosedCandles(candles, now))
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
		key := domain.RSIHeatmapCacheKey(ex, quote, sortBy, interval, period)
		if prev, ok := s.rsiHeat.GetStale(key); !ok || prev == nil || len(out.Items) >= len(prev.Items) {
			s.rsiHeat.Set(key, out)
		}
	}
	return out, nil
}
