package market

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// rsiHeatWarmEvery is the default-map refresh period (overridden in tests).
var rsiHeatWarmEvery = domain.RSIHeatmapCacheTTL


// StartRSIHeatmapWarmer keeps the default Binance USDT 1h Top-100 map warm
// so the first /heatmap?view=rsi click is a cache hit.
func (s *Service) StartRSIHeatmapWarmer(ctx context.Context) {
	if s == nil {
		return
	}
	s.warmDefaultRSIHeatmap(ctx)
	every := rsiHeatWarmEvery
	if every <= 0 {
		every = domain.RSIHeatmapCacheTTL
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.warmDefaultRSIHeatmap(ctx)
		}
	}
}

func (s *Service) warmDefaultRSIHeatmap(ctx context.Context) {
	wctx, cancel := context.WithTimeout(ctx, domain.RSIHeatmapBuildTimeout)
	defer cancel()
	_, err := s.GetRSIHeatmap(
		wctx,
		string(domain.ExchangeBinance),
		"USDT",
		domain.RSIHeatmapDefaultInterval,
		string(domain.SpotSortMarketCapCirculating),
		domain.RSIHeatmapDefaultLimit,
		domain.DefaultRSIPeriod,
	)
	if err != nil && slog.Default() != nil {
		slog.Debug("rsi heatmap warm", "err", err)
	}
}
