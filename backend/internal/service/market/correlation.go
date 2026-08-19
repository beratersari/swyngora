package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const corrDisclaimer = "Similarity is Pearson correlation of bar-to-bar percent returns versus BTC and ETH on the same venue. Beta is how much the coin typically moves when the reference moves 1%. A lag means the coin has been following a few minutes later — not a prediction. Informational only — not financial advice."

// GetCorrelation compares a coin's recent price path to BTC and ETH.
func (s *Service) GetCorrelation(ctx context.Context, exchange, symbol string) (*domain.CorrelationReport, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	btcSym, ethSym := domain.CorrelationRefs(ex, symbol)
	now := time.Now().UTC()
	out := &domain.CorrelationReport{
		Symbol:   symbol,
		Exchange: string(ex),
		AsOf:     now,
		Windows:  make([]domain.CorrelationWindow, 0, len(domain.CorrelationWindows)),
		Note:     corrDisclaimer,
	}

	need1m, need5m := 0, 0
	for _, spec := range domain.CorrelationWindows {
		switch spec.Interval {
		case domain.Interval1m:
			if spec.Bars > need1m {
				need1m = spec.Bars
			}
		case domain.Interval5m:
			if spec.Bars > need5m {
				need5m = spec.Bars
			}
		}
	}
	// A few extra bars so a partial last candle does not starve the window.
	if need1m > 0 {
		need1m += 10
	}
	if need5m > 0 {
		need5m += 12
	}

	var (
		asset1m, asset5m []domain.PricePoint
		btc1m, btc5m     []domain.PricePoint
		eth1m, eth5m     []domain.PricePoint
		wg               sync.WaitGroup
	)
	load := func(dst *[]domain.PricePoint, sym, interval string, limit int) {
		defer wg.Done()
		*dst = s.corrCloses(ctx, string(ex), sym, interval, limit)
	}
	wg.Add(6)
	go load(&asset1m, symbol, "1m", need1m)
	go load(&asset5m, symbol, "5m", need5m)
	go load(&btc1m, btcSym, "1m", need1m)
	go load(&btc5m, btcSym, "5m", need5m)
	go load(&eth1m, ethSym, "1m", need1m)
	go load(&eth5m, ethSym, "5m", need5m)
	wg.Wait()

	// Same-quote pair missing (e.g. BTCUSDC) — try USDT majors.
	if fallbackBTC, fallbackETH, ok := corrUSDTFallback(ex, btcSym, ethSym); ok {
		if len(btc1m) < domain.CorrelationWindows[0].Bars {
			btc1m = s.corrCloses(ctx, string(ex), fallbackBTC, "1m", need1m)
			btc5m = s.corrCloses(ctx, string(ex), fallbackBTC, "5m", need5m)
			if len(btc1m) > 0 {
				btcSym = fallbackBTC
			}
		}
		if len(eth1m) < domain.CorrelationWindows[0].Bars {
			eth1m = s.corrCloses(ctx, string(ex), fallbackETH, "1m", need1m)
			eth5m = s.corrCloses(ctx, string(ex), fallbackETH, "5m", need5m)
			if len(eth1m) > 0 {
				ethSym = fallbackETH
			}
		}
	}

	for _, spec := range domain.CorrelationWindows {
		var asset, btc, eth []domain.PricePoint
		switch spec.Interval {
		case domain.Interval1m:
			asset, btc, eth = asset1m, btc1m, eth1m
		default:
			asset, btc, eth = asset5m, btc5m, eth5m
		}
		out.Windows = append(out.Windows, domain.BuildCorrelationWindow(symbol, spec, asset, btc, eth, btcSym, ethSym))
	}
	out.Summary = domain.ExplainCorrelationReport(symbol, out.Windows)
	return out, nil
}

func (s *Service) corrCloses(ctx context.Context, exchange, symbol, interval string, limit int) []domain.PricePoint {
	if symbol == "" || limit <= 0 {
		return nil
	}
	candles, err := s.GetCandles(ctx, exchange, symbol, interval, limit, nil, nil)
	if err != nil {
		return nil
	}
	return domain.PricePointsFromCandles(candles)
}

func corrUSDTFallback(ex domain.Exchange, btcSym, ethSym string) (btc, eth string, ok bool) {
	if ex == domain.ExchangeCoinbase {
		return "", "", false
	}
	usdtBTC, usdtETH := "BTCUSDT", "ETHUSDT"
	if btcSym == usdtBTC && ethSym == usdtETH {
		return "", "", false
	}
	return usdtBTC, usdtETH, true
}
