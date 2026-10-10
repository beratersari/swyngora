package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationOpenLevels returns estimated liquidation bands that stay at
// the price where that open-interest slice was placed until a later candle
// trades through them. range matches the liquidation heatmap.
// exchange=all returns Binance, Bybit, OKX, and combined.
// rawAnchor, when set, adds a click anchor. It does not change the window.
func (s *Service) GetLiquidationOpenLevels(ctx context.Context, exchange, symbol, rawRange, rawAnchor string) (*domain.OpenLevelsReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	spec, err := domain.ParseHuntHeatmapRange(rawRange)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	click, err := domain.ParseCVDAnchor(rawAnchor, now)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	if s.liqWatch != nil {
		s.liqWatch.Watch(symbol)
	}
	from := now.Add(-spec.Window)
	want := s.futuresWant(ex)
	venues := make([]domain.HuntHeatmapVenueSeries, 0, len(want))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ser := s.huntHeatmapVenueSeries(ctx, v, symbol, spec, from, now)
			mu.Lock()
			venues = append(venues, ser)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	got := domain.BuildOpenLevels(domain.OpenLevelsInput{
		Symbol: symbol, Spec: spec, To: now, Click: click, Venues: venues,
	})
	return &got, nil
}
