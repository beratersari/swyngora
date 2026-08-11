package domain

import (
	"testing"
)

func TestScoreBookLiquidity_DeepBookHigh(t *testing.T) {
	raw := RawOrderBook{
		Bids: []PriceLevel{
			{Price: 99.95, Quantity: 2000}, // inside 0.1% of 100
			{Price: 99.6, Quantity: 5000},  // inside 0.5%
			{Price: 99.0, Quantity: 8000},  // inside 1%
		},
		Asks: []PriceLevel{
			{Price: 100.05, Quantity: 2000},
			{Price: 100.4, Quantity: 5000},
			{Price: 101.0, Quantity: 8000},
		},
	}
	got := ScoreBookLiquidity(raw, 100)
	if len(got.Bands) != 3 {
		t.Fatalf("bands=%d", len(got.Bands))
	}
	if got.Bands[0].RangePct != 0.1 || got.Bands[1].RangePct != 0.5 || got.Bands[2].RangePct != 1 {
		t.Fatalf("pcts %+v", got.Bands)
	}
	if got.Score < 50 {
		t.Fatalf("expected solid score for ~$400k near depth, got %v grade=%s", got.Score, got.Grade)
	}
	if got.WeakerSide != LiquidityWeakerBalanced {
		t.Fatalf("symmetric book weaker=%s", got.WeakerSide)
	}
}

func TestScoreBookLiquidity_ThinAskIsWeakerSell(t *testing.T) {
	raw := RawOrderBook{
		Bids: []PriceLevel{{Price: 99.95, Quantity: 100}},
		Asks: []PriceLevel{{Price: 100.05, Quantity: 1}},
	}
	got := ScoreBookLiquidity(raw, 100)
	if got.WeakerSide != LiquidityWeakerSell {
		t.Fatalf("want sell weaker, got %s imb=%v", got.WeakerSide, got.Weakness)
	}
	if got.Weakness < OrderBookBalancedAbs {
		t.Fatalf("weakness %v", got.Weakness)
	}
}

func TestScoreBookLiquidity_Empty(t *testing.T) {
	got := ScoreBookLiquidity(RawOrderBook{}, 0)
	if got.Score != 0 || got.Grade != LiquidityGradeVeryLow {
		t.Fatalf("%+v", got)
	}
}

func TestScoreNotionalUSD_Anchors(t *testing.T) {
	if s := scoreNotionalUSD(0); s != 0 {
		t.Fatalf("0 → %v", s)
	}
	if s := scoreNotionalUSD(1000); s < 19 || s > 21 {
		t.Fatalf("$1k → %v", s)
	}
	justOver := scoreNotionalUSD(1001)
	atFloor := scoreNotionalUSD(1000)
	if justOver <= atFloor {
		t.Fatalf("$1001 (%v) must score above $1000 (%v)", justOver, atFloor)
	}
	if s := scoreNotionalUSD(1_000_000); s < 79 || s > 81 {
		t.Fatalf("$1M → %v (want 80: 20 + 20*log10(1000))", s)
	}
	if s := scoreNotionalUSD(10_000_000); s != 100 {
		t.Fatalf("$10M → %v", s)
	}
	prev := 0.0
	for _, usd := range []float64{1, 100, 999, 1000, 1001, 10_000, 100_000, 1_000_000} {
		s := scoreNotionalUSD(usd)
		if s < prev {
			t.Fatalf("score dropped: $%g → %v after %v", usd, s, prev)
		}
		prev = s
	}
}

func TestScoreBookLiquidity_OmitsUnreachableBands(t *testing.T) {
	// Only ±0.05% of depth — must not publish 0.1 / 0.5 / 1 as if they were full.
	raw := RawOrderBook{
		Bids: []PriceLevel{{Price: 99.95, Quantity: 10}},
		Asks: []PriceLevel{{Price: 100.05, Quantity: 10}},
	}
	got := ScoreBookLiquidity(raw, 100)
	if len(got.Bands) != 0 {
		t.Fatalf("no preset band is covered: %+v", got.Bands)
	}
	if got.UsedRangePct > 0.06 {
		t.Fatalf("used=%v", got.UsedRangePct)
	}
}

func TestScoreBookLiquidity_StopsAtHalfPercent(t *testing.T) {
	raw := RawOrderBook{
		Bids: []PriceLevel{{Price: 99.95, Quantity: 5}, {Price: 99.50, Quantity: 5}},
		Asks: []PriceLevel{{Price: 100.05, Quantity: 5}, {Price: 100.50, Quantity: 5}},
	}
	got := ScoreBookLiquidity(raw, 100)
	if len(got.Bands) != 2 || got.Bands[0].RangePct != 0.1 || got.Bands[1].RangePct != 0.5 {
		t.Fatalf("want 0.1 and 0.5 only: %+v", got.Bands)
	}
}

func TestScoreMarketLiquidity_UsesCommonReach(t *testing.T) {
	deep := VenueRawBook{Exchange: ExchangeBinance, Book: RawOrderBook{
		Bids: []PriceLevel{{Price: 99.95, Quantity: 10}, {Price: 99.0, Quantity: 50}},
		Asks: []PriceLevel{{Price: 100.05, Quantity: 10}, {Price: 101.0, Quantity: 50}},
	}}
	shallow := VenueRawBook{Exchange: ExchangeBybit, Book: RawOrderBook{
		Bids: []PriceLevel{{Price: 99.95, Quantity: 10}},
		Asks: []PriceLevel{{Price: 100.05, Quantity: 10}},
	}}
	got := ScoreMarketLiquidity([]VenueRawBook{deep, shallow})
	if len(got.Bands) != 0 {
		t.Fatalf("common reach is ~0.05%%, no preset band: %+v", got)
	}
	// Give shallow enough to cover 0.1% both ways but not 0.5%.
	shallow.Book.Bids = append(shallow.Book.Bids, PriceLevel{Price: 99.89, Quantity: 1})
	shallow.Book.Asks = append(shallow.Book.Asks, PriceLevel{Price: 100.11, Quantity: 1})
	got = ScoreMarketLiquidity([]VenueRawBook{deep, shallow})
	if len(got.Bands) != 1 || got.Bands[0].RangePct != 0.1 {
		t.Fatalf("common should be 0.1%% only: %+v", got)
	}
}
