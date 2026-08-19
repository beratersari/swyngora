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

const positioningDisclaimer = "Positioning is a classic price + open interest model (buildup / unwinding / covering), not a prediction or financial advice. Funding and account long/short only corroborate the label. Informational only."

// GetPositioning classifies long/short buildup and unwinding per venue and combined.
func (s *Service) GetPositioning(ctx context.Context, exchange, symbol string) (*domain.PositioningReport, error) {
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
	out := &domain.PositioningReport{
		Symbol:   symbol,
		Exchange: ex,
		AsOf:     now,
		Venues:   make([]domain.PositioningVenueReport, 0, len(want)),
		Note:     positioningDisclaimer,
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, v := range want {
		wg.Add(1)
		go func(v domain.Exchange) {
			defer wg.Done()
			ven := s.positioningOne(ctx, v, symbol, now)
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
		out.Combined = domain.CombinePositioningReports(out.Venues)
	}
	return out, nil
}

func (s *Service) positioningOne(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) domain.PositioningVenueReport {
	in := domain.PositioningInputs{
		Exchange:    ex,
		Symbol:      symbol,
		Price1hPct:  math.NaN(),
		Price4hPct:  math.NaN(),
		Price24hPct: math.NaN(),
		OI1hPct:     math.NaN(),
		OI4hPct:     math.NaN(),
		OI24hPct:    math.NaN(),
	}
	var oiErr error
	var oiSer *domain.OpenInterestSeries

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
			if err != nil || ser == nil {
				return
			}
			pt := domain.NormalizeLongShortPoint(ser.Current)
			in.LongShare, in.ShortShare = pt.LongShare, pt.ShortShare
		}
	}()
	go func() {
		defer wg.Done()
		if p := s.fundingPort(ex); p != nil {
			ser, err := p.GetFundingSeries(ctx, symbol, domain.DefaultFundingHistoryLimit)
			if err != nil || ser == nil {
				return
			}
			in.FundingRate = ser.Current.Rate
		}
	}()
	go func() {
		defer wg.Done()
		// 1h candles for 1h/4h/24h price change; ticker as 24h fallback.
		candles, err := s.GetCandles(ctx, string(ex), symbol, "1h", 30, nil, nil)
		if err == nil && len(candles) > 0 {
			closes := domain.ClosesFromCandles(candles)
			in.Price1hPct = domain.PriceChangeOverBars(closes, 1)
			in.Price4hPct = domain.PriceChangeOverBars(closes, 4)
			in.Price24hPct = domain.PriceChangeOverBars(closes, 24)
			if len(closes) > 0 {
				in.Price = closes[len(closes)-1]
			}
		}
		if tkr, err := s.GetTicker24h(ctx, string(ex), symbol); err == nil && tkr != nil {
			if in.Price <= 0 {
				if px, perr := strconv.ParseFloat(tkr.LastPrice, 64); perr == nil {
					in.Price = px
				}
			}
			if math.IsNaN(in.Price24hPct) {
				in.Price24hPct = domain.ParseTickerPriceChangePct(tkr.PriceChangePercent)
			}
		}
	}()
	wg.Wait()

	if oiSer != nil {
		in.OIValue = oiSer.Current.Value
		in.OI1hPct = domain.OIChangePctFromSeries(oiSer, time.Hour, now)
		in.OI4hPct = domain.OIChangePctFromSeries(oiSer, 4*time.Hour, now)
		in.OI24hPct = domain.OIChangePctFromSeries(oiSer, 24*time.Hour, now)
		if in.Price <= 0 && oiSer.Current.Contracts > 0 && oiSer.Current.Value > 0 {
			in.Price = oiSer.Current.Value / oiSer.Current.Contracts
		}
	}

	got := domain.BuildPositioningVenue(in)
	if oiErr != nil && in.OIValue == 0 {
		got.Error = oiErr.Error()
	} else if math.IsNaN(in.Price1hPct) && math.IsNaN(in.Price4hPct) && math.IsNaN(in.Price24hPct) &&
		math.IsNaN(in.OI1hPct) && math.IsNaN(in.OI4hPct) {
		got.Error = fmt.Sprintf("insufficient price/OI history for %s", ex)
	}
	return got
}
