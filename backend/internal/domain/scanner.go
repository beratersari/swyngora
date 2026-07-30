package domain

import (
	"context"
	"fmt"
	"math"
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
)

// ScannerRule is a client-owned rule evaluated against watchlist symbols.
type ScannerRule struct {
	ID        string
	ClientID  string
	Type      ScannerRuleType
	Interval  string // candle interval, e.g. 1h
	Enabled   bool
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

// ScannerBacktest is a background historical test of one rule on one symbol.
type ScannerBacktest struct {
	ID             string
	ClientID       string
	RuleID         string
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
	ErrorMessage   string
	CreatedAt      time.Time
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

// ScannerBacktestSignal is one historical match with optional forward returns (%).
type ScannerBacktestSignal struct {
	ID          string
	BacktestID  string
	SignalAt    time.Time
	ClosePrice  float64
	Summary     string
	Return1d    *float64 // percent change after 1 calendar day; nil if unavailable
	Return5d    *float64
	Return20d   *float64
	Metrics     map[string]float64
}

// ScannerPort persists scanner rules, live results, and historical backtests.
type ScannerPort interface {
	CreateRule(ctx context.Context, r ScannerRule) (*ScannerRule, error)
	GetRule(ctx context.Context, clientID, id string) (*ScannerRule, error)
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
	// FinishBacktest sets terminal status (completed/canceled/failed).
	FinishBacktest(ctx context.Context, id string, status ScannerBacktestStatus, signalCount int, errMsg string, at time.Time) error
	// CancelBacktest sets canceled if pending or running.
	CancelBacktest(ctx context.Context, clientID, id string, at time.Time) (*ScannerBacktest, error)
	// GetBacktestStatus returns only status (for cancel checks during run).
	GetBacktestStatus(ctx context.Context, id string) (ScannerBacktestStatus, error)

	InsertBacktestSignal(ctx context.Context, sig ScannerBacktestSignal) error
	ListBacktestSignals(ctx context.Context, backtestID string, limit, offset int) ([]ScannerBacktestSignal, error)
	CountBacktestSignals(ctx context.Context, backtestID string) (int, error)
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

// IsValidScannerRuleType reports known rule types.
func IsValidScannerRuleType(s string) bool {
	switch ScannerRuleType(s) {
	case ScannerRuleRSI, ScannerRuleMACrossover, ScannerRuleVolumeIncrease:
		return true
	default:
		return false
	}
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

// EvaluateScannerRule evaluates the latest bar(s) of candles for a rule.
// Returns nil when the condition is not met or data is insufficient.
func EvaluateScannerRule(rule ScannerRule, candles []Candle) (*ScannerMatch, error) {
	if len(candles) == 0 {
		return nil, nil
	}
	switch rule.Type {
	case ScannerRuleRSI:
		return evalRSI(rule, candles)
	case ScannerRuleMACrossover:
		return evalMACrossover(rule, candles)
	case ScannerRuleVolumeIncrease:
		return evalVolumeIncrease(rule, candles)
	default:
		return nil, fmt.Errorf("%w: unknown scanner rule type", ErrInvalidArgument)
	}
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
			"volume":     last,
			"avgVolume":  avg,
			"ratio":      ratio,
			"lookback":   float64(lookback),
			"minRatio":   rule.VolumeMinRatio,
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
