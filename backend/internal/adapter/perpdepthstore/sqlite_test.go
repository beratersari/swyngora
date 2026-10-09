package perpdepthstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLiteInsertListPurge(t *testing.T) {
	t.Parallel()
	s, err := Open(filepath.Join(t.TempDir(), "perpdepth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	now := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	n, err := s.InsertSamples(context.Background(), []domain.PerpDepthSample{
		{Time: now, Exchange: "all", RangePct: 1, Symbol: "BTCUSDT", Base: "BTC", Bid: 80, Ask: 20},
		{Time: now, Exchange: "all", RangePct: 1, Symbol: "ETHUSDT", Base: "ETH", Bid: 10, Ask: 10},
	})
	if err != nil || n != 2 {
		t.Fatalf("insert %d %v", n, err)
	}
	got, err := s.ListSamples(context.Background(), "all", 1, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil || len(got) != 2 {
		t.Fatalf("list %+v %v", got, err)
	}
	totals, err := s.ListMinuteTotals(context.Background(), "all", 1, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil || len(totals) != 1 || totals[0].Total != 120 {
		t.Fatalf("totals %+v %v", totals, err)
	}
	buckets, err := s.ListBucketTotals(context.Background(), "all", 1, now.Add(-time.Hour), now.Add(time.Minute), time.Hour)
	if err != nil || len(buckets) != 1 || buckets[0].Total != 120 {
		t.Fatalf("buckets %+v %v", buckets, err)
	}
	bn, err := s.InsertBookBands(context.Background(), []domain.PerpBookBandSample{
		{Time: now, Exchange: "all", Symbol: "BTCUSDT", RangePct: 0.1, Bid: 3, Ask: 1, Mid: 100},
		{Time: now.Add(20 * time.Minute), Exchange: "all", Symbol: "BTCUSDT", RangePct: 1, Bid: 8, Ask: 2, Mid: 101},
	})
	if err != nil || bn != 2 {
		t.Fatalf("bands %d %v", bn, err)
	}
	if err := s.InsertBookGap(context.Background(), domain.PerpBookGap{Time: now.Add(-time.Minute), Symbol: "BTCUSDT", Missing: []string{"okx"}}); err != nil {
		t.Fatal(err)
	}
	folded, err := s.ListBookBandBuckets(context.Background(), "all", "BTCUSDT", now.Add(-time.Hour), now.Add(time.Hour), time.Hour)
	if err != nil || len(folded) != 2 {
		t.Fatalf("folded %+v %v", folded, err)
	}
	dropped, err := s.PurgeOlderThan(context.Background(), now.Add(time.Hour))
	if err != nil || dropped < 4 {
		t.Fatalf("purge %d %v", dropped, err)
	}
}

func TestSQLiteGapIgnoredWhenMinuteHasSamples(t *testing.T) {
	t.Parallel()
	s, err := Open(filepath.Join(t.TempDir(), "perpdepth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	now := time.Date(2026, 9, 28, 12, 4, 0, 0, time.UTC)
	if err := s.InsertGap(context.Background(), domain.PerpDepthHistoryGap{
		Time: now, Exchange: "all", RangePct: 1, Missing: []string{"okx"}, Stale: []string{"bybit"},
	}); err != nil {
		t.Fatal(err)
	}
	gaps, err := s.ListGaps(context.Background(), "all", 1, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil || len(gaps) != 1 || gaps[0].Missing[0] != "okx" || gaps[0].Stale[0] != "bybit" {
		t.Fatalf("%+v %v", gaps, err)
	}
	if _, err := s.InsertSamples(context.Background(), []domain.PerpDepthSample{
		{Time: now, Exchange: "all", RangePct: 1, Symbol: "BTCUSDT", Base: "BTC", Bid: 1, Ask: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearGap(context.Background(), "all", 1, now); err != nil {
		t.Fatal(err)
	}
	gaps, err = s.ListGaps(context.Background(), "all", 1, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil || len(gaps) != 0 {
		t.Fatalf("cleared %+v %v", gaps, err)
	}
	if err := s.InsertGap(context.Background(), domain.PerpDepthHistoryGap{
		Time: now, Exchange: "all", RangePct: 1, Missing: []string{"okx"},
	}); err != nil {
		t.Fatal(err)
	}
	gaps, err = s.ListGaps(context.Background(), "all", 1, now.Add(-time.Minute), now.Add(time.Minute))
	if err != nil || len(gaps) != 0 {
		t.Fatalf("sample minute kept a gap %+v %v", gaps, err)
	}
}

func TestMemoryInsertList(t *testing.T) {
	t.Parallel()
	m := &Memory{}
	now := time.Date(2026, 9, 21, 16, 0, 0, 0, time.UTC)
	_, _ = m.InsertSamples(context.Background(), []domain.PerpDepthSample{
		{Time: now, Exchange: "all", RangePct: 1, Symbol: "BTCUSDT", Base: "BTC", Bid: 1, Ask: 1},
	})
	got, err := m.ListSamples(context.Background(), "all", 1, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil || len(got) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
}
