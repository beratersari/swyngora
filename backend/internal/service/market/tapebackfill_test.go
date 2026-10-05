package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type tapeMem struct {
	cells    []domain.FlowAtPricesCell
	buckets  []domain.TakerBucket
	earliest time.Time
	hasEarly bool
	scan     time.Time
	hasScan  bool
}

func (s *tapeMem) AddFlowCells(context.Context, []domain.FlowAtPricesCell) error { return nil }
func (s *tapeMem) ListFlowCells(context.Context, domain.Exchange, string, time.Time, time.Time) ([]domain.FlowAtPricesCell, error) {
	return nil, nil
}
func (s *tapeMem) FlowCursor(context.Context, domain.Exchange, string) (time.Time, error) {
	return time.Time{}, nil
}
func (s *tapeMem) SetFlowCursor(context.Context, domain.Exchange, string, time.Time) error {
	return nil
}
func (s *tapeMem) PurgeFlowCells(context.Context, time.Time) (int, error) { return 0, nil }
func (s *tapeMem) EarliestFlow(context.Context, domain.Exchange, string) (time.Time, bool, error) {
	return s.earliest, s.hasEarly, nil
}
func (s *tapeMem) FlowScan(context.Context, domain.Exchange, string) (time.Time, bool, error) {
	return s.scan, s.hasScan, nil
}
func (s *tapeMem) SetFlowScan(_ context.Context, _ domain.Exchange, _ string, until time.Time) error {
	s.scan = until
	s.hasScan = true
	return nil
}
func (s *tapeMem) ReplaceFlowWindow(_ context.Context, _ domain.Exchange, _ string, from, to time.Time, cells []domain.FlowAtPricesCell) error {
	kept := s.cells[:0]
	for _, c := range s.cells {
		if !c.Minute.Before(from) && c.Minute.Before(to) {
			continue
		}
		kept = append(kept, c)
	}
	s.cells = append(kept, cells...)
	return nil
}
func (s *tapeMem) UpsertTakerBuckets(_ context.Context, recs []domain.TakerBucket) (int, error) {
	s.buckets = append(s.buckets, recs...)
	return len(recs), nil
}
func (s *tapeMem) ListTakerBuckets(context.Context, string, string, time.Time, time.Time) ([]domain.TakerBucket, error) {
	return nil, nil
}
func (s *tapeMem) PurgeTakerBuckets(context.Context, time.Time) (int, error) { return 0, nil }

type scriptWindow struct {
	calls [][2]time.Time
	err   error
	full  int
	empty bool
}

func (p *scriptWindow) ListAggTradeWindow(_ context.Context, _ string, from, to time.Time) ([]domain.TakerPrint, error) {
	p.calls = append(p.calls, [2]time.Time{from, to})
	if p.full > 0 {
		p.full--
		return nil, domain.ErrTapeWindowFull
	}
	if p.err != nil {
		return nil, p.err
	}
	if p.empty {
		return nil, nil
	}
	at := from.Add(time.Minute)
	return []domain.TakerPrint{
		{Side: domain.TakerSideBuy, Price: 100, Quantity: 1, Notional: 80, Time: at},
		{Side: domain.TakerSideSell, Price: 100, Quantity: 1, Notional: 20, Time: at.Add(time.Second)},
		{Side: domain.TakerSideBuy, Price: 100, Quantity: 1, Notional: 999, Time: to},
	}, nil
}

func TestBackfillBinanceTape_WritesBeforeStoredMinute(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	earliest := now.Add(-20 * time.Minute).Truncate(time.Minute)
	store := &tapeMem{earliest: earliest, hasEarly: true}
	svc := &Service{flowHist: store, takerStore: store}
	port := &scriptWindow{}
	res, err := svc.BackfillBinanceTape(context.Background(), port, "btcusdt", now)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Advanced || res.Done || res.Stop {
		t.Fatalf("result %+v", res)
	}
	if len(port.calls) != 1 || !port.calls[0][1].Equal(earliest) {
		t.Fatalf("window %+v", port.calls)
	}
	if len(store.buckets) != 1 || store.buckets[0].BuyNotional != 80 || store.buckets[0].SellNotional != 20 {
		t.Fatalf("buckets %+v", store.buckets)
	}
	if len(store.cells) != 1 || store.cells[0].Buy != 80 || store.cells[0].Sell != 20 {
		t.Fatalf("cells %+v", store.cells)
	}
	if !store.cells[0].Minute.Before(earliest) {
		t.Fatalf("cell landed on the stored minute %+v", store.cells[0])
	}
	if !store.hasScan || !store.scan.Equal(port.calls[0][0]) {
		t.Fatalf("scan %s", store.scan)
	}
}

func TestBackfillBinanceTape_EmptyHourAdvancesScan(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	store := &tapeMem{}
	svc := &Service{flowHist: store, takerStore: store}
	res, err := svc.BackfillBinanceTape(context.Background(), &scriptWindow{empty: true}, "BTCUSDT", now)
	if err != nil || !res.Advanced || len(store.buckets) != 0 || len(store.cells) != 0 || !store.hasScan {
		t.Fatalf("result %+v err %v buckets %d scan %v", res, err, len(store.buckets), store.hasScan)
	}
}

func TestBackfillBinanceTape_RateLimitDoesNotAdvance(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	store := &tapeMem{}
	svc := &Service{flowHist: store, takerStore: store}
	port := &scriptWindow{err: domain.ErrRateLimited}
	res, err := svc.BackfillBinanceTape(context.Background(), port, "BTCUSDT", now)
	if !res.Stop || !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("result %+v err %v", res, err)
	}
	if store.hasScan || len(store.buckets) != 0 || len(store.cells) != 0 {
		t.Fatalf("stored on pause scan=%v buckets=%d cells=%d", store.hasScan, len(store.buckets), len(store.cells))
	}
}

func TestBackfillBinanceTape_ShrinksAFullWindow(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	store := &tapeMem{}
	svc := &Service{flowHist: store, takerStore: store}
	port := &scriptWindow{full: 1}
	res, err := svc.BackfillBinanceTape(context.Background(), port, "ETHUSDT", now)
	if err != nil || !res.Advanced {
		t.Fatalf("result %+v err %v", res, err)
	}
	if len(port.calls) != 2 || !port.calls[1][0].After(port.calls[0][0]) {
		t.Fatalf("windows %+v", port.calls)
	}
}

func TestBackfillBinanceTape_DoneAtLookback(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	store := &tapeMem{scan: now.Add(-domain.BinanceTapeBackfillLookback), hasScan: true}
	svc := &Service{flowHist: store, takerStore: store}
	port := &scriptWindow{}
	res, err := svc.BackfillBinanceTape(context.Background(), port, "BTCUSDT", now)
	if err != nil || !res.Done || res.Advanced || len(port.calls) != 0 {
		t.Fatalf("result %+v calls %d err %v", res, len(port.calls), err)
	}
}

func TestBinanceTapeBackfill_PassStopsOnRateLimit(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	store := &tapeMem{}
	svc := &Service{flowHist: store, takerStore: store}
	port := &scriptWindow{err: domain.ErrRateLimited}
	w := &BinanceTapeBackfill{
		Svc:     svc,
		Prints:  port,
		Symbols: func() []string { return []string{"BTCUSDT", "ETHUSDT"} },
		now:     func() time.Time { return now },
	}
	paused, idle := w.Pass(context.Background())
	if !paused || idle || len(port.calls) != 1 {
		t.Fatalf("paused=%v idle=%v calls=%d", paused, idle, len(port.calls))
	}
}
