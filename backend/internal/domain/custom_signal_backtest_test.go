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
	bars := synthTrend(220, 50, 0.8)
	bars[len(bars)-1].Volume = 20000
	daily := synthTrend(220, 40, 1.5)
	start := bars[80].OpenTime
	end := bars[len(bars)-1].OpenTime
	hits, err := ReplayCustomSignalBacktest(CustomBacktestInput{
		Exchange: ExchangeBinance, Symbol: "ETHUSDT",
		Primary: bars, Higher: daily, BTCPrimary: daily, BTCHigher: daily,
		RangeStart: start, RangeEnd: end,
		Weights: DefaultCustomSignalWeights(), Threshold: CustomBacktestThreshold,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one custom-mix onset on a strong uptrend")
	}
	if hits[0].Score < CustomBacktestThreshold {
		t.Fatalf("hit score %.1f", hits[0].Score)
	}
	if hits[0].At.Before(start) || hits[0].At.After(end) {
		t.Fatalf("hit outside range %s", hits[0].At)
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
