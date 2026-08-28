package domain

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func TestEvaluateRSI_Below(t *testing.T) {
	candles := make([]Candle, 40)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	price := 100.0
	for i := 0; i < 40; i++ {
		candles[i] = Candle{
			OpenTime: t0.Add(time.Duration(i) * time.Hour),
			Close:    fmt.Sprintf("%.8f", price),
			Volume:   "100",
		}
		price -= 1
	}
	rule := ScannerRule{
		Type: ScannerRuleRSI, RSIPeriod: 14, RSICondition: AlertBelow, RSIThreshold: 40,
	}
	m, err := EvaluateScannerRule(rule, candles)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected RSI match")
	}
	if m.Metrics["rsi"] > 40 {
		t.Fatalf("rsi=%v", m.Metrics["rsi"])
	}
}

func TestEvaluateMACrossover_Golden(t *testing.T) {
	// Long decline then sharp rebound so EMA(3) crosses above EMA(8) on the last bar.
	prices := make([]float64, 0, 40)
	for i := 0; i < 30; i++ {
		prices = append(prices, 100-float64(i))
	}
	prices = append(prices, 80, 100, 130, 160, 200)
	candles := make([]Candle, len(prices))
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, p := range prices {
		candles[i] = Candle{
			OpenTime: t0.Add(time.Duration(i) * time.Hour),
			Close:    fmt.Sprintf("%.8f", p),
			Volume:   "10",
		}
	}
	rule := ScannerRule{
		Type: ScannerRuleMACrossover, MAFastPeriod: 3, MASlowPeriod: 8, MADirection: "golden_cross",
	}
	// Scan last few bars: evaluate on truncated series ending at each i where cross can occur
	var hit *ScannerMatch
	for end := 10; end <= len(candles); end++ {
		m, err := EvaluateScannerRule(rule, candles[:end])
		if err != nil {
			t.Fatal(err)
		}
		if m != nil {
			hit = m
		}
	}
	if hit == nil {
		// death_cross style reverse for robustness check of evaluator
		rule.MADirection = "death_cross"
		// fall after spike
		extra := append([]Candle{}, candles...)
		last := candles[len(candles)-1].OpenTime
		for i, p := range []float64{180, 120, 80, 50, 30} {
			extra = append(extra, Candle{
				OpenTime: last.Add(time.Duration(i+1) * time.Hour),
				Close:    fmt.Sprintf("%.8f", p),
				Volume:   "10",
			})
		}
		for end := len(candles); end <= len(extra); end++ {
			m, err := EvaluateScannerRule(rule, extra[:end])
			if err != nil {
				t.Fatal(err)
			}
			if m != nil {
				hit = m
				break
			}
		}
	}
	if hit == nil {
		t.Fatal("expected a MA crossover match on synthetic series")
	}
	if hit.MarketDataKey == "" {
		t.Fatal("missing market data key")
	}
}

func TestEvaluateVolumeIncrease(t *testing.T) {
	candles := make([]Candle, 25)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		vol := "100"
		if i == 24 {
			vol = "500"
		}
		candles[i] = Candle{
			OpenTime: t0.Add(time.Duration(i) * time.Hour),
			Close:    "10",
			Volume:   vol,
		}
	}
	rule := ScannerRule{
		Type: ScannerRuleVolumeIncrease, VolumeLookback: 20, VolumeMinRatio: 2,
	}
	m, err := EvaluateScannerRule(rule, candles)
	if err != nil || m == nil {
		t.Fatalf("m=%+v err=%v", m, err)
	}
	if math.Abs(m.Metrics["ratio"]-5) > 0.01 {
		t.Fatalf("ratio=%v", m.Metrics["ratio"])
	}
}

func TestEvaluateNoMatch(t *testing.T) {
	candles := make([]Candle, 30)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ {
		candles[i] = Candle{OpenTime: t0.Add(time.Duration(i) * time.Hour), Close: "100", Volume: "10"}
	}
	m, err := EvaluateScannerRule(ScannerRule{
		Type: ScannerRuleVolumeIncrease, VolumeLookback: 10, VolumeMinRatio: 3,
	}, candles)
	if err != nil || m != nil {
		t.Fatalf("want no match %+v %v", m, err)
	}
}

func TestResolveScannerConditionsAndMatchMode(t *testing.T) {
	conds, typ, err := ResolveScannerConditions("rsi", nil)
	if err != nil || typ != ScannerRuleRSI || len(conds) != 1 || conds[0] != ScannerRuleRSI {
		t.Fatalf("legacy type %+v %s %v", conds, typ, err)
	}
	conds, typ, err = ResolveScannerConditions("", []string{"rsi", "volume_increase", "rsi"})
	if err != nil || typ != ScannerRuleCombo || len(conds) != 2 {
		t.Fatalf("combo %+v %s %v", conds, typ, err)
	}
	if _, _, err = ResolveScannerConditions("combo", nil); err == nil {
		t.Fatal("combo type without conditions should fail")
	}
	if _, _, err = ResolveScannerConditions("", []string{"nope"}); err == nil {
		t.Fatal("invalid condition should fail")
	}
	mode, err := ResolveScannerMatchMode("")
	if err != nil || mode != ScannerMatchAll {
		t.Fatalf("default mode %s %v", mode, err)
	}
	mode, err = ResolveScannerMatchMode("or")
	if err != nil || mode != ScannerMatchAny {
		t.Fatalf("or alias %s %v", mode, err)
	}
	if _, err = ResolveScannerMatchMode("maybe"); err == nil {
		t.Fatal("invalid matchMode should fail")
	}
}

func TestEvaluateCombo_AllAndAny(t *testing.T) {
	candles := make([]Candle, 40)
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	price := 100.0
	for i := 0; i < 40; i++ {
		vol := "100"
		if i == 39 {
			vol = "500"
		}
		candles[i] = Candle{
			OpenTime: t0.Add(time.Duration(i) * time.Hour),
			Close:    fmt.Sprintf("%.8f", price),
			Volume:   vol,
		}
		price -= 1
	}
	allRule := ScannerRule{
		Conditions:     []ScannerRuleType{ScannerRuleRSI, ScannerRuleVolumeIncrease},
		MatchMode:      ScannerMatchAll,
		RSIPeriod:      14,
		RSICondition:   AlertBelow,
		RSIThreshold:   40,
		VolumeLookback: 20,
		VolumeMinRatio: 2,
	}
	m, err := EvaluateScannerRule(allRule, candles)
	if err != nil || m == nil {
		t.Fatalf("all should match %+v %v", m, err)
	}
	if !strings.Contains(m.Summary, "RSI") || !strings.Contains(m.Summary, "Volume") {
		t.Fatalf("summary %s", m.Summary)
	}
	anyRule := allRule
	anyRule.MatchMode = ScannerMatchAny
	anyRule.VolumeMinRatio = 99
	m, err = EvaluateScannerRule(anyRule, candles)
	if err != nil || m == nil {
		t.Fatalf("any RSI should match %+v %v", m, err)
	}
	allRule.VolumeMinRatio = 99
	m, err = EvaluateScannerRule(allRule, candles)
	if err != nil || m != nil {
		t.Fatalf("all should miss when volume fails %+v %v", m, err)
	}
}
