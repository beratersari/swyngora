package domain

import (
	"fmt"
	"math"
	"strings"
)

// NormalizeAlertKind returns price for empty input; validates otherwise.
func NormalizeAlertKind(s string) (AlertKind, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return AlertKindPrice, true
	}
	switch AlertKind(s) {
	case AlertKindPrice, AlertKindImbalance, AlertKindWall:
		return AlertKind(s), true
	default:
		return "", false
	}
}

// IsBookAlert is true for imbalance/wall kinds (checked against the live book).
func IsBookAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && (k == AlertKindImbalance || k == AlertKindWall)
}

// EffectiveAlertKind treats empty as price (old rows).
func EffectiveAlertKind(a PriceAlert) AlertKind {
	k, ok := NormalizeAlertKind(string(a.Kind))
	if !ok {
		return AlertKindPrice
	}
	return k
}

// ValidateAlertSpec checks kind + condition + target (+ optional rangePct).
func ValidateAlertSpec(kind AlertKind, condition string, target, rangePct float64) error {
	kind, ok := NormalizeAlertKind(string(kind))
	if !ok {
		return fmt.Errorf("%w: kind must be price, imbalance, or wall", ErrInvalidArgument)
	}
	cond := strings.ToLower(strings.TrimSpace(condition))
	if rangePct < 0 || math.IsNaN(rangePct) || math.IsInf(rangePct, 0) {
		return fmt.Errorf("%w: rangePct must be >= 0", ErrInvalidArgument)
	}
	switch kind {
	case AlertKindPrice:
		if !IsValidAlertCondition(cond) {
			return fmt.Errorf("%w: condition must be above or below", ErrInvalidArgument)
		}
		if target <= 0 || math.IsNaN(target) || math.IsInf(target, 0) {
			return fmt.Errorf("%w: targetPrice must be a positive number", ErrInvalidArgument)
		}
	case AlertKindImbalance:
		if cond != string(AlertAbove) && cond != string(AlertBelow) {
			return fmt.Errorf("%w: imbalance condition must be above (buy) or below (sell)", ErrInvalidArgument)
		}
		if target < MinImbalanceAlertThreshold || target > MaxImbalanceAlertThreshold || math.IsNaN(target) {
			return fmt.Errorf("%w: imbalance threshold must be between %.2f and %.2f", ErrInvalidArgument, MinImbalanceAlertThreshold, MaxImbalanceAlertThreshold)
		}
	case AlertKindWall:
		if cond != string(AlertWallBid) && cond != string(AlertWallAsk) && cond != string(AlertWallAny) {
			return fmt.Errorf("%w: wall condition must be bid, ask, or any", ErrInvalidArgument)
		}
		if target < 0 || target > 1 || math.IsNaN(target) || math.IsInf(target, 0) {
			return fmt.Errorf("%w: wall minShare must be between 0 and 1", ErrInvalidArgument)
		}
	}
	return nil
}

// BookAlertObservation is whether a live book currently satisfies a book alert,
// plus the metric stored as triggeredPrice (imbalance or best wall share).
func BookAlertObservation(a PriceAlert, an OrderBookAnalysis) (met bool, metric float64) {
	switch EffectiveAlertKind(a) {
	case AlertKindImbalance:
		metric = an.Imbalance
		thr := a.TargetPrice
		if a.Condition == AlertAbove {
			return metric >= thr, metric
		}
		if a.Condition == AlertBelow {
			return metric <= -thr, metric
		}
		return false, metric
	case AlertKindWall:
		return wallAlertObservation(a, an)
	default:
		return false, 0
	}
}

func wallAlertObservation(a PriceAlert, an OrderBookAnalysis) (bool, float64) {
	want := strings.ToLower(strings.TrimSpace(string(a.Condition)))
	best := 0.0
	found := false
	for _, w := range an.Walls {
		side := strings.ToLower(strings.TrimSpace(w.Side))
		if want == string(AlertWallBid) && side != "bid" {
			continue
		}
		if want == string(AlertWallAsk) && side != "ask" {
			continue
		}
		if a.TargetPrice > 0 && w.Share+1e-12 < a.TargetPrice {
			continue
		}
		found = true
		if w.Share > best {
			best = w.Share
		}
	}
	return found, best
}

// EvaluateBookAlert applies one live-book snapshot to an imbalance/wall alert.
func EvaluateBookAlert(a PriceAlert, an OrderBookAnalysis) (AlertEvalResult, float64) {
	met, metric := BookAlertObservation(a, an)
	return EvaluateAlertState(a, met), metric
}
