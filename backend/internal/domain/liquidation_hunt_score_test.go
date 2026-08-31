package domain

import (
	"math"
	"testing"
)

func fullHuntSignals() HuntSignals {
	return HuntSignals{
		HasPrice:       true,
		Has1hPrice:     true,
		Has4hPrice:     true,
		Price1hPct:     0.8,
		Price4hPct:     2.1,
		HasOI:          true,
		OI1hPct:        0.6,
		OI4hPct:        1.8,
		HasTaker:       true,
		TakerBuy1h:     70,
		TakerSell1h:    30,
		HasLiqWindows:  true,
		LiqFeedPresent: true,
		ShortLiq1h:     3_000_000,
		LongLiq1h:      500_000,
		HasBook:        true,
		HasLongShort:   true,
		HasFunding:     true,
	}
}

func testHuntInputsCrowdedShorts() HuntInputs {
	return HuntInputs{
		Exchange:    ExchangeBinance,
		Symbol:      "BTCUSDT",
		Price:       100,
		OIValue:     10_000_000,
		LongShare:   0.35,
		ShortShare:  0.65,
		FundingRate: -0.0002,
		Asks: []ImpactSourceLevel{
			{Price: 100.10, Quantity: 1},
			{Price: 101.00, Quantity: 1},
			{Price: 102.00, Quantity: 2},
			{Price: 105.00, Quantity: 2},
			{Price: 110.00, Quantity: 2},
		},
		Bids: []ImpactSourceLevel{
			{Price: 99.90, Quantity: 1},
			{Price: 99.00, Quantity: 1},
			{Price: 90.00, Quantity: 5},
		},
	}
}

func TestAttachHuntDirectionScores_DoesNotChangeZones(t *testing.T) {
	got := BuildHuntVenue(testHuntInputsCrowdedShorts())
	beforeUp := got.UpHunt
	beforeDown := got.DownHunt
	beforeBands := len(got.UpPressure) + len(got.DownPressure)
	AttachHuntDirectionScores(&got, fullHuntSignals())
	if got.UpHunt.EstLiquidated != beforeUp.EstLiquidated || got.UpHunt.NetWithCascade != beforeUp.NetWithCascade {
		t.Fatalf("up hunt mutated: before %+v after %+v", beforeUp, got.UpHunt)
	}
	if got.DownHunt.Spot.Notional != beforeDown.Spot.Notional || got.DownHunt.HouseEdge != beforeDown.HouseEdge {
		t.Fatalf("down hunt mutated: before %+v after %+v", beforeDown, got.DownHunt)
	}
	if len(got.UpPressure)+len(got.DownPressure) != beforeBands {
		t.Fatal("pressure bands changed")
	}
}

func TestAttachHuntDirectionScores_UpEasierWhenShortsCrowdedAndTrendUp(t *testing.T) {
	got := BuildHuntVenue(testHuntInputsCrowdedShorts())
	AttachHuntDirectionScores(&got, fullHuntSignals())
	if got.UpScore.Score <= got.DownScore.Score {
		t.Fatalf("want up > down: up=%v down=%v reasons=%v / %v", got.UpScore, got.DownScore, got.UpScore.Reasons, got.DownScore.Reasons)
	}
	if got.Bias.Lean != HuntLeanUp {
		t.Fatalf("lean=%s margin=%v up=%v down=%v", got.Bias.Lean, got.Bias.Margin, got.UpScore.Score, got.DownScore.Score)
	}
	if len(got.UpScore.Reasons) == 0 || got.UpScore.Level == "" {
		t.Fatalf("missing up reasons/level: %+v", got.UpScore)
	}
}

func TestAttachHuntDirectionScores_DownEasierWhenLongsCrowdedAndTrendDown(t *testing.T) {
	in := testHuntInputsCrowdedShorts()
	in.LongShare, in.ShortShare = 0.72, 0.28
	in.FundingRate = 0.0003
	// Thicker asks, thinner bids so walking down is cheaper.
	in.Asks = []ImpactSourceLevel{
		{Price: 100.10, Quantity: 40},
		{Price: 101.00, Quantity: 40},
		{Price: 105.00, Quantity: 80},
	}
	in.Bids = []ImpactSourceLevel{
		{Price: 99.90, Quantity: 1},
		{Price: 99.00, Quantity: 2},
		{Price: 97.00, Quantity: 3},
		{Price: 90.00, Quantity: 4},
	}
	got := BuildHuntVenue(in)
	sig := fullHuntSignals()
	sig.Price1hPct = -1.1
	sig.Price4hPct = -3.2
	sig.OI4hPct = 2.0
	sig.TakerBuy1h, sig.TakerSell1h = 25, 75
	sig.LongLiq1h, sig.ShortLiq1h = 4_000_000, 400_000
	AttachHuntDirectionScores(&got, sig)
	if got.DownScore.Score <= got.UpScore.Score {
		t.Fatalf("want down > up: up=%v down=%v", got.UpScore, got.DownScore)
	}
	if got.Bias.Lean != HuntLeanDown {
		t.Fatalf("lean=%s up=%v down=%v", got.Bias.Lean, got.UpScore.Score, got.DownScore.Score)
	}
}

func TestAttachHuntDirectionScores_MissingTapeStillScoresBook(t *testing.T) {
	got := BuildHuntVenue(testHuntInputsCrowdedShorts())
	AttachHuntDirectionScores(&got, HuntSignals{HasBook: true, HasLongShort: true, HasFunding: true})
	if got.UpScore.Score <= 0 || got.DownScore.Score <= 0 {
		t.Fatalf("book-only scores should still land: %+v %+v", got.UpScore, got.DownScore)
	}
	for _, f := range got.UpScore.Factors {
		if f.ID == "trend" || f.ID == "flow" {
			t.Fatalf("missing tape should drop trend/flow: %+v", got.UpScore.Factors)
		}
	}
	if got.Coverage.Score >= 85 || got.Coverage.Level == HuntCoverageComplete {
		t.Fatalf("missing tape should not look complete: %+v", got.Coverage)
	}
}

func TestAttachHuntDirectionScores_ThinCoverageShrinksVsFullTape(t *testing.T) {
	full := BuildHuntVenue(testHuntInputsCrowdedShorts())
	thin := BuildHuntVenue(testHuntInputsCrowdedShorts())
	AttachHuntDirectionScores(&full, fullHuntSignals())
	AttachHuntDirectionScores(&thin, HuntSignals{HasBook: true})
	spreadFull := math.Abs(full.UpScore.Score - full.DownScore.Score)
	spreadThin := math.Abs(thin.UpScore.Score - thin.DownScore.Score)
	if spreadThin >= spreadFull {
		t.Fatalf("thin coverage should shrink the lean: full=%v thin=%v", spreadFull, spreadThin)
	}
	if thin.Coverage.Usable && thin.Coverage.Score >= full.Coverage.Score {
		t.Fatalf("thin coverage score should be lower: full=%v thin=%v", full.Coverage, thin.Coverage)
	}
}

func TestCombineHuntBias_SkipsErroredVenue(t *testing.T) {
	good := HuntVenueReport{
		Exchange:          ExchangeBinance,
		Price:             100,
		OpenInterestValue: 5,
		UpScore:           HuntDirectionScore{Score: 80},
		DownScore:         HuntDirectionScore{Score: 30},
		Coverage:          HuntCoverage{Score: 90, Level: HuntCoverageComplete, Usable: true},
	}
	bad := HuntVenueReport{
		Exchange:          ExchangeBybit,
		Price:             100,
		OpenInterestValue: 50,
		Error:             "book: timeout",
		UpScore:           HuntDirectionScore{Score: 10},
		DownScore:         HuntDirectionScore{Score: 95},
		Coverage:          HuntCoverage{Score: 20, Level: HuntCoverageInsufficient, Usable: false},
	}
	got := CombineHuntBias([]HuntVenueReport{good, bad})
	if got == nil || got.Lean != HuntLeanUp {
		t.Fatalf("errored bybit must not flip combined: %+v", got)
	}
	if len(got.Excluded) != 1 || got.Excluded[0] != "bybit" {
		t.Fatalf("excluded=%v", got.Excluded)
	}
	if got.UpScore < 70 {
		t.Fatalf("combined should follow binance: %+v", got)
	}
}

func TestCombineHuntBias_OIWeighted(t *testing.T) {
	a := HuntVenueReport{Exchange: ExchangeBinance, Price: 100, OpenInterestValue: 9, UpScore: HuntDirectionScore{Score: 80}, DownScore: HuntDirectionScore{Score: 30}}
	b := HuntVenueReport{Exchange: ExchangeBybit, Price: 100, OpenInterestValue: 1, UpScore: HuntDirectionScore{Score: 20}, DownScore: HuntDirectionScore{Score: 80}}
	got := CombineHuntBias([]HuntVenueReport{a, b})
	if got == nil || got.Lean != HuntLeanUp {
		t.Fatalf("%+v", got)
	}
	if got.UpScore <= 70 {
		t.Fatalf("OI-weighted up should stay high: %+v", got)
	}
}

func TestHuntLeanFromScores_EvenInsideMargin(t *testing.T) {
	lean, margin := HuntLeanFromScores(52, 50)
	if lean != HuntLeanEven || margin != 2 {
		t.Fatalf("%s %v", lean, margin)
	}
	if HuntEaseFromScore(72) != HuntEaseEasier || HuntEaseFromScore(20) != HuntEaseHard {
		t.Fatal("levels")
	}
	if _, m := HuntLeanFromScores(70, 50); math.Abs(m-20) > 0.01 {
		t.Fatalf("margin %v", m)
	}
}

func TestCombineHuntBias_SkipsEmptyVenues(t *testing.T) {
	if CombineHuntBias(nil) != nil {
		t.Fatal("empty")
	}
	got := CombineHuntBias([]HuntVenueReport{{Exchange: ExchangeBinance, Error: "down"}})
	if got == nil || got.Coverage.Usable || len(got.Excluded) != 1 {
		t.Fatalf("errored-only set should not lean: %+v", got)
	}
}
