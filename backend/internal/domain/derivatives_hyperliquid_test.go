package domain

import (
	"strings"
	"testing"
	"time"
)

func TestBuildDerivativesBoard_HyperliquidUsesTapeAndBook(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	coin := DerivativesCoinInput{
		Symbol:           "BTCUSDT",
		CVDSplitKind:     "price_up_cvd_down",
		CVDSplitDuration: "15m",
		CVDSplitSeconds:  900,
		CVDSplitOK:       true,
		Venues: []DerivativesVenueInput{{
			Exchange:           ExchangeHyperliquid,
			OIValue:            200,
			OIAt:               now,
			OIOK:               true,
			FundingRate:        -0.000005,
			FundingAt:          now,
			FundingOK:          true,
			MarkPrice:          100,
			LiqLong:            999,
			LiqShort:           999,
			LongShare:          0.6,
			LongShortOK:        true,
			TakerBuy:           10,
			TakerSell:          1,
			TakerOK:            true,
			CVDDay:             50,
			CVDAt:              now,
			CVDDayOK:           true,
			CVDSplitKind:       "price_up_cvd_down",
			CVDSplitDuration:   "15m",
			CVDSplitSeconds:    900,
			CVDSplitAt:         now,
			CVDSplitOK:         true,
			PrintNotional:      200,
			PrintCount:         2,
			PrintPriorNotional: 100,
			PrintPriorCount:    2,
			PrintAt:            now,
			PrintOK:            true,
			OrderBuy:           3,
			OrderSell:          1,
			OrderAt:            now,
			OrderOK:            true,
			BookOK:             true,
			BookAt:             now,
			BidNotional:        80,
			AskNotional:        20,
		}},
	}
	got := BuildDerivativesBoard("hyperliquid", []DerivativesCoinInput{coin}, now)
	if got.Exchange != "hyperliquid" || len(got.Coins) != 1 {
		t.Fatalf("board %+v", got)
	}
	if !strings.Contains(got.Note, "20 levels") {
		t.Fatalf("note %s", got.Note)
	}
	row := got.Coins[0]
	if !row.OpenInterestDefinite || row.OpenInterest == "" || !row.FundingDefinite || row.FundingRatePct == "" {
		t.Fatalf("mark columns %+v", row)
	}
	if row.Price == "" || row.PriceExchange != "hyperliquid" {
		t.Fatalf("price %s %s", row.Price, row.PriceExchange)
	}
	if !row.TakerDefinite || row.TakerBuyPct == "" || !row.CVDDayDefinite || row.CVDDay == "" || !row.BidDefinite || row.BidPct != "80" {
		t.Fatalf("tape or book %+v", row)
	}
	if !row.PrintDefinite || row.PrintChangePct == "" || !row.OrderDefinite || row.OrderBuy != 3 || !row.CVDSplitDefinite {
		t.Fatalf("print or split %+v", row)
	}
	if row.LiqDefinite || row.LiqLong != "" || row.LongDefinite || row.GapDefinite || row.LiqHeatDefinite {
		t.Fatalf("liquidation or crowd leaked %+v", row)
	}
	venue := row.Venues[0]
	for _, status := range []string{venue.LiqStatus, venue.LongStatus, venue.GapStatus, venue.LiqHeatStatus} {
		if status != VenueReadUnsupported {
			t.Fatalf("status %s row %+v", status, venue)
		}
	}
	if venue.TakerStatus != VenueReadFresh || venue.CVDStatus != VenueReadFresh || venue.BookStatus != VenueReadFresh || venue.OrderStatus != VenueReadFresh {
		t.Fatalf("fresh tape %+v", venue)
	}
	if venue.LiqLong != "" || venue.TakerBuyPct == "" || venue.CVDDay == "" || venue.BidPct != "80" {
		t.Fatalf("cells %+v", venue)
	}
}
