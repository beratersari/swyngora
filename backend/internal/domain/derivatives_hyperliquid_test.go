package domain

import (
	"testing"
	"time"
)

func TestBuildDerivativesBoard_HyperliquidKeepsTapeAndLiquidationsBlank(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	coin := DerivativesCoinInput{
		Symbol: "BTCUSDT",
		Venues: []DerivativesVenueInput{{
			Exchange:    ExchangeHyperliquid,
			OIValue:     200,
			OIAt:        now,
			OIOK:        true,
			FundingRate: -0.000005,
			FundingAt:   now,
			FundingOK:   true,
			MarkPrice:   100,
			LiqLong:     999,
			LiqShort:    999,
			LongShare:   0.6,
			LongShortOK: true,
			TakerBuy:    10,
			TakerSell:   1,
			TakerOK:     true,
			CVDDay:      50,
			CVDDayOK:    true,
			BookOK:      true,
			BidNotional: 80,
			AskNotional: 20,
		}},
	}
	got := BuildDerivativesBoard("hyperliquid", []DerivativesCoinInput{coin}, now)
	if got.Exchange != "hyperliquid" || len(got.Coins) != 1 {
		t.Fatalf("board %+v", got)
	}
	row := got.Coins[0]
	if !row.OpenInterestDefinite || row.OpenInterest == "" || !row.FundingDefinite || row.FundingRatePct == "" {
		t.Fatalf("mark columns %+v", row)
	}
	if row.Price == "" || row.PriceExchange != "hyperliquid" {
		t.Fatalf("price %s %s", row.Price, row.PriceExchange)
	}
	if row.LiqDefinite || row.LiqLong != "" || row.LongDefinite || row.TakerDefinite || row.CVDDayDefinite || row.BidDefinite || row.GapDefinite {
		t.Fatalf("unsupported columns leaked %+v", row)
	}
	venue := row.Venues[0]
	for _, status := range []string{venue.LiqStatus, venue.LongStatus, venue.TakerStatus, venue.CVDStatus, venue.BookStatus, venue.GapStatus, venue.OrderStatus} {
		if status != VenueReadUnsupported {
			t.Fatalf("status %s row %+v", status, venue)
		}
	}
	if venue.LiqLong != "" || venue.TakerBuyPct != "" || venue.CVDDay != "" || venue.BidPct != "" {
		t.Fatalf("blank cells %+v", venue)
	}
}
