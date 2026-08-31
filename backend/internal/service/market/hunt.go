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
	out.Bias = domain.CombineHuntBias(out.Venues)
	if out.Bias != nil {
		cov := out.Bias.Coverage
		out.Coverage = &cov
	}
	return out, nil
}

const huntDisclaimer = "Hypothetical model only — not evidence that any exchange moves the market, and not financial advice. Long/short is account count, not position size. Leverage mix is assumed. USD-M mark uses a multi-venue index, so one spot book may not move mark 1:1. Exchanges usually match users rather than take the other side; liquidationTake is an insurance-fund-like stand-in. bookOnlyPnl is the spot tour if you unwind on the current opposite side (usually a loss). netWithCascade assumes part of estimated liquidations becomes exit flow at the target. upScore / downScore rank which side looks easier or more likely from zone distance, visible book cost, price+OI trend, crowding/funding, and recent taker/liquidation flow. coverage says how complete those inputs are; a failed venue is shown but excluded from the combined lean — not a prediction."

func (s *Service) huntOne(ctx context.Context, ex domain.Exchange, symbol string, now time.Time) domain.HuntVenueReport {
	in := domain.HuntInputs{Exchange: ex, Symbol: symbol}
	sig := domain.HuntSignals{
		Price1hPct:  math.NaN(),
		Price4hPct:  math.NaN(),
		Price24hPct: math.NaN(),
		OI1hPct:     math.NaN(),
		OI4hPct:     math.NaN(),
	}
	var wg sync.WaitGroup
	var sigMu sync.Mutex
	var oiErr, lsErr, fundErr, bookErr error
	var oiSer *domain.OpenInterestSeries

	wg.Add(6)
	go func() {
		defer wg.Done()
		if p := s.oiPort(ex); p != nil {
			ser, err := p.GetOpenInterestSeries(ctx, symbol)
			if err != nil {
				oiErr = err
				return
			}
			oiSer = ser
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
				pt := domain.NormalizeLongShortPoint(ser.Current)
				in.LongShare, in.ShortShare = pt.LongShare, pt.ShortShare
				sigMu.Lock()
				sig.HasLongShort = true
				sigMu.Unlock()
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
				sigMu.Lock()
				sig.HasFunding = true
				sigMu.Unlock()
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
		if len(in.Asks)+len(in.Bids) > 0 {
			sigMu.Lock()
			sig.HasBook = true
			sigMu.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		candles, err := s.GetCandles(ctx, string(ex), symbol, "1h", 30, nil, nil)
		if err != nil || len(candles) == 0 {
			return
		}
		closes := domain.ClosesFromCandles(candles)
		p1 := domain.PriceChangeOverBars(closes, 1)
		p4 := domain.PriceChangeOverBars(closes, 4)
		p24 := domain.PriceChangeOverBars(closes, 24)
		sigMu.Lock()
		sig.Price1hPct = p1
		sig.Price4hPct = p4
		sig.Price24hPct = p24
		sig.HasPrice = true
		sig.Has1hPrice = !math.IsNaN(p1)
		sig.Has4hPrice = !math.IsNaN(p4)
		sig.PriceSpan1h = domain.HuntPriceLookbackSpan(len(closes), 1, time.Hour)
		sig.PriceSpan4h = domain.HuntPriceLookbackSpan(len(closes), 4, 4*time.Hour)
		sigMu.Unlock()
	}()
	go func() {
		defer wg.Done()
		p := s.takerPort(ex)
		if p == nil {
			return
		}
		flow, err := p.GetTakerFlow(ctx, symbol)
		if err != nil || flow == nil {
			return
		}
		for _, w := range flow.Windows {
			if w.Window != domain.TakerWindow1h {
				continue
			}
			sigMu.Lock()
			sig.TakerBuy1h, sig.TakerSell1h = w.BuyNotional, w.SellNotional
			sig.HasTaker = w.BuyNotional+w.SellNotional > 0
			sig.TakerSpan = domain.HuntSpanFromTakerWindow(w)
			sigMu.Unlock()
			return
		}
	}()
	wg.Wait()
	if bookErr != nil {
		sig.BookError = bookErr.Error()
	}
	if lsErr != nil {
		sig.LSError = lsErr.Error()
	}
	if fundErr != nil {
		sig.FundError = fundErr.Error()
	}

	if s.liq != nil {
		in.Liquidations = s.liq.Events(string(ex), symbol, now.Add(-24*time.Hour))
		sig.LiqFeedPresent = true
		sig.HasLiqWindows = len(in.Liquidations) > 0
		if snap := s.liq.Snapshot(string(ex), symbol); snap != nil {
			for _, w := range snap.Windows {
				switch w.Window {
				case domain.LiquidationWindow1h:
					sig.LiqSpan1h = domain.HuntSpanFromLiqWindow(w, time.Hour)
				case domain.LiquidationWindow4h:
					sig.LiqSpan4h = domain.HuntSpanFromLiqWindow(w, 4*time.Hour)
				}
			}
		}
		cut1h := now.Add(-time.Hour)
		cut4h := now.Add(-4 * time.Hour)
		for _, e := range in.Liquidations {
			switch {
			case e.Side == domain.LiquidationSideShort:
				if !e.Time.Before(cut1h) {
					sig.ShortLiq1h += e.Notional
				}
				if !e.Time.Before(cut4h) {
					sig.ShortLiq4h += e.Notional
				}
			default:
				if !e.Time.Before(cut1h) {
					sig.LongLiq1h += e.Notional
				}
				if !e.Time.Before(cut4h) {
					sig.LongLiq4h += e.Notional
				}
			}
		}
	}
	if in.Price <= 0 {
		if tkr, err := s.GetTicker24h(ctx, string(ex), symbol); err == nil && tkr != nil {
			if px, perr := strconv.ParseFloat(tkr.LastPrice, 64); perr == nil {
				in.Price = px
			}
			if !sig.HasPrice {
				if pct, perr := strconv.ParseFloat(tkr.PriceChangePercent, 64); perr == nil {
					sig.Price24hPct = pct
					sig.HasPrice = true
					sig.PriceFromTicker = true
				}
			}
		}
	}
	if oiSer != nil {
		sig.OI1hPct, sig.OISpan1h = domain.HuntOILookback(oiSer, time.Hour, now)
		sig.OI4hPct, sig.OISpan4h = domain.HuntOILookback(oiSer, 4*time.Hour, now)
		if !math.IsNaN(sig.OI1hPct) || !math.IsNaN(sig.OI4hPct) || sig.OISpan1h.Stale || sig.OISpan4h.Stale || sig.OISpan1h.CoverPct > 0 || sig.OISpan4h.CoverPct > 0 {
			sig.HasOI = true
		}
	}

	if oiErr != nil && in.OIValue == 0 {
		sig.OIError = oiErr.Error()
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
	domain.AttachHuntDirectionScores(&got, sig)
	if got.Error != "" {
		got.Coverage.Usable = false
		got.Bias.Coverage = got.Coverage
	}
	_ = lsErr
	_ = fundErr
	return got
}
