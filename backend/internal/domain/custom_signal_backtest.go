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
	Kind            string             `json:"kind"`
	Name            string             `json:"name"`
	Weights         map[string]float64 `json:"weights"`
	Threshold       float64            `json:"threshold"`
	Scope           string             `json:"scope,omitempty"`
	Markets         []BacktestMarket   `json:"markets,omitempty"`
	StartingBalance float64            `json:"startingBalance,omitempty"`
	RiskPct         float64            `json:"riskPct,omitempty"`
	MaxOpenRiskPct  float64            `json:"maxOpenRiskPct,omitempty"`
	UseTP2          bool               `json:"useTp2,omitempty"`
	TP1ClosePct     float64            `json:"tp1ClosePct,omitempty"`
	StopAfterTP1    string             `json:"stopAfterTp1,omitempty"`
	FeePct          float64            `json:"feePct,omitempty"`
	SlipPct         float64            `json:"slipPct,omitempty"`
}

// EncodeCustomProfileSnapshot freezes mix weights for a backtest job.
func EncodeCustomProfileSnapshot(name string, w CustomSignalWeights) (string, error) {
	return encodeCustomProfileSnapshot(name, w, BacktestScopeSymbol, nil, 0, 0, 0, BacktestScale{})
}

// EncodeWatchlistCustomProfileSnapshot freezes mix weights plus the watchlist coins.
func EncodeWatchlistCustomProfileSnapshot(name string, w CustomSignalWeights, markets []BacktestMarket, startingBalance, riskPct, maxOpenRiskPct float64) (string, error) {
	return EncodeWatchlistCustomProfileSnapshotScale(name, w, markets, startingBalance, riskPct, maxOpenRiskPct, BacktestScale{})
}

// EncodeWatchlistCustomProfileSnapshotScale is EncodeWatchlistCustomProfileSnapshot plus TP scale.
func EncodeWatchlistCustomProfileSnapshotScale(name string, w CustomSignalWeights, markets []BacktestMarket, startingBalance, riskPct, maxOpenRiskPct float64, scale BacktestScale) (string, error) {
	return encodeCustomProfileSnapshot(name, w, BacktestScopeWatchlist, markets, startingBalance, riskPct, maxOpenRiskPct, scale)
}

// EncodeCustomProfileSnapshotScale freezes mix weights and Lab TP scale for a one-coin job.
func EncodeCustomProfileSnapshotScale(name string, w CustomSignalWeights, scale BacktestScale) (string, error) {
	return encodeCustomProfileSnapshot(name, w, BacktestScopeSymbol, nil, 0, 0, 0, scale)
}

func encodeCustomProfileSnapshot(name string, w CustomSignalWeights, scope string, markets []BacktestMarket, startingBalance, riskPct, maxOpenRiskPct float64, scale BacktestScale) (string, error) {
	pct := w.Pct
	if pct == nil {
		pct = map[string]float64{}
	}
	th := CustomBacktestThreshold
	scale = NormalizeBacktestScale(scale)
	if scope == BacktestScopeWatchlist {
		startingBalance = NormalizeBacktestStartBalance(startingBalance)
		riskPct = NormalizeBacktestRiskPct(riskPct)
		maxOpenRiskPct = NormalizeBacktestMaxOpenRiskPct(maxOpenRiskPct)
	}
	b, err := json.Marshal(customProfileSnapshot{
		Kind: BacktestKindCustom, Name: strings.TrimSpace(name),
		Weights: pct, Threshold: th, Scope: scope, Markets: markets,
		StartingBalance: startingBalance, RiskPct: riskPct, MaxOpenRiskPct: maxOpenRiskPct,
		UseTP2: scale.UseTP2, TP1ClosePct: scale.TP1ClosePct, StopAfterTP1: scale.StopAfterTP1,
		FeePct: scale.FeePct, SlipPct: scale.SlipPct,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseCustomProfileSnapshot(raw string) (customProfileSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return customProfileSnapshot{}, fmt.Errorf("%w: missing custom profile snapshot", ErrInvalidArgument)
	}
	var snap customProfileSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return customProfileSnapshot{}, fmt.Errorf("%w: invalid custom profile snapshot", ErrInvalidArgument)
	}
	if snap.Kind != BacktestKindCustom {
		return customProfileSnapshot{}, fmt.Errorf("%w: not a custom profile snapshot", ErrInvalidArgument)
	}
	return snap, nil
}

// DecodeCustomProfileSnapshot restores a queued custom mix.
func DecodeCustomProfileSnapshot(raw string) (name string, w CustomSignalWeights, threshold float64, err error) {
	snap, err := parseCustomProfileSnapshot(raw)
	if err != nil {
		return "", CustomSignalWeights{}, 0, err
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

// CustomBacktestMarkets is the coin list for a job (watchlist snapshot or the single pair).
func CustomBacktestMarkets(job ScannerBacktest) []BacktestMarket {
	if snap, err := parseCustomProfileSnapshot(job.RuleJSON); err == nil && len(snap.Markets) > 0 {
		return append([]BacktestMarket(nil), snap.Markets...)
	}
	if job.Symbol != "" && !strings.EqualFold(job.Symbol, BacktestWatchlistSymbol) {
		return []BacktestMarket{{Exchange: job.Exchange, Symbol: job.Symbol}}
	}
	return nil
}

// CustomBacktestStartBalance is the shared pot snapshotted on a watchlist job.
func CustomBacktestStartBalance(job ScannerBacktest) float64 {
	if snap, err := parseCustomProfileSnapshot(job.RuleJSON); err == nil && snap.StartingBalance > 0 {
		return NormalizeBacktestStartBalance(snap.StartingBalance)
	}
	if IsWatchlistBacktest(job) {
		return BacktestDefaultStartBalance
	}
	return 0
}

// CustomBacktestRiskPct is the stop-loss share of equity on a watchlist job.
func CustomBacktestRiskPct(job ScannerBacktest) float64 {
	if snap, err := parseCustomProfileSnapshot(job.RuleJSON); err == nil && snap.RiskPct > 0 {
		return NormalizeBacktestRiskPct(snap.RiskPct)
	}
	if IsWatchlistBacktest(job) {
		return BacktestDefaultRiskPct
	}
	return 0
}

// CustomBacktestScale is the snapshotted TP1/TP2 Lab config.
func CustomBacktestScale(job ScannerBacktest) BacktestScale {
	return BacktestScaleFromJob(job)
}

// BacktestScaleFromJob reads scale config from a custom or rule snapshot.
func BacktestScaleFromJob(job ScannerBacktest) BacktestScale {
	if snap, err := parseCustomProfileSnapshot(job.RuleJSON); err == nil {
		return NormalizeBacktestScale(BacktestScale{UseTP2: snap.UseTP2, TP1ClosePct: snap.TP1ClosePct, StopAfterTP1: snap.StopAfterTP1, FeePct: snap.FeePct, SlipPct: snap.SlipPct})
	}
	if snap, err := parseScannerRuleSnapshot(job.RuleJSON); err == nil {
		return NormalizeBacktestScale(BacktestScale{UseTP2: snap.UseTP2, TP1ClosePct: snap.TP1ClosePct, StopAfterTP1: snap.StopAfterTP1, FeePct: snap.FeePct, SlipPct: snap.SlipPct})
	}
	return BacktestScale{}
}

// CustomBacktestMaxOpenRiskPct is the combined open-stop cap on a watchlist job.
func CustomBacktestMaxOpenRiskPct(job ScannerBacktest) float64 {
	if snap, err := parseCustomProfileSnapshot(job.RuleJSON); err == nil && snap.MaxOpenRiskPct > 0 {
		return NormalizeBacktestMaxOpenRiskPct(snap.MaxOpenRiskPct)
	}
	if IsWatchlistBacktest(job) {
		return BacktestDefaultMaxOpenRiskPct
	}
	return 0
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

// CustomBacktestFactor is one selected mix factor at the signal bar.
type CustomBacktestFactor struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Score     float64 `json:"score"`
	WeightPct float64 `json:"weightPct"`
	SharePct  float64 `json:"sharePct,omitempty"`
	Effect    float64 `json:"effect,omitempty"`
	Status    string  `json:"status"`
	Detail    string  `json:"detail,omitempty"`
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
	Factors []CustomBacktestFactor
	Side    string
	Trade   *BacktestTrade
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
	Touch      BacktestTouchOpts
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
	var openTrade *BacktestTrade
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
			factors := customBacktestFactors(scored)
			side := NormalizeTradeSide(dec.Side)
			var trade *BacktestTrade
			entryAt := ot
			if i+1 < len(in.Primary) && !in.Primary[i+1].OpenTime.IsZero() {
				entryAt = in.Primary[i+1].OpenTime.UTC()
			} else if !asOf.IsZero() {
				entryAt = asOf.UTC()
			}
			if !TradeOpenAt(openTrade, entryAt) {
				trade = SimulateBacktestTradeOpts(in.Primary, i, side, swingLevelsOrEmpty(dec.Levels), in.Touch)
				if trade == nil && dec.Levels == nil {
					trade = PlanBacktestTradeOpts(in.Primary, i, side, in.Touch)
				}
				if trade != nil {
					openTrade = trade
				}
			}
			out.Hits = append(out.Hits, CustomBacktestHit{
				Index: i, At: ot, Close: bar.Close, Score: scored.Score, Grade: scored.Grade,
				Summary: customBacktestWhy(scored, in.Threshold, factors),
				Metrics: map[string]float64{"score": scored.Score, "threshold": in.Threshold},
				Factors: factors, Side: side, Trade: trade,
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

func customBacktestFactors(scored CustomSignalScore) []CustomBacktestFactor {
	out := make([]CustomBacktestFactor, 0, len(scored.Factors))
	for _, f := range scored.Factors {
		if f.RequestedPct <= 0 {
			continue
		}
		out = append(out, CustomBacktestFactor{
			ID: f.ID, Label: f.Label, Score: f.Score,
			WeightPct: f.RequestedPct, SharePct: f.SharePct,
			Effect: f.Effect, Status: f.Status, Detail: f.Detail,
		})
	}
	return out
}

func customBacktestWhy(scored CustomSignalScore, threshold float64, factors []CustomBacktestFactor) string {
	head := fmt.Sprintf("Custom mix %s (%.0f) crossed %.0f", scored.Grade, scored.Score, threshold)
	var used []string
	for _, f := range factors {
		if f.Status != CustomFactorUsed {
			continue
		}
		used = append(used, fmt.Sprintf("%s %.0f (%.0f%%)", f.Label, f.Score, f.WeightPct))
	}
	if len(used) == 0 {
		return head
	}
	return head + ". Used: " + strings.Join(used, ", ")
}

func swingLevelsOrEmpty(lv *SwingLevels) SwingLevels {
	if lv == nil {
		return SwingLevels{}
	}
	return *lv
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
