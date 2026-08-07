package portfolio

import (
	"context"
	"errors"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestRiskLimits_BlockConcentrationBuyAllowSell(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100", "binance|ETHUSDT": "50"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "risk-w", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	// Drift first (no limit) — user may already be overweight when they turn the rule on.
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-w", Symbol: "BTCUSDT", Side: "buy", Quantity: 80}); err != nil {
		t.Fatal(err)
	}
	w := 30.0
	if _, err := svc.SetRiskLimits(ctx, RiskLimitsInput{ClientID: "risk-w", MaxAssetWeightPct: &w}); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-w", Symbol: "BTCUSDT", Side: "buy", Quantity: 1})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want forbidden BTC buy: %v", err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-w", Symbol: "BTCUSDT", Side: "sell", Quantity: 1}); err != nil {
		t.Fatalf("sell must stay allowed: %v", err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-w", Symbol: "ETHUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatalf("small ETH buy should pass: %v", err)
	}
	view, err := svc.GetRiskLimitsView(ctx, "risk-w")
	if err != nil || view.Limits.MaxAssetWeightPct == nil {
		t.Fatalf("%+v %v", view, err)
	}
	var btcOver bool
	for _, a := range view.Status.Assets {
		if a.Asset == "BTC" && a.AtOrOverLimit {
			btcOver = true
		}
	}
	if !btcOver {
		t.Fatalf("expected BTC over weight: %+v", view.Status.Assets)
	}
}

func TestRiskLimits_DailyLossBlocksBuyAndMargin(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "risk-d", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	d := 5.0
	if _, err := svc.SetRiskLimits(ctx, RiskLimitsInput{ClientID: "risk-d", MaxDailyLossPct: &d}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-d", Symbol: "BTCUSDT", Side: "buy", Quantity: 50}); err != nil {
		t.Fatal(err)
	}
	px.prices["binance|BTCUSDT"] = "90" // equity ~ 5000 cash + 4500 mark = 9500 → -5%
	_, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-d", Symbol: "BTCUSDT", Side: "buy", Quantity: 1})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("want daily loss block: %v", err)
	}
	_, _, err = svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "risk-d", Symbol: "BTCUSDT", Side: "long", Quantity: 0.1, Leverage: 2,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("margin open blocked: %v", err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-d", Symbol: "BTCUSDT", Side: "sell", Quantity: 1}); err != nil {
		t.Fatalf("sell still allowed: %v", err)
	}
	if err := svc.ClearRiskLimits(ctx, "risk-d"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{ClientID: "risk-d", Symbol: "BTCUSDT", Side: "buy", Quantity: 1}); err != nil {
		t.Fatalf("after clear buy: %v", err)
	}
}
