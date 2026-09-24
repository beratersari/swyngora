package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	AlertKindFunding      AlertKind = "funding"
	AlertKindOpenInterest AlertKind = "open_interest"
	AlertKindTaker        AlertKind = "taker"
	AlertKindFearGreed    AlertKind = "fear_greed"
)

// IsTapeAlert reports funding, open interest, taker buy share, or fear and greed.
func IsTapeAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && (k == AlertKindFunding || k == AlertKindOpenInterest || k == AlertKindTaker || k == AlertKindFearGreed)
}

// IsFearGreedAlert reports the market-wide 0–100 score.
func IsFearGreedAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && k == AlertKindFearGreed
}

func validateTapeAlertSpec(kind AlertKind, condition string, target float64) error {
	if condition != string(AlertAbove) && condition != string(AlertBelow) {
		return fmt.Errorf("%w: condition must be above or below", ErrInvalidArgument)
	}
	if math.IsNaN(target) || math.IsInf(target, 0) {
		return fmt.Errorf("%w: targetPrice must be a number", ErrInvalidArgument)
	}
	switch kind {
	case AlertKindFunding:
		if target < -5 || target > 5 {
			return fmt.Errorf("%w: funding target is a percent between -5 and 5", ErrInvalidArgument)
		}
	case AlertKindOpenInterest:
		if target <= 0 {
			return fmt.Errorf("%w: open interest target is USDT and must be positive", ErrInvalidArgument)
		}
	case AlertKindTaker:
		if target < 0 || target > 100 {
			return fmt.Errorf("%w: taker target is the 5m buy share percent, 0 to 100", ErrInvalidArgument)
		}
	case AlertKindFearGreed:
		if target < 0 || target > 100 {
			return fmt.Errorf("%w: fear and greed target is a score from 0 to 100", ErrInvalidArgument)
		}
	}
	return nil
}

// FundingAlertPercent is the live funding rate in percent (0.01 = 0.01%).
func FundingAlertPercent(snap *FundingSnapshot) (float64, bool) {
	if snap == nil {
		return 0, false
	}
	if snap.Current != nil {
		if v, ok := parseAlertPercent(snap.Current.RatePct); ok {
			return v, true
		}
	}
	for _, v := range snap.Venues {
		if p, ok := parseAlertPercent(v.Current.RatePct); ok {
			return p, true
		}
	}
	return 0, false
}

// OpenInterestAlertUSDT is the current open-interest notional.
func OpenInterestAlertUSDT(snap *OpenInterestSnapshot) (float64, bool) {
	if snap == nil {
		return 0, false
	}
	if v, ok := parseAlertFloat(snap.Current.Value); ok && v > 0 {
		return v, true
	}
	for _, v := range snap.Venues {
		if n, ok := parseAlertFloat(v.Current.Value); ok && n > 0 {
			return n, true
		}
	}
	return 0, false
}

// TakerAlertBuyPercent is the 5-minute taker buy share, 0–100.
func TakerAlertBuyPercent(rep *TakerFlowReport) (float64, bool) {
	if rep == nil {
		return 0, false
	}
	venue := rep.Combined
	if venue == nil && len(rep.Venues) > 0 {
		venue = &rep.Venues[0]
	}
	if venue == nil {
		return 0, false
	}
	for _, w := range venue.Windows {
		if w.Window != "5m" {
			continue
		}
		if w.BuyNotional+w.SellNotional <= 0 {
			return 0, false
		}
		return w.BuyShare * 100, true
	}
	return 0, false
}

func parseAlertPercent(raw string) (float64, bool) {
	s := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(raw), "%"))
	return parseAlertFloat(s)
}

func parseAlertFloat(raw string) (float64, bool) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}
