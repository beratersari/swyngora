package portfoliostore

import (
	"context"
	"errors"
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
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, RemainingQuantity: 1,
		TriggerPrice: 90, ReservedCash: 90, Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil || o.Status != domain.PendingStatusOpen || o.ReservedCash != 90 {
		t.Fatalf("%+v %v", o, err)
	}
	sum, err := s.SumReservedCash(ctx, "c")
	if err != nil || sum != 90 {
		t.Fatalf("reserved cash=%v err=%v", sum, err)
	}
	_ = s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	all, err := s2.ListAllOpenPendingOrders(ctx)
	if err != nil || len(all) != 1 || all[0].ReservedCash != 90 {
		t.Fatalf("%+v %v", all, err)
	}
	// Cancel releases reservation
	canceled, err := s2.CancelPendingOrder(ctx, "c", "po1", now, domain.CancelReasonUser)
	if err != nil || canceled.Status != domain.PendingStatusCanceled || canceled.ReservedCash != 0 {
		t.Fatalf("%+v %v", canceled, err)
	}
	if canceled.CancelReason != domain.CancelReasonUser {
		t.Fatalf("cancel reason=%q", canceled.CancelReason)
	}
	sum, _ = s2.SumReservedCash(ctx, "c")
	if sum != 0 {
		t.Fatalf("reserved after cancel=%v", sum)
	}
	if _, err := s2.CancelPendingOrder(ctx, "c", "po1", now, domain.CancelReasonUser); err != domain.ErrNotFound {
		t.Fatalf("want not found: %v", err)
	}

	// Partial then full fill
	o2 := domain.PendingOrder{
		ID: "po2", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 2, RemainingQuantity: 2,
		TriggerPrice: 100, ReservedCash: 200, Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s2.CreatePendingOrder(ctx, o2); err != nil {
		t.Fatal(err)
	}
	// Partial fill 1
	o2.FilledQuantity = 1
	o2.RemainingQuantity = 1
	o2.ReservedCash = 100
	o2.Status = domain.PendingStatusOpen
	o2.FillTradeID = "t-partial"
	o2.FillPrice = 100
	if err := s2.ExecutePendingFill(ctx, &o2, &domain.Portfolio{
		ClientID: "c", CashBalance: 900, RealizedPnLTotal: 0,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100,
	}, domain.Trade{
		ID: "t-partial", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, PendingOrderID: "po2", CreatedAt: now,
	}, now, nil); err != nil {
		t.Fatal(err)
	}
	got, err := s2.GetPendingOrder(ctx, "c", "po2")
	if err != nil || got.Status != domain.PendingStatusOpen || got.RemainingQuantity != 1 || got.ReservedCash != 100 {
		t.Fatalf("partial %+v %v", got, err)
	}
	// Complete fill
	o2.FilledQuantity = 2
	o2.RemainingQuantity = 0
	o2.ReservedCash = 0
	o2.Status = domain.PendingStatusFilled
	o2.FillTradeID = "t-full"
	o2.FillPrice = 100
	ft := now
	o2.FilledAt = &ft
	if err := s2.ExecutePendingFill(ctx, &o2, &domain.Portfolio{
		ClientID: "c", CashBalance: 800, RealizedPnLTotal: 0,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 2, AvgCost: 100,
	}, domain.Trade{
		ID: "t-full", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, PendingOrderID: "po2", CreatedAt: now,
	}, now, nil); err != nil {
		t.Fatal(err)
	}
	got, err = s2.GetPendingOrder(ctx, "c", "po2")
	if err != nil || got.Status != domain.PendingStatusFilled || got.FillTradeID != "t-full" {
		t.Fatalf("%+v %v", got, err)
	}
	// Double fill fails
	if err := s2.ExecutePendingFill(ctx, &o2, &domain.Portfolio{
		ClientID: "c", CashBalance: 700,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 3, AvgCost: 100,
	}, domain.Trade{
		ID: "t-dup", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, PendingOrderID: "po2", CreatedAt: now,
	}, now, nil); err != domain.ErrNotFound {
		t.Fatalf("want not found on double fill: %v", err)
	}
	tr, err := s2.ListTrades(ctx, "c", 10, 0)
	if err != nil || len(tr) != 2 {
		t.Fatalf("trades=%+v err=%v", tr, err)
	}
	if tr[0].PendingOrderID != "po2" || tr[1].PendingOrderID != "po2" {
		t.Fatalf("pending order id missing: %+v", tr)
	}
	if _, err := s2.CancelPendingOrder(ctx, "c", "po2", now, domain.CancelReasonUser); err != domain.ErrNotFound {
		t.Fatalf("cancel filled: %v", err)
	}
}

func TestSQLite_AmendPendingOrderCAS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "amend.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePendingOrder(ctx, domain.PendingOrder{
		ID: "po-a", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, RemainingQuantity: 1,
		TriggerPrice: 90, ReservedCash: 90, Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.AmendPendingOrder(ctx, "c", "po-a", domain.PendingOrderAmend{
		RemainingQuantity: 0.5, TriggerPrice: 80, Quantity: 0.5,
		ReservedCash: 40, ExpectedRemaining: 1, ExpectedTrigger: 90, At: now,
	})
	if err != nil || got.RemainingQuantity != 0.5 || got.TriggerPrice != 80 || got.ReservedCash != 40 {
		t.Fatalf("amend %+v %v", got, err)
	}
	sum, _ := s.SumReservedCash(ctx, "c")
	if sum != 40 {
		t.Fatalf("reserved=%v", sum)
	}
	// Stale CAS
	_, err = s.AmendPendingOrder(ctx, "c", "po-a", domain.PendingOrderAmend{
		RemainingQuantity: 0.4, TriggerPrice: 80, Quantity: 0.4,
		ReservedCash: 32, ExpectedRemaining: 1, ExpectedTrigger: 90, At: now,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("stale CAS want conflict: %v", err)
	}
	// Missing
	if _, err := s.AmendPendingOrder(ctx, "c", "nope", domain.PendingOrderAmend{
		RemainingQuantity: 1, TriggerPrice: 1, Quantity: 1, ExpectedRemaining: 1, ExpectedTrigger: 1, At: now,
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	// Not open
	if _, err := s.CancelPendingOrder(ctx, "c", "po-a", now, domain.CancelReasonUser); err != nil {
		t.Fatal(err)
	}
	_, err = s.AmendPendingOrder(ctx, "c", "po-a", domain.PendingOrderAmend{
		RemainingQuantity: 0.5, TriggerPrice: 80, Quantity: 0.5,
		ReservedCash: 40, ExpectedRemaining: 0.5, ExpectedTrigger: 80, At: now,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("canceled amend want conflict: %v", err)
	}
}

func TestSQLite_CancelOpenPendingOrdersMarketAndAll(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "cancel-all.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 10000, CashBalance: 10000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	mk := func(id, sym string, reserve float64) {
		t.Helper()
		if _, err := s.CreatePendingOrder(ctx, domain.PendingOrder{
			ID: id, ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: sym,
			Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, RemainingQuantity: 1,
			TriggerPrice: reserve, ReservedCash: reserve, Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	mk("a", "BTCUSDT", 90)
	mk("b", "BTCUSDT", 80)
	mk("c1", "ETHUSDT", 50)
	// Inactive bracket exit on BTC should also cancel with the market wipe
	if _, err := s.CreatePendingOrder(ctx, domain.PendingOrder{
		ID: "exit", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitSell, Side: domain.TradeSideSell, Quantity: 0, RemainingQuantity: 0,
		TriggerPrice: 120, Status: domain.PendingStatusPending, BracketID: "br", BracketRole: domain.BracketRoleTakeProfit,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := s.CancelOpenPendingOrders(ctx, "c", domain.ExchangeBinance, "BTCUSDT", now, domain.CancelReasonUser)
	if err != nil || len(got) != 3 {
		t.Fatalf("btc cancel n=%d err=%v", len(got), err)
	}
	sum, _ := s.SumReservedCash(ctx, "c")
	if sum != 50 {
		t.Fatalf("eth reserve left=%v", sum)
	}
	rest, err := s.CancelOpenPendingOrders(ctx, "c", "", "", now.Add(time.Second), domain.CancelReasonUser)
	if err != nil || len(rest) != 1 || rest[0].Symbol != "ETHUSDT" {
		t.Fatalf("all remaining %+v err=%v", rest, err)
	}
	sum, _ = s.SumReservedCash(ctx, "c")
	if sum != 0 {
		t.Fatalf("reserved after all=%v", sum)
	}
	none, err := s.CancelOpenPendingOrders(ctx, "c", "", "", now, domain.CancelReasonUser)
	if err != nil || len(none) != 0 {
		t.Fatalf("empty cancel %+v err=%v", none, err)
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
	}, nil); err != nil {
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
