package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const volDisclaimer = "Volatility is the high–low range and the noisiness of recent bars, not a prediction. Higher than normal compares this window to earlier windows of the same length. Expanding means the range just got bigger than the previous window. Informational only — not financial advice."

// GetVolatility measures how much a coin has been moving vs its own history and vs BTC/ETH.
func (s *Service) GetVolatility(ctx context.Context, exchange, symbol string) (*domain.VolatilityReport, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	btcSym, ethSym := domain.CorrelationRefs(ex, symbol)
	need1m, need5m := 0, 0
	for _, spec := range domain.VolatilityWindows {
		need := spec.Bars * 5 // current + previous + a few for "normal"
		switch spec.Interval {
		case domain.Interval1m:
			if need > need1m {
				need1m = need
			}
		default:
			if need > need5m {
				need5m = need
			}
		}
	}
	if need1m > 1000 {
		need1m = 1000
	}
	if need5m > 1000 {
		need5m = 1000
	}

	var (
		asset1m, asset5m []domain.OHLCBar
		btc1m, btc5m     []domain.OHLCBar
		eth1m, eth5m     []domain.OHLCBar
		wg               sync.WaitGroup
	)
	load := func(dst *[]domain.OHLCBar, sym, interval string, limit int) {
		defer wg.Done()
		*dst = s.volBars(ctx, string(ex), sym, interval, limit)
	}
	wg.Add(6)
	go load(&asset1m, symbol, "1m", need1m)
	go load(&asset5m, symbol, "5m", need5m)
	go load(&btc1m, btcSym, "1m", need1m)
	go load(&btc5m, btcSym, "5m", need5m)
	go load(&eth1m, ethSym, "1m", need1m)
	go load(&eth5m, ethSym, "5m", need5m)
	wg.Wait()

	if fallbackBTC, fallbackETH, ok := corrUSDTFallback(ex, btcSym, ethSym); ok {
		if len(btc1m) < domain.VolatilityWindows[0].Bars {
			btc1m = s.volBars(ctx, string(ex), fallbackBTC, "1m", need1m)
			btc5m = s.volBars(ctx, string(ex), fallbackBTC, "5m", need5m)
		}
		if len(eth1m) < domain.VolatilityWindows[0].Bars {
			eth1m = s.volBars(ctx, string(ex), fallbackETH, "1m", need1m)
			eth5m = s.volBars(ctx, string(ex), fallbackETH, "5m", need5m)
		}
	}

	now := time.Now().UTC()
	out := &domain.VolatilityReport{
		Symbol:   symbol,
		Exchange: string(ex),
		AsOf:     now,
		Windows:  make([]domain.VolWindow, 0, len(domain.VolatilityWindows)),
		Note:     volDisclaimer,
	}
	for _, spec := range domain.VolatilityWindows {
		var asset, btc, eth []domain.OHLCBar
		if spec.Interval == domain.Interval1m {
			asset, btc, eth = asset1m, btc1m, eth1m
		} else {
			asset, btc, eth = asset5m, btc5m, eth5m
		}
		out.Windows = append(out.Windows, domain.BuildVolWindow(symbol, spec, asset, btc, eth))
	}
	out.Summary = domain.ExplainVolatilityReport(symbol, out.Windows)
	return out, nil
}

func (s *Service) volBars(ctx context.Context, exchange, symbol, interval string, limit int) []domain.OHLCBar {
	if symbol == "" || limit <= 0 {
		return nil
	}
	candles, err := s.GetCandles(ctx, exchange, symbol, interval, limit, nil, nil)
	if err != nil {
		return nil
	}
	return domain.BarsFromCandles(candles)
}
