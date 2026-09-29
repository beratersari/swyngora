package bookhiststore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestHourlySpreadPctAveragesWithoutLadder(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "book.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	now := time.Date(2026, 9, 29, 15, 30, 0, 0, time.UTC)
	hour := now.Truncate(time.Hour)
	insert := func(at time.Time, pct float64) {
		t.Helper()
		if _, err := s.InsertSnapshot(ctx, domain.BookHistorySnapshot{
			Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", SampledAt: at, SpreadPct: pct,
		}); err != nil {
			t.Fatal(err)
		}
	}
	insert(hour.Add(time.Minute), 1)
	insert(hour.Add(2*time.Minute), 3)
	insert(hour.Add(-time.Hour), 4)
	insert(now.Add(-10*24*time.Hour), 9)
	got, err := s.HourlySpreadPct(ctx, "binance", "BTCUSDT", now.Add(-7*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("%+v", got)
	}
	if got[0].Value != 4 || got[1].Value != 2 {
		t.Fatalf("%+v", got)
	}
}
