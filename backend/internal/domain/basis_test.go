package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestComputeBasis(t *testing.T) {
	d, pct, kind := ComputeBasis(101, 100)
	if kind != BasisKindPremium || math.Abs(d-1) > 1e-12 || math.Abs(pct-1) > 1e-12 {
		t.Fatalf("%v %v %s", d, pct, kind)
	}
	_, _, kind = ComputeBasis(99, 100)
	if kind != BasisKindDiscount {
		t.Fatal(kind)
	}
	_, _, kind = ComputeBasis(100.005, 100)
	if kind != BasisKindFlat {
		t.Fatal(kind)
	}
}

func TestBasisTrendFromChange(t *testing.T) {
	if BasisTrendFromChange(0.08, 0.02) != BasisTrendExpanding {
		t.Fatal("expand")
	}
	if BasisTrendFromChange(0.02, 0.08) != BasisTrendShrinking {
		t.Fatal("shrink")
	}
	if BasisTrendFromChange(0.05, -0.05) != BasisTrendFlipped {
		t.Fatal("flip")
	}
}

func TestBuildBasisVenue_FiveMinuteKlineOpenTime(t *testing.T) {
	now := time.Date(2026, 8, 15, 7, 59, 0, 0, time.UTC)
	q := BasisQuote{
		Exchange: ExchangeBinance, FuturesLast: 100.1, FuturesMark: 100.1, SpotIndex: 100,
		Time: now,
		History: []BasisHistPoint{
			// 5m kline open at 07:50 — 4m before the 5m lookback target (07:54).
			{Time: now.Add(-9 * time.Minute), Mark: 100.08, Index: 100, Basis: 0.08, BasisPct: 0.08},
		},
	}
	got := BuildBasisVenue(q)
	var five *BasisWindowChange
	for i := range got.Windows {
		if got.Windows[i].Window == TakerWindow5m {
			five = &got.Windows[i]
		}
	}
	if five == nil || !five.Complete {
		t.Fatalf("5m window %+v", got.Windows)
	}
}

func TestBuildBasisVenue_PremiumExpanding(t *testing.T) {
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	q := BasisQuote{
		Exchange:    ExchangeBinance,
		Symbol:      "BTCUSDT",
		FuturesLast: 64100,
		FuturesMark: 64080,
		SpotIndex:   64000,
		Time:        now,
		FundingRate: 0.0001,
		OIChange1h:  1.2,
		History: []BasisHistPoint{
			{Time: now.Add(-time.Hour), Mark: 64020, Index: 64000, Basis: 20, BasisPct: 20.0 / 64000 * 100},
			{Time: now.Add(-5 * time.Minute), Mark: 64040, Index: 64000, Basis: 40, BasisPct: 40.0 / 64000 * 100},
		},
	}
	got := BuildBasisVenue(q)
	if got.Last.Kind != BasisKindPremium {
		t.Fatalf("kind %s", got.Last.Kind)
	}
	if got.Trend != BasisTrendExpanding {
		t.Fatalf("trend %s windows=%+v", got.Trend, got.Windows)
	}
	if !strings.Contains(strings.ToLower(got.Summary), "premium") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestCompareBasisVenues_Opposite(t *testing.T) {
	a := BuildBasisVenue(BasisQuote{Exchange: ExchangeBinance, FuturesLast: 101, SpotIndex: 100})
	b := BuildBasisVenue(BasisQuote{Exchange: ExchangeBybit, FuturesLast: 99, SpotIndex: 100})
	got := CompareBasisVenues([]BasisVenueReport{a, b})
	if got.Alignment != AlignOpposite {
		t.Fatalf("%+v", got)
	}
}

func TestCompareBasisVenues_Same(t *testing.T) {
	a := BuildBasisVenue(BasisQuote{Exchange: ExchangeBinance, FuturesLast: 100.2, SpotIndex: 100})
	b := BuildBasisVenue(BasisQuote{Exchange: ExchangeBybit, FuturesLast: 100.18, SpotIndex: 100})
	got := CompareBasisVenues([]BasisVenueReport{a, b})
	if got.Alignment != AlignSame {
		t.Fatalf("%+v", got)
	}
}

func TestCompareBasisVenues_VeryDifferent(t *testing.T) {
	a := BuildBasisVenue(BasisQuote{Exchange: ExchangeBinance, FuturesLast: 100.20, SpotIndex: 100})
	b := BuildBasisVenue(BasisQuote{Exchange: ExchangeBybit, FuturesLast: 100.02, SpotIndex: 100})
	got := CompareBasisVenues([]BasisVenueReport{a, b})
	if got.Alignment != AlignMixed {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(strings.ToLower(got.Title), "different") {
		t.Fatalf("title %s", got.Title)
	}
}

func TestBuildBasisCurve_ContangoFarRicher(t *testing.T) {
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	got := BuildBasisCurve(BasisQuote{
		Symbol: "BTCUSDT", FuturesLast: 64000, FuturesMark: 64000, SpotIndex: 64000, Time: now,
		Contracts: []BasisContractQuote{
			{Symbol: "BTCUSDT", Kind: BasisContractPerp, Last: 64000, Mark: 64000, Index: 64000},
			{Symbol: "BTCUSDT_260925", Kind: BasisContractCurrentQuarter, Expiry: now.Add(42 * 24 * time.Hour), Last: 64200, Mark: 64200, Index: 64000},
			{Symbol: "BTCUSDT_261225", Kind: BasisContractNextQuarter, Expiry: now.Add(133 * 24 * time.Hour), Last: 64600, Mark: 64600, Index: 64000},
		},
	})
	if !got.Complete || got.Shape != BasisCurveContango || got.Slope != BasisSlopeUp || got.Richer != BasisRicherFar {
		t.Fatalf("%+v", got)
	}
	if got.Near == nil || got.Far == nil || got.Near.Symbol != "BTCUSDT_260925" || got.Far.Symbol != "BTCUSDT_261225" {
		t.Fatalf("ends %+v %+v", got.Near, got.Far)
	}
	if got.SpreadPct <= 0 || !strings.Contains(got.Summary, "contango") {
		t.Fatalf("summary %s spread %v", got.Summary, got.SpreadPct)
	}
}

func TestBuildBasisCurve_BackwardationNearRicher(t *testing.T) {
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	got := BuildBasisCurve(BasisQuote{
		Symbol: "ETHUSDT", FuturesLast: 4000, FuturesMark: 4000, SpotIndex: 4000, Time: now,
		Contracts: []BasisContractQuote{
			{Symbol: "ETHUSDT", Kind: BasisContractPerp, Mark: 4000, Index: 4000},
			{Symbol: "ETHUSDT_260925", Kind: BasisContractCurrentQuarter, Expiry: now.Add(40 * 24 * time.Hour), Mark: 4050, Index: 4000},
			{Symbol: "ETHUSDT_261225", Kind: BasisContractNextQuarter, Expiry: now.Add(130 * 24 * time.Hour), Mark: 3980, Index: 4000},
		},
	})
	if !got.Complete || got.Slope != BasisSlopeDown || got.Richer != BasisRicherNear {
		t.Fatalf("%+v", got)
	}
	if got.Shape != BasisCurveBackwardation && got.Shape != BasisCurveHumped {
		t.Fatalf("shape %s", got.Shape)
	}
	if !strings.Contains(got.Summary, "backwardation") {
		t.Fatalf("summary %s", got.Summary)
	}
}

func TestBuildBasisCurve_PerpVsOneDated(t *testing.T) {
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	got := BuildBasisCurve(BasisQuote{
		Symbol: "SOLUSDT", FuturesLast: 140, FuturesMark: 140, SpotIndex: 140, Time: now,
		Contracts: []BasisContractQuote{
			{Symbol: "SOL-26DEC26", Kind: BasisContractDated, Expiry: now.Add(120 * 24 * time.Hour), Mark: 144, Index: 140},
		},
	})
	if !got.Complete || got.Richer != BasisRicherFar || got.Near == nil || got.Near.Kind != BasisContractPerp {
		t.Fatalf("%+v", got)
	}
}

func TestBuildBasisHistory(t *testing.T) {
	t0 := time.UnixMilli(1_700_000_000_000).UTC()
	marks := []PriceSample{{Time: t0, Price: 101}, {Time: t0.Add(time.Minute), Price: 102}}
	idx := []PriceSample{{Time: t0, Price: 100}, {Time: t0.Add(time.Minute), Price: 100}}
	got := BuildBasisHistory(marks, idx)
	if len(got) != 2 || got[0].BasisPct != 1 {
		t.Fatalf("%+v", got)
	}
}
