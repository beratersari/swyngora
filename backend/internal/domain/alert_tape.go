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
// A single-venue snapshot uses that rate. A combined snapshot uses the
// highest venue rate (above). Prefer FundingAlertReading when the condition
// is known.
func FundingAlertPercent(snap *FundingSnapshot) (float64, bool) {
	pct, _, ok := FundingAlertReading(snap, AlertAbove)
	return pct, ok
}

// FundingAlertReading is the funding percent to compare with an alert, and the
// venue that printed it. exchange=all has no single current rate: above uses
// the highest venue, below uses the lowest, so one venue crossing the level
// is enough. A single-venue snapshot uses that venue only.
func FundingAlertReading(snap *FundingSnapshot, cond AlertCondition) (pct float64, venue string, ok bool) {
	if snap == nil {
		return 0, "", false
	}
	if snap.Current != nil {
		if v, parsed := parseAlertPercent(snap.Current.RatePct); parsed {
			return v, fundingAlertVenue(snap, ""), true
		}
	}
	wantMin := cond == AlertBelow
	found := false
	var best float64
	var bestVenue string
	for _, v := range snap.Venues {
		p, parsed := parseAlertPercent(v.Current.RatePct)
		if !parsed {
			continue
		}
		if !found || (wantMin && p < best) || (!wantMin && p > best) {
			found = true
			best = p
			bestVenue = v.Exchange
		}
	}
	if !found {
		return 0, "", false
	}
	return best, fundingAlertVenue(snap, bestVenue), true
}

func fundingAlertVenue(snap *FundingSnapshot, picked string) string {
	if picked != "" {
		return picked
	}
	if snap == nil {
		return ""
	}
	ex := strings.TrimSpace(string(snap.Exchange))
	if ex != "" && !strings.EqualFold(ex, "all") {
		return ex
	}
	if len(snap.Venues) == 1 {
		return snap.Venues[0].Exchange
	}
	return ""
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
