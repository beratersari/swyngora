package market

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetSpotPerpLiquidity_PerpThickerThanSpot(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetSpotPerpLiquidity(context.Background(), "BTCUSDT", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.ThinnerBook != domain.SpotPerpSpot {
		t.Fatalf("%+v summary %s", got, got.Summary)
	}
	if got.Spot.Total <= 0 || got.Perp.Total <= got.Spot.Total {
		t.Fatalf("spot %v perp %v", got.Spot.Total, got.Perp.Total)
	}
	if !strings.Contains(got.Summary, "spot book is thinner") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestGetSpotPerpLiquidity_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetSpotPerpLiquidity(context.Background(), "", 1); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
