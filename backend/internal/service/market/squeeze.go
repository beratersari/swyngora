package market

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const squeezeDisclaimer = "Squeeze risk is a model, not a prediction or financial advice. Long/short is account count (not position size). Nearby liquidation pockets use an assumed leverage mix. Liquidation heat only includes events this process has seen. Informational only."

// GetSqueezeRisk scores long-squeeze and short-squeeze risk per venue and combined.
// exchange=all returns Binance + Bybit plus an OI-weighted combined block.
func (s *Service) GetSqueezeRisk(ctx context.Context, exchange, symbol string) (*domain.SqueezeReport, error) {
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
	out := &domain.SqueezeReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.SqueezeVenueReport, 0, len(want)),
		Note:     squeezeDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.squeezeOne(ctx, v, symbol, now)
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
		out.Combined = domain.CombineSqueezeReports(out.Venues)
	}
	return out, nil
}

func (s *Service) squeezeOne(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) domain.SqueezeVenueReport {
	in := domain.SqueezeInputs{
		Exchange:      ex,
		Symbol:        symbol,
		OIChange1hPct: math.NaN(),
		OIChange4hPct: math.NaN(),
	}
	var oiErr, lsErr, fundErr error
	var oiSer *domain.OpenInterestSeries
	var lsSer *domain.LongShortSeries
	var fundSer *domain.FundingSeries
	var price24 float64
	var hasPrice24 bool

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		if p := s.oiPort(ex); p != nil {
			ser, err := p.GetOpenInterestSeries(ctx, symbol)
			if err != nil {
				oiErr = err
				return
			}
			oiSer = ser
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
			lsSer = ser
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
			fundSer = ser
		}
	}()
	go func() {
		defer wg.Done()
		if tkr, err := s.GetTicker24h(ctx, string(ex), symbol); err == nil && tkr != nil {
			if px, perr := strconv.ParseFloat(tkr.LastPrice, 64); perr == nil {
				in.Price = px
			}
			if pct, perr := strconv.ParseFloat(tkr.PriceChangePercent, 64); perr == nil {
				price24 = pct
				hasPrice24 = true
			}
		}
	}()
	wg.Wait()

	if oiSer != nil {
		in.OIValue = oiSer.Current.Value
		in.OIChange1hPct = domain.OIChangePctFromSeries(oiSer, time.Hour, now)
		in.OIChange4hPct = domain.OIChangePctFromSeries(oiSer, 4*time.Hour, now)
		if in.Price <= 0 && oiSer.Current.Contracts > 0 && oiSer.Current.Value > 0 {
			in.Price = oiSer.Current.Value / oiSer.Current.Contracts
		}
	}
	if lsSer != nil {
		p := domain.NormalizeLongShortPoint(lsSer.Current)
		in.LongShare, in.ShortShare = p.LongShare, p.ShortShare
		if share, ok := domain.LongShareAbout(lsSer.Current, lsSer.History, now.Add(-time.Hour)); ok {
			in.LongShare1hAgo = share
			in.HasLSHistory = true
		}
	}
	if fundSer != nil {
		in.FundingRate = fundSer.Current.Rate
		if avg, ok := domain.AverageFundingRate(fundSer.History, 3); ok {
			in.FundingAvg3 = avg
			in.HasFundingAvg = true
		}
	}
	if hasPrice24 {
		in.PriceChange24hPct = price24
	} else {
		in.PriceChange24hPct = math.NaN()
	}

	if s.liq != nil {
		ev := s.liq.Events(string(ex), symbol, now.Add(-24*time.Hour))
		in.LongLiq1h = domain.SumLiquidationNotional(ev, domain.LiquidationSideLong, now.Add(-time.Hour))
		in.ShortLiq1h = domain.SumLiquidationNotional(ev, domain.LiquidationSideShort, now.Add(-time.Hour))
		in.LongLiq24h = domain.SumLiquidationNotional(ev, domain.LiquidationSideLong, now.Add(-24*time.Hour))
		in.ShortLiq24h = domain.SumLiquidationNotional(ev, domain.LiquidationSideShort, now.Add(-24*time.Hour))
	}
	in.LongPressureNear = domain.NearLiquidationPressureShare(true, 2, in.FundingRate)
	in.ShortPressureNear = domain.NearLiquidationPressureShare(false, 2, in.FundingRate)

	got := domain.BuildSqueezeVenue(in)
	switch {
	case oiErr != nil && in.OIValue == 0 && lsErr != nil:
		got.Error = fmt.Sprintf("open interest: %v; long/short: %v", oiErr, lsErr)
	case oiErr != nil && in.OIValue == 0:
		got.Error = oiErr.Error()
	case lsErr != nil && in.LongShare == 0 && in.ShortShare == 0:
		got.Error = lsErr.Error()
	}
	_ = fundErr
	return got
}
