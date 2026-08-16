package market

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const levelsDisclaimer = "Support and resistance are areas from recent price swings plus volume, then checked against the live order book. Tests count distinct visits, not guaranteed holds. Breakout score uses recent volume, resting liquidity, and taker buy/sell — not a prediction. Informational only — not financial advice."

// GetLevels finds support/resistance areas and breakout strength for one coin.
func (s *Service) GetLevels(ctx context.Context, exchange, symbol string) (*domain.LevelsReport, error) {
	ex, err := s.ResolveExchange(exchange)
	if err != nil {
		return nil, err
	}
	symbol = normalizeSymbolForExchange(ex, symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}

	var (
		bars  []domain.OHLCBar
		book  *domain.RawOrderBook
		taker *domain.TakerVenueFlow
		last  float64
		wg    sync.WaitGroup
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		if c, err := s.GetCandles(ctx, string(ex), symbol, "1h", 220, nil, nil); err == nil {
			bars = domain.BarsFromCandles(c)
		}
	}()
	go func() {
		defer wg.Done()
		if p, err := s.port(ex); err == nil {
			book, _ = p.GetOrderBook(ctx, domain.OrderBookQuery{Symbol: symbol, Limit: domain.MaxOrderBookRawLimit})
		}
	}()
	go func() {
		defer wg.Done()
		if tkr, err := s.GetTicker24h(ctx, string(ex), symbol); err == nil && tkr != nil {
			last, _ = strconv.ParseFloat(tkr.LastPrice, 64)
		}
	}()
	go func() {
		defer wg.Done()
		// Futures taker uses the same USDT pair when the venue has a port.
		futEx := ex
		if ex == domain.ExchangeCoinbase {
			futEx = domain.ExchangeBinance
		}
		if p := s.takerPort(futEx); p != nil {
			taker, _ = p.GetTakerFlow(ctx, domain.NormalizeSymbol(domain.ExchangeBinance, symbol))
		}
	}()
	wg.Wait()

	if last <= 0 && len(bars) > 0 {
		last = bars[len(bars)-1].Close
	}
	out := domain.BuildLevelsReport(symbol, string(ex), bars, book, taker, last)
	out.Note = levelsDisclaimer
	return &out, nil
}
