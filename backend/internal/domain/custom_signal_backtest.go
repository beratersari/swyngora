package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// BacktestKindRule is a historical scanner-rule job.
	BacktestKindRule = "scanner_rule"
	// BacktestKindCustom is a historical custom-signal profile job.
	BacktestKindCustom = "custom_profile"
	// CustomBacktestBuiltInID is the Lab id for the built-in swing mix.
	CustomBacktestBuiltInID = "built_in"
	// CustomBacktestThreshold is the onset line (same as engine trigger).
	CustomBacktestThreshold = SwingTriggerMinScore
)

// customProfileSnapshot is the queued mix used by a custom backtest.
type customProfileSnapshot struct {
	Kind      string             `json:"kind"`
	Name      string             `json:"name"`
	Weights   map[string]float64 `json:"weights"`
	Threshold float64            `json:"threshold"`
}

// EncodeCustomProfileSnapshot freezes mix weights for a backtest job.
func EncodeCustomProfileSnapshot(name string, w CustomSignalWeights) (string, error) {
	pct := w.Pct
	if pct == nil {
		pct = map[string]float64{}
	}
	th := CustomBacktestThreshold
	b, err := json.Marshal(customProfileSnapshot{
		Kind: BacktestKindCustom, Name: strings.TrimSpace(name),
		Weights: pct, Threshold: th,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DecodeCustomProfileSnapshot restores a queued custom mix.
func DecodeCustomProfileSnapshot(raw string) (name string, w CustomSignalWeights, threshold float64, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", CustomSignalWeights{}, 0, fmt.Errorf("%w: missing custom profile snapshot", ErrInvalidArgument)
	}
	var snap customProfileSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return "", CustomSignalWeights{}, 0, fmt.Errorf("%w: invalid custom profile snapshot", ErrInvalidArgument)
	}
	if snap.Kind != BacktestKindCustom {
		return "", CustomSignalWeights{}, 0, fmt.Errorf("%w: not a custom profile snapshot", ErrInvalidArgument)
	}
	parsed, err := ParseCustomSignalWeightMap(snap.Weights)
	if err != nil {
		return "", CustomSignalWeights{}, 0, err
	}
	th := snap.Threshold
	if th <= 0 {
		th = CustomBacktestThreshold
	}
	return snap.Name, parsed, th, nil
}

// IsCustomProfileBacktest reports whether the job is a custom-mix replay.
func IsCustomProfileBacktest(job ScannerBacktest) bool {
	if job.Kind == BacktestKindCustom {
		return true
	}
	raw := strings.TrimSpace(job.RuleJSON)
	return strings.Contains(raw, `"kind":"custom_profile"`)
}

// CustomSignalCrossedTrigger is true when score first reaches the line.
func CustomSignalCrossedTrigger(prev, curr, threshold float64) bool {
	if threshold <= 0 {
		threshold = CustomBacktestThreshold
	}
	return prev < threshold && curr >= threshold
}

// CustomBacktestHit is one onset during a custom-mix replay.
type CustomBacktestHit struct {
	Index   int
	At      time.Time
	Close   float64
	Score   float64
	Grade   string
	Summary string
	Metrics map[string]float64
}

// CustomBacktestInput is one symbol's candles plus the queued mix.
type CustomBacktestInput struct {
	Exchange   Exchange
	Symbol     string
	Primary    []OHLC
	Higher     []OHLC
	BTCPrimary []OHLC
	BTCHigher  []OHLC
	RangeStart time.Time
	RangeEnd   time.Time
	Weights    CustomSignalWeights
	Threshold  float64
	Interval   time.Duration
}

// ReplayCustomSignalBacktest walks closed primary bars in [start, end] and
// emits a hit when the custom mix first crosses the threshold (onset).
// Hunt and tape factors are omitted — they cannot be rebuilt from candles.
func ReplayCustomSignalBacktest(in CustomBacktestInput) ([]CustomBacktestHit, error) {
	if in.Interval <= 0 {
		in.Interval = 4 * time.Hour
	}
	if in.Threshold <= 0 {
		in.Threshold = CustomBacktestThreshold
	}
	start := in.RangeStart.UTC()
	end := in.RangeEnd.UTC()
	var hits []CustomBacktestHit
	prev := 0.0
	for i := range in.Primary {
		bar := in.Primary[i]
		ot := bar.OpenTime.UTC()
		if ot.Before(start) {
			continue
		}
		if ot.After(end) {
			break
		}
		asOf := bar.CloseTime
		if asOf.IsZero() {
			asOf = ot.Add(in.Interval)
		}
		dec, err := EvaluateSwing(SwingScanInput{
			Exchange:   in.Exchange,
			Symbol:     in.Symbol,
			Primary:    in.Primary[:i+1],
			Higher:     in.Higher,
			BTCPrimary: in.BTCPrimary,
			BTCHigher:  in.BTCHigher,
			PrimaryTF:  string(Interval4h),
			HigherTF:   string(Interval1d),
			Now:        asOf,
		})
		if err != nil {
			return hits, err
		}
		scored := ScoreCustomSignal(BuildCustomSignalRaws(dec.Patterns, nil), in.Weights)
		if CustomSignalCrossedTrigger(prev, scored.Score, in.Threshold) {
			hits = append(hits, CustomBacktestHit{
				Index: i, At: ot, Close: bar.Close, Score: scored.Score, Grade: scored.Grade,
				Summary: fmt.Sprintf("Custom mix %s (%.0f) crossed %.0f", scored.Grade, scored.Score, in.Threshold),
				Metrics: map[string]float64{"score": scored.Score, "threshold": in.Threshold},
			})
		}
		prev = scored.Score
	}
	return hits, nil
}
