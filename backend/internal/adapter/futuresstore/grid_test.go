package futuresstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestListGridSnapshots_KeepsLatestSettledInTheHour(t *testing.T) {
	s := openGridDB(t)
	hour := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	insertSnap := func(at time.Time, predicted bool, rate float64) {
		t.Helper()
		_, err := s.InsertSnapshot(context.Background(), domain.FuturesSnapshot{
			Metric: domain.FuturesMetricFunding, Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
			SampledAt: at, Predicted: predicted, FundingRate: rate,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	insertSnap(hour.Add(5*time.Minute), true, 0.01)
	insertSnap(hour.Add(10*time.Minute), false, 0.0001)
	insertSnap(hour.Add(40*time.Minute), false, 0.0004)
	insertSnap(hour.Add(50*time.Minute), true, 0.02)
	got, err := s.ListGridSnapshots(context.Background(), domain.GridRead{
		Metric: domain.FuturesMetricFunding, Symbols: []string{"BTCUSDT", "ETHUSDT"},
		From: hour, To: hour.Add(time.Hour), Step: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want settled and predicted, got %+v", got)
	}
	var settled domain.GridSnapshot
	for _, row := range got {
		if !row.Predicted {
			settled = row
		}
		if !row.Bucket.Equal(hour) || row.Metric != domain.FuturesMetricFunding {
			t.Fatalf("row %+v", row)
		}
	}
	if settled.FundingRate != 0.0004 {
		t.Fatalf("settled %+v", settled)
	}
}

func TestListGridTakerSums_FoldsTheHour(t *testing.T) {
	s := openGridDB(t)
	hour := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if _, err := s.UpsertTakerBuckets(context.Background(), []domain.TakerBucket{{
			Exchange: domain.ExchangeBybit, Symbol: "ETHUSDT",
			Start:       hour.Add(time.Duration(i) * time.Minute),
			BuyNotional: 5, SellNotional: 1,
		}}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ListGridTakerSums(context.Background(), domain.GridRead{
		Symbols: []string{"ETHUSDT"}, From: hour, To: hour.Add(time.Hour), Step: time.Hour,
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	if got[0].Buy != 15 || got[0].Sell != 3 || got[0].Minutes != 3 || !got[0].First.Equal(hour) || !got[0].Last.Equal(hour.Add(2*time.Minute)) {
		t.Fatalf("%+v", got[0])
	}
}

func TestListGridLiquidations_SumsTheHour(t *testing.T) {
	s := openGridDB(t)
	hour := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	for i, n := range []float64{100, 250} {
		_, err := s.InsertLiquidation(context.Background(), domain.LiquidationEvent{
			Exchange: domain.ExchangeOKX, Symbol: "SOLUSDT", Side: "long",
			Time: hour.Add(time.Duration(i) * time.Minute), Price: 100, Quantity: 1, Notional: n,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ListGridLiquidations(context.Background(), domain.GridRead{
		Symbols: []string{"SOLUSDT"}, From: hour, To: hour.Add(time.Hour), Step: time.Hour,
	})
	if err != nil || len(got) != 1 || got[0].Notional != 350 || !got[0].Bucket.Equal(hour) {
		t.Fatalf("%+v %v", got, err)
	}
}

func openGridDB(t *testing.T) *SQLite {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "grid.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
