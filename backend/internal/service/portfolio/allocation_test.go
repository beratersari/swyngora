package portfolio

import (
	"context"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestAllocationBasket_CreatePreviewRebalance(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100",
		"binance|ETHUSDT": "50",
	}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "bsk-1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	// 80 BTC @ 100 = 8000 + 2000 cash. No ETH yet.
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "bsk-1", Symbol: "BTCUSDT", Side: "buy", Quantity: 80}); err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateAllocationBasket(ctx, AllocationBasketCreateInput{
		ClientID: "bsk-1", Name: "Core 50/30/20",
		Targets: []domain.AllocationTarget{
			{Asset: "BTC", WeightPct: 50},
			{Asset: "ETH", WeightPct: 30},
			{Asset: "USDT", WeightPct: 20},
		},
	})
	if err != nil || b.Name != "Core 50/30/20" || len(b.Targets) != 3 {
		t.Fatalf("%+v %v", b, err)
	}
	prev, err := svc.PreviewAllocationRebalance(ctx, "bsk-1", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(prev.Plan.Legs) == 0 {
		t.Fatalf("expected legs, lines=%+v", prev.Plan.Lines)
	}
	// Preview must not trade
	v1, _ := svc.View(ctx, "bsk-1")
	if math.Abs(v1.CashBalance-2000) > 1 {
		t.Fatalf("preview mutated cash=%v", v1.CashBalance)
	}
	view, trades, err := svc.ExecuteAllocationRebalance(ctx, "bsk-1", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(trades) == 0 {
		t.Fatalf("expected trades note=%s legs=%+v", view.Note, view.Plan.Legs)
	}
	v2, _ := svc.View(ctx, "bsk-1")
	// BTC should be closer to 50% of ~10k = 5000 → qty ~50
	var btcQty float64
	for _, p := range v2.Positions {
		if p.Symbol == "BTCUSDT" {
			btcQty = p.Quantity
		}
	}
	if btcQty > 55 || btcQty < 45 {
		t.Fatalf("btc qty after rebalance=%v cash=%v pos=%+v", btcQty, v2.CashBalance, v2.Positions)
	}
	// Drift is still allowed: updating targets does not trade
	newName := "Keep more BTC later"
	_, err = svc.UpdateAllocationBasket(ctx, AllocationBasketUpdateInput{
		ClientID: "bsk-1", BasketID: b.ID, Name: &newName,
		Targets: []domain.AllocationTarget{
			{Asset: "BTC", WeightPct: 65},
			{Asset: "ETH", WeightPct: 20},
			{Asset: "USDT", WeightPct: 15},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	v3, _ := svc.View(ctx, "bsk-1")
	if math.Abs(v3.CashBalance-v2.CashBalance) > 1e-6 {
		t.Fatal("update basket must not trade")
	}
}

func TestAllocationBasket_ListDelete(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "bsk-2", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateAllocationBasket(ctx, AllocationBasketCreateInput{
		ClientID: "bsk-2", Name: "All cash",
		Targets: []domain.AllocationTarget{{Asset: "USDT", WeightPct: 100}},
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.ListAllocationBaskets(ctx, "bsk-2")
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	if err := svc.DeleteAllocationBasket(ctx, "bsk-2", b.ID); err != nil {
		t.Fatal(err)
	}
	list, _ = svc.ListAllocationBaskets(ctx, "bsk-2")
	if len(list) != 0 {
		t.Fatalf("left %v", list)
	}
}
