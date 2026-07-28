package portfoliostore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_PendingOrderLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	o, err := s.CreatePendingOrder(ctx, domain.PendingOrder{
		ID: "po1", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, TriggerPrice: 90,
		Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || o.Status != domain.PendingStatusOpen {
		t.Fatalf("%+v %v", o, err)
	}
	_ = s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	all, err := s2.ListAllOpenPendingOrders(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("%+v %v", all, err)
	}
	// Cancel
	canceled, err := s2.CancelPendingOrder(ctx, "c", "po1", now)
	if err != nil || canceled.Status != domain.PendingStatusCanceled {
		t.Fatalf("%+v %v", canceled, err)
	}
	// Second cancel fails
	if _, err := s2.CancelPendingOrder(ctx, "c", "po1", now); err != domain.ErrNotFound {
		t.Fatalf("want not found: %v", err)
	}
	// Create another and fill
	_, _ = s2.CreatePendingOrder(ctx, domain.PendingOrder{
		ID: "po2", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, TriggerPrice: 100,
		Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
	})
	if err := s2.ExecutePendingFill(ctx, "po2", &domain.Portfolio{
		ClientID: "c", CashBalance: 900, RealizedPnLTotal: 0,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100,
	}, domain.Trade{
		ID: "t-fill", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, CreatedAt: now,
	}, 100, now); err != nil {
		t.Fatal(err)
	}
	// Double fill fails
	if err := s2.ExecutePendingFill(ctx, "po2", &domain.Portfolio{
		ClientID: "c", CashBalance: 800, RealizedPnLTotal: 0,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 2, AvgCost: 100,
	}, domain.Trade{
		ID: "t-dup", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, CreatedAt: now,
	}, 100, now); err != domain.ErrNotFound {
		t.Fatalf("want not found on double fill: %v", err)
	}
	got, err := s2.GetPendingOrder(ctx, "c", "po2")
	if err != nil || got.Status != domain.PendingStatusFilled || got.FillTradeID != "t-fill" {
		t.Fatalf("%+v %v", got, err)
	}
	// Cancel after fill fails
	if _, err := s2.CancelPendingOrder(ctx, "c", "po2", now); err != domain.ErrNotFound {
		t.Fatalf("cancel filled: %v", err)
	}
}

func TestSQLite_PortfolioRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.ExecuteTrade(ctx, &domain.Portfolio{
		ClientID: "c", CashBalance: 900, RealizedPnLTotal: 0, UpdatedAt: now,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100, UpdatedAt: now,
	}, domain.Trade{
		ID: "t1", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	got, err := s2.GetPortfolio(ctx, "c")
	if err != nil || got.CashBalance != 900 {
		t.Fatalf("%+v %v", got, err)
	}
	pos, err := s2.ListPositions(ctx, "c")
	if err != nil || len(pos) != 1 || pos[0].Quantity != 1 {
		t.Fatalf("%+v %v", pos, err)
	}
	tr, err := s2.ListTrades(ctx, "c", 10, 0)
	if err != nil || len(tr) != 1 {
		t.Fatalf("%+v %v", tr, err)
	}
}