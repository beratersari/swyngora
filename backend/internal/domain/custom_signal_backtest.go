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

// CustomBacktestReplay is hits plus bars skipped because a selected factor had no data.
type CustomBacktestReplay struct {
	Hits           []CustomBacktestHit
	EvaluatedBars  int
	SkippedBars    int
	MissingFactors []string
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

// ReplayCustomSignalBacktest walks closed primary bars in [start, end].
// The mix score still uses whatever factors have data. A bar does not emit a
// signal when any selected (weighted) factor is missing historical data.
func ReplayCustomSignalBacktest(in CustomBacktestInput) (CustomBacktestReplay, error) {
	out := CustomBacktestReplay{}
	if in.Interval <= 0 {
		in.Interval = 4 * time.Hour
	}
	if in.Threshold <= 0 {
		in.Threshold = CustomBacktestThreshold
	}
	start := in.RangeStart.UTC()
	end := in.RangeEnd.UTC()
	prev := 0.0
	missingSet := map[string]struct{}{}
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
			return out, err
		}
		barsOK := len(ClosedBars(in.Primary[:i+1], asOf)) >= SwingMinClosedBars
		scored := ScoreCustomSignal(buildCustomSignalRawsForBacktest(dec.Patterns, barsOK), in.Weights)
		out.EvaluatedBars++
		if miss := selectedMissingFactors(scored, in.Weights); len(miss) > 0 {
			out.SkippedBars++
			for _, id := range miss {
				missingSet[id] = struct{}{}
			}
			continue
		}
		if CustomSignalCrossedTrigger(prev, scored.Score, in.Threshold) {
			out.Hits = append(out.Hits, CustomBacktestHit{
				Index: i, At: ot, Close: bar.Close, Score: scored.Score, Grade: scored.Grade,
				Summary: fmt.Sprintf("Custom mix %s (%.0f) crossed %.0f", scored.Grade, scored.Score, in.Threshold),
				Metrics: map[string]float64{"score": scored.Score, "threshold": in.Threshold},
			})
		}
		prev = scored.Score
	}
	out.MissingFactors = missingFactorIDs(missingSet)
	return out, nil
}

func buildCustomSignalRawsForBacktest(patterns []SwingPattern, barsOK bool) map[string]CustomSignalRaw {
	raws := BuildCustomSignalRaws(patterns, nil)
	if barsOK {
		return raws
	}
	for _, id := range CustomSignalSwingFactorIDs {
		raws[id] = CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "not enough historical bars"}
	}
	return raws
}

func selectedMissingFactors(scored CustomSignalScore, w CustomSignalWeights) []string {
	var out []string
	for _, f := range scored.Factors {
		if w.pctOf(f.ID) <= 0 || f.Status != CustomFactorMissing {
			continue
		}
		out = append(out, f.ID)
	}
	return out
}

func missingFactorIDs(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for _, id := range CustomSignalFactorIDs {
		if _, ok := set[id]; ok {
			out = append(out, id)
		}
	}
	return out
}
