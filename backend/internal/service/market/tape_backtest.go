package market

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const tapeCandlePages = 12

// GetTapeBacktest replays stored CVD hours, liquidation prints, or spot-book
// samples and scores the next 1h, 4h, and 24h spot close.
func (s *Service) GetTapeBacktest(ctx context.Context, exchange, symbol, condition, side string, threshold float64, from, to time.Time) (*domain.TapeBacktestReport, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	q, err := domain.ParseTapeBacktestQuery(exchange, symbol, condition, side, threshold, from, to, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	in := domain.TapeBacktestInput{
		Query: q, CandleStep: 5 * time.Minute, VenueErrors: map[string]string{},
	}
	switch q.Condition {
	case domain.TapeConditionCVD:
		if err := s.loadTapeCVD(ctx, &in); err != nil {
			return nil, err
		}
	case domain.TapeConditionLiquidation:
		if err := s.loadTapeLiquidations(ctx, &in); err != nil {
			return nil, err
		}
	case domain.TapeConditionBook:
		if err := s.loadTapeBooks(ctx, &in); err != nil {
			return nil, err
		}
	}
	s.loadTapeCandles(ctx, &in)
	return domain.BuildTapeBacktest(in)
}

func (s *Service) loadTapeCVD(ctx context.Context, in *domain.TapeBacktestInput) error {
	q := in.Query
	var all []domain.TakerBucket
	for _, ex := range q.Expected {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rows, err := s.tapeMinutes(ctx, ex, q.Symbol, q.From, q.To)
		if err != nil {
			in.VenueErrors[string(ex)] = err.Error()
			continue
		}
		if len(rows) == 0 && s.takerStore == nil && s.bucketPort(ex) == nil {
			in.VenueErrors[string(ex)] = "taker minutes are not stored"
			continue
		}
		all = append(all, rows...)
	}
	in.Buckets = all
	return nil
}

func (s *Service) tapeMinutes(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) ([]domain.TakerBucket, error) {
	var stored []domain.TakerBucket
	var err error
	if s.takerStore != nil {
		stored, err = s.takerStore.ListTakerBuckets(ctx, string(ex), symbol, from, to)
	}
	var live []domain.TakerBucket
	if p := s.bucketPort(ex); p != nil {
		if got, lerr := p.GetTakerBuckets(ctx, symbol); lerr == nil {
			live = got
		}
	}
	if err != nil && len(live) == 0 {
		return nil, err
	}
	return filterTapeBuckets(domain.MergeTakerBuckets(stored, live), ex, from, to), nil
}

func filterTapeBuckets(rows []domain.TakerBucket, ex domain.Exchange, from, to time.Time) []domain.TakerBucket {
	out := make([]domain.TakerBucket, 0, len(rows))
	for _, b := range rows {
		if b.Exchange != "" && b.Exchange != ex {
			continue
		}
		if b.Exchange == "" {
			b.Exchange = ex
		}
		if b.Start.IsZero() {
			continue
		}
		start := b.Start.UTC()
		if !from.IsZero() && start.Before(from) {
			continue
		}
		if !to.IsZero() && start.After(to) {
			continue
		}
		b.Start = start
		b.Symbol = domain.NormalizeLiquidationSymbol(b.Symbol)
		out = append(out, b)
	}
	return out
}

func (s *Service) loadTapeLiquidations(ctx context.Context, in *domain.TapeBacktestInput) error {
	if s.futHist == nil {
		for _, ex := range in.Query.Expected {
			in.VenueErrors[string(ex)] = "liquidation history is not stored"
		}
		return nil
	}
	raw, err := s.futHist.History(ctx, domain.FuturesHistoryQuery{
		Metric: "liquidations", Exchange: in.Query.Exchange, Symbol: in.Query.Symbol,
		From: in.Query.From, To: in.Query.To, Limit: domain.MaxFuturesHistoryLimit,
	})
	if err != nil {
		return err
	}
	events, _ := raw.([]domain.LiquidationEvent)
	if len(events) >= domain.MaxFuturesHistoryLimit {
		in.LiquidationCapped = true
	}
	in.Liquidations = events
	return nil
}

func (s *Service) loadTapeBooks(ctx context.Context, in *domain.TapeBacktestInput) error {
	if s.bookHist == nil {
		for _, ex := range in.Query.Expected {
			in.VenueErrors[string(ex)] = "order book history is not stored"
		}
		return nil
	}
	var all []domain.BookHistorySnapshot
	for _, ex := range in.Query.Expected {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		sym := domain.CrossVenueSymbol(ex, in.Query.Symbol)
		rows, err := s.bookHist.List(ctx, domain.BookHistoryQuery{
			Exchange: string(ex), Symbol: sym, From: in.Query.From, To: in.Query.To,
			Limit: domain.MaxBookHistoryLimit,
		})
		if err != nil {
			in.VenueErrors[string(ex)] = err.Error()
			continue
		}
		if len(rows) >= domain.MaxBookHistoryLimit {
			in.BookCapped = true
		}
		for i := range rows {
			if rows[i].Exchange == "" {
				rows[i].Exchange = ex
			}
		}
		all = append(all, rows...)
	}
	in.Books = all
	return nil
}

func (s *Service) loadTapeCandles(ctx context.Context, in *domain.TapeBacktestInput) {
	ex := s.tapePriceExchange()
	if ex == "" {
		return
	}
	symbol := domain.CrossVenueSymbol(ex, in.Query.Symbol)
	if symbol == "" {
		return
	}
	end := in.Query.To.Add(24 * time.Hour)
	if end.After(in.Query.Now) {
		end = in.Query.Now
	}
	candles, err := s.pageTapeCandles(ctx, ex, symbol, in.Query.From, end)
	if err != nil || len(candles) == 0 {
		return
	}
	in.Candles = candles
	in.PriceExchange = string(ex)
}

func (s *Service) tapePriceExchange() domain.Exchange {
	for _, ex := range []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit, domain.ExchangeCoinbase} {
		if _, err := s.port(ex); err == nil {
			return ex
		}
	}
	return ""
}

func (s *Service) pageTapeCandles(ctx context.Context, ex domain.Exchange, symbol string, from, to time.Time) ([]domain.Candle, error) {
	p, err := s.port(ex)
	if err != nil {
		return nil, err
	}
	var all []domain.Candle
	start := from
	seen := map[int64]struct{}{}
	for page := 0; page < tapeCandlePages; page++ {
		if ctx.Err() != nil {
			return all, ctx.Err()
		}
		if !start.Before(to) {
			break
		}
		batch, err := p.GetCandles(ctx, domain.CandleQuery{
			Symbol: symbol, Interval: domain.Interval5m, Limit: domain.VolumeProfilePageLimit,
			StartTime: start, EndTime: to,
		})
		if err != nil {
			if len(all) > 0 {
				return all, nil
			}
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		added := 0
		var lastOpen time.Time
		for _, c := range batch {
			if c.OpenTime.IsZero() {
				continue
			}
			ms := c.OpenTime.UTC().UnixMilli()
			if _, ok := seen[ms]; ok {
				continue
			}
			seen[ms] = struct{}{}
			all = append(all, c)
			added++
			if c.OpenTime.After(lastOpen) {
				lastOpen = c.OpenTime
			}
		}
		if added == 0 || len(batch) < domain.VolumeProfilePageLimit || lastOpen.IsZero() {
			break
		}
		next := lastOpen.UTC().Add(time.Millisecond)
		if !next.After(start) {
			break
		}
		start = next
	}
	return all, nil
}
