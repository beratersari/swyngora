package market

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const liquiditySweepDisclaimer = "A liquidity sweep is a brief poke through a prior high or low that had already turned price back, then a return to the other side. The level is the clustered swing high/low (at least two tests). Excursion is how far price went through; duration is how long it stayed beyond before coming back. Volume is quote notional of the bars in that poke (buy/sell when the venue publishes taker-buy). 15-minute spot candles, last ~7 days. A poke that stays through for more than 2 hours is treated as a breakout, not a sweep. Informational only — not financial advice."

const sweepInterval = 15 * time.Minute

// GetLiquiditySweeps returns recent high/low sweeps for Binance and/or Bybit.
func (s *Service) GetLiquiditySweeps(ctx context.Context, exchange, symbol string) (*domain.LiquiditySweepReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	now := time.Now().UTC()
	out := &domain.LiquiditySweepReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.LiquiditySweepVenue, 0, len(want)),
		Note:     liquiditySweepDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.liquiditySweepOne(ctx, v, symbol)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	out.Summary = domain.ExplainLiquiditySweepReport(*out)
	return out, nil
}

func (s *Service) liquiditySweepOne(ctx context.Context, ex domain.Exchange, symbol string) domain.LiquiditySweepVenue {
	out := domain.LiquiditySweepVenue{Exchange: ex, Symbol: symbol, Interval: "15m", Sweeps: []domain.LiquiditySweep{}}
	candles, err := s.GetCandles(ctx, string(ex), symbol, "15m", domain.SweepCandleLimit, nil, nil)
	if err != nil {
		out.Error = err.Error()
		out.Summary = err.Error()
		return out
	}
	bars := domain.SweepBarsFromCandles(candles)
	var last float64
	if tkr, err := s.GetTicker24h(ctx, string(ex), symbol); err == nil && tkr != nil {
		last, _ = strconv.ParseFloat(tkr.LastPrice, 64)
	}
	return domain.BuildLiquiditySweepVenue(ex, symbol, bars, last, sweepInterval)
}
