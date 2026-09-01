package futuresstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestInsertSnapshot_NoDuplicates(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	at := time.Date(2026, 8, 11, 16, 15, 0, 0, time.UTC)
	rec := domain.FuturesSnapshot{
		Metric: domain.FuturesMetricOpenInterest, Exchange: domain.ExchangeBinance,
		Symbol: "BTCUSDT", SampledAt: at, Contracts: 100, Value: 1000,
	}
	ok, err := s.InsertSnapshot(context.Background(), rec)
	if err != nil || !ok {
		t.Fatalf("first insert %v %v", ok, err)
	}
	rec.Contracts = 999
	ok, err = s.InsertSnapshot(context.Background(), rec)
	if err != nil || ok {
		t.Fatalf("dup should ignore, inserted=%v err=%v", ok, err)
	}
	got, err := s.ListSnapshots(context.Background(), domain.FuturesHistoryQuery{
		Metric: domain.FuturesMetricOpenInterest, Symbol: "BTCUSDT", Limit: 10,
	})
	if err != nil || len(got) != 1 || got[0].Contracts != 100 {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestInsertSnapshot_VenuesIndependent(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	at := time.Date(2026, 8, 11, 16, 15, 0, 0, time.UTC)
	_, _ = s.InsertSnapshot(context.Background(), domain.FuturesSnapshot{
		Metric: domain.FuturesMetricFunding, Exchange: domain.ExchangeBinance,
		Symbol: "ETHUSDT", SampledAt: at, FundingRate: 0.0001,
	})
	_, _ = s.InsertSnapshot(context.Background(), domain.FuturesSnapshot{
		Metric: domain.FuturesMetricFunding, Exchange: domain.ExchangeBybit,
		Symbol: "ETHUSDT", SampledAt: at, FundingRate: 0.0002,
	})
	got, err := s.ListSnapshots(context.Background(), domain.FuturesHistoryQuery{
		Metric: domain.FuturesMetricFunding, Symbol: "ETHUSDT", Exchange: "all",
	})
	if err != nil || len(got) != 2 {
		t.Fatalf("want 2 venues, got %d %v", len(got), err)
	}
}

func TestTakerBuckets_UpsertListPurge(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	n, err := s.UpsertTakerBuckets(context.Background(), []domain.TakerBucket{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: t0, BuyNotional: 10, SellNotional: 2},
	})
	if err != nil || n != 1 {
		t.Fatalf("%d %v", n, err)
	}
	_, err = s.UpsertTakerBuckets(context.Background(), []domain.TakerBucket{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Start: t0, BuyNotional: 12, SellNotional: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ListTakerBuckets(context.Background(), "binance", "BTCUSDT", t0.Add(-time.Minute), t0.Add(time.Minute))
	if err != nil || len(got) != 1 || got[0].BuyNotional != 12 {
		t.Fatalf("%+v %v", got, err)
	}
	purged, err := s.PurgeTakerBuckets(context.Background(), t0.Add(time.Second))
	if err != nil || purged != 1 {
		t.Fatalf("purge %d %v", purged, err)
	}
}

func TestInsertLiquidation_NoDuplicatesAndPurge(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	at := time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC)
	ev := domain.LiquidationEvent{
		Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
		Price: 64000, Quantity: 1, Notional: 64000, Time: at,
	}
	ok, err := s.InsertLiquidation(context.Background(), ev)
	if err != nil || !ok {
		t.Fatalf("first %v %v", ok, err)
	}
	ok, err = s.InsertLiquidation(context.Background(), ev)
	if err != nil || ok {
		t.Fatalf("dup %v %v", ok, err)
	}
	got, err := s.ListLiquidations(context.Background(), "bybit", "BTCUSDT", time.Time{}, time.Time{}, 10)
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	n1, n2, err := s.PurgeOlderThan(context.Background(), at.Add(time.Hour))
	if err != nil || n2 != 1 {
		t.Fatalf("purge snaps=%d liq=%d err=%v", n1, n2, err)
	}
}

func TestListLiquidationsSince_PagesAllVenues(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	from := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		ex := domain.ExchangeBinance
		if i%2 == 1 {
			ex = domain.ExchangeBybit
		}
		_, err := s.InsertLiquidation(context.Background(), domain.LiquidationEvent{
			Exchange: ex, Symbol: "BTCUSDT", Side: domain.LiquidationSideLong,
			Price: 1, Quantity: float64(i + 1), Notional: float64(i + 1),
			Time: from.Add(time.Duration(i) * time.Minute),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.ListLiquidationsSince(context.Background(), from, 0)
	if err != nil || len(all) != 12 {
		t.Fatalf("unlimited got %d %v", len(all), err)
	}
	var bn, bb int
	for _, e := range all {
		if e.Exchange == domain.ExchangeBinance {
			bn++
		}
		if e.Exchange == domain.ExchangeBybit {
			bb++
		}
	}
	if bn != 6 || bb != 6 {
		t.Fatalf("venues bn=%d bb=%d", bn, bb)
	}
	capped, err := s.ListLiquidationsSince(context.Background(), from, 5)
	if err != nil || len(capped) != 5 {
		t.Fatalf("cap %d %v", len(capped), err)
	}
}

func TestLiquidationCoverage_Reopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "futures.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	err = s.UpsertLiquidationCoverage(context.Background(), []domain.LiquidationCoverage{
		{Exchange: domain.ExchangeBinance, FirstWatch: first, Live: 2 * time.Hour},
		{Exchange: domain.ExchangeBybit, Symbol: "ETHUSDT", FirstWatch: first, Live: time.Hour},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = s.UpsertLiquidationCoverage(context.Background(), []domain.LiquidationCoverage{
		{Exchange: domain.ExchangeBinance, FirstWatch: first.Add(time.Hour), Live: 3 * time.Hour},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	got, err := s2.ListLiquidationCoverage(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	var bn domain.LiquidationCoverage
	for _, row := range got {
		if row.Exchange == domain.ExchangeBinance && row.Symbol == "" {
			bn = row
		}
	}
	if !bn.FirstWatch.Equal(first) || bn.Live != 3*time.Hour {
		t.Fatalf("merge %+v", bn)
	}
}

func TestReopenKeepsRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "futures.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.InsertSnapshot(context.Background(), domain.FuturesSnapshot{
		Metric: domain.FuturesMetricLongShort, Exchange: domain.ExchangeBinance,
		Symbol: "BTCUSDT", SampledAt: time.Date(2026, 8, 11, 16, 0, 0, 0, time.UTC),
		LongShare: 0.6, ShortShare: 0.4, Ratio: 1.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	got, err := s2.ListSnapshots(context.Background(), domain.FuturesHistoryQuery{
		Metric: domain.FuturesMetricLongShort, Symbol: "BTCUSDT",
	})
	if err != nil || len(got) != 1 || got[0].Ratio != 1.5 {
		t.Fatalf("reopen %+v %v", got, err)
	}
}
