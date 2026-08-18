package bookhiststore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestInsertSnapshot_NoDuplicatesAndNearest(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "books.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	t0 := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	rec := domain.BookHistorySnapshot{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", SampledAt: t0,
		Mid: 100, Spread: 0.2, BidNotional: 5000, AskNotional: 4000,
		Bids: []domain.BookHistoryLevel{{Price: 99.9, Quantity: 1, Notional: 99.9}},
		Asks: []domain.BookHistoryLevel{{Price: 100.1, Quantity: 1, Notional: 100.1}},
	}
	ok, err := s.InsertSnapshot(context.Background(), rec)
	if err != nil || !ok {
		t.Fatalf("first %v %v", ok, err)
	}
	rec.Mid = 999
	ok, err = s.InsertSnapshot(context.Background(), rec)
	if err != nil || ok {
		t.Fatalf("dup %v %v", ok, err)
	}
	got, err := s.NearestAt(context.Background(), "binance", "btcusdt", t0.Add(30*time.Second))
	if err != nil || got == nil || got.Mid != 100 || len(got.Bids) != 1 {
		t.Fatalf("%+v %v", got, err)
	}
	later := rec
	later.SampledAt = t0.Add(2 * time.Minute)
	later.Mid = 101
	if _, err := s.InsertSnapshot(context.Background(), later); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListSnapshots(context.Background(), domain.BookHistoryQuery{
		Exchange: "binance", Symbol: "BTCUSDT", Limit: 10,
	})
	if err != nil || len(list) != 2 || list[0].Mid != 101 {
		t.Fatalf("%+v %v", list, err)
	}
	n, err := s.PurgeOlderThan(context.Background(), t0.Add(time.Minute))
	if err != nil || n != 1 {
		t.Fatalf("purge %d %v", n, err)
	}
}

func TestInsertSnapshot_VenuesIndependent(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "books.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	at := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	_, _ = s.InsertSnapshot(context.Background(), domain.BookHistorySnapshot{
		Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT", SampledAt: at, Mid: 1,
	})
	_, _ = s.InsertSnapshot(context.Background(), domain.BookHistorySnapshot{
		Exchange: domain.ExchangeBybit, Symbol: "ETHUSDT", SampledAt: at, Mid: 2,
	})
	bn, _ := s.NearestAt(context.Background(), "binance", "ETHUSDT", at)
	bb, _ := s.NearestAt(context.Background(), "bybit", "ETHUSDT", at)
	if bn == nil || bb == nil || bn.Mid != 1 || bb.Mid != 2 {
		t.Fatalf("bn=%+v bb=%+v", bn, bb)
	}
}
