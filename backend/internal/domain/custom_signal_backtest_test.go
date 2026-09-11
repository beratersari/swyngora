package domain

import (
	"strings"
	"testing"
)

func TestCustomSignalCrossedTrigger(t *testing.T) {
	t.Parallel()
	if CustomSignalCrossedTrigger(50, 61, 62) {
		t.Fatal("still below")
	}
	if !CustomSignalCrossedTrigger(50, 62, 62) {
		t.Fatal("must fire at the line")
	}
	if CustomSignalCrossedTrigger(70, 80, 62) {
		t.Fatal("already above is not a new onset")
	}
}

func TestEncodeDecodeCustomProfileSnapshot(t *testing.T) {
	t.Parallel()
	w := DefaultCustomSignalWeights()
	raw, err := EncodeCustomProfileSnapshot("Desk mix", w)
	if err != nil || !strings.Contains(raw, BacktestKindCustom) {
		t.Fatalf("%q %v", raw, err)
	}
	name, got, th, err := DecodeCustomProfileSnapshot(raw)
	if err != nil || name != "Desk mix" || th != CustomBacktestThreshold {
		t.Fatalf("%s %+v %.0f %v", name, got, th, err)
	}
	if got.Pct[CustomFactorEMACrossover] != w.Pct[CustomFactorEMACrossover] {
		t.Fatalf("%+v", got)
	}
	if _, _, _, err := DecodeCustomProfileSnapshot(`{"type":"rsi"}`); err == nil {
		t.Fatal("rule snapshot must not decode as custom")
	}
}

func TestReplayCustomSignalBacktest_OnsetOnUptrend(t *testing.T) {
	t.Parallel()
	bars := synthTrend(221, 50, 0.8)
	bars[219].Volume = 20000
	daily := synthTrend(221, 40, 1.5)
	start := bars[80].OpenTime
	end := bars[219].OpenTime
	rep, err := ReplayCustomSignalBacktest(CustomBacktestInput{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT",
		Primary: bars, Higher: daily, BTCPrimary: daily, BTCHigher: daily,
		RangeStart: start, RangeEnd: end,
		Weights: DefaultCustomSignalWeights(), Threshold: CustomBacktestThreshold,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Hits) == 0 {
		t.Fatal("expected at least one custom-mix onset on a strong uptrend")
	}
	if rep.Hits[0].Score < CustomBacktestThreshold {
		t.Fatalf("hit score %.1f", rep.Hits[0].Score)
	}
	if rep.Hits[0].At.Before(start) || rep.Hits[0].At.After(end) {
		t.Fatalf("hit outside range %s", rep.Hits[0].At)
	}
	if rep.SkippedBars != 0 || len(rep.MissingFactors) != 0 {
		t.Fatalf("built-in mix should not skip: %+v", rep)
	}
	if len(rep.Hits[0].Factors) == 0 {
		t.Fatal("hit must list selected factors")
	}
	var used int
	for _, f := range rep.Hits[0].Factors {
		if f.WeightPct <= 0 {
			t.Fatalf("disabled factor in hit: %+v", f)
		}
		if f.Status == CustomFactorUsed {
			used++
			if f.Score <= 0 {
				t.Fatalf("used factor needs a score: %+v", f)
			}
		}
	}
	if used == 0 {
		t.Fatal("expected at least one used factor")
	}
	if !strings.Contains(rep.Hits[0].Summary, "Used:") {
		t.Fatalf("summary %q", rep.Hits[0].Summary)
	}
	if rep.Hits[0].Trade == nil || rep.Hits[0].Trade.Entry <= 0 || rep.Hits[0].Trade.Leverage < 1 {
		t.Fatalf("hit must plan a trade: %+v", rep.Hits[0].Trade)
	}
	if rep.Hits[0].Side != BacktestTradeSideLong {
		t.Fatalf("side %s", rep.Hits[0].Side)
	}
	switch rep.Hits[0].Trade.Outcome {
	case BacktestTradeOutcomeTP, BacktestTradeOutcomeSL, BacktestTradeOutcomeOpen,
		BacktestTradeOutcomeTP1, BacktestTradeOutcomeTP1TP2, BacktestTradeOutcomeTP1SL, BacktestTradeOutcomeTP1Open:
	default:
		t.Fatalf("outcome %q", rep.Hits[0].Trade.Outcome)
	}
}

func TestReplayCustomSignalBacktest_SkipsWhenSelectedFactorMissing(t *testing.T) {
	t.Parallel()
	bars := synthTrend(220, 50, 0.8)
	daily := synthTrend(220, 40, 1.5)
	w, err := ParseCustomSignalWeightMap(map[string]float64{
		CustomFactorEMACrossover: 50,
		CustomFactorHuntUp:       50,
	})
	if err != nil {
		t.Fatal(err)
	}
	start := bars[80].OpenTime
	end := bars[len(bars)-1].OpenTime
	rep, err := ReplayCustomSignalBacktest(CustomBacktestInput{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT",
		Primary: bars, Higher: daily, BTCPrimary: daily, BTCHigher: daily,
		RangeStart: start, RangeEnd: end, Weights: w, Threshold: CustomBacktestThreshold,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Hits) != 0 {
		t.Fatalf("must not fire while hunt data is missing: %+v", rep.Hits)
	}
	if rep.SkippedBars == 0 || rep.SkippedBars != rep.EvaluatedBars {
		t.Fatalf("every bar should skip: %+v", rep)
	}
	if len(rep.MissingFactors) != 1 || rep.MissingFactors[0] != CustomFactorHuntUp {
		t.Fatalf("missing %+v", rep.MissingFactors)
	}
}

func TestIsCustomProfileBacktest(t *testing.T) {
	t.Parallel()
	if IsCustomProfileBacktest(ScannerBacktest{Kind: BacktestKindRule}) {
		t.Fatal("rule")
	}
	if !IsCustomProfileBacktest(ScannerBacktest{Kind: BacktestKindCustom}) {
		t.Fatal("kind")
	}
	raw, _ := EncodeCustomProfileSnapshot("x", DefaultCustomSignalWeights())
	if !IsCustomProfileBacktest(ScannerBacktest{RuleJSON: raw}) {
		t.Fatal("json")
	}
}
