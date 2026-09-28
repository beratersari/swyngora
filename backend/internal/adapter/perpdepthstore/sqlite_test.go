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
	dropped, err := s.PurgeOlderThan(context.Background(), now.Add(time.Minute))
	if err != nil || dropped != 2 {
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
