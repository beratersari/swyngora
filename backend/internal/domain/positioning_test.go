package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestClassifyPositioningRegime(t *testing.T) {
	cases := []struct {
		p, o float64
		want string
	}{
		{1, 2, RegimeLongBuildup},
		{-1, 2, RegimeShortBuildup},
		{-1, -2, RegimeLongUnwinding},
		{1, -2, RegimeShortCovering},
		{0.05, 2, RegimeNeutral}, // price flat
		{1, 0.1, RegimeNeutral},  // OI flat
	}
	for _, tc := range cases {
		got, _, _ := ClassifyPositioningRegime(tc.p, tc.o)
		if got != tc.want {
			t.Fatalf("p=%v o=%v got %s want %s", tc.p, tc.o, got, tc.want)
		}
	}
}

func TestBuildPositioningVenue_ShortBuildupReasons(t *testing.T) {
	got := BuildPositioningVenue(PositioningInputs{
		Exchange:    ExchangeBinance,
		Symbol:      "BTCUSDT",
		Price:       64000,
		OIValue:     1e9,
		Price1hPct:  -0.8,
		Price4hPct:  -1.5,
		Price24hPct: -3,
		OI1hPct:     1.2,
		OI4hPct:     2.5,
		OI24hPct:    4,
		LongShare:   0.42,
		ShortShare:  0.58,
		FundingRate: -0.0002,
	})
	if got.Regime != RegimeShortBuildup {
		t.Fatalf("regime %s primary %+v", got.Regime, got.Primary)
	}
	if got.Label != "Short buildup" {
		t.Fatalf("label %s", got.Label)
	}
	if len(got.Reasons) < 2 {
		t.Fatalf("reasons %v", got.Reasons)
	}
	found := false
	for _, r := range got.Reasons {
		if strings.Contains(strings.ToLower(r), "short") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected short explanation: %v", got.Reasons)
	}
}

func TestBuildPositioningVenue_CrowdVsSize(t *testing.T) {
	got := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT",
		Price: 100, OIValue: 1e6,
		Price1hPct: 0.5, Price4hPct: 2, Price24hPct: 4,
		OI1hPct: 1, OI4hPct: 3, OI24hPct: 5,
		LongShare: 0.68, ShortShare: 0.32,
		CrowdingSplit: LongShortSplit{
			Alignment: AlignOpposite, Important: true,
			General: "long", TopPositions: "short", SizeBias: "short",
			Detail: "All accounts are long (68% long) while top-trader positions are short (32% long). Size is with the bigger books, not the crowd.",
		},
	})
	found := false
	for _, r := range got.Reasons {
		if strings.Contains(r, "bigger books") {
			found = true
		}
	}
	if !found {
		t.Fatalf("reasons %+v", got.Reasons)
	}
}

func TestBuildPositioningVenue_LongBuildup(t *testing.T) {
	got := BuildPositioningVenue(PositioningInputs{
		Exchange:    ExchangeBybit,
		Symbol:      "ETHUSDT",
		Price4hPct:  2,
		OI4hPct:     3,
		LongShare:   0.62,
		FundingRate: 0.0001,
	})
	if got.Regime != RegimeLongBuildup {
		t.Fatalf("%s", got.Regime)
	}
}

func TestCombinePositioningReports_Agree(t *testing.T) {
	a := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBinance, OIValue: 9e9, Price4hPct: 1.2, OI4hPct: 2,
	})
	b := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBybit, OIValue: 1e9, Price4hPct: 1.0, OI4hPct: 1.5,
	})
	c := CombinePositioningReports([]PositioningVenueReport{a, b})
	if c == nil || c.Regime != RegimeLongBuildup || c.Agreement != "agree" {
		t.Fatalf("%+v", c)
	}
	if c.DominantVenue != "binance" {
		t.Fatalf("dom %s", c.DominantVenue)
	}
}

func TestCombinePositioningReports_WithholdsUnusableVenue(t *testing.T) {
	a := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBinance, OIValue: 9e9, Price4hPct: 1.2, OI4hPct: 2,
	})
	b := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBybit, OIValue: 1e9, Price4hPct: 1.0, OI4hPct: 1.5,
	})
	b.Error = "oi down"
	c := CombinePositioningReports([]PositioningVenueReport{a, b})
	if c == nil || c.Agreement != "withheld" || c.Regime != "" {
		t.Fatalf("%+v", c)
	}
	if !strings.Contains(c.Summary, "bybit") {
		t.Fatalf("summary %s", c.Summary)
	}
	if len(c.Missing) != 1 || c.Missing[0] != "bybit" {
		t.Fatalf("missing %+v", c.Missing)
	}
	b.Error = ""
	b.OpenInterest = 0
	c = CombinePositioningReports([]PositioningVenueReport{a, b})
	if c == nil || c.Agreement != "withheld" || len(c.Missing) != 1 || c.Missing[0] != "bybit" {
		t.Fatalf("zero oi %+v", c)
	}
}

func TestCombinePositioningReports_Mixed(t *testing.T) {
	a := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBinance, OIValue: 5e9, Price4hPct: 1.2, OI4hPct: 2,
	})
	b := BuildPositioningVenue(PositioningInputs{
		Exchange: ExchangeBybit, OIValue: 4e9, Price4hPct: -1.2, OI4hPct: 2,
	})
	c := CombinePositioningReports([]PositioningVenueReport{a, b})
	if c == nil || c.Agreement != "mixed" {
		t.Fatalf("%+v", c)
	}
}

func TestPriceChangeOverBars(t *testing.T) {
	closes := []float64{100, 101, 102, 104}
	// 1 bar: 104 vs 102 ≈ +1.96%
	p1 := PriceChangeOverBars(closes, 1)
	want := (104.0 - 102.0) / 102.0 * 100
	if math.IsNaN(p1) || math.Abs(p1-want) > 0.01 {
		t.Fatalf("got %v want %v", p1, want)
	}
	if !math.IsNaN(PriceChangeOverBars(closes, 10)) {
		t.Fatal("expected nan")
	}
}

func TestClosesFromCandles_Sorts(t *testing.T) {
	c := []Candle{
		{OpenTime: time.Unix(2, 0), Close: "102"},
		{OpenTime: time.Unix(1, 0), Close: "100"},
		{OpenTime: time.Unix(3, 0), Close: "105"},
	}
	got := ClosesFromCandles(c)
	if len(got) != 3 || got[0] != 100 || got[2] != 105 {
		t.Fatalf("%v", got)
	}
	pct := PriceChangePctFromCloses(got)
	want := (105 - 100) / 100.0 * 100
	if math.Abs(pct-want) > 1e-9 {
		t.Fatalf("%v want %v", pct, want)
	}
}
