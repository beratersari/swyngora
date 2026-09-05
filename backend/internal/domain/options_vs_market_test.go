package domain

import (
	"math"
	"testing"
	"time"
)

func sampleOptionsSnap(lean string, atmIV, dvol float64, pcr float64) *OptionsSnapshot {
	asOf := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	callOI, putOI := 100.0, 100.0
	switch lean {
	case OptionsLeanPuts:
		putOI = 200
	case OptionsLeanCalls:
		callOI = 250
		putOI = 100
	}
	if pcr > 0 && lean == "" {
		putOI = pcr * callOI
	}
	snap := &OptionsSnapshot{
		Asset: OptionsAssetBTC, IndexPrice: 100000, AsOf: asOf,
		CallOI: callOI, PutOI: putOI, Lean: lean,
		DVOL: dvol, HasDVOL: dvol > 0,
		ATM: &OptionsATM{IV: atmIV, Expiry: exp, DaysToExpiry: 26, Strike: 100000},
		Expiries: []OptionsExpiryBucket{{
			Expiry: exp, Label: "27 Mar 2026", DaysToExpiry: 26,
			AtmIV: atmIV, HasAtmIV: atmIV > 0,
			MaxPain: 100000, HasMaxPain: true,
		}},
	}
	if callOI > 0 {
		snap.PutCallOI = putOI / callOI
		snap.HasPutCallOI = true
		if snap.Lean == "" {
			snap.Lean = optionsLean(snap.PutCallOI)
		}
	}
	if atmIV > 0 {
		yf := OptionsYearFraction(26, exp, asOf)
		abs, pct, low, high, ok := OptionsExpectedMove(100000, atmIV, yf)
		if ok {
			snap.Expiries[0].ExpectedMoveAbs = abs
			snap.Expiries[0].ExpectedMovePct = pct
			snap.Expiries[0].ExpectedLow = low
			snap.Expiries[0].ExpectedHigh = high
			snap.Expiries[0].HasExpectedMove = true
		}
	}
	return snap
}

func TestCompareOptionsToMarket_NilWhenTapeEmpty(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanBalanced, 50, 49, 1)
	if got := CompareOptionsToMarket(snap, OptionsMarketTape{}); got != nil {
		t.Fatalf("empty tape: %+v", got)
	}
	if got := CompareOptionsToMarket(nil, OptionsMarketTape{HasRange: true, Range24hPct: 2}); got != nil {
		t.Fatalf("nil snap: %+v", got)
	}
}

func TestCompareOptionsToMarket_AlignedHasNoFindings(t *testing.T) {
	t.Parallel()
	// 2.6% 24h range annualizes to ~50%, matching 50% IV. PCR 1.0, LS 50, funding 0.
	snap := sampleOptionsSnap(OptionsLeanBalanced, 50, 50, 1)
	got := CompareOptionsToMarket(snap, OptionsMarketTape{
		SpotLast: 100000, HasSpot: true,
		Range24hPct: 2.62, HasRange: true,
		FundingRate: 0, HasFunding: true,
		LongPct: 50, HasLongPct: true,
	})
	if got == nil || !got.Compared {
		t.Fatal("expected compared tape")
	}
	if len(got.Findings) != 0 {
		t.Fatalf("aligned findings=%+v", got.Findings)
	}
	if got.Summary != "" {
		t.Fatalf("summary=%q", got.Summary)
	}
}

func TestCompareOptionsToMarket_IVRicher(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanBalanced, 70, 68, 1)
	got := CompareOptionsToMarket(snap, OptionsMarketTape{
		Range24hPct: 2.0, HasRange: true, SpotLast: 100000, HasSpot: true,
	})
	if got == nil || !hasFinding(got, OptionsFindingIVRicher) {
		t.Fatalf("want iv_richer got %+v", got)
	}
}

func TestCompareOptionsToMarket_IVCheaper(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanBalanced, 28, 27, 1)
	got := CompareOptionsToMarket(snap, OptionsMarketTape{
		Range24hPct: 4.0, HasRange: true, SpotLast: 100000, HasSpot: true,
	})
	if got == nil || !hasFinding(got, OptionsFindingIVCheaper) {
		t.Fatalf("want iv_cheaper got %+v", got)
	}
}

func TestCompareOptionsToMarket_DVOLRisingSpotCalm(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanBalanced, 50, 50, 1)
	snap.HasDVOLChange = true
	snap.DVOLChange24h = 2.4
	got := CompareOptionsToMarket(snap, OptionsMarketTape{
		Range24hPct: 1.5, HasRange: true,
		PrevRange24hPct: 2.2, HasPrevRange: true,
	})
	if got == nil || !hasFinding(got, OptionsFindingIVRisingCalm) {
		t.Fatalf("want iv_rising_spot_calm got %+v", got)
	}
}

func TestCompareOptionsToMarket_PutsVsFuturesLong(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanPuts, 50, 50, 2)
	got := CompareOptionsToMarket(snap, OptionsMarketTape{
		LongPct: 63, HasLongPct: true,
		FundingRate: 0.00012, HasFunding: true,
	})
	if got == nil || !hasFinding(got, OptionsFindingPutsVsLongs) {
		t.Fatalf("want puts_vs_futures_long got %+v", got)
	}
	if f := finding(got, OptionsFindingPutsVsLongs); f == nil || f.Severity != OptionsFindingNotable {
		t.Fatalf("severity %+v", f)
	}
}

func TestCompareOptionsToMarket_CallsVsFuturesShort(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanCalls, 50, 50, 0.4)
	got := CompareOptionsToMarket(snap, OptionsMarketTape{
		LongPct: 40, HasLongPct: true,
		FundingRate: -0.00015, HasFunding: true,
	})
	if got == nil || !hasFinding(got, OptionsFindingCallsVsShorts) {
		t.Fatalf("want calls_vs_futures_short got %+v", got)
	}
}

func TestCompareOptionsToMarket_DoesNotFlagMatchingLean(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanPuts, 50, 50, 2)
	got := CompareOptionsToMarket(snap, OptionsMarketTape{
		LongPct: 40, HasLongPct: true,
		FundingRate: -0.0002, HasFunding: true,
	})
	if got == nil {
		t.Fatal("expected compared")
	}
	if hasFinding(got, OptionsFindingPutsVsLongs) || hasFinding(got, OptionsFindingCallsVsShorts) {
		t.Fatalf("matching lean should stay quiet %+v", got.Findings)
	}
}

func TestCompareOptionsToMarket_MaxPainAway(t *testing.T) {
	t.Parallel()
	snap := sampleOptionsSnap(OptionsLeanBalanced, 50, 50, 1)
	snap.Expiries[0].MaxPain = 88000
	got := CompareOptionsToMarket(snap, OptionsMarketTape{Range24hPct: 2.6, HasRange: true})
	if got == nil || !hasFinding(got, OptionsFindingPainAway) {
		t.Fatalf("want max_pain_away got %+v", got)
	}
}

func TestCompareOptionsToMarket_SpotSpentNearExpiry(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 3, 27, 8, 0, 0, 0, time.UTC)
	snap := &OptionsSnapshot{
		Asset: OptionsAssetBTC, IndexPrice: 100000, AsOf: asOf,
		ATM: &OptionsATM{IV: 40, Expiry: exp, DaysToExpiry: 3},
		Expiries: []OptionsExpiryBucket{{
			Expiry: exp, Label: "27 Mar 2026", DaysToExpiry: 3,
			HasExpectedMove: true, ExpectedMovePct: 3.5,
		}},
	}
	got := CompareOptionsToMarket(snap, OptionsMarketTape{Range24hPct: 5.0, HasRange: true})
	if got == nil || !hasFinding(got, OptionsFindingMoveSpent) {
		t.Fatalf("want spot_spent_expected_move got %+v", got)
	}
}

func TestOptionsAnnualizeRange(t *testing.T) {
	t.Parallel()
	got := OptionsAnnualizeRange(2)
	want := 2 * math.Sqrt(365)
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
	if OptionsAnnualizeRange(0) != 0 {
		t.Fatal("zero")
	}
}

func TestOptionsSpotSymbol(t *testing.T) {
	t.Parallel()
	if OptionsSpotSymbol(OptionsAssetETH) != "ETHUSDT" {
		t.Fatal("eth")
	}
	if OptionsSpotSymbol(OptionsAssetBTC) != "BTCUSDT" {
		t.Fatal("btc")
	}
}

func hasFinding(got *OptionsVsMarket, kind string) bool {
	return finding(got, kind) != nil
}

func finding(got *OptionsVsMarket, kind string) *OptionsVsFinding {
	if got == nil {
		return nil
	}
	for i := range got.Findings {
		if got.Findings[i].Kind == kind {
			return &got.Findings[i]
		}
	}
	return nil
}
