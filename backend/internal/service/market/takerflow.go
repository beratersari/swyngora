package market

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const takerDisclaimer = "Taker buy/sell is aggressive market-order volume (who hit the book), not how many accounts are already long or short. Binance uses the public taker buy/sell series; Bybit uses live trades (4h is complete after this process has been collecting). Informational only — not financial advice."

// GetTakerFlow returns aggressive futures buy vs sell volume for 5m / 1h / 4h.
func (s *Service) GetTakerFlow(ctx context.Context, exchange, symbol string) (*domain.TakerFlowReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseTakerExchange(exchange)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	if s.liqWatch != nil {
		s.liqWatch.Watch(symbol)
	}
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	if ex != "all" {
		want = []domain.Exchange{domain.Exchange(ex)}
	}
	now := time.Now().UTC()
	out := &domain.TakerFlowReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.TakerVenueFlow, 0, len(want)),
		Note:     takerDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.takerOne(ctx, v, symbol)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	if ex == "all" && len(out.Venues) > 0 {
		out.Combined = domain.CombineTakerVenues(symbol, out.Venues)
		ctxAll := s.takerContext(ctx, "all", symbol)
		if out.Combined != nil {
			out.Combined.Summary = domain.ExplainTakerFlow(*out.Combined, ctxAll)
		}
	}
	return out, nil
}

func (s *Service) takerOne(ctx context.Context, ex domain.Exchange, symbol string) domain.TakerVenueFlow {
	out := domain.TakerVenueFlow{Exchange: ex, Symbol: symbol, Windows: []domain.TakerWindowFlow{}}
	p := s.takerPort(ex)
	if p == nil {
		out.Error = "taker flow not configured"
		out.Summary = out.Error
		return out
	}
	got, err := p.GetTakerFlow(ctx, symbol)
	if err != nil {
		out.Error = err.Error()
		out.Summary = err.Error()
		return out
	}
	if got != nil {
		out = *got
	}
	out.Summary = domain.ExplainTakerFlow(out, s.takerContext(ctx, string(ex), symbol))
	return out
}

func (s *Service) takerPort(ex domain.Exchange) domain.TakerFlowPort {
	if s == nil || s.taker == nil {
		return nil
	}
	return s.taker[ex]
}

func (s *Service) takerContext(ctx context.Context, exchange, symbol string) domain.TakerFlowContext {
	tc := domain.TakerFlowContext{
		PriceChange1hPct: math.NaN(),
		OIChange1hPct:    math.NaN(),
	}
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		ex := domain.Exchange(exchange)
		if exchange == "all" {
			ex = domain.ExchangeBinance
		}
		if p := s.fundingPort(ex); p != nil {
			if ser, err := p.GetFundingSeries(ctx, symbol, 3); err == nil && ser != nil {
				tc.FundingRate = ser.Current.Rate
			}
		}
	}()
	go func() {
		defer wg.Done()
		ex := domain.Exchange(exchange)
		if exchange == "all" {
			ex = domain.ExchangeBinance
		}
		if p := s.oiPort(ex); p != nil {
			if ser, err := p.GetOpenInterestSeries(ctx, symbol); err == nil && ser != nil {
				tc.OIChange1hPct = domain.OIChangePctFromSeries(ser, time.Hour, time.Now().UTC())
			}
		}
	}()
	go func() {
		defer wg.Done()
		ex := exchange
		if ex == "all" {
			ex = string(domain.ExchangeBinance)
		}
		if candles, err := s.GetCandles(ctx, ex, symbol, "1h", 3, nil, nil); err == nil {
			closes := domain.ClosesFromCandles(candles)
			tc.PriceChange1hPct = domain.PriceChangeOverBars(closes, 1)
		}
		if p := s.longShortPort(domain.Exchange(ex)); p != nil {
			if ser, err := p.GetLongShortSeries(ctx, symbol, 4); err == nil && ser != nil {
				tc.LongShare = ser.Current.LongShare
			}
		}
	}()
	wg.Wait()
	return tc
}
