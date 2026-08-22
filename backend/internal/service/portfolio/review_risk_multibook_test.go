package portfolio

import (
	"context"
	"testing"
)

// Risk helpers view the resolved book (owner + book id), so GET/SET limits
// and guarded buys still work after a second book exists.
func TestReview_RiskLimitsBrokenAfterSecondBook(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	main, err := svc.Create(ctx, CreateInput{ClientID: "risk-mb", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	loss := 10.0
	if _, err := svc.SetRiskLimits(ctx, RiskLimitsInput{ClientID: "risk-mb", PortfolioID: main.ID, MaxDailyLossPct: &loss}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "risk-mb", Name: "Alt", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.GetRiskLimitsView(ctx, "risk-mb", main.ID); err != nil {
		t.Fatalf("GetRisk after second book: %v", err)
	}
	if _, err := svc.SetRiskLimits(ctx, RiskLimitsInput{ClientID: "risk-mb", PortfolioID: main.ID, MaxDailyLossPct: &loss}); err != nil {
		t.Fatalf("SetRisk after second book: %v", err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "risk-mb", PortfolioID: main.ID, Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); err != nil {
		t.Fatalf("PlaceOrder after second book: %v", err)
	}
}

func TestReview_RiskLimitsCannotBeSetOnExtraBook(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "risk-x", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	alt, err := svc.Create(ctx, CreateInput{ClientID: "risk-x", Name: "Alt", StartingBalance: 5000})
	if err != nil {
		t.Fatal(err)
	}
	w := 40.0
	view, err := svc.SetRiskLimits(ctx, RiskLimitsInput{ClientID: "risk-x", PortfolioID: alt.ID, MaxAssetWeightPct: &w})
	if err != nil {
		t.Fatalf("SetRiskLimits on extra book: %v", err)
	}
	if view.Limits.MaxAssetWeightPct == nil || *view.Limits.MaxAssetWeightPct != w {
		t.Fatalf("extra book limits: %+v", view.Limits)
	}
}
