package portfolio

import (
	"context"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestPreviewOrder_MarketBuyDoesNotPlace(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-buy", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-buy", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.FillsNow || got.FillPrice != 100 || got.Fee != 0 || math.Abs(got.CashDebit-100) > 1e-9 {
		t.Fatalf("quote %+v", got)
	}
	if math.Abs(got.CashAfter-9900) > 1e-9 || got.InsufficientCash || got.Risk.Blocked {
		t.Fatalf("after %+v risk %+v", got, got.Risk)
	}
	view, err := svc.View(ctx, "prev-buy")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-10000) > 1e-9 || len(view.Positions) != 0 {
		t.Fatalf("preview wrote the book: cash=%v positions=%d", view.CashBalance, len(view.Positions))
	}
}

func TestPreviewOrder_FeeAndSlippageMatchTheFill(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px).WithPaperCosts(domain.TradingCostFor)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-fee", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-fee", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	fill, _, fee, debit, _ := domain.QuoteFill(domain.TradeSideBuy, 100, 1, domain.TradingCostFor(domain.ExchangeBinance))
	if math.Abs(got.FillPrice-fill) > 1e-9 || math.Abs(got.Fee-fee) > 1e-9 || math.Abs(got.CashDebit-debit) > 1e-9 {
		t.Fatalf("preview fill=%v fee=%v debit=%v want %v %v %v", got.FillPrice, got.Fee, got.CashDebit, fill, fee, debit)
	}
}

func TestPreviewOrder_WeightBlocksBuyAndSellStaysOpen(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-w", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "prev-w", Symbol: "BTCUSDT", Side: "buy", Quantity: 20}); err != nil {
		t.Fatal(err)
	}
	w := 25.0
	if _, err := svc.SetRiskLimits(ctx, RiskLimitsInput{ClientID: "prev-w", MaxAssetWeightPct: &w}); err != nil {
		t.Fatal(err)
	}
	buy, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-w", Symbol: "BTCUSDT", Side: "buy", Quantity: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !buy.Risk.Blocked || !buy.Risk.WeightLimitHit {
		t.Fatalf("buy should be blocked: %+v", buy.Risk)
	}
	px.prices["binance|BTCUSDT"] = "110"
	sell, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-w", Symbol: "BTCUSDT", Side: "sell", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sell.Risk.Blocked || sell.Risk.Applies {
		t.Fatalf("sell blocked: %+v", sell.Risk)
	}
	if sell.EstimatedRealizedPnL == nil || math.Abs(*sell.EstimatedRealizedPnL-10) > 1e-6 {
		t.Fatalf("pnl=%v", sell.EstimatedRealizedPnL)
	}
	if sell.InsufficientQuantity {
		t.Fatal("had the coin")
	}
}

func TestPreviewOrder_LimitBuyReservesWithoutSpending(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-lim", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-lim", Symbol: "BTCUSDT", Type: "limit_buy", Quantity: 2, TriggerPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.FillsNow || math.Abs(got.ReservedCash-180) > 1e-9 || math.Abs(got.CashAfter-10000) > 1e-9 {
		t.Fatalf("reserve %+v", got)
	}
	if math.Abs(got.AvailableAfter-9820) > 1e-9 || got.InsufficientCash {
		t.Fatalf("available %+v", got)
	}
	orders, err := svc.ListPendingOrders(ctx, "prev-lim", "open", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 0 {
		t.Fatalf("preview created %d orders", len(orders))
	}
}

func TestPreviewOrder_MarginShowsLiquidationAndDoesNotOpen(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-m", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-m", Product: "margin", Symbol: "BTCUSDT", Side: "long", Type: "market",
		Quantity: 1, Leverage: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.InitialMargin-50) > 1e-9 || math.Abs(got.CashDebit-50) > 1e-9 {
		t.Fatalf("margin %+v", got)
	}
	if got.LiquidationPrice < 50 || got.LiquidationPrice > 51 {
		t.Fatalf("liq=%v", got.LiquidationPrice)
	}
	if !got.Risk.Applies || got.Risk.Blocked || got.InsufficientCash {
		t.Fatalf("risk %+v cash %v", got.Risk, got.InsufficientCash)
	}
	list, err := svc.ListMarginPositions(ctx, "prev-m")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("preview opened %d positions", len(list))
	}
}

func TestPreviewOrder_InsufficientCashIsAWarning(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-cash", StartingBalance: 50}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-cash", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.InsufficientCash || math.Abs(got.CashDebit-100) > 1e-9 {
		t.Fatalf("%+v", got)
	}
}
