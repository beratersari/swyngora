package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// situationAtFutureSlack is how far past now a candle close may sit.
// A forming candle is clamped on the client. Anything further ahead is rejected
// rather than answered with the live card.
const situationAtFutureSlack = time.Minute

// GetCoinSituationAt reads the stored five-minute minute at a candle close.
// The estimated liquidation pocket is not part of this read.
// Zero at, and a clock more than a minute ahead of now, are rejected.
func (s *Service) GetCoinSituationAt(ctx context.Context, symbol string, at time.Time) (*domain.CoinSituation, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if at.IsZero() {
		return nil, fmt.Errorf("%w: at is required", domain.ErrInvalidArgument)
	}
	at = at.UTC()
	if at.After(time.Now().UTC().Add(situationAtFutureSlack)) {
		return nil, fmt.Errorf("%w: at is in the future", domain.ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil {
		return nil, ctx.Err()
	}
	key := symbol + "|" + at.Format(time.RFC3339Nano)
	v, err, _ := s.situationSF.Do(key, func() (any, error) {
		return s.loadCoinSituationAt(ctx, symbol, at)
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.(*domain.CoinSituation)
	return got, nil
}

func (s *Service) loadCoinSituationAt(ctx context.Context, symbol string, at time.Time) (*domain.CoinSituation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var (
		cvd      *domain.CVDReport
		oi       []domain.FuturesSnapshot
		funding  []domain.FuturesSnapshot
		crowd    []domain.FuturesSnapshot
		minutes  []domain.SituationPrintMinute
		coverage []domain.LiquidationCoverage
		wg       sync.WaitGroup
	)
	wg.Add(5)
	go func() {
		defer wg.Done()
		cvd = s.situationCVDAt(ctx, symbol, at)
	}()
	go func() {
		defer wg.Done()
		oi = s.situationSnapshots(ctx, domain.FuturesMetricOpenInterest, symbol, at.Add(-26*time.Hour), at)
	}()
	go func() {
		defer wg.Done()
		funding = s.situationSnapshots(ctx, domain.FuturesMetricFunding, symbol, at.Add(-domain.FundingPressureLookback), at)
	}()
	go func() {
		defer wg.Done()
		crowd = s.situationSnapshots(ctx, domain.FuturesMetricLongShort, symbol, at.Add(-domain.LongShortGapLookback), at)
	}()
	go func() {
		defer wg.Done()
		minutes, coverage = s.situationPrintMinutesAt(ctx, symbol, at)
	}()
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return domain.BuildSituationAt(domain.SituationAtInput{
		At:           at,
		Symbol:       symbol,
		CVD:          cvd,
		OpenInterest: oi,
		Funding:      funding,
		FundingOI:    oi,
		Crowd:        crowd,
		Prints:       minutes,
		Coverage:     coverage,
	}), nil
}

// situationCVDAt builds the 24h futures tape ending at the candle.
// It lists stored buckets and does not upsert, and it does not subscribe the coin.
// A missing store leaves the price and size clauses blank.
func (s *Service) situationCVDAt(ctx context.Context, symbol string, at time.Time) *domain.CVDReport {
	if s == nil || s.takerStore == nil {
		return nil
	}
	span := domain.DefaultCVDSpan()
	from := at.Add(-span.Lookback)
	prices := s.situationCVDPrices(ctx, symbol, span, from, at)
	want := domain.FuturesVenues()
	series := make([]domain.CVDVenueSeries, 0, len(want))
	for _, ex := range want {
		series = append(series, s.situationCVDVenue(ctx, ex, symbol, prices, at, from, span))
	}
	return &domain.CVDReport{
		Symbol:   symbol,
		Exchange: "all",
		Range:    span.ID,
		Bar:      span.BarLabel(),
		AsOf:     at,
		Venues:   series,
		Combined: domain.CombineCVDVenues(symbol, series, prices, at),
	}
}

func (s *Service) situationCVDVenue(ctx context.Context, ex domain.Exchange, symbol string, prices []domain.CVDPrice, at, from time.Time, span domain.CVDSpan) domain.CVDVenueSeries {
	var stored []domain.TakerBucket
	if s != nil && s.takerStore != nil {
		rows, err := s.takerStore.ListTakerBuckets(ctx, string(ex), symbol, from, at)
		if err == nil {
			stored = rows
		}
	}
	var live []domain.TakerBucket
	if p := s.bucketPort(ex); p != nil {
		got, err := p.GetTakerBuckets(ctx, symbol)
		if err == nil {
			live = got
		}
	}
	merged := domain.MergeTakerBuckets(situationBucketsBefore(stored, at), situationBucketsBefore(live, at))
	started := at
	for _, b := range merged {
		if !b.Start.IsZero() && b.Start.Before(started) {
			started = b.Start
		}
	}
	return domain.BuildCVDSeriesSpan(ex, symbol, merged, prices, at, started, span)
}

// situationBucketsBefore drops a bar that starts at or after the candle close.
// CVD bars are labeled by their start and cover [start, start+5m). The bar
// that opens on the close belongs to the next candle.
func situationBucketsBefore(rows []domain.TakerBucket, at time.Time) []domain.TakerBucket {
	if len(rows) == 0 || at.IsZero() {
		return nil
	}
	out := make([]domain.TakerBucket, 0, len(rows))
	for _, row := range rows {
		if row.Start.IsZero() || !row.Start.UTC().Before(at) {
			continue
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Service) situationCVDPrices(ctx context.Context, symbol string, span domain.CVDSpan, from, at time.Time) []domain.CVDPrice {
	interval := span.CandleInterval
	limit := span.CandleLimit
	if interval == "" {
		interval = "5m"
	}
	if limit <= 0 {
		limit = 320
	}
	start, end := from, at
	bars, err := s.GetCandles(ctx, string(domain.ExchangeBinance), symbol, interval, limit, &start, &end)
	if err != nil || len(bars) == 0 {
		bars, err = s.GetCandles(ctx, string(domain.ExchangeBybit), symbol, interval, limit, &start, &end)
		if err != nil || len(bars) == 0 {
			return nil
		}
	}
	kept := make([]domain.Candle, 0, len(bars))
	for _, bar := range bars {
		if bar.OpenTime.After(at) {
			continue
		}
		kept = append(kept, bar)
	}
	return domain.CVDPricesFromCandles(kept)
}

func (s *Service) situationSnapshots(ctx context.Context, metric, symbol string, from, to time.Time) []domain.FuturesSnapshot {
	if s == nil || s.futHist == nil {
		return nil
	}
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric:   metric,
		Exchange: "all",
		Symbol:   symbol,
		From:     from,
		To:       to,
		Limit:    domain.MaxFuturesHistoryLimit,
	})
	if err != nil {
		return nil
	}
	rows := futuresSnapshots(raw)
	if len(rows) == 0 {
		return nil
	}
	kept := make([]domain.FuturesSnapshot, 0, len(rows))
	for _, row := range rows {
		if row.SampledAt.IsZero() || row.SampledAt.After(to) {
			continue
		}
		kept = append(kept, row)
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// situationPrintMinutesAt loads stored print buckets through the candle close.
// A failed or missing archive returns no minutes and no coverage, so a quiet
// book is not treated as a recorded window. An empty successful history is a
// real quiet when the coverage clock reaches the window.
func (s *Service) situationPrintMinutesAt(ctx context.Context, symbol string, at time.Time) ([]domain.SituationPrintMinute, []domain.LiquidationCoverage) {
	if s == nil || s.futHist == nil {
		return nil, nil
	}
	from := at.Add(-48 * time.Hour)
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric:   "liquidations",
		Exchange: "all",
		Symbol:   symbol,
		From:     from,
		To:       at,
		Bucket:   5 * time.Minute,
	})
	if err != nil || raw == nil {
		return nil, nil
	}
	events, ok := raw.([]domain.LiquidationEvent)
	if !ok {
		return nil, nil
	}
	latest := map[domain.Exchange]time.Time{}
	kept := make([]domain.LiquidationEvent, 0, len(events))
	for _, event := range events {
		if event.Time.IsZero() || event.Time.After(at) {
			continue
		}
		kept = append(kept, event)
		if event.Notional <= 0 {
			continue
		}
		if last := latest[event.Exchange]; last.IsZero() || event.Time.After(last) {
			latest[event.Exchange] = event.Time.UTC()
		}
	}
	if s.liq != nil {
		for _, event := range s.liq.Events("all", symbol, from) {
			if event.Time.IsZero() || event.Notional <= 0 || event.Time.Before(from) || event.Time.After(at) {
				continue
			}
			if last, found := latest[event.Exchange]; found && !event.Time.After(last) {
				continue
			}
			kept = append(kept, event)
		}
	}
	var coverage []domain.LiquidationCoverage
	if s.liq != nil {
		coverage = s.liq.CoverageSnapshot(at)
	}
	return domain.PrintMinutes(kept), coverage
}
