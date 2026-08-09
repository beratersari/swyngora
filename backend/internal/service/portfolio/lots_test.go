package portfolio

import (
	"context"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestCash_LotsFIFOAndLIFO(t *testing.T) {
	ctx := context.Background()
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)

	if _, err := svc.Create(ctx, CreateInput{ClientID: "lot-a", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "lot-a", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "200"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "lot-a", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	lots, err := svc.ListLots(ctx, "lot-a", "binance", "BTCUSDT", "open")
	if err != nil || len(lots) != 2 {
		t.Fatalf("lots=%d err=%v", len(lots), err)
	}

	px.prices["binance|BTCUSDT"] = "180"
	tr, view, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "lot-a", Symbol: "BTCUSDT", Side: "sell", Quantity: 1, LotMethod: "fifo"})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tr.RealizedPnL-80) > 1e-6 {
		t.Fatalf("fifo realized=%v", tr.RealizedPnL)
	}
	if math.Abs(view.CashBalance-(10000-100-200+180)) > 1e-6 {
		t.Fatalf("cash=%v", view.CashBalance)
	}
	if len(view.Positions) != 1 || math.Abs(view.Positions[0].Quantity-1) > 1e-9 {
		t.Fatalf("pos=%+v", view.Positions)
	}
	if math.Abs(view.Positions[0].AvgCost-200) > 1e-6 {
		t.Fatalf("remaining avg fifo=%v", view.Positions[0].AvgCost)
	}
	if len(tr.LotFills) != 1 || tr.LotFills[0].LotID != lots[0].ID {
		t.Fatalf("fills=%+v want lot %s", tr.LotFills, lots[0].ID)
	}

	if _, err := svc.Create(ctx, CreateInput{ClientID: "lot-b", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "100"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "lot-b", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "200"
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "lot-b", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "180"
	tr, view, err = svc.PlaceOrder(ctx, OrderInput{ClientID: "lot-b", Symbol: "BTCUSDT", Side: "sell", Quantity: 1, LotMethod: "lifo"})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(tr.RealizedPnL-(-20)) > 1e-6 {
		t.Fatalf("lifo realized=%v", tr.RealizedPnL)
	}
	if math.Abs(view.Positions[0].AvgCost-100) > 1e-6 {
		t.Fatalf("remaining avg lifo=%v", view.Positions[0].AvgCost)
	}
	hist, total, err := svc.ListTrades(ctx, "lot-b", 10, 0)
	if err != nil || total != 3 || len(hist) != 3 {
		t.Fatalf("history total=%d n=%d err=%v", total, len(hist), err)
	}
}

func TestCash_LotsPendingSellAndPartialLot(t *testing.T) {
	ctx := context.Background()
	px := &fakePx{prices: map[string]string{"binance|ETHUSDT": "10"}}
	svc := newSvc(t, px)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "lot-p", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "lot-p", Exchange: "binance", Symbol: "ETHUSDT", Side: "buy", Quantity: 4}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "lot-p", Symbol: "ETHUSDT", Type: "limit_sell", Quantity: 1.5, TriggerPrice: 12, LotMethod: "fifo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.LotMethod != domain.LotMethodFIFO {
		t.Fatalf("pending lotMethod=%s", o.LotMethod)
	}
	got, ok, err := svc.TryFillPendingOrder(ctx, *o, 12, 0)
	if err != nil || !ok || got == nil {
		t.Fatalf("fill %+v ok=%v err=%v", got, ok, err)
	}
	lots, err := svc.ListLots(ctx, "lot-p", "binance", "ETHUSDT", "open")
	if err != nil || len(lots) != 1 {
		t.Fatalf("open lots=%d err=%v %+v", len(lots), err, lots)
	}
	if math.Abs(lots[0].Quantity-2.5) > 1e-9 {
		t.Fatalf("remaining lot qty=%v", lots[0].Quantity)
	}
	if lots[0].OriginalQuantity != 4 {
		t.Fatalf("original=%v", lots[0].OriginalQuantity)
	}
}
