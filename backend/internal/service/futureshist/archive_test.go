package futureshist

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type archiveStore struct {
	snaps []domain.FuturesSnapshot
	liqs  []domain.LiquidationEvent
}

func (m *archiveStore) InsertSnapshot(_ context.Context, rec domain.FuturesSnapshot) (bool, error) {
	for _, s := range m.snaps {
		if s.Metric == rec.Metric && s.Exchange == rec.Exchange && s.Symbol == rec.Symbol && s.SampledAt.Equal(rec.SampledAt) && s.Predicted == rec.Predicted {
			return false, nil
		}
	}
	m.snaps = append(m.snaps, rec)
	return true, nil
}

func (m *archiveStore) InsertLiquidation(_ context.Context, e domain.LiquidationEvent) (bool, error) {
	m.liqs = append(m.liqs, e)
	return true, nil
}

func (m *archiveStore) ListSnapshots(context.Context, domain.FuturesHistoryQuery) ([]domain.FuturesSnapshot, error) {
	return nil, nil
}
func (m *archiveStore) ListLiquidations(context.Context, string, string, time.Time, time.Time, int) ([]domain.LiquidationEvent, error) {
	return nil, nil
}
func (m *archiveStore) ListLiquidationsSince(context.Context, time.Time, int) ([]domain.LiquidationEvent, error) {
	return nil, nil
}
func (m *archiveStore) PurgeOlderThan(context.Context, time.Time) (int, int, error) {
	return 0, 0, nil
}

func (m *archiveStore) EarliestSnapshot(_ context.Context, metric string, exchange domain.Exchange, symbol string) (time.Time, bool, error) {
	var earliest time.Time
	found := false
	for _, s := range m.snaps {
		if s.Metric != metric || s.Exchange != exchange || s.Symbol != symbol || s.Predicted {
			continue
		}
		if !found || s.SampledAt.Before(earliest) {
			earliest = s.SampledAt
			found = true
		}
	}
	return earliest, found, nil
}

type histOI struct {
	points []domain.OpenInterestPoint
	calls  int
	err    error
}

func (h *histOI) GetOpenInterestSeries(context.Context, string) (*domain.OpenInterestSeries, error) {
	return nil, nil
}
func (h *histOI) ListOpenInterestHistory(context.Context, string, time.Time, time.Time) ([]domain.OpenInterestPoint, error) {
	h.calls++
	return h.points, h.err
}

type histFund struct {
	points []domain.FundingPoint
	calls  int
}

func (h *histFund) GetFundingSeries(context.Context, string, int) (*domain.FundingSeries, error) {
	return nil, nil
}
func (h *histFund) ListFundingHistory(context.Context, string, time.Time, time.Time) ([]domain.FundingPoint, error) {
	h.calls++
	return h.points, nil
}

type histLS struct {
	points []domain.LongShortHistoryPoint
	calls  int
}

func (h *histLS) GetLongShortSeries(context.Context, string, int) (*domain.LongShortSeries, error) {
	return nil, nil
}
func (h *histLS) ListLongShortHistory(context.Context, string, time.Time, time.Time) ([]domain.LongShortHistoryPoint, error) {
	h.calls++
	return h.points, nil
}

type histLiq struct {
	events []domain.LiquidationEvent
}

func (h *histLiq) ListLiquidationHistory(context.Context, domain.LiquidationHistoryQuery) (domain.LiquidationHistoryResult, error) {
	return domain.LiquidationHistoryResult{Events: h.events}, nil
}

func TestArchiveBackfillInsertsAndSkipsWhenCovered(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	from := now.Add(-30 * 24 * time.Hour)
	store := &archiveStore{}
	oi := &histOI{points: []domain.OpenInterestPoint{
		{Time: from.Add(time.Hour), Contracts: 10, Value: 1000},
		{Time: now.Add(-time.Hour), Contracts: 12, Value: 1200},
	}}
	fund := &histFund{points: []domain.FundingPoint{
		{Time: from.Add(2 * time.Hour), Rate: 0.0001},
		{Time: now.Add(-8 * time.Hour), Rate: 0.0001},
	}}
	ls := &histLS{points: []domain.LongShortHistoryPoint{
		{Time: from.Add(3 * time.Hour), LongShare: 0.55, ShortShare: 0.45, Ratio: 1.2},
		{Time: now.Add(-2 * time.Hour), LongShare: 0.6, ShortShare: 0.4, Ratio: 1.5},
	}}
	liq := &histLiq{events: []domain.LiquidationEvent{{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Side: "long",
		Time: now.Add(-time.Hour), Price: 100, Quantity: 1, Notional: 100,
	}}}
	svc := &Service{
		Store: store,
		OI:    map[domain.Exchange]domain.OpenInterestPort{domain.ExchangeBinance: oi},
		Funding: map[domain.Exchange]domain.FundingRatePort{
			domain.ExchangeBinance: fund,
		},
		LS:    map[domain.Exchange]domain.LongShortRatioPort{domain.ExchangeBinance: ls},
		Seeds: []string{"BTCUSDT"},
	}
	arch := &Archive{
		Hist: svc, Lookback: 30 * 24 * time.Hour, now: func() time.Time { return now },
		Liq: map[domain.Exchange]domain.LiquidationHistoryPort{domain.ExchangeBinance: liq},
	}
	arch.Start(context.Background())
	if oi.calls != 1 || fund.calls != 1 || ls.calls != 1 {
		t.Fatalf("calls oi=%d fund=%d ls=%d", oi.calls, fund.calls, ls.calls)
	}
	if len(store.snaps) != 6 || len(store.liqs) != 1 {
		t.Fatalf("snaps=%d liqs=%d", len(store.snaps), len(store.liqs))
	}
	arch.Start(context.Background())
	if oi.calls != 1 || fund.calls != 1 || ls.calls != 1 {
		t.Fatalf("second pass should skip, oi=%d fund=%d ls=%d", oi.calls, fund.calls, ls.calls)
	}
}

func TestArchiveStopsWhenRateLimited(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	oi := &histOI{err: domain.ErrRateLimited}
	fund := &histFund{points: []domain.FundingPoint{{Time: now.Add(-time.Hour), Rate: 0.01}}}
	svc := &Service{
		Store:   &archiveStore{},
		OI:      map[domain.Exchange]domain.OpenInterestPort{domain.ExchangeBinance: oi},
		Funding: map[domain.Exchange]domain.FundingRatePort{domain.ExchangeBinance: fund},
		Seeds:   []string{"BTCUSDT"},
	}
	(&Archive{Hist: svc, Lookback: 24 * time.Hour, now: func() time.Time { return now }}).Start(context.Background())
	if fund.calls != 0 {
		t.Fatalf("funding should not run after a rate limit, calls=%d", fund.calls)
	}
	if !errors.Is(oi.err, domain.ErrRateLimited) {
		t.Fatal(oi.err)
	}
}
