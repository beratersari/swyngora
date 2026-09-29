package flowstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_AddsAndListsFlowCells(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "flow.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	minute := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	cell := domain.FlowAtPricesCell{
		Exchange: domain.ExchangeBybit, Symbol: "BTCUSDT", Minute: minute,
		Price: 65000, Buy: 10, Sell: 4, BuyN: 2, SellN: 1,
	}
	if err := s.AddFlowCells(ctx, []domain.FlowAtPricesCell{cell, cell}); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListFlowCells(ctx, domain.ExchangeBybit, "BTCUSDT", minute, minute.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Buy != 20 || got[0].Sell != 8 || got[0].BuyN != 4 || got[0].SellN != 2 {
		t.Fatalf("%+v", got)
	}
	if err := s.SetFlowCursor(ctx, domain.ExchangeBybit, "BTCUSDT", minute.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	cur, err := s.FlowCursor(ctx, domain.ExchangeBybit, "BTCUSDT")
	if err != nil || !cur.Equal(minute.Add(time.Second)) {
		t.Fatalf("cursor %s %v", cur, err)
	}
	earliest, ok, err := s.EarliestFlow(ctx, domain.ExchangeBybit, "BTCUSDT")
	if err != nil || !ok || !earliest.Equal(minute) {
		t.Fatalf("earliest %s ok=%v err=%v", earliest, ok, err)
	}
}
