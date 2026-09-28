package domain

import (
	"fmt"
	"math"
	"time"
)

// Optional paper risk-limit bounds (user-set; never auto-close positions).
const (
	MinRiskLimitPct = 0.01
	MaxRiskLimitPct = 100.0
	// MaxExposureLimitPct allows a gross book above 100% of equity when margin is open.
	MaxExposureLimitPct = 1000.0
)

// RiskLimits are per-client optional brakes on new risk (buys / new margin opens).
// Nil percents mean that rule is off. Existing positions are never closed.
type RiskLimits struct {
	ClientID           string
	MaxDailyLossPct    *float64 // e.g. 5 = block new risk if today is down >= 5%
	MaxAssetWeightPct  *float64 // e.g. 30 = block new buys that push one coin over 30%
	MaxExposurePct     *float64 // e.g. 60 = block new buys once all coins together exceed 60% of equity
	DayKey             string   // UTC date YYYY-MM-DD for start-of-day equity snapshot
	DayStartEquity     float64
	UpdatedAt          time.Time
}

// RiskAssetWeight is one coin's live share of portfolio equity (spot + margin notional).
type RiskAssetWeight struct {
	Asset       string
	Value       float64
	WeightPct   float64
	AtOrOverLimit bool
}

// RiskStatus is live telemetry for the settings / block UI.
type RiskStatus struct {
	DayKey             string
	StartOfDayEquity   float64
	Equity             float64
	DailyPnL           float64
	DailyPnLPct        float64
	DailyLossLimitHit  bool
	ExposureValue      float64
	ExposurePct        float64
	ExposureLimitHit   bool
	Assets             []RiskAssetWeight
	CanOpenSpotBuy     bool
	CanOpenMargin      bool
	BlockReasons       []string
}

// UTCDayKey is the UTC calendar day used for daily-loss windows.
func UTCDayKey(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

// ValidateOptionalExposurePct accepts nil (disabled) or (0, 1000].
func ValidateOptionalExposurePct(p *float64) error {
	if p == nil {
		return nil
	}
	v := *p
	if v < MinRiskLimitPct || v > MaxExposureLimitPct || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%w: maxExposurePct must be between %g and %g, or omitted to disable", ErrInvalidArgument, MinRiskLimitPct, MaxExposureLimitPct)
	}
	return nil
}

// ValidateOptionalRiskPct accepts nil (disabled) or (0, 100].
func ValidateOptionalRiskPct(p *float64, field string) error {
	if p == nil {
		return nil
	}
	v := *p
	if v < MinRiskLimitPct || v > MaxRiskLimitPct || math.IsNaN(v) || math.IsInf(v, 0) {
		return fmt.Errorf("%w: %s must be between %g and %g, or omitted to disable", ErrInvalidArgument, field, MinRiskLimitPct, MaxRiskLimitPct)
	}
	return nil
}

// DailyPnLPct is (equityNow - start)/start * 100. Zero start → 0.
func DailyPnLPct(startEquity, equityNow float64) float64 {
	if startEquity <= PositionEpsilon {
		return 0
	}
	return 100 * (equityNow - startEquity) / startEquity
}

// DailyLossLimitHit reports whether mark-to-market daily loss reached the user cap.
func DailyLossLimitHit(startEquity, equityNow, maxLossPct float64) bool {
	if maxLossPct <= 0 || startEquity <= PositionEpsilon {
		return false
	}
	pnlPct := DailyPnLPct(startEquity, equityNow)
	return pnlPct <= -maxLossPct+1e-12
}

// ProjectedAssetWeightPct is (currentAssetValue + additionalNotional) / equity * 100.
// Buying swaps cash for coin so equity is unchanged; additionalNotional increases that coin's sleeve.
func ProjectedAssetWeightPct(equity, currentAssetValue, additionalNotional float64) float64 {
	if equity <= PositionEpsilon {
		return 0
	}
	v := currentAssetValue + additionalNotional
	if v < 0 {
		v = 0
	}
	return 100 * v / equity
}

// AssetWeightLimitHit reports whether a new add would exceed maxWeightPct.
func AssetWeightLimitHit(equity, currentAssetValue, additionalNotional, maxWeightPct float64) bool {
	if maxWeightPct <= 0 {
		return false
	}
	return ProjectedAssetWeightPct(equity, currentAssetValue, additionalNotional) > maxWeightPct+1e-9
}

// ProjectedExposurePct is all coins together, plus the new order, as a percent of equity.
func ProjectedExposurePct(equity, currentExposure, additionalNotional float64) float64 {
	return ProjectedAssetWeightPct(equity, currentExposure, additionalNotional)
}

// ExposureLimitHit reports whether a new add would push total coin exposure over the cap.
func ExposureLimitHit(equity, currentExposure, additionalNotional, maxExposurePct float64) bool {
	if maxExposurePct <= 0 {
		return false
	}
	return ProjectedExposurePct(equity, currentExposure, additionalNotional) > maxExposurePct+1e-9
}

// RiskLimitMessages formats user-facing block reasons (empty if none fire).
func RiskLimitMessages(limits RiskLimits, startEquity, equityNow float64, asset string, assetValue, exposureValue, additionalNotional float64) []string {
	var reasons []string
	if limits.MaxDailyLossPct != nil && DailyLossLimitHit(startEquity, equityNow, *limits.MaxDailyLossPct) {
		reasons = append(reasons, fmt.Sprintf("daily loss limit reached (%.2f%% loss >= %.2f%%)",
			math.Abs(DailyPnLPct(startEquity, equityNow)), *limits.MaxDailyLossPct))
	}
	if limits.MaxAssetWeightPct != nil && asset != "" &&
		AssetWeightLimitHit(equityNow, assetValue, additionalNotional, *limits.MaxAssetWeightPct) {
		proj := ProjectedAssetWeightPct(equityNow, assetValue, additionalNotional)
		reasons = append(reasons, fmt.Sprintf("%s would be %.2f%% of the portfolio (limit %.2f%%)",
			asset, proj, *limits.MaxAssetWeightPct))
	}
	if limits.MaxExposurePct != nil && additionalNotional > 0 &&
		ExposureLimitHit(equityNow, exposureValue, additionalNotional, *limits.MaxExposurePct) {
		proj := ProjectedExposurePct(equityNow, exposureValue, additionalNotional)
		reasons = append(reasons, fmt.Sprintf("total exposure would be %.2f%% of the portfolio (limit %.2f%%)",
			proj, *limits.MaxExposurePct))
	}
	return reasons
}
