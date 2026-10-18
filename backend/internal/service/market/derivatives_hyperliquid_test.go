package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetDerivativesBoard_HyperliquidIsMarkOpenInterestAndFunding(t *testing.T) {
	now := time.Now().UTC()
	oi := &domain.OpenInterestSeries{
		Current: domain.OpenInterestPoint{Time: now, Value: 200, Contracts: 2},
		History: []domain.OpenInterestPoint{
			{Time: now.Add(-time.Hour), Value: 180, Contracts: 1.8},
			{Time: now.Add(-24 * time.Hour), Value: 100, Contracts: 1},
		},
	}
	fund := &domain.FundingSeries{
		Current:       domain.FundingPoint{Time: now, Rate: -0.000005, MarkPrice: 100, Predicted: true},
		IntervalHours: 1,
	}
	svc := New(nil, nil).
		WithOpenInterest(map[domain.Exchange]domain.OpenInterestPort{
			domain.ExchangeHyperliquid: boardOI{series: oi},
		}).
		WithFundingRate(map[domain.Exchange]domain.FundingRatePort{
			domain.ExchangeHyperliquid: boardFunding{series: fund},
		})
	got, err := svc.GetDerivativesBoard(context.Background(), "hyperliquid", 5)
	if err != nil {
		t.Fatal(err)
	}
	if got.Universe != 5 || !got.Coins[0].OpenInterestDefinite || !got.Coins[0].FundingDefinite {
		t.Fatalf("%+v", got.Coins[0])
	}
	if got.Coins[0].PriceExchange != "hyperliquid" || got.Coins[0].LiqDefinite || got.Coins[0].TakerDefinite || got.Coins[0].CVDDayDefinite || got.Coins[0].BidDefinite {
		t.Fatalf("columns %+v", got.Coins[0])
	}
	venue := got.Coins[0].Venues[0]
	if venue.LiqStatus != domain.VenueReadUnsupported || venue.LongStatus != domain.VenueReadUnsupported || venue.GapStatus != domain.VenueReadUnsupported {
		t.Fatalf("liquidation or crowd %+v", venue)
	}
	if venue.BookStatus != domain.VenueReadMissing || venue.TakerStatus != domain.VenueReadMissing || venue.CVDStatus != domain.VenueReadMissing {
		t.Fatalf("a missing tape is not unsupported %+v", venue)
	}
	if _, err := svc.GetDerivativesBoard(context.Background(), "coinbase", 5); err == nil {
		t.Fatal("coinbase is not a derivatives venue")
	}
}
