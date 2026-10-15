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

func TestPreviewOrder_RiskPercentSizesSpotBuyAndDoesNotPlace(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-risk", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-risk", Symbol: "BTCUSDT", Side: "buy", RiskPercent: 1, StopLossPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Sizing == nil || math.Abs(got.Quantity-10) > 1e-9 || math.Abs(got.CashDebit-1000) > 1e-6 {
		t.Fatalf("sized %+v debit=%v", got.Sizing, got.CashDebit)
	}
	if got.Sizing.StopAttached || got.Sizing.StopBeyondLiquidation {
		t.Fatalf("spot market stop is not an order: %+v", got.Sizing)
	}
	if math.Abs(got.Sizing.EstimatedLoss-100) > 1e-6 || math.Abs(got.Sizing.Equity-10000) > 1e-6 {
		t.Fatalf("sizing %+v", got.Sizing)
	}
	view, err := svc.View(ctx, "prev-risk")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(view.CashBalance-10000) > 1e-9 || len(view.Positions) != 0 {
		t.Fatalf("preview wrote the book: cash=%v positions=%d", view.CashBalance, len(view.Positions))
	}
}

func TestPreviewOrder_RiskPercentUsesLimitAndIgnoresSentQuantity(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-risk-lim", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-risk-lim", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 1, TriggerPrice: 80, StopLossPrice: 70, RiskPercent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 1% of 10000 is 100. The stop is 10 under the limit, so the size is 10, not the sent 1.
	if math.Abs(got.Quantity-10) > 1e-9 || math.Abs(got.ReservedCash-800) > 1e-6 || got.FillsNow {
		t.Fatalf("%+v", got)
	}
	if got.Sizing == nil || got.Sizing.StopAttached {
		t.Fatalf("limit buy does not place the stop: %+v", got.Sizing)
	}
}

func TestPreviewOrder_RiskPercentSizesBracketAndAttachesStop(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-risk-br", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-risk-br", Symbol: "BTCUSDT", Type: "bracket",
		TriggerPrice: 90, TakeProfitPrice: 120, StopLossPrice: 80, RiskPercent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Sizing == nil || !got.Sizing.StopAttached || math.Abs(got.Quantity-10) > 1e-9 {
		t.Fatalf("%+v qty=%v", got.Sizing, got.Quantity)
	}
	if len(got.Legs) != 2 {
		t.Fatalf("legs=%d", len(got.Legs))
	}
}

func TestPreviewOrder_RiskPercentSizesMarginAndFlagsLiquidation(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-risk-m", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	safe, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-risk-m", Product: "margin", Symbol: "BTCUSDT", Side: "long",
		Leverage: 2, RiskPercent: 1, StopLossPrice: 90,
	})
	if err != nil {
		t.Fatal(err)
	}
	if safe.Sizing == nil || !safe.Sizing.StopAttached || safe.Sizing.StopBeyondLiquidation {
		t.Fatalf("stop should be the exit: %+v liq=%v", safe.Sizing, safe.LiquidationPrice)
	}
	if math.Abs(safe.Quantity-10) > 1e-9 || math.Abs(safe.InitialMargin-500) > 1e-6 {
		t.Fatalf("qty=%v margin=%v", safe.Quantity, safe.InitialMargin)
	}
	wide, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-risk-m", Product: "margin", Symbol: "BTCUSDT", Side: "long",
		Leverage: 10, RiskPercent: 1, StopLossPrice: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	if wide.Sizing == nil || !wide.Sizing.StopBeyondLiquidation || wide.LiquidationPrice <= 80 {
		t.Fatalf("liq=%v sizing=%+v", wide.LiquidationPrice, wide.Sizing)
	}
	list, err := svc.ListMarginPositions(ctx, "prev-risk-m")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("preview opened %d positions", len(list))
	}
}

func TestPreviewOrder_RiskPercentRejectsSpotSell(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "prev-risk-sell", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.PreviewOrder(ctx, OrderPreviewInput{
		ClientID: "prev-risk-sell", Symbol: "BTCUSDT", Side: "sell", RiskPercent: 1, StopLossPrice: 110,
	})
	if err == nil {
		t.Fatal("expected a sell to be rejected")
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
