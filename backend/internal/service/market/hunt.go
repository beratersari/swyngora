package market

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetLiquidationHunt builds a hypothetical per-venue “house hunt” report.
// exchange=all returns Binance and Bybit separately (never averaged).
func (s *Service) GetLiquidationHunt(ctx context.Context, exchange, symbol string) (*domain.HuntReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
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
	out := &domain.HuntReport{
		Symbol:      symbol,
		Exchange:    ex,
		AsOf:        now,
		Assumptions: domain.DefaultHuntAssumptions(),
		Venues:      make([]domain.HuntVenueReport, 0, len(want)),
		Note:        huntDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.huntOne(ctx, v, symbol, now)
			mu.Lock()
			out.Venues = append(out.Venues, ven)
			mu.Unlock()
		}(v)
	}
	wg.Wait()
	sort.Slice(out.Venues, func(i, j int) bool {
		return string(out.Venues[i].Exchange) < string(out.Venues[j].Exchange)
	})
	return out, nil
}

const huntDisclaimer = "Hypothetical model only — not evidence that any exchange moves the market, and not financial advice. Long/short is account count, not position size. Leverage mix is assumed. USD-M mark uses a multi-venue index, so one spot book may not move mark 1:1. Exchanges usually match users rather than take the other side; liquidationTake is an insurance-fund-like stand-in. bookOnlyPnl is the spot tour if you unwind on the current opposite side (usually a loss). netWithCascade assumes part of estimated liquidations becomes exit flow at the target."

func (s *Service) huntOne(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) domain.HuntVenueReport {
	in := domain.HuntInputs{Exchange: ex, Symbol: symbol}
	var wg sync.WaitGroup
	var oiErr, lsErr, fundErr, bookErr error

	wg.Add(4)
	go func() {
		defer wg.Done()
		if p := s.oiPort(ex); p != nil {
			ser, err := p.GetOpenInterestSeries(ctx, symbol)
			if err != nil {
				oiErr = err
				return
			}
			if ser != nil {
				in.OIValue = ser.Current.Value
			}
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.longShortPort(ex); p != nil {
			ser, err := p.GetLongShortSeries(ctx, symbol, domain.DefaultLongShortHistoryLimit)
			if err != nil {
				lsErr = err
				return
			}
			if ser != nil {
				p := domain.NormalizeLongShortPoint(ser.Current)
				in.LongShare, in.ShortShare = p.LongShare, p.ShortShare
			}
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.fundingPort(ex); p != nil {
			ser, err := p.GetFundingSeries(ctx, symbol, domain.DefaultFundingHistoryLimit)
			if err != nil {
				fundErr = err
				return
			}
			if ser != nil {
				in.FundingRate = ser.Current.Rate
			}
		}
	}()
	go func() {
		defer wg.Done()
		books, err := s.fetchVenueBooks(ctx, symbol, []domain.Exchange{ex})
		if err != nil {
			bookErr = err
			return
		}
		in.Price = domain.ImpactBookMid(books)
		in.Asks = domain.CollectImpactLevels(domain.ImpactSideBuy, books)
		in.Bids = domain.CollectImpactLevels(domain.ImpactSideSell, books)
	}()
	wg.Wait()

	if s.liq != nil {
		in.Liquidations = s.liq.Events(string(ex), symbol, now.Add(-24*time.Hour))
	}
	if in.Price <= 0 {
		if tkr, err := s.GetTicker24h(ctx, string(ex), symbol); err == nil && tkr != nil {
			if px, perr := strconv.ParseFloat(tkr.LastPrice, 64); perr == nil {
				in.Price = px
			}
		}
	}

	got := domain.BuildHuntVenue(in)
	switch {
	case bookErr != nil && oiErr != nil:
		got.Error = fmt.Sprintf("book: %v; open interest: %v", bookErr, oiErr)
	case bookErr != nil:
		got.Error = bookErr.Error()
	case oiErr != nil && in.OIValue == 0:
		got.Error = oiErr.Error()
	}
	_ = lsErr
	_ = fundErr
	return got
}
