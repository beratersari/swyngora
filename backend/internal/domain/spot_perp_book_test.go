package domain

import (
	"strings"
	"testing"
	"time"
)

func spotPerpBook(touchBid, touchAsk, farBid, farAsk float64) RawOrderBook {
	return RawOrderBook{
		Bids: []PriceLevel{{Price: 100, Quantity: touchBid}, {Price: 98, Quantity: farBid}},
		Asks: []PriceLevel{{Price: 100.05, Quantity: touchAsk}, {Price: 102, Quantity: farAsk}},
	}
}

func TestCompareSpotPerp_SpotIsThinnerAndAsksAreThin(t *testing.T) {
	spot := []VenueRawBook{{
		Exchange: ExchangeBinance,
		Book:     spotPerpBook(2, 0.2, 2, 0.2),
	}}
	perp := []VenueRawBook{{
		Exchange: ExchangeBinance,
		Book:     spotPerpBook(50, 10, 50, 10),
	}}
	got := CompareSpotPerpLiquidity("BTCUSDT", spot, perp, 1, time.Unix(0, 0).UTC())
	if got.ThinnerBook != SpotPerpSpot || got.StrongerBook != SpotPerpPerp {
		t.Fatalf("book %+v", got)
	}
	if got.Spot.ThinnerSide != SpotPerpAsk || got.Perp.ThinnerSide != SpotPerpAsk {
		t.Fatalf("sides spot=%s perp=%s", got.Spot.ThinnerSide, got.Perp.ThinnerSide)
	}
	if !got.Spot.Complete || got.Ratio <= 1 {
		t.Fatalf("ratio %v complete %v", got.Ratio, got.Spot.Complete)
	}
	if len(got.Venues) != 1 || got.Venues[0].ThinnerBook != SpotPerpSpot {
		t.Fatalf("venues %+v", got.Venues)
	}
	if !strings.Contains(got.Summary, "spot book is thinner") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestCompareSpotPerp_EvenBooks(t *testing.T) {
	book := spotPerpBook(10, 10, 10, 10)
	spot := []VenueRawBook{{Exchange: ExchangeBinance, Book: book}}
	perp := []VenueRawBook{{Exchange: ExchangeBybit, Book: book}}
	got := CompareSpotPerpLiquidity("ETHUSDT", spot, perp, 1, time.Unix(0, 0).UTC())
	if got.ThinnerBook != SpotPerpEven || got.Spot.ThinnerSide != SpotPerpBalanced {
		t.Fatalf("%+v", got)
	}
}

func TestCompareSpotPerp_SkipsBandTheBookDoesNotReach(t *testing.T) {
	// Levels sit about 0.4% from mid, so 0.1% counts and 0.5% / 1% do not.
	near := RawOrderBook{
		Bids: []PriceLevel{{Price: 99.6, Quantity: 10}},
		Asks: []PriceLevel{{Price: 100.4, Quantity: 10}},
	}
	spot := []VenueRawBook{{Exchange: ExchangeBinance, Book: near}}
	perp := []VenueRawBook{{Exchange: ExchangeBinance, Book: near}}
	got := CompareSpotPerpLiquidity("BTCUSDT", spot, perp, 1, time.Unix(0, 0).UTC())
	if len(got.Bands) != 3 {
		t.Fatalf("bands %+v", got.Bands)
	}
	if !got.Bands[0].Complete || got.Bands[1].Complete || got.Bands[2].Complete {
		t.Fatalf("bands %+v", got.Bands)
	}
	if got.RangePct != 0.1 {
		t.Fatalf("headline %v", got.RangePct)
	}
}

func TestParseSpotPerpRangePct(t *testing.T) {
	if v, err := ParseSpotPerpRangePct(0); err != nil || v != 1 {
		t.Fatalf("%v %v", v, err)
	}
	if _, err := ParseSpotPerpRangePct(2); err == nil {
		t.Fatal("expected error")
	}
}
