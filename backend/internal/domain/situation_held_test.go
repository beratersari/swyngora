package domain

import (
	"testing"
	"time"
)

func TestHeldSince_NamesTheChangeAndStaysBlankAtTheEdge(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	at := func(m int) time.Time { return t0.Add(time.Duration(m) * time.Minute) }
	samples := []HeldSample{
		{At: at(0), Kind: "short"},
		{At: at(5), Kind: "long"},
		{At: at(10), Kind: "long"},
	}
	if got := HeldSince(samples, "long"); !got.Equal(at(5)) {
		t.Fatalf("since %s", got)
	}
	if got := HeldSince(samples, "short"); !got.IsZero() {
		t.Fatalf("newest is not short: %s", got)
	}
	edge := []HeldSample{{At: at(0), Kind: "long"}, {At: at(5), Kind: "long"}}
	if got := HeldSince(edge, "long"); !got.IsZero() {
		t.Fatalf("archive edge should stay blank: %s", got)
	}
	gap := []HeldSample{{At: at(0), Kind: "short"}, {At: at(20), Kind: "long"}}
	if got := HeldSince(gap, "long"); !got.IsZero() {
		t.Fatalf("hole should stay blank: %s", got)
	}
	unread := []HeldSample{{At: at(0), Kind: "short"}, {At: at(5), Kind: ""}, {At: at(10), Kind: "long"}}
	if got := HeldSince(unread, "long"); !got.IsZero() {
		t.Fatalf("unreadable minute should stay blank: %s", got)
	}
}

func TestStampSituationHeld_NamesEachClockTheArchiveReaches(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	bar := func(i int, price, delta, whale, retail float64, sized bool) CVDPoint {
		return CVDPoint{
			Time: t0.Add(time.Duration(i) * 5 * time.Minute), Price: price, Delta: delta,
			WhaleDelta: whale, RetailDelta: retail, SizeKnown: sized,
		}
	}
	points := []CVDPoint{
		bar(0, 100, 10, 0, 10, true),
		bar(1, 101, 10, 0, 10, true),
		bar(2, 102, -10, 20, -5, true),
		bar(3, 103, -8, 20, -5, true),
	}
	annotateCVDPoints(points)

	oiAt := t0.Add(-3 * time.Hour)
	var oi []FuturesSnapshot
	for step := 0; step <= 36; step++ {
		at := oiAt.Add(time.Duration(step) * 5 * time.Minute)
		contracts := 100.0
		if !at.Before(t0.Add(-30 * time.Minute)) {
			contracts = 120
		}
		for _, ex := range FuturesVenues() {
			oi = append(oi, FuturesSnapshot{
				Exchange: ex, SampledAt: at, Contracts: contracts, Value: contracts,
			})
		}
	}

	fundAt := t0.Add(-20 * time.Minute)
	funding := []FundingPressurePoint{
		{Time: fundAt, Payer: "short", VenueCount: 3},
		{Time: fundAt.Add(5 * time.Minute), Payer: "long", VenueCount: 3},
		{Time: fundAt.Add(10 * time.Minute), Payer: "long", VenueCount: 3},
	}
	crowdAt := t0.Add(-10 * time.Minute)
	crowd := []LongShortVenueSnap{
		{
			Exchange: string(ExchangeBinance),
			GapHistory: []LongShortGapPoint{
				{Time: crowdAt, GapPct: "+10"},
				{Time: crowdAt.Add(5 * time.Minute), GapPct: "0"},
			},
		},
		{
			Exchange: string(ExchangeOKX),
			GapHistory: []LongShortGapPoint{
				{Time: crowdAt, GapPct: "+10"},
				{Time: crowdAt.Add(5 * time.Minute), GapPct: "0"},
			},
		},
	}

	var coverage []LiquidationCoverage
	for _, ex := range FuturesVenues() {
		coverage = append(coverage, LiquidationCoverage{
			Exchange: ex, Symbol: "BTCUSDT", FirstWatch: t0.Add(-72 * time.Hour),
		})
	}
	shortAt := t0.Add(-30 * time.Hour)
	longAt := t0.Add(-time.Hour)
	prints := []SituationPrintMinute{
		{At: shortAt, Exchange: ExchangeBinance, Short: 100},
		{At: longAt, Exchange: ExchangeBybit, Long: 100},
	}

	sit := &CoinSituation{
		Symbol:       "BTCUSDT",
		PriceCVD:     &SituationPriceCVD{Kind: SituationCVDPriceUp},
		OpenInterest: &SituationOpenInterest{Flow: SituationOIOpened, Window: "1h"},
		Liquidation:  &SituationLiquidation{Side: SituationLiqLong},
		Prints:       &SituationPrints{Side: SituationPrintLong, Window: "24h"},
		Funding:      &SituationFunding{Payer: "long"},
		Crowd:        &SituationCrowd{Lean: LongShortGapLeanEven},
		Size:         &SituationSize{Kind: SituationSizeWhaleBuyRetailSell},
	}
	StampSituationHeld(sit, SituationHeldInput{
		Now: t0, Symbol: "BTCUSDT", CVD: points, OpenInterest: oi, Funding: funding,
		Crowd: crowd, Prints: prints, Coverage: coverage,
	})

	if !sit.PriceCVD.Since.Equal(points[2].Time) {
		t.Fatalf("price cvd since %s want %s", sit.PriceCVD.Since, points[2].Time)
	}
	if !sit.Size.Since.Equal(points[2].Time) {
		t.Fatalf("size since %s", sit.Size.Since)
	}
	if !sit.OpenInterest.Since.Equal(t0.Add(-30 * time.Minute)) {
		t.Fatalf("oi since %s", sit.OpenInterest.Since)
	}
	if !sit.Funding.Since.Equal(fundAt.Add(5 * time.Minute)) {
		t.Fatalf("funding since %s", sit.Funding.Since)
	}
	if !sit.Crowd.Since.Equal(crowdAt.Add(5 * time.Minute)) {
		t.Fatalf("crowd since %s", sit.Crowd.Since)
	}
	if sit.Prints.Since.IsZero() {
		t.Fatal("prints since blank")
	}
	if !sit.Liquidation.Since.IsZero() {
		t.Fatalf("liquidation model has no stored minute: %s", sit.Liquidation.Since)
	}

	same := CompareCoinSituation(sit, &CoinSituation{
		PriceCVD:     &SituationPriceCVD{Kind: SituationCVDPriceUp},
		OpenInterest: &SituationOpenInterest{Flow: SituationOIOpened},
		Liquidation:  &SituationLiquidation{Side: SituationLiqLong},
		Prints:       &SituationPrints{Side: SituationPrintLong},
		Funding:      &SituationFunding{Payer: "long"},
		Crowd:        &SituationCrowd{Lean: LongShortGapLeanEven},
		Size:         &SituationSize{Kind: SituationSizeWhaleBuyRetailSell},
	})
	if !same.Same {
		t.Fatalf("since must not count as leaving: %+v", same.Clauses)
	}

	ClearSituationHeld(sit)
	if !sit.PriceCVD.Since.IsZero() || !sit.Prints.Since.IsZero() || !sit.Funding.Since.IsZero() {
		t.Fatal("clear should drop the clocks")
	}
}

func TestWithoutSituationHeld_LeavesTheLiveReadAlone(t *testing.T) {
	since := time.Date(2026, 10, 12, 11, 0, 0, 0, time.UTC)
	live := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees, Since: since},
		Changes:  []SituationChange{{Clause: "funding", At: since, Kind: "short"}},
	}
	stored := WithoutSituationHeld(live)
	if stored == nil || stored.PriceCVD == nil || !stored.PriceCVD.Since.IsZero() || stored.PriceCVD.Kind != SituationCVDAgrees {
		t.Fatalf("stored %+v", stored)
	}
	if len(stored.Changes) != 0 {
		t.Fatalf("stored changes %+v", stored.Changes)
	}
	if !live.PriceCVD.Since.Equal(since) || len(live.Changes) != 1 {
		t.Fatal("live clock was cleared")
	}
}

func TestStampSituationHeld_BlankWhenTheArchiveDoesNotReach(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	points := []CVDPoint{
		{Time: t0, Price: 100, Delta: 5, SizeKnown: true, WhaleDelta: 5},
		{Time: t0.Add(5 * time.Minute), Price: 101, Delta: 5, SizeKnown: true, WhaleDelta: 5},
	}
	annotateCVDPoints(points)
	funding := []FundingPressurePoint{
		{Time: t0, Payer: "short", VenueCount: 3},
		{Time: t0.Add(5 * time.Minute), Payer: "long", VenueCount: 1},
		{Time: t0.Add(10 * time.Minute), Payer: "long", VenueCount: 3},
	}
	sit := &CoinSituation{
		PriceCVD: &SituationPriceCVD{Kind: SituationCVDAgrees},
		Funding:  &SituationFunding{Payer: "long"},
		Size:     &SituationSize{Kind: situationSizeKind(5, 0)},
	}
	StampSituationHeld(sit, SituationHeldInput{Now: t0, CVD: points, Funding: funding})
	if !sit.PriceCVD.Since.IsZero() {
		t.Fatalf("agrees across the whole tape should stay blank: %s", sit.PriceCVD.Since)
	}
	if !sit.Funding.Since.IsZero() {
		t.Fatalf("a missing venue inside the run should stay blank: %s", sit.Funding.Since)
	}
	if !sit.Size.Since.IsZero() {
		t.Fatalf("one size across the tape should stay blank: %s", sit.Size.Since)
	}
}
