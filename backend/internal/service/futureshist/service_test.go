package futureshist

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type memStore struct {
	mu   sync.Mutex
	snap []domain.FuturesSnapshot
	liq  []domain.LiquidationEvent
}

func (m *memStore) InsertSnapshot(_ context.Context, rec domain.FuturesSnapshot) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.snap {
		if e.Metric == rec.Metric && e.Exchange == rec.Exchange && e.Symbol == rec.Symbol &&
			e.SampledAt.Equal(rec.SampledAt) && e.Predicted == rec.Predicted {
			return false, nil
		}
	}
	m.snap = append(m.snap, rec)
	return true, nil
}

func (m *memStore) InsertLiquidation(_ context.Context, e domain.LiquidationEvent) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.liq {
		if x.Exchange == e.Exchange && x.Symbol == e.Symbol && x.Side == e.Side &&
			x.Time.Equal(e.Time) && x.Price == e.Price && x.Quantity == e.Quantity {
			return false, nil
		}
	}
	m.liq = append(m.liq, e)
	return true, nil
}

func (m *memStore) ListSnapshots(_ context.Context, q domain.FuturesHistoryQuery) ([]domain.FuturesSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.FuturesSnapshot
	for _, e := range m.snap {
		if e.Metric == q.Metric && e.Symbol == q.Symbol {
			if q.Exchange != "" && q.Exchange != "all" && string(e.Exchange) != q.Exchange {
				continue
			}
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *memStore) ListLiquidations(_ context.Context, _, symbol string, _, _ time.Time, _ int) ([]domain.LiquidationEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.LiquidationEvent
	for _, e := range m.liq {
		if e.Symbol == symbol {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *memStore) ListLiquidationsSince(_ context.Context, from time.Time, _ int) ([]domain.LiquidationEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.LiquidationEvent
	for _, e := range m.liq {
		if from.IsZero() || !e.Time.Before(from) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *memStore) PurgeOlderThan(context.Context, time.Time) (int, int, error) {
	return 0, 0, nil
}

type fakeOI struct {
	ser *domain.OpenInterestSeries
	err error
}

func (f *fakeOI) GetOpenInterestSeries(context.Context, string) (*domain.OpenInterestSeries, error) {
	return f.ser, f.err
}

type fakeFund struct {
	ser *domain.FundingSeries
	err error
}

func (f *fakeFund) GetFundingSeries(context.Context, string, int) (*domain.FundingSeries, error) {
	return f.ser, f.err
}

func (f *fakeFund) ListFundingHistory(context.Context, string, time.Time, time.Time) ([]domain.FundingPoint, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.ser == nil {
		return nil, nil
	}
	return f.ser.History, nil
}

type fakeLS struct {
	ser *domain.LongShortSeries
	err error
}

func (f *fakeLS) GetLongShortSeries(context.Context, string, int) (*domain.LongShortSeries, error) {
	return f.ser, f.err
}

func TestSaveSymbol_OneVenueFailsOtherSaved(t *testing.T) {
	st := &memStore{}
	now := time.Date(2026, 8, 11, 16, 15, 0, 0, time.UTC)
	svc := &Service{
		Store: st,
		Seeds: []string{"BTCUSDT"},
		OI: map[domain.Exchange]domain.OpenInterestPort{
			domain.ExchangeBinance: &fakeOI{ser: &domain.OpenInterestSeries{
				Current: domain.OpenInterestPoint{Time: now, Contracts: 10, Value: 100},
			}},
			domain.ExchangeBybit: &fakeOI{err: errors.New("bybit down")},
		},
	}
	n, err := svc.SaveSymbol(context.Background(), domain.ExchangeBinance, "BTCUSDT", now)
	if err != nil || n < 1 {
		t.Fatalf("binance n=%d err=%v", n, err)
	}
	n, err = svc.SaveSymbol(context.Background(), domain.ExchangeBybit, "BTCUSDT", now)
	if err == nil {
		t.Fatal("expected bybit error")
	}
	if n != 0 {
		t.Fatalf("bybit inserted %d", n)
	}
	got, _ := st.ListSnapshots(context.Background(), domain.FuturesHistoryQuery{
		Metric: domain.FuturesMetricOpenInterest, Symbol: "BTCUSDT",
	})
	if len(got) != 1 || got[0].Exchange != domain.ExchangeBinance {
		t.Fatalf("%+v", got)
	}
}

func TestSaveSymbol_DuplicateTickIgnored(t *testing.T) {
	st := &memStore{}
	now := time.Date(2026, 8, 11, 16, 17, 0, 0, time.UTC) // buckets to 16:15
	oi := &fakeOI{ser: &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Contracts: 10, Value: 100},
	}}
	svc := &Service{Store: st, OI: map[domain.Exchange]domain.OpenInterestPort{domain.ExchangeBinance: oi}}
	n1, _ := svc.SaveSymbol(context.Background(), domain.ExchangeBinance, "BTCUSDT", now)
	n2, _ := svc.SaveSymbol(context.Background(), domain.ExchangeBinance, "BTCUSDT", now.Add(30*time.Second))
	if n1 < 1 || n2 != 0 {
		t.Fatalf("n1=%d n2=%d", n1, n2)
	}
}

func TestWorker_RunOnceIndependentVenues(t *testing.T) {
	st := &memStore{}
	now := time.Date(2026, 8, 11, 16, 15, 0, 0, time.UTC)
	svc := &Service{
		Store: st,
		Seeds: []string{"BTCUSDT"},
		OI: map[domain.Exchange]domain.OpenInterestPort{
			domain.ExchangeBinance: &fakeOI{ser: &domain.OpenInterestSeries{
				Current: domain.OpenInterestPoint{Time: now, Contracts: 1, Value: 1},
			}},
			domain.ExchangeBybit: &fakeOI{err: errors.New("down")},
		},
	}
	w := &Worker{Hist: svc, now: func() time.Time { return now }}
	w.RunOnce(context.Background())
	got, _ := st.ListSnapshots(context.Background(), domain.FuturesHistoryQuery{
		Metric: domain.FuturesMetricOpenInterest, Symbol: "BTCUSDT",
	})
	if len(got) != 1 {
		t.Fatalf("want binance row only, got %+v", got)
	}
}
