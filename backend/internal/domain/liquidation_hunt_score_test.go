package domain

import (
	"math"
	"testing"
)

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
	AttachHuntDirectionScores(&got, HuntSignals{
		HasPrice:      true,
		Price1hPct:    1.2,
		Price4hPct:    2.4,
		HasOI:         true,
		OI4hPct:       1.5,
		HasTaker:      true,
		TakerBuy1h:    80,
		TakerSell1h:   20,
		HasLiqWindows: true,
		ShortLiq1h:    2_000_000,
		LongLiq1h:     400_000,
	})
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
	AttachHuntDirectionScores(&got, HuntSignals{
		HasPrice:      true,
		Price1hPct:    0.8,
		Price4hPct:    2.1,
		HasOI:         true,
		OI1hPct:       0.6,
		OI4hPct:       1.8,
		HasTaker:      true,
		TakerBuy1h:    70,
		TakerSell1h:   30,
		HasLiqWindows: true,
		ShortLiq1h:    3_000_000,
		LongLiq1h:     500_000,
	})
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
	AttachHuntDirectionScores(&got, HuntSignals{
		HasPrice:      true,
		Price1hPct:    -1.1,
		Price4hPct:    -3.2,
		HasOI:         true,
		OI4hPct:       2.0,
		HasTaker:      true,
		TakerBuy1h:    25,
		TakerSell1h:   75,
		HasLiqWindows: true,
		LongLiq1h:     4_000_000,
		ShortLiq1h:    400_000,
	})
	if got.DownScore.Score <= got.UpScore.Score {
		t.Fatalf("want down > up: up=%v down=%v", got.UpScore, got.DownScore)
	}
	if got.Bias.Lean != HuntLeanDown {
		t.Fatalf("lean=%s up=%v down=%v", got.Bias.Lean, got.UpScore.Score, got.DownScore.Score)
	}
}

func TestAttachHuntDirectionScores_MissingTapeStillScoresBook(t *testing.T) {
	got := BuildHuntVenue(testHuntInputsCrowdedShorts())
	AttachHuntDirectionScores(&got, HuntSignals{})
	if got.UpScore.Score <= 0 || got.DownScore.Score <= 0 {
		t.Fatalf("book-only scores should still land: %+v %+v", got.UpScore, got.DownScore)
	}
	for _, f := range got.UpScore.Factors {
		if f.ID == "trend" || f.ID == "flow" {
			t.Fatalf("missing tape should drop trend/flow: %+v", got.UpScore.Factors)
		}
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
	if got != nil {
		t.Fatalf("no-price empty score should skip: %+v", got)
	}
}
