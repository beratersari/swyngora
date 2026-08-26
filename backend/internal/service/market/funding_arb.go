package market

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// FundingArbParams is the application input for one-coin funding arb.
type FundingArbParams struct {
	Symbol        string
	Notional      float64
	HoldHours     float64
	FeeBinancePct *float64
	FeeBybitPct   *float64
}

// FundingArbScanParams ranks liquid coins by after-fee funding.
type FundingArbScanParams struct {
	Quote         string
	Notional      float64
	HoldHours     float64
	FeeBinancePct *float64
	FeeBybitPct   *float64
	SymbolLimit   int
}

// GetFundingArb sizes a Binance/Bybit long-short funding trade for one coin.
func (s *Service) GetFundingArb(ctx context.Context, in FundingArbParams) (*domain.FundingArbReport, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(in.Symbol)
	if err != nil {
		return nil, err
	}
	notional, hold, fees, err := resolveFundingArbSizing(in.Notional, in.HoldHours, in.FeeBinancePct, in.FeeBybitPct)
	if err != nil {
		return nil, err
	}
	s.noteFutures(symbol)
	legs := s.fundingArbLegs(ctx, symbol, fees)
	return domain.BuildFundingArbReport(symbol, legs, notional, hold, time.Now().UTC()), nil
}

// ScanFundingArb ranks top-volume USDT pairs by after-fee horizon payout.
func (s *Service) ScanFundingArb(ctx context.Context, in FundingArbScanParams) (*domain.FundingArbScan, error) {
	notional, hold, fees, err := resolveFundingArbSizing(in.Notional, in.HoldHours, in.FeeBinancePct, in.FeeBybitPct)
	if err != nil {
		return nil, err
	}
	limit := domain.ClampFundingArbScanLimit(in.SymbolLimit)
	quote := in.Quote
	if quote == "" {
		quote = "USDT"
	}
	now := time.Now().UTC()
	out := domain.NewFundingArbScan(notional, hold, limit, now)
	spot, err := s.ListSpotMarkets(ctx, string(domain.ExchangeBinance), domain.SpotListQuery{
		QuoteAsset: quote,
		SortBy:     domain.SpotSortQuoteVolume,
		Order:      domain.SortDesc,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	var (
		mu      sync.Mutex
		hits    = []domain.FundingArbHit{}
		skipped int
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 5)
	)
	for _, m := range spot.Items {
		symbol := m.Symbol
		if symbol == "" {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()
			s.noteFutures(symbol)
			legs := s.fundingArbLegs(ctx, symbol, fees)
			rep := domain.BuildFundingArbReport(symbol, legs, notional, hold, now)
			hit, ok := domain.FundingArbHitFromReport(rep, 0)
			if ok {
				hit.RankScore = rep.HorizonNet
			}
			mu.Lock()
			defer mu.Unlock()
			if !ok {
				skipped++
				return
			}
			hits = append(hits, hit)
		}()
	}
	wg.Wait()
	domain.SortFundingArbHits(hits)
	out.Hits = hits
	out.Skipped = skipped
	return out, nil
}

type fundingArbFees struct {
	binance float64
	bybit   float64
}

func resolveFundingArbSizing(notional, holdHours float64, feeBinancePct, feeBybitPct *float64) (float64, float64, fundingArbFees, error) {
	n, err := domain.ResolveFundingArbNotional(notional)
	if err != nil {
		return 0, 0, fundingArbFees{}, err
	}
	h, err := domain.ResolveFundingArbHoldHours(holdHours)
	if err != nil {
		return 0, 0, fundingArbFees{}, err
	}
	fb, err := domain.ResolveFundingArbFeeRate(domain.ExchangeBinance, feeBinancePct)
	if err != nil {
		return 0, 0, fundingArbFees{}, err
	}
	fy, err := domain.ResolveFundingArbFeeRate(domain.ExchangeBybit, feeBybitPct)
	if err != nil {
		return 0, 0, fundingArbFees{}, err
	}
	return n, h, fundingArbFees{binance: fb, bybit: fy}, nil
}

func (s *Service) fundingArbLegs(ctx context.Context, symbol string, fees fundingArbFees) []domain.FundingArbVenueInput {
	want := []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit}
	out := make([]domain.FundingArbVenueInput, len(want))
	var wg sync.WaitGroup
	for i, ex := range want {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = s.fundingArbOne(ctx, ex, symbol, fees)
		}()
	}
	wg.Wait()
	return out
}

func (s *Service) fundingArbOne(ctx context.Context, ex domain.Exchange, symbol string, fees fundingArbFees) domain.FundingArbVenueInput {
	in := domain.FundingArbVenueInput{Exchange: ex, FeeRate: feeForFundingArb(ex, fees)}
	var (
		fundErr, basisErr error
		ser               *domain.FundingSeries
		q                 *domain.BasisQuote
		wg                sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		p := s.fundingPort(ex)
		if p == nil {
			fundErr = errFundingNotConfigured
			return
		}
		ser, fundErr = p.GetFundingSeries(ctx, symbol, 3)
	}()
	go func() {
		defer wg.Done()
		p := s.basisPort(ex)
		if p == nil {
			return
		}
		q, basisErr = p.GetBasisQuote(ctx, symbol)
	}()
	wg.Wait()
	if fundErr != nil {
		in.Error = "funding unavailable"
		return in
	}
	if ser == nil {
		in.Error = "empty funding"
		return in
	}
	in.FundingRate = ser.Current.Rate
	in.IntervalHours = ser.IntervalHours
	in.NextFundingTime = ser.NextFundingTime
	if avg, ok := domain.AverageFundingRate(ser.History, 3); ok {
		in.AvgLast3 = avg
		in.HasAvgLast3 = true
	}
	if q != nil && basisErr == nil {
		in.PerpLast = q.FuturesLast
		in.PerpMark = q.FuturesMark
		in.SpotIndex = q.SpotIndex
		in.SpotLast = q.SpotLast
	}
	return in
}

func feeForFundingArb(ex domain.Exchange, fees fundingArbFees) float64 {
	if ex == domain.ExchangeBybit {
		return fees.bybit
	}
	return fees.binance
}

var errFundingNotConfigured = errFunding{"funding not configured"}

type errFunding struct{ msg string }

func (e errFunding) Error() string { return e.msg }
