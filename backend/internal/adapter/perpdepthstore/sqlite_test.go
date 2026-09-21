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
	dropped, err := s.PurgeOlderThan(context.Background(), now.Add(time.Minute))
	if err != nil || dropped != 2 {
		t.Fatalf("purge %d %v", dropped, err)
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
