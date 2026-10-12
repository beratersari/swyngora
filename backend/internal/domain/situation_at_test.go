package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildSituationAt_ReadsTheStoredMinute(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	got := BuildSituationAt(SituationAtInput{
		At:           at,
		Symbol:       "BTCUSDT",
		CVD:          situationAtCVD(at.Add(-5*time.Minute), 200_000, -50),
		OpenInterest: situationAtOI(at, 100, 110),
		Funding:      situationAtFunding(at, 0.0001),
		FundingOI:    situationAtOIValue(at, 1_000_000),
		Crowd:        situationAtCrowd(at, 0.40, 0.60),
		Prints:       situationAtPrints(at.Add(-5*time.Minute), 1_000, 0),
		Coverage:     situationAtCoverage(at.Add(-48 * time.Hour)),
	})
	if got == nil {
		t.Fatal("nil situation")
	}
	if !got.AsOf.Equal(at) {
		t.Fatalf("asOf %s", got.AsOf)
	}
	if got.Note == situationNote || !strings.Contains(got.Note, "not part of this read") {
		t.Fatalf("note %q", got.Note)
	}
	if got.PriceCVD == nil || got.PriceCVD.Kind != SituationCVDAgrees {
		t.Fatalf("price %+v missing %v", got.PriceCVD, got.PriceCVDMissing)
	}
	if got.Size == nil || got.Size.Kind != SituationSizeWhaleBuyRetailSell {
		t.Fatalf("size %+v missing %v", got.Size, got.SizeMissing)
	}
	if got.OpenInterest == nil || got.OpenInterest.Flow != SituationOIOpened || got.OpenInterest.Window != OpenInterestWindow1h {
		t.Fatalf("oi %+v missing %v", got.OpenInterest, got.OpenInterestMissing)
	}
	if got.Funding == nil || got.Funding.Payer != "long" || got.Funding.RatePct != 0.01 {
		t.Fatalf("funding %+v missing %v", got.Funding, got.FundingMissing)
	}
	if got.Crowd == nil || got.Crowd.Lean != LongShortGapLeanTopLong || got.Crowd.GapPct != 20 {
		t.Fatalf("crowd %+v missing %v", got.Crowd, got.CrowdMissing)
	}
	if got.Prints == nil || got.Prints.Side != SituationPrintLong || got.Prints.LongNotional != 3000 || got.Prints.Window != "24h" {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
	if !got.Prints.Since.Equal(at) {
		t.Fatalf("prints since %s", got.Prints.Since)
	}
	// Two open-interest samples, and one minute of the other clauses, do not
	// reach a different kind. The clock stays off. Zero time still serializes.
	if !got.PriceCVD.Since.IsZero() || !got.OpenInterest.Since.IsZero() || !got.Funding.Since.IsZero() || !got.Crowd.Since.IsZero() || !got.Size.Since.IsZero() {
		t.Fatalf("sparse clocks price %s oi %s funding %s crowd %s size %s", got.PriceCVD.Since, got.OpenInterest.Since, got.Funding.Since, got.Crowd.Since, got.Size.Since)
	}
	if got.Liquidation != nil || len(got.LiquidationMissing) != 0 {
		t.Fatalf("pocket %+v missing %v", got.Liquidation, got.LiquidationMissing)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["liquidation"]; ok {
		t.Fatalf("pocket leaked into json %s", raw)
	}
	if _, ok := body["liquidationMissing"]; ok {
		t.Fatalf("pocket missing leaked into json %s", raw)
	}
}

func TestBuildSituationAt_StaleMinuteNamesTheVenue(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	stale := at.Add(-15 * time.Minute)
	got := BuildSituationAt(SituationAtInput{
		At:           at,
		Symbol:       "BTCUSDT",
		CVD:          situationAtCVD(stale, 200_000, -50),
		OpenInterest: situationAtOI(stale, 100, 110),
		Funding:      situationAtFunding(stale, 0.0001),
		FundingOI:    situationAtOIValue(stale, 1_000_000),
		Crowd:        situationAtCrowd(stale, 0.40, 0.60),
	})
	if got.PriceCVD != nil || len(got.PriceCVDMissing) != 3 {
		t.Fatalf("price %+v missing %v", got.PriceCVD, got.PriceCVDMissing)
	}
	if got.Size != nil || len(got.SizeMissing) != 3 {
		t.Fatalf("size %+v missing %v", got.Size, got.SizeMissing)
	}
	if got.OpenInterest != nil || len(got.OpenInterestMissing) != 3 {
		t.Fatalf("oi %+v missing %v", got.OpenInterest, got.OpenInterestMissing)
	}
	if got.Funding != nil || len(got.FundingMissing) != 3 {
		t.Fatalf("funding %+v missing %v", got.Funding, got.FundingMissing)
	}
	if got.Crowd != nil || len(got.CrowdMissing) != 2 {
		t.Fatalf("crowd %+v missing %v", got.Crowd, got.CrowdMissing)
	}
	if got.Liquidation != nil || len(got.LiquidationMissing) != 0 {
		t.Fatalf("pocket %+v missing %v", got.Liquidation, got.LiquidationMissing)
	}
}

func TestBuildSituationAt_NamesTheVenueTheArchiveDoesNotHave(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	cvd := situationAtCVD(at.Add(-5*time.Minute), 200_000, -50)
	cvd.Combined.Complete = false
	cvd.Venues[2].Complete = false
	cvd.Venues[2].Points = nil
	oi := situationAtOI(at, 100, 110)
	oi = oi[:4] // binance and bybit, both ends
	funding := situationAtFunding(at, 0.0001)
	funding = funding[:2]
	crowd := situationAtCrowd(at, 0.40, 0.60)
	crowd = crowd[:1]
	got := BuildSituationAt(SituationAtInput{
		At:           at,
		Symbol:       "btc-usd",
		CVD:          cvd,
		OpenInterest: oi,
		Funding:      funding,
		FundingOI:    situationAtOIValue(at, 1_000_000),
		Crowd:        crowd,
		Prints:       situationAtPrints(at.Add(-time.Minute), 100, 0),
		Coverage:     situationAtCoverage(at.Add(-48 * time.Hour))[:2],
	})
	if got.Symbol != "BTCUSDT" {
		t.Fatalf("symbol %s", got.Symbol)
	}
	if got.PriceCVD != nil || len(got.PriceCVDMissing) != 1 || got.PriceCVDMissing[0] != string(ExchangeOKX) {
		t.Fatalf("price %+v missing %v", got.PriceCVD, got.PriceCVDMissing)
	}
	if got.OpenInterest != nil || len(got.OpenInterestMissing) != 1 || got.OpenInterestMissing[0] != string(ExchangeOKX) {
		t.Fatalf("oi %+v missing %v", got.OpenInterest, got.OpenInterestMissing)
	}
	if got.Funding != nil || len(got.FundingMissing) != 1 || got.FundingMissing[0] != string(ExchangeOKX) {
		t.Fatalf("funding %+v missing %v", got.Funding, got.FundingMissing)
	}
	if got.Crowd != nil || len(got.CrowdMissing) != 1 || got.CrowdMissing[0] != string(ExchangeOKX) {
		t.Fatalf("crowd %+v missing %v", got.Crowd, got.CrowdMissing)
	}
	if got.Prints != nil || len(got.PrintsMissing) != 1 || got.PrintsMissing[0] != string(ExchangeOKX) {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
}

func TestBuildSituationAt_PrintBucketAtTheCloseStaysOut(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	minutes := situationAtPrints(at, 5_000, 0)
	got := BuildSituationAt(SituationAtInput{
		At:       at,
		Symbol:   "BTCUSDT",
		Prints:   minutes,
		Coverage: situationAtCoverage(at.Add(-48 * time.Hour)),
	})
	if got.Prints == nil || got.Prints.Side != SituationPrintQuiet {
		t.Fatalf("prints %+v missing %v", got.Prints, got.PrintsMissing)
	}
}

func TestBuildSituationAt_FundingOIMayUseThePreviousBucket(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	got := BuildSituationAt(SituationAtInput{
		At:        at,
		Symbol:    "BTCUSDT",
		Funding:   situationAtFunding(at, -0.0002),
		FundingOI: situationAtOIValue(at.Add(-5*time.Minute), 2_000_000),
	})
	if got.Funding == nil || got.Funding.Payer != "short" {
		t.Fatalf("funding %+v missing %v", got.Funding, got.FundingMissing)
	}
}

func TestBuildSituationAt_BybitIsNotACrowdVenue(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	crowd := situationAtCrowd(at, 0.50, 0.50)
	crowd = append(crowd, FuturesSnapshot{
		Exchange: ExchangeBybit, SampledAt: at, LongShare: 0.10, ShortShare: 0.90, SizeLongShare: 0.90,
	})
	got := BuildSituationAt(SituationAtInput{At: at, Symbol: "BTCUSDT", Crowd: crowd})
	if got.Crowd == nil || got.Crowd.Lean != LongShortGapLeanEven {
		t.Fatalf("crowd %+v missing %v", got.Crowd, got.CrowdMissing)
	}
}

func TestBuildSituationAt_SinceStopsAtTheCandle(t *testing.T) {
	at := time.Date(2026, 6, 1, 16, 0, 0, 0, time.UTC)
	prev := at.Add(-5 * time.Minute)
	crowd := append(situationAtCrowd(prev, 0.50, 0.50), situationAtCrowd(at, 0.40, 0.60)...)
	got := BuildSituationAt(SituationAtInput{At: at, Symbol: "BTCUSDT", Crowd: crowd})
	if got.Crowd == nil || got.Crowd.Lean != LongShortGapLeanTopLong {
		t.Fatalf("crowd %+v", got.Crowd)
	}
	if !got.Crowd.Since.Equal(at) {
		t.Fatalf("since %s", got.Crowd.Since)
	}
}

func TestBuildSituationAt_ZeroClockIsNotARead(t *testing.T) {
	if BuildSituationAt(SituationAtInput{Symbol: "BTCUSDT"}) != nil {
		t.Fatal("zero at built a read")
	}
}

func situationAtCVD(at time.Time, whale, retail float64) *CVDReport {
	point := CVDPoint{Time: at, Price: 100, CVD: 1, SizeKnown: true, WhaleDelta: whale, RetailDelta: retail}
	venue := func(ex Exchange) CVDVenueSeries {
		return CVDVenueSeries{
			Exchange: ex, Market: CVDMarketFutures, Complete: true, LastPrice: 100,
			Points: []CVDPoint{point},
		}
	}
	return &CVDReport{
		Combined: &CVDVenueSeries{Complete: true, LastPrice: 100, Points: []CVDPoint{point}},
		Venues: []CVDVenueSeries{
			venue(ExchangeBinance), venue(ExchangeBybit), venue(ExchangeOKX),
		},
	}
}

func situationAtOI(at time.Time, past, cur float64) []FuturesSnapshot {
	var out []FuturesSnapshot
	for _, ex := range FuturesVenues() {
		out = append(out,
			FuturesSnapshot{Exchange: ex, SampledAt: at.Add(-time.Hour), Contracts: past, Value: past},
			FuturesSnapshot{Exchange: ex, SampledAt: at, Contracts: cur, Value: cur},
		)
	}
	return out
}

func situationAtOIValue(at time.Time, value float64) []FuturesSnapshot {
	var out []FuturesSnapshot
	for _, ex := range FuturesVenues() {
		out = append(out, FuturesSnapshot{Exchange: ex, SampledAt: at, Value: value, Contracts: 1})
	}
	return out
}

func situationAtFunding(at time.Time, rate float64) []FuturesSnapshot {
	var out []FuturesSnapshot
	for _, ex := range FuturesVenues() {
		out = append(out, FuturesSnapshot{Exchange: ex, SampledAt: at, Predicted: true, FundingRate: rate})
	}
	return out
}

func situationAtCrowd(at time.Time, crowd, top float64) []FuturesSnapshot {
	return []FuturesSnapshot{
		{Exchange: ExchangeBinance, SampledAt: at, LongShare: crowd, ShortShare: 1 - crowd, SizeLongShare: top},
		{Exchange: ExchangeOKX, SampledAt: at, LongShare: crowd, ShortShare: 1 - crowd, SizeLongShare: top},
	}
}

func situationAtPrints(at time.Time, longN, shortN float64) []SituationPrintMinute {
	var out []SituationPrintMinute
	for _, ex := range FuturesVenues() {
		out = append(out, SituationPrintMinute{At: at, Exchange: ex, Long: longN, Short: shortN})
	}
	return out
}

func situationAtCoverage(first time.Time) []LiquidationCoverage {
	var out []LiquidationCoverage
	for _, ex := range FuturesVenues() {
		out = append(out, LiquidationCoverage{Exchange: ex, Symbol: "BTCUSDT", FirstWatch: first})
	}
	return out
}
