package market

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/bookhiststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/futuresstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/bookhist"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/futureshist"
)

func TestGetUnusualWiresStoredHistory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := futuresstore.Open(filepath.Join(dir, "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	books, err := bookhiststore.Open(filepath.Join(dir, "book.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = books.Close() })

	now := time.Now().UTC()
	last := now.Add(-8 * time.Hour).Truncate(time.Hour)
	for i := 0; i < 25; i++ {
		if _, err := store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricFunding, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			SampledAt: last.Add(-time.Duration(24-i) * 8 * time.Hour), FundingRate: float64(i+1) / 100000,
		}); err != nil {
			t.Fatal(err)
		}
	}
	hour := now.Truncate(time.Hour)
	for i := 24; i >= 0; i-- {
		if _, err := books.InsertSnapshot(ctx, domain.BookHistorySnapshot{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			SampledAt: hour.Add(-time.Duration(i) * time.Hour), SpreadPct: 0.01,
		}); err != nil {
			t.Fatal(err)
		}
	}

	svc := New(nil, nil).
		WithTakerBucketStore(store).
		WithBookHistory(&bookhist.Service{Store: books})
	svc.SetFuturesHistory(&futureshist.Service{Store: store, TakerStore: store})

	got, err := svc.GetUnusual(ctx, "all", "BTCUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Exchange != "all" || len(got.Venues) != 3 {
		t.Fatalf("%+v", got)
	}
	var binance domain.UnusualVenue
	for _, v := range got.Venues {
		if v.Exchange == domain.ExchangeBinance {
			binance = v
		}
		if v.Exchange == domain.ExchangeOKX {
			spread := findReading(t, v, domain.UnusualMetricSpread)
			if spread.Reason != domain.UnusualReasonNoBook {
				t.Fatalf("okx spread %+v", spread)
			}
		}
		printSize := findReading(t, v, domain.UnusualMetricPrintSize)
		if printSize.Reason != domain.UnusualReasonNoStoredHistory {
			t.Fatalf("print %+v", printSize)
		}
	}
	funding := findReading(t, binance, domain.UnusualMetricFunding)
	if funding.Band != domain.UnusualBandExtremeHigh || funding.Label != domain.UnusualLabelSettled || funding.Samples != 24 {
		t.Fatalf("%+v", funding)
	}
	spread := findReading(t, binance, domain.UnusualMetricSpread)
	if spread.Band != domain.UnusualBandTypical || spread.Samples != 24 {
		t.Fatalf("%+v", spread)
	}
}

func TestScanUnusualRanksStoredFunding(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := futuresstore.Open(filepath.Join(dir, "futures.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	now := time.Now().UTC()
	last := now.Add(-8 * time.Hour).Truncate(time.Hour)
	for i := 0; i < 25; i++ {
		if _, err := store.InsertSnapshot(ctx, domain.FuturesSnapshot{
			Metric: domain.FuturesMetricFunding, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			SampledAt: last.Add(-time.Duration(24-i) * 8 * time.Hour), FundingRate: float64(i+1) / 100000,
		}); err != nil {
			t.Fatal(err)
		}
	}
	svc := New(nil, nil)
	svc.SetFuturesHistory(&futureshist.Service{Store: store, TakerStore: store})
	got, err := svc.ScanUnusual(ctx, "binance", "funding", 5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Metric != "funding" || got.Exchange != "binance" || got.Universe != 5 || got.Scanned != 5 {
		t.Fatalf("%+v", got)
	}
	if len(got.Hits) != 1 || got.Hits[0].Symbol != "BTCUSDT" || got.Hits[0].Band != domain.UnusualBandExtremeHigh {
		t.Fatalf("%+v", got.Hits)
	}
	if got.Quiet != 4 {
		t.Fatalf("quiet %d", got.Quiet)
	}
}

func TestScanUnusualRejectsPrintSize(t *testing.T) {
	svc := New(nil, nil)
	if _, err := svc.ScanUnusual(context.Background(), "all", "print_size", 5); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetUnusualRejectsBadExchange(t *testing.T) {
	svc := New(nil, nil)
	if _, err := svc.GetUnusual(context.Background(), "coinbase", "BTCUSDT"); err == nil {
		t.Fatal("expected error")
	}
}

func findReading(t *testing.T, venue domain.UnusualVenue, metric string) domain.UnusualReading {
	t.Helper()
	for _, r := range venue.Readings {
		if r.Metric == metric {
			return r
		}
	}
	t.Fatalf("missing %s", metric)
	return domain.UnusualReading{}
}
