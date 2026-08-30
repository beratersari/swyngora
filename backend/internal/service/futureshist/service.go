package futureshist

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const snapshotBucket = 5 * time.Minute

// Service writes and reads durable futures history.
type Service struct {
	Store      domain.FuturesHistoryStore
	TakerStore domain.TakerBucketStore
	OI         map[domain.Exchange]domain.OpenInterestPort
	Funding    map[domain.Exchange]domain.FundingRatePort
	LS         map[domain.Exchange]domain.LongShortRatioPort
	Taker      map[domain.Exchange]domain.TakerBucketPort
	Logger     *slog.Logger
	Seeds      []string

	mu   sync.Mutex
	seen map[string]time.Time
}

// NoteSymbol records a pair that users or streams have asked about.
func (s *Service) NoteSymbol(symbol string) {
	if s == nil {
		return
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if symbol == "" {
		return
	}
	s.mu.Lock()
	if s.seen == nil {
		s.seen = map[string]time.Time{}
	}
	s.seen[symbol] = time.Now().UTC()
	s.mu.Unlock()
}

// Symbols is the worker universe: seeds + recently seen pairs.
func (s *Service) Symbols() []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]string{}, s.Seeds...)
	for sym := range s.seen {
		out = append(out, sym)
	}
	return domain.NormalizeFuturesSymbols(out)
}

// History returns stored snapshots or liquidation events.
func (s *Service) History(ctx context.Context, q domain.FuturesHistoryQuery) (any, error) {
	if s == nil || s.Store == nil {
		return nil, nil
	}
	symbol, err := domain.ValidateOpenInterestSymbol(q.Symbol)
	if err != nil {
		return nil, err
	}
	q.Symbol = symbol
	metric, err := domain.ParseFuturesMetric(q.Metric)
	if err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(q.Exchange)
	if err != nil {
		return nil, err
	}
	q.Exchange = ex
	q.Limit = domain.ClampFuturesHistoryLimit(q.Limit)
	if metric == "liquidations" {
		return s.Store.ListLiquidations(ctx, ex, symbol, q.From, q.To, q.Limit)
	}
	q.Metric = metric
	return s.Store.ListSnapshots(ctx, q)
}

// SaveSymbol fetches one venue independently and upserts samples.
// A failure on one venue does not stop the other.
func (s *Service) SaveSymbol(ctx context.Context, exchange domain.Exchange, symbol string, now time.Time) (inserted int, err error) {
	if s == nil || s.Store == nil {
		return 0, nil
	}
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if symbol == "" {
		return 0, nil
	}
	now = now.UTC()
	n := 0
	if p := s.OI[exchange]; p != nil {
		ser, e := p.GetOpenInterestSeries(ctx, symbol)
		if e != nil {
			err = e
		} else if ser != nil {
			n += s.saveOI(ctx, exchange, symbol, ser, now)
		}
	}
	if p := s.Funding[exchange]; p != nil {
		ser, e := p.GetFundingSeries(ctx, symbol, domain.DefaultFundingHistoryLimit)
		if e != nil {
			err = e
		} else if ser != nil {
			n += s.saveFunding(ctx, exchange, symbol, ser, now)
		}
	}
	if p := s.Taker[exchange]; p != nil && s.TakerStore != nil {
		if b, e := p.GetTakerBuckets(ctx, symbol); e != nil {
			err = e
		} else if len(b) > 0 {
			if _, e := s.TakerStore.UpsertTakerBuckets(ctx, b); e == nil {
				n++
			}
		}
	}
	if p := s.LS[exchange]; p != nil {
		ser, e := p.GetLongShortSeries(ctx, symbol, domain.DefaultLongShortHistoryLimit)
		if e != nil {
			err = e
		} else if ser != nil {
			n += s.saveLS(ctx, exchange, symbol, ser)
		}
	}
	return n, err
}

func (s *Service) saveOI(ctx context.Context, ex domain.Exchange, symbol string, ser *domain.OpenInterestSeries, now time.Time) int {
	n := 0
	curAt := domain.TruncateToBucket(ser.Current.Time, snapshotBucket)
	if curAt.IsZero() {
		curAt = domain.TruncateToBucket(now, snapshotBucket)
	}
	if ok, _ := s.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
		Metric: domain.FuturesMetricOpenInterest, Exchange: ex, Symbol: symbol,
		SampledAt: curAt, Contracts: ser.Current.Contracts, Value: ser.Current.Value,
	}); ok {
		n++
	}
	for _, p := range ser.History {
		at := domain.TruncateToBucket(p.Time, snapshotBucket)
		if at.IsZero() {
			continue
		}
		if ok, _ := s.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricOpenInterest, Exchange: ex, Symbol: symbol,
			SampledAt: at, Contracts: p.Contracts, Value: p.Value,
		}); ok {
			n++
		}
	}
	return n
}

func (s *Service) saveFunding(ctx context.Context, ex domain.Exchange, symbol string, ser *domain.FundingSeries, now time.Time) int {
	n := 0
	predAt := domain.TruncateToBucket(now, snapshotBucket)
	if ok, _ := s.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
		Metric: domain.FuturesMetricFunding, Exchange: ex, Symbol: symbol,
		SampledAt: predAt, Predicted: true, FundingRate: ser.Current.Rate,
		IntervalHours: ser.IntervalHours, NextFunding: ser.NextFundingTime,
	}); ok {
		n++
	}
	for _, p := range ser.History {
		if p.Time.IsZero() {
			continue
		}
		if ok, _ := s.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricFunding, Exchange: ex, Symbol: symbol,
			SampledAt: p.Time.UTC(), Predicted: false, FundingRate: p.Rate,
			IntervalHours: ser.IntervalHours,
		}); ok {
			n++
		}
	}
	return n
}

func (s *Service) saveLS(ctx context.Context, ex domain.Exchange, symbol string, ser *domain.LongShortSeries) int {
	n := 0
	pts := append([]domain.LongShortPoint{ser.Current}, ser.History...)
	for _, p := range pts {
		at := domain.TruncateToBucket(p.Time, snapshotBucket)
		if at.IsZero() {
			continue
		}
		if ok, _ := s.Store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricLongShort, Exchange: ex, Symbol: symbol,
			SampledAt: at, LongShare: p.LongShare, ShortShare: p.ShortShare, Ratio: p.Ratio,
		}); ok {
			n++
		}
	}
	return n
}

// SaveLiquidation persists one event (ignore duplicates).
func (s *Service) SaveLiquidation(ctx context.Context, e domain.LiquidationEvent) {
	if s == nil || s.Store == nil {
		return
	}
	s.NoteSymbol(e.Symbol)
	_, _ = s.Store.InsertLiquidation(ctx, e)
}

// LoadRecentLiquidations returns every stored print since cutoff (no 20k cap).
func (s *Service) LoadRecentLiquidations(ctx context.Context, since time.Time) []domain.LiquidationEvent {
	if s == nil || s.Store == nil {
		return nil
	}
	ev, err := s.Store.ListLiquidationsSince(ctx, since, 0)
	if err != nil {
		return nil
	}
	return ev
}

type liquidationCoverageStore interface {
	UpsertLiquidationCoverage(ctx context.Context, rows []domain.LiquidationCoverage) error
	ListLiquidationCoverage(ctx context.Context) ([]domain.LiquidationCoverage, error)
}

// SaveCoverage writes live-socket clocks so they survive a restart.
func (s *Service) SaveCoverage(ctx context.Context, rows []domain.LiquidationCoverage) {
	if s == nil || len(rows) == 0 {
		return
	}
	cs, ok := s.Store.(liquidationCoverageStore)
	if !ok {
		return
	}
	if err := cs.UpsertLiquidationCoverage(ctx, rows); err != nil {
		s.log().Warn("liquidation coverage save failed", "err", err)
	}
}

// LoadCoverage returns persisted venue/pair clocks.
func (s *Service) LoadCoverage(ctx context.Context) []domain.LiquidationCoverage {
	if s == nil {
		return nil
	}
	cs, ok := s.Store.(liquidationCoverageStore)
	if !ok {
		return nil
	}
	rows, err := cs.ListLiquidationCoverage(ctx)
	if err != nil {
		s.log().Warn("liquidation coverage load failed", "err", err)
		return nil
	}
	return rows
}

func (s *Service) log() *slog.Logger {
	if s != nil && s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}
