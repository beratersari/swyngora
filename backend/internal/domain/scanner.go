package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Scanner limits.
const (
	MaxScannerRulesPerClient   = 30
	MaxScannerResultsPerClient = 500
	DefaultScannerInterval     = "1h"
	MinVolumeLookback          = 2
	MaxVolumeLookback          = 200
	MinVolumeRatio             = 1.01
	MaxVolumeRatio             = 100.0
)

// ScannerRuleType is the kind of technical scan rule.
type ScannerRuleType string

const (
	ScannerRuleRSI            ScannerRuleType = "rsi"
	ScannerRuleMACrossover    ScannerRuleType = "ma_crossover"
	ScannerRuleVolumeIncrease ScannerRuleType = "volume_increase"
	ScannerRuleCombo          ScannerRuleType = "combo"
)

// ScannerMatchMode is how selected conditions are combined.
type ScannerMatchMode string

const (
	ScannerMatchAll ScannerMatchMode = "all"
	ScannerMatchAny ScannerMatchMode = "any"
)

// ScannerRule is a client-owned rule evaluated against watchlist symbols.
type ScannerRule struct {
	ID         string
	ClientID   string
	Type       ScannerRuleType
	Conditions []ScannerRuleType
	MatchMode  ScannerMatchMode // all | any
	Interval   string           // candle interval, e.g. 1h
	Enabled    bool
	// RSI
	RSIPeriod    int
	RSICondition AlertCondition // above | below
	RSIThreshold float64
	// MA crossover (EMA)
	MAFastPeriod int
	MASlowPeriod int
	MADirection  string // golden_cross | death_cross
	// Volume increase
	VolumeLookback int
	VolumeMinRatio float64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ScannerResult is one saved match for a rule/symbol/market-data key.
type ScannerResult struct {
	ID            string
	ClientID      string
	RuleID        string
	Exchange      Exchange
	Symbol        string
	RuleType      ScannerRuleType
	Interval      string
	MarketDataKey string // candle open time (unique per rule/symbol bar)
	MatchedAt     time.Time
	Summary       string
	// Snapshot metrics at match (JSON-serializable map).
	Metrics map[string]float64
}

// ScannerMatch is an in-memory evaluation hit before persistence.
type ScannerMatch struct {
	MarketDataKey string
	Summary       string
	Metrics       map[string]float64
}

// Backtest horizons (calendar days after the signal bar).
var BacktestForwardDays = []int{1, 5, 20}

// ScannerBacktestStatus is the lifecycle of a historical rule test job.
type ScannerBacktestStatus string

const (
	BacktestPending   ScannerBacktestStatus = "pending"
	BacktestRunning   ScannerBacktestStatus = "running"
	BacktestCompleted ScannerBacktestStatus = "completed"
	BacktestCanceled  ScannerBacktestStatus = "canceled"
	BacktestFailed    ScannerBacktestStatus = "failed"
)

// MaxScannerBacktestsPerClient caps stored jobs per client.
const MaxScannerBacktestsPerClient = 50

const (
	// BacktestScopeSymbol is one pair (default).
	BacktestScopeSymbol = "symbol"
	// BacktestScopeWatchlist replays a custom mix on every watchlist coin.
	BacktestScopeWatchlist = "watchlist"
	// BacktestWatchlistSymbol is the job.Symbol sentinel for a watchlist run.
	BacktestWatchlistSymbol = "watchlist"
)

// BacktestMarket is one exchange+symbol in a watchlist Lab job.
type BacktestMarket struct {
	Exchange Exchange `json:"exchange"`
	Symbol   string   `json:"symbol"`
}

// ScannerBacktest is a background historical test of one rule or custom mix.
type ScannerBacktest struct {
	ID             string
	ClientID       string
	RuleID         string
	Kind           string // scanner_rule | custom_profile
	Exchange       Exchange
	Symbol         string
	Interval       string
	RangeStart     time.Time
	RangeEnd       time.Time
	Status         ScannerBacktestStatus
	ProgressPct    float64 // 0–100
	ProcessedBars  int
	TotalBars      int
	SignalCount    int
	SkippedBars    int
	MissingFactors []string
	ErrorMessage   string
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
	// RuleJSON is the rule or custom mix as queued. The worker uses this, not the live source.
	RuleJSON string
	// Run is per-coin status and the shared cash book (persisted after the worker walks coins).
	Run *BacktestRunState
	// Performance is filled when loading a job or its signals (not persisted).
	Performance *BacktestPerformance
}

// IsWatchlistBacktest reports a custom-mix job that covers the client's watchlist.
func IsWatchlistBacktest(job ScannerBacktest) bool {
	if strings.EqualFold(strings.TrimSpace(job.Symbol), BacktestWatchlistSymbol) {
		return true
	}
	if snap, err := parseCustomProfileSnapshot(job.RuleJSON); err == nil && snap.Scope == BacktestScopeWatchlist {
		return true
	}
	return false
}

// NormalizeBacktestScope maps API input. symbol=watchlist|all implies watchlist.
func NormalizeBacktestScope(scope, symbol string) string {
	s := strings.ToLower(strings.TrimSpace(scope))
	if s == BacktestScopeWatchlist {
		return BacktestScopeWatchlist
	}
	sym := strings.ToLower(strings.TrimSpace(symbol))
	if sym == BacktestWatchlistSymbol || sym == "all" {
		return BacktestScopeWatchlist
	}
	return BacktestScopeSymbol
}

// BacktestHorizonStats is one forward-return horizon across a job's signals.
type BacktestHorizonStats struct {
	Days        int      `json:"days"`
	SampleCount int      `json:"sampleCount"`
	Average     *float64 `json:"average,omitempty"`
	Median      *float64 `json:"median,omitempty"`
	WinRate     *float64 `json:"winRate,omitempty"` // 0–100, share of returns > 0
}

// BacktestPerformance is 1d / 5d / 20d summary stats plus first-touch trades.
type BacktestPerformance struct {
	SignalCount int                         `json:"signalCount"`
	Return1d    BacktestHorizonStats        `json:"return1d"`
	Return5d    BacktestHorizonStats        `json:"return5d"`
	Return20d   BacktestHorizonStats        `json:"return20d"`
	Trades      BacktestTradeStats          `json:"trades"`
	Markets     []BacktestMarketPerformance `json:"markets,omitempty"`
	Book        *BacktestBook               `json:"book,omitempty"`
}

// BacktestMarketPerformance is one coin's slice of a watchlist Lab job.
type BacktestMarketPerformance struct {
	Exchange    string               `json:"exchange"`
	Symbol      string               `json:"symbol"`
	Status      string               `json:"status,omitempty"`
	Error       string               `json:"error,omitempty"`
	SignalCount int                  `json:"signalCount"`
	Return1d    BacktestHorizonStats `json:"return1d"`
	Return5d    BacktestHorizonStats `json:"return5d"`
	Return20d   BacktestHorizonStats `json:"return20d"`
	Trades      BacktestTradeStats   `json:"trades"`
}

// ScannerBacktestSignal is one historical match with optional forward returns (%).
type ScannerBacktestSignal struct {
	ID         string
	BacktestID string
	Exchange   Exchange
	Symbol     string
	SignalAt   time.Time
	ClosePrice float64
	Summary    string
	Return1d   *float64 // percent change after 1 calendar day; nil if unavailable
	Return5d   *float64
	Return20d  *float64
	Metrics    map[string]float64
	Factors    []CustomBacktestFactor
	Trade      *BacktestTrade
}

// ScannerPort persists scanner rules, live results, and historical backtests.
type ScannerPort interface {
	CreateRule(ctx context.Context, r ScannerRule) (*ScannerRule, error)
	GetRule(ctx context.Context, clientID, id string) (*ScannerRule, error)
	UpdateRule(ctx context.Context, r ScannerRule) (*ScannerRule, error)
	ListRules(ctx context.Context, clientID string) ([]ScannerRule, error)
	ListEnabledRules(ctx context.Context) ([]ScannerRule, error)
	DeleteRule(ctx context.Context, clientID, id string) error
	CountRules(ctx context.Context, clientID string) (int, error)

	// InsertResult inserts if the (rule_id, exchange, symbol, market_data_key) is new.
	// Returns (nil, false, nil) when a duplicate already exists.
	InsertResult(ctx context.Context, res ScannerResult) (*ScannerResult, bool, error)
	ListResults(ctx context.Context, clientID string, limit, offset int) ([]ScannerResult, error)
	CountResults(ctx context.Context, clientID string) (int, error)

	// Backtests
	CreateBacktest(ctx context.Context, b ScannerBacktest) (*ScannerBacktest, error)
	GetBacktest(ctx context.Context, clientID, id string) (*ScannerBacktest, error)
	// FindActiveBacktest finds pending/running/completed job with same fingerprint.
	FindActiveBacktest(ctx context.Context, clientID, ruleID string, exchange Exchange, symbol string, rangeStart, rangeEnd time.Time) (*ScannerBacktest, error)
	ListBacktests(ctx context.Context, clientID string, limit, offset int) ([]ScannerBacktest, error)
	CountBacktests(ctx context.Context, clientID string) (int, error)
	ListPendingBacktests(ctx context.Context) ([]ScannerBacktest, error)
	// ClaimBacktest moves pending → running if still pending. Returns false if not claimed.
	ClaimBacktest(ctx context.Context, id string, at time.Time) (bool, error)
	UpdateBacktestProgress(ctx context.Context, id string, processed, total, signalCount int, progressPct float64) error
	// UpdateBacktestSkipStats records bars skipped because a selected factor had no data.
	UpdateBacktestSkipStats(ctx context.Context, id string, skipped int, missing []string) error
	// UpdateBacktestRun stores per-coin status and the shared cash book.
	UpdateBacktestRun(ctx context.Context, id string, run BacktestRunState) error
	// FinishBacktest sets terminal status (completed/canceled/failed).
	FinishBacktest(ctx context.Context, id string, status ScannerBacktestStatus, signalCount int, errMsg string, at time.Time) error
	// CancelBacktest sets canceled if pending or running.
	CancelBacktest(ctx context.Context, clientID, id string, at time.Time) (*ScannerBacktest, error)
	// GetBacktestStatus returns only status (for cancel checks during run).
	GetBacktestStatus(ctx context.Context, id string) (ScannerBacktestStatus, error)

	InsertBacktestSignal(ctx context.Context, sig ScannerBacktestSignal) error
	ListBacktestSignals(ctx context.Context, backtestID string, limit, offset int) ([]ScannerBacktestSignal, error)
	CountBacktestSignals(ctx context.Context, backtestID string) (int, error)
	// DeleteBacktest removes a job and its signals (CASCADE). Owner-scoped.
	DeleteBacktest(ctx context.Context, clientID, id string) error
	// PurgeClient deletes rules, results, and backtests for clientID.
	PurgeClient(ctx context.Context, clientID string) error
}

// ForwardReturnPct is (futureClose - signalClose) / signalClose * 100.
// days is calendar days after the signal bar open time.
// Returns nil when no future candle is available.
func ForwardReturnPct(candles []Candle, signalIdx int, days int) *float64 {
	if signalIdx < 0 || signalIdx >= len(candles) || days <= 0 {
		return nil
	}
	sigClose, err := parseClose(candles[signalIdx].Close)
	if err != nil || sigClose <= 0 {
		return nil
	}
	target := candles[signalIdx].OpenTime.UTC().Add(time.Duration(days) * 24 * time.Hour)
	for j := signalIdx + 1; j < len(candles); j++ {
		if candles[j].OpenTime.UTC().Before(target) {
			continue
		}
		fut, err := parseClose(candles[j].Close)
		if err != nil || fut <= 0 {
			return nil
		}
		pct := (fut - sigClose) / sigClose * 100
		return &pct
	}
	return nil
}

// SummarizeBacktestReturns averages / medians / win-rates each horizon.
// A return is a win only when it is greater than 0. Missing horizons are omitted from that column.
func SummarizeBacktestReturns(sigs []ScannerBacktestSignal) BacktestPerformance {
	one := make([]float64, 0, len(sigs))
	five := make([]float64, 0, len(sigs))
	twenty := make([]float64, 0, len(sigs))
	for _, s := range sigs {
		if s.Return1d != nil && !math.IsNaN(*s.Return1d) {
			one = append(one, *s.Return1d)
		}
		if s.Return5d != nil && !math.IsNaN(*s.Return5d) {
			five = append(five, *s.Return5d)
		}
		if s.Return20d != nil && !math.IsNaN(*s.Return20d) {
			twenty = append(twenty, *s.Return20d)
		}
	}
	return BacktestPerformance{
		SignalCount: len(sigs),
		Return1d:    backtestHorizonStats(1, one),
		Return5d:    backtestHorizonStats(5, five),
		Return20d:   backtestHorizonStats(20, twenty),
		Trades:      summarizeBacktestTrades(sigs),
		Markets:     summarizeBacktestMarkets(sigs),
	}
}

func summarizeBacktestMarkets(sigs []ScannerBacktestSignal) []BacktestMarketPerformance {
	order := make([]string, 0)
	groups := map[string][]ScannerBacktestSignal{}
	for _, s := range sigs {
		ex, sym := string(s.Exchange), strings.TrimSpace(s.Symbol)
		if ex == "" && sym == "" {
			continue
		}
		key := ex + "|" + strings.ToUpper(sym)
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], s)
	}
	if len(order) == 0 {
		return nil
	}
	out := make([]BacktestMarketPerformance, 0, len(order))
	for _, key := range order {
		g := groups[key]
		one := make([]float64, 0, len(g))
		five := make([]float64, 0, len(g))
		twenty := make([]float64, 0, len(g))
		for _, s := range g {
			if s.Return1d != nil && !math.IsNaN(*s.Return1d) {
				one = append(one, *s.Return1d)
			}
			if s.Return5d != nil && !math.IsNaN(*s.Return5d) {
				five = append(five, *s.Return5d)
			}
			if s.Return20d != nil && !math.IsNaN(*s.Return20d) {
				twenty = append(twenty, *s.Return20d)
			}
		}
		out = append(out, BacktestMarketPerformance{
			Exchange: string(g[0].Exchange), Symbol: g[0].Symbol,
			SignalCount: len(g),
			Return1d:    backtestHorizonStats(1, one),
			Return5d:    backtestHorizonStats(5, five),
			Return20d:   backtestHorizonStats(20, twenty),
			Trades:      summarizeBacktestTrades(g),
		})
	}
	return out
}

func backtestHorizonStats(days int, xs []float64) BacktestHorizonStats {
	out := BacktestHorizonStats{Days: days, SampleCount: len(xs)}
	if len(xs) == 0 {
		return out
	}
	sum := 0.0
	wins := 0
	for _, v := range xs {
		sum += v
		if v > 0 {
			wins++
		}
	}
	avg := round1(sum / float64(len(xs)))
	med := round1(medianFloats(xs))
	wr := round1(float64(wins) / float64(len(xs)) * 100)
	out.Average = &avg
	out.Median = &med
	out.WinRate = &wr
	return out
}

func medianFloats(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	cp := append([]float64(nil), xs...)
	sort.Float64s(cp)
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

// IsValidScannerRuleType reports known rule types.
func IsValidScannerRuleType(s string) bool {
	switch ScannerRuleType(s) {
	case ScannerRuleRSI, ScannerRuleMACrossover, ScannerRuleVolumeIncrease, ScannerRuleCombo:
		return true
	default:
		return false
	}
}

func isAtomicScannerType(t ScannerRuleType) bool {
	switch t {
	case ScannerRuleRSI, ScannerRuleMACrossover, ScannerRuleVolumeIncrease:
		return true
	default:
		return false
	}
}

// ResolveScannerMatchMode defaults to all. Accepts all | any.
func ResolveScannerMatchMode(s string) (ScannerMatchMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "all", "and":
		return ScannerMatchAll, nil
	case "any", "or":
		return ScannerMatchAny, nil
	default:
		return "", fmt.Errorf("%w: matchMode must be all or any", ErrInvalidArgument)
	}
}

// ResolveScannerConditions builds the selected set from conditions and/or type.
// Legacy type-only rules keep a single condition.
func ResolveScannerConditions(typ string, conds []string) ([]ScannerRuleType, ScannerRuleType, error) {
	seen := map[ScannerRuleType]bool{}
	out := make([]ScannerRuleType, 0, 3)
	add := func(raw string) error {
		t := ScannerRuleType(strings.ToLower(strings.TrimSpace(raw)))
		if !isAtomicScannerType(t) {
			return fmt.Errorf("%w: condition must be rsi, ma_crossover, or volume_increase", ErrInvalidArgument)
		}
		if seen[t] {
			return nil
		}
		seen[t] = true
		out = append(out, t)
		return nil
	}
	for _, c := range conds {
		if strings.TrimSpace(c) == "" {
			continue
		}
		if err := add(c); err != nil {
			return nil, "", err
		}
	}
	if len(out) == 0 {
		t := ScannerRuleType(strings.ToLower(strings.TrimSpace(typ)))
		if t == ScannerRuleCombo {
			return nil, "", fmt.Errorf("%w: conditions are required when type is combo", ErrInvalidArgument)
		}
		if !isAtomicScannerType(t) {
			return nil, "", fmt.Errorf("%w: type must be rsi, ma_crossover, or volume_increase", ErrInvalidArgument)
		}
		out = []ScannerRuleType{t}
	}
	stored := ScannerRuleCombo
	if len(out) == 1 {
		stored = out[0]
	}
	return out, stored, nil
}

const (
	scannerRSIWarmup     = 30
	scannerMAWarmup      = 30
	scannerVolumePad     = 5
	scannerCandleNeedMin = 20
	scannerCandleNeedMax = 1000
)

// ScannerCandleNeed is how many latest candles EvaluateScannerRule needs.
// Combo rules use the longest selected condition (not a fixed 100).
func ScannerCandleNeed(rule ScannerRule) int {
	need := 0
	for _, c := range rule.SelectedConditions() {
		n := 0
		switch c {
		case ScannerRuleRSI:
			p := rule.RSIPeriod
			if p <= 0 {
				p = DefaultRSIPeriod
			}
			n = p + scannerRSIWarmup
		case ScannerRuleMACrossover:
			slow := rule.MASlowPeriod
			if slow <= 0 {
				slow = DefaultEMASlow
			}
			n = slow + scannerMAWarmup
		case ScannerRuleVolumeIncrease:
			lb := rule.VolumeLookback
			if lb <= 0 {
				lb = 20
			}
			n = lb + scannerVolumePad
		}
		if n > need {
			need = n
		}
	}
	if need == 0 {
		return 100
	}
	if need < scannerCandleNeedMin {
		return scannerCandleNeedMin
	}
	if need > scannerCandleNeedMax {
		return scannerCandleNeedMax
	}
	return need
}

// SelectedConditions is the set EvaluateScannerRule runs. Empty Conditions
// falls back to Type for rules created before combo support.
func (r ScannerRule) SelectedConditions() []ScannerRuleType {
	if len(r.Conditions) > 0 {
		return r.Conditions
	}
	if isAtomicScannerType(r.Type) {
		return []ScannerRuleType{r.Type}
	}
	return nil
}

// IsValidMADirection reports golden_cross | death_cross.
func IsValidMADirection(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "golden_cross", "death_cross":
		return true
	default:
		return false
	}
}

// EvaluateScannerRuleOnset matches only when the latest bar newly satisfies
// the rule and the previous bar did not (false → true). A condition that
// stays true does not produce another hit until it goes false and true again.
func EvaluateScannerRuleOnset(rule ScannerRule, candles []Candle) (*ScannerMatch, error) {
	curr, err := EvaluateScannerRule(rule, candles)
	if err != nil || curr == nil {
		return curr, err
	}
	if len(candles) < 2 {
		return curr, nil
	}
	prev, err := EvaluateScannerRule(rule, candles[:len(candles)-1])
	if err != nil {
		return nil, err
	}
	if prev != nil {
		return nil, nil
	}
	return curr, nil
}

// scannerRuleSnapshot is the queued copy of a rule used by backtests.
type scannerRuleSnapshot struct {
	Type           string   `json:"type"`
	Conditions     []string `json:"conditions"`
	MatchMode      string   `json:"matchMode"`
	Interval       string   `json:"interval"`
	RSIPeriod      int      `json:"rsiPeriod"`
	RSICondition   string   `json:"rsiCondition"`
	RSIThreshold   float64  `json:"rsiThreshold"`
	MAFastPeriod   int      `json:"maFastPeriod"`
	MASlowPeriod   int      `json:"maSlowPeriod"`
	MADirection    string   `json:"maDirection"`
	VolumeLookback int      `json:"volumeLookback"`
	VolumeMinRatio float64  `json:"volumeMinRatio"`
	UseTP2         bool     `json:"useTp2,omitempty"`
	TP1ClosePct    float64  `json:"tp1ClosePct,omitempty"`
	StopAfterTP1   string   `json:"stopAfterTp1,omitempty"`
	FeePct         float64  `json:"feePct,omitempty"`
	SlipPct        float64  `json:"slipPct,omitempty"`
}

// EncodeScannerRuleSnapshot freezes rule params for a backtest job.
func EncodeScannerRuleSnapshot(r ScannerRule) (string, error) {
	return EncodeScannerRuleSnapshotScale(r, BacktestScale{})
}

// EncodeScannerRuleSnapshotScale freezes the rule plus Lab TP scale.
func EncodeScannerRuleSnapshotScale(r ScannerRule, scale BacktestScale) (string, error) {
	conds := r.SelectedConditions()
	names := make([]string, 0, len(conds))
	for _, c := range conds {
		names = append(names, string(c))
	}
	mode := string(r.MatchMode)
	if mode == "" {
		mode = string(ScannerMatchAll)
	}
	scale = NormalizeBacktestScale(scale)
	b, err := json.Marshal(scannerRuleSnapshot{
		Type: string(r.Type), Conditions: names, MatchMode: mode, Interval: r.Interval,
		RSIPeriod: r.RSIPeriod, RSICondition: string(r.RSICondition), RSIThreshold: r.RSIThreshold,
		MAFastPeriod: r.MAFastPeriod, MASlowPeriod: r.MASlowPeriod, MADirection: r.MADirection,
		VolumeLookback: r.VolumeLookback, VolumeMinRatio: r.VolumeMinRatio,
		UseTP2: scale.UseTP2, TP1ClosePct: scale.TP1ClosePct, StopAfterTP1: scale.StopAfterTP1,
		FeePct: scale.FeePct, SlipPct: scale.SlipPct,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseScannerRuleSnapshot(raw string) (scannerRuleSnapshot, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return scannerRuleSnapshot{}, fmt.Errorf("%w: missing scanner rule snapshot", ErrInvalidArgument)
	}
	var snap scannerRuleSnapshot
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		return scannerRuleSnapshot{}, fmt.Errorf("%w: invalid scanner rule snapshot", ErrInvalidArgument)
	}
	return snap, nil
}

// DecodeScannerRuleSnapshot restores a queued rule. Empty input is an error.
func DecodeScannerRuleSnapshot(raw string) (*ScannerRule, error) {
	snap, err := parseScannerRuleSnapshot(raw)
	if err != nil {
		return nil, err
	}
	conds, typ, err := ResolveScannerConditions(snap.Type, snap.Conditions)
	if err != nil {
		return nil, err
	}
	mode, err := ResolveScannerMatchMode(snap.MatchMode)
	if err != nil {
		return nil, err
	}
	return &ScannerRule{
		Type: typ, Conditions: conds, MatchMode: mode, Interval: snap.Interval,
		RSIPeriod: snap.RSIPeriod, RSICondition: AlertCondition(snap.RSICondition), RSIThreshold: snap.RSIThreshold,
		MAFastPeriod: snap.MAFastPeriod, MASlowPeriod: snap.MASlowPeriod, MADirection: snap.MADirection,
		VolumeLookback: snap.VolumeLookback, VolumeMinRatio: snap.VolumeMinRatio,
	}, nil
}

// EvaluateScannerRule evaluates the latest bar(s) of candles for a rule.
// Returns nil when the condition is not met or data is insufficient.
func EvaluateScannerRule(rule ScannerRule, candles []Candle) (*ScannerMatch, error) {
	if len(candles) == 0 {
		return nil, nil
	}
	conds := rule.SelectedConditions()
	if len(conds) == 0 {
		return nil, fmt.Errorf("%w: unknown scanner rule type", ErrInvalidArgument)
	}
	mode := rule.MatchMode
	if mode == "" {
		mode = ScannerMatchAll
	}
	hits := make([]*ScannerMatch, 0, len(conds))
	for _, c := range conds {
		sub := rule
		sub.Type = c
		var (
			m   *ScannerMatch
			err error
		)
		switch c {
		case ScannerRuleRSI:
			m, err = evalRSI(sub, candles)
		case ScannerRuleMACrossover:
			m, err = evalMACrossover(sub, candles)
		case ScannerRuleVolumeIncrease:
			m, err = evalVolumeIncrease(sub, candles)
		default:
			return nil, fmt.Errorf("%w: unknown scanner rule type", ErrInvalidArgument)
		}
		if err != nil {
			return nil, err
		}
		if m != nil {
			hits = append(hits, m)
		} else if mode == ScannerMatchAll {
			return nil, nil
		}
	}
	if len(hits) == 0 {
		return nil, nil
	}
	return combineScannerMatches(hits, mode), nil
}

func combineScannerMatches(hits []*ScannerMatch, mode ScannerMatchMode) *ScannerMatch {
	if len(hits) == 1 {
		return hits[0]
	}
	sep := " and "
	if mode == ScannerMatchAny {
		sep = " or "
	}
	out := &ScannerMatch{
		MarketDataKey: hits[0].MarketDataKey,
		Metrics:       map[string]float64{},
	}
	parts := make([]string, 0, len(hits))
	for _, h := range hits {
		if h == nil {
			continue
		}
		if h.MarketDataKey != "" {
			out.MarketDataKey = h.MarketDataKey
		}
		parts = append(parts, h.Summary)
		for k, v := range h.Metrics {
			out.Metrics[k] = v
		}
	}
	out.Summary = strings.Join(parts, sep)
	return out
}

func evalRSI(rule ScannerRule, candles []Candle) (*ScannerMatch, error) {
	period := rule.RSIPeriod
	if period <= 0 {
		period = DefaultRSIPeriod
	}
	closes, err := ParseClosePrices(candles)
	if err != nil {
		return nil, err
	}
	rsi := RSIWilder(closes, period)
	i := len(rsi) - 1
	if i < 0 || rsi[i] == nil {
		return nil, nil
	}
	v := *rsi[i]
	ok := false
	switch rule.RSICondition {
	case AlertAbove:
		ok = v >= rule.RSIThreshold-1e-12
	case AlertBelow:
		ok = v <= rule.RSIThreshold+1e-12
	default:
		return nil, fmt.Errorf("%w: invalid rsi condition", ErrInvalidArgument)
	}
	if !ok {
		return nil, nil
	}
	bar := candles[i]
	return &ScannerMatch{
		MarketDataKey: marketDataKey(bar),
		Summary:       fmt.Sprintf("RSI(%d)=%.2f %s %.2f", period, v, rule.RSICondition, rule.RSIThreshold),
		Metrics: map[string]float64{
			"rsi":       v,
			"period":    float64(period),
			"threshold": rule.RSIThreshold,
			"close":     closes[i],
		},
	}, nil
}

func evalMACrossover(rule ScannerRule, candles []Candle) (*ScannerMatch, error) {
	fastP, slowP := rule.MAFastPeriod, rule.MASlowPeriod
	if fastP <= 0 {
		fastP = DefaultEMAFast
	}
	if slowP <= 0 {
		slowP = DefaultEMASlow
	}
	if fastP >= slowP {
		return nil, fmt.Errorf("%w: ma fastPeriod must be < slowPeriod", ErrInvalidArgument)
	}
	closes, err := ParseClosePrices(candles)
	if err != nil {
		return nil, err
	}
	fast := EMA(closes, fastP)
	slow := EMA(closes, slowP)
	n := len(closes)
	if n < 2 {
		return nil, nil
	}
	i := n - 1
	j := n - 2
	if fast[i] == nil || slow[i] == nil || fast[j] == nil || slow[j] == nil {
		return nil, nil
	}
	f0, s0 := *fast[j], *slow[j]
	f1, s1 := *fast[i], *slow[i]
	dir := strings.ToLower(strings.TrimSpace(rule.MADirection))
	hit := false
	switch dir {
	case "golden_cross":
		hit = f0 <= s0+1e-12 && f1 > s1+1e-12
	case "death_cross":
		hit = f0 >= s0-1e-12 && f1 < s1-1e-12
	default:
		return nil, fmt.Errorf("%w: invalid ma direction", ErrInvalidArgument)
	}
	if !hit {
		return nil, nil
	}
	bar := candles[i]
	return &ScannerMatch{
		MarketDataKey: marketDataKey(bar),
		Summary:       fmt.Sprintf("EMA(%d)/%d %s (%.4f/%.4f)", fastP, slowP, dir, f1, s1),
		Metrics: map[string]float64{
			"emaFast":    f1,
			"emaSlow":    s1,
			"fastPeriod": float64(fastP),
			"slowPeriod": float64(slowP),
			"close":      closes[i],
		},
	}, nil
}

func evalVolumeIncrease(rule ScannerRule, candles []Candle) (*ScannerMatch, error) {
	lookback := rule.VolumeLookback
	if lookback <= 0 {
		lookback = 20
	}
	if len(candles) < lookback+1 {
		return nil, nil
	}
	vols := make([]float64, len(candles))
	for i, c := range candles {
		v, err := parseVolume(c.Volume)
		if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("%w: invalid volume at candle index %d", ErrUpstream, i)
		}
		vols[i] = v
	}
	i := len(vols) - 1
	last := vols[i]
	var sum float64
	for k := i - lookback; k < i; k++ {
		sum += vols[k]
	}
	avg := sum / float64(lookback)
	if avg <= 0 {
		return nil, nil
	}
	ratio := last / avg
	if ratio+1e-12 < rule.VolumeMinRatio {
		return nil, nil
	}
	bar := candles[i]
	return &ScannerMatch{
		MarketDataKey: marketDataKey(bar),
		Summary:       fmt.Sprintf("Volume %.2fx avg of prior %d bars", ratio, lookback),
		Metrics: map[string]float64{
			"volume":    last,
			"avgVolume": avg,
			"ratio":     ratio,
			"lookback":  float64(lookback),
			"minRatio":  rule.VolumeMinRatio,
		},
	}, nil
}

func marketDataKey(c Candle) string {
	if !c.OpenTime.IsZero() {
		return c.OpenTime.UTC().Format(time.RFC3339Nano)
	}
	if !c.CloseTime.IsZero() {
		return c.CloseTime.UTC().Format(time.RFC3339Nano)
	}
	return strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}

func parseVolume(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(s, 64)
}
