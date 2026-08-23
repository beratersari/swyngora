package portfolio

import (
	"context"
	"errors"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Extra-book risk limits must actually block overweight buys on that book.
func TestHunt_ExtraBookRiskLimitsAreEnforced(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "risk-eb", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	alt, err := svc.Create(ctx, CreateInput{ClientID: "risk-eb", Name: "Alt", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "risk-eb", PortfolioID: alt.ID, Symbol: "BTCUSDT", Side: "buy", Quantity: 80,
	}); err != nil {
		t.Fatal(err)
	}
	w := 30.0
	if _, err := svc.SetRiskLimits(ctx, RiskLimitsInput{
		ClientID: "risk-eb", PortfolioID: alt.ID, MaxAssetWeightPct: &w,
	}); err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.PlaceOrder(ctx, OrderInput{
		ClientID: "risk-eb", PortfolioID: alt.ID, Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("extra-book weight limit not enforced: %v", err)
	}
}
