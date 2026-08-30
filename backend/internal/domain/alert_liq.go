package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// AlertKindLiqFeed fires when a liquidation websocket is down or stalled too long.
	AlertKindLiqFeed AlertKind = "liquidation_feed"
	// AlertKindLiqCascade fires when a coin (or the market) is in a cascade.
	AlertKindLiqCascade AlertKind = "liquidation_cascade"
	// AlertKindLiqNotional fires when long/short/total notional in a window
	// crosses a dollar threshold (one fire per wave).
	AlertKindLiqNotional AlertKind = "liquidation_notional"

	// DefaultLiqNotionalWindow is 5 minutes.
	DefaultLiqNotionalWindow = 5 * time.Minute

	// DefaultLiqFeedAlertSeconds is how long a venue may stay unhealthy before fire.
	DefaultLiqFeedAlertSeconds = 300
	MinLiqFeedAlertSeconds     = 30
	MaxLiqFeedAlertSeconds     = 24 * 60 * 60

	// LiqAlertSymbolAll is stored for venue-wide / market-wide liquidation alerts.
	LiqAlertSymbolAll = "ALL"

	// DefaultLiqCascadeAlertGrade is the minimum "big cascade" grade.
	DefaultLiqCascadeAlertGrade = CascadeGradeCascade
)

// IsLiqFeedAlert reports kind=liquidation_feed.
func IsLiqFeedAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && k == AlertKindLiqFeed
}

// IsLiqCascadeAlert reports kind=liquidation_cascade.
func IsLiqCascadeAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && k == AlertKindLiqCascade
}

// IsLiqNotionalAlert reports kind=liquidation_notional.
func IsLiqNotionalAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && k == AlertKindLiqNotional
}

// IsLiquidationAlert is feed, cascade, or notional.
func IsLiquidationAlert(k AlertKind) bool {
	return IsLiqFeedAlert(k) || IsLiqCascadeAlert(k) || IsLiqNotionalAlert(k)
}

// LiqFeedAlertThreshold is the min unhealthy duration (targetPrice as seconds).
func LiqFeedAlertThreshold(a PriceAlert) time.Duration {
	sec := a.TargetPrice
	if sec <= 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		sec = DefaultLiqFeedAlertSeconds
	}
	if sec < MinLiqFeedAlertSeconds {
		sec = MinLiqFeedAlertSeconds
	}
	if sec > MaxLiqFeedAlertSeconds {
		sec = MaxLiqFeedAlertSeconds
	}
	return time.Duration(sec) * time.Second
}

// LiqCascadeAlertMinGrade is cascade (default) or extreme / elevated.
func LiqCascadeAlertMinGrade(a PriceAlert) string {
	switch strings.ToLower(strings.TrimSpace(string(a.Condition))) {
	case CascadeGradeExtreme:
		return CascadeGradeExtreme
	case CascadeGradeElevated:
		return CascadeGradeElevated
	default:
		return DefaultLiqCascadeAlertGrade
	}
}

// CascadeGradeAtLeast is true when grade is at least min (quiet < elevated < cascade < extreme).
func CascadeGradeAtLeast(grade, min string) bool {
	return CascadeGradeRank(grade) >= CascadeGradeRank(min)
}

// LiqFeedAlertDetail is which venues failed and for how long.
type LiqFeedAlertDetail struct {
	Exchange         string
	UnhealthySeconds float64
	ThresholdSeconds float64
	Missing          []string
	LastSeenAt       time.Time
	LastEventAt      time.Time
	Live             bool
	Venues           []LiquidationVenueHealth
}

// LiqCascadeAlertDetail is the coin and venue that crossed the min grade.
type LiqCascadeAlertDetail struct {
	Exchange string
	Symbol   string
	Grade    string
	Side     string
	Score    float64
	Hottest  string
	Summary  string
	Both     bool
}

// FeedAlertObservation is whether the liquidation feed has been unhealthy long enough.
// A live feed is healthy even if an old unfilled history hole remains.
func FeedAlertObservation(a PriceAlert, feed LiquidationFeed, now time.Time) (bool, float64, LiqFeedAlertDetail) {
	thr := LiqFeedAlertThreshold(a)
	detail := LiqFeedAlertDetail{
		Exchange:         string(a.Exchange),
		ThresholdSeconds: thr.Seconds(),
		Missing:          []string{},
		Venues:           []LiquidationVenueHealth{},
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	want := liquidationVenues()
	if ex := strings.ToLower(strings.TrimSpace(string(a.Exchange))); ex != "" && ex != "all" {
		want = []Exchange{Exchange(ex)}
	}
	byEx := map[string]LiquidationVenueHealth{}
	for _, v := range feed.Venues {
		byEx[strings.ToLower(v.Exchange)] = v
	}
	var worst float64
	for _, ex := range want {
		v, ok := byEx[string(ex)]
		if !ok {
			v = LiquidationVenueHealth{Exchange: string(ex), Live: false}
		}
		d := venueUnhealthyFor(v, now, a.CreatedAt)
		if d < thr {
			continue
		}
		sec := d.Seconds()
		if sec > worst {
			worst = sec
			detail.LastSeenAt = v.LastSeenAt
			detail.LastEventAt = v.LastEventAt
			detail.Live = v.Live
		}
		detail.Missing = append(detail.Missing, string(ex))
		detail.Venues = append(detail.Venues, v)
	}
	if len(detail.Missing) == 0 {
		return false, 0, detail
	}
	if len(detail.Missing) == 1 {
		detail.Exchange = detail.Missing[0]
	}
	detail.UnhealthySeconds = worst
	return true, worst, detail
}

func venueUnhealthyFor(v LiquidationVenueHealth, now, created time.Time) time.Duration {
	if v.Live {
		return 0
	}
	var d time.Duration
	if !v.LastSeenAt.IsZero() && now.After(v.LastSeenAt) {
		d = now.Sub(v.LastSeenAt)
	} else if !created.IsZero() && now.After(created) {
		d = now.Sub(created)
	}
	for _, g := range v.Gaps {
		if !g.To.IsZero() {
			continue
		}
		if !g.From.IsZero() && now.After(g.From) {
			if age := now.Sub(g.From); age > d {
				d = age
			}
		}
	}
	if d < 0 {
		return 0
	}
	return d
}

// CascadeAlertObservation checks one coin report (not a market scan).
func CascadeAlertObservation(a PriceAlert, rep *CascadeReport) (bool, float64, LiqCascadeAlertDetail) {
	detail := LiqCascadeAlertDetail{
		Exchange: string(a.Exchange),
		Symbol:   a.Symbol,
	}
	if rep == nil {
		return false, 0, detail
	}
	min := LiqCascadeAlertMinGrade(a)
	want := strings.ToLower(strings.TrimSpace(string(a.Exchange)))
	var best *CascadeVenue
	for i := range rep.Venues {
		v := &rep.Venues[i]
		if want != "" && want != "all" && string(v.Exchange) != want {
			continue
		}
		if !CascadeGradeAtLeast(v.Grade, min) {
			continue
		}
		if best == nil || v.Score > best.Score {
			best = v
		}
	}
	if best == nil {
		return false, 0, detail
	}
	detail.Exchange = string(best.Exchange)
	if best.Symbol != "" {
		detail.Symbol = best.Symbol
	} else if rep.Symbol != "" && !strings.EqualFold(rep.Symbol, "all") {
		detail.Symbol = rep.Symbol
	}
	detail.Grade = best.Grade
	detail.Side = best.Side
	detail.Score = best.Score
	detail.Hottest = best.Hottest
	detail.Summary = best.Summary
	if rep.Both != nil && rep.Both.Agree && CascadeGradeAtLeast(rep.Both.Grade, min) {
		detail.Both = true
	}
	return true, best.Score, detail
}

// CascadeScanAlertObservation checks a market scan (symbol=all).
func CascadeScanAlertObservation(a PriceAlert, scan *CascadeScan) (bool, float64, LiqCascadeAlertDetail) {
	if scan == nil {
		return false, 0, LiqCascadeAlertDetail{Exchange: string(a.Exchange), Symbol: LiqAlertSymbolAll}
	}
	min := LiqCascadeAlertMinGrade(a)
	var best *CascadeHit
	for i := range scan.Hits {
		h := &scan.Hits[i]
		if !CascadeGradeAtLeast(h.Grade, min) {
			continue
		}
		if best == nil || h.Score > best.Score {
			best = h
		}
	}
	if best == nil {
		// Market-wide pooled read.
		return CascadeAlertObservation(a, &scan.Market)
	}
	detail := LiqCascadeAlertDetail{
		Exchange: string(a.Exchange),
		Symbol:   best.Symbol,
		Grade:    best.Grade,
		Side:     best.Side,
		Score:    best.Score,
		Hottest:  best.Hottest,
		Summary:  best.Summary,
		Both:     best.Both,
	}
	if detail.Exchange == "" || detail.Exchange == "all" {
		// Hit does not name a venue; leave all unless both is set.
		if best.Both {
			detail.Exchange = "all"
		}
	}
	return true, best.Score, detail
}

// LiqNotionalWindow is the lookback for a notional alert (RangePct = minutes).
func LiqNotionalWindow(a PriceAlert) time.Duration {
	d, _, _ := ParseLiqNotionalWindow(a.RangePct, "")
	return d
}

// ParseLiqNotionalWindow accepts 1m/5m/15m/1h, or RangePct as minutes (1, 5, 15, 60).
func ParseLiqNotionalWindow(rangePct float64, window string) (time.Duration, string, error) {
	w := strings.ToLower(strings.TrimSpace(window))
	if w != "" {
		switch w {
		case "1m", "1min", "1":
			return time.Minute, "1m", nil
		case "5m", "5min", "5":
			return 5 * time.Minute, "5m", nil
		case "15m", "15min", "15":
			return 15 * time.Minute, "15m", nil
		case "1h", "60m", "60":
			return time.Hour, "1h", nil
		default:
			return 0, "", fmt.Errorf("%w: window must be 1m, 5m, 15m, or 1h", ErrInvalidArgument)
		}
	}
	switch {
	case rangePct == 0:
		return DefaultLiqNotionalWindow, "5m", nil
	case rangePct == 1:
		return time.Minute, "1m", nil
	case rangePct == 5:
		return 5 * time.Minute, "5m", nil
	case rangePct == 15:
		return 15 * time.Minute, "15m", nil
	case rangePct == 60:
		return time.Hour, "1h", nil
	default:
		return 0, "", fmt.Errorf("%w: window must be 1m, 5m, 15m, or 1h", ErrInvalidArgument)
	}
}

// LiqNotionalSide is long, short, or both (total).
func LiqNotionalSide(a PriceAlert) string {
	switch strings.ToLower(strings.TrimSpace(string(a.Condition))) {
	case LiquidationSideLong:
		return LiquidationSideLong
	case LiquidationSideShort:
		return LiquidationSideShort
	default:
		return "both"
	}
}

// LiqNotionalAlertDetail is the window total that crossed the dollar line.
type LiqNotionalAlertDetail struct {
	Exchange      string
	Symbol        string
	Side          string
	Window        string
	Notional      float64
	LongNotional  float64
	ShortNotional float64
	Threshold     float64
	Count         int
}

// NotionalAlertObservation sums prints in the lookback. Met when the chosen
// side (long, short, or both=total) is at or above targetPrice USDT.
// exchange=all sums Binance + Bybit and never substitutes a missing venue.
func NotionalAlertObservation(a PriceAlert, events []LiquidationEvent, now time.Time) (bool, float64, LiqNotionalAlertDetail) {
	win, winID, _ := ParseLiqNotionalWindow(a.RangePct, "")
	if win <= 0 {
		win = DefaultLiqNotionalWindow
		winID = "5m"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	cut := now.Add(-win)
	side := LiqNotionalSide(a)
	wantEx := strings.ToLower(strings.TrimSpace(string(a.Exchange)))
	detail := LiqNotionalAlertDetail{
		Exchange:  string(a.Exchange),
		Symbol:    a.Symbol,
		Side:      side,
		Window:    winID,
		Threshold: a.TargetPrice,
	}
	var longN, shortN float64
	for _, e := range events {
		if e.Notional <= 0 || e.Time.IsZero() || e.Time.Before(cut) {
			continue
		}
		if wantEx != "" && wantEx != "all" && string(e.Exchange) != wantEx {
			continue
		}
		switch e.Side {
		case LiquidationSideLong:
			longN += e.Notional
		case LiquidationSideShort:
			shortN += e.Notional
		default:
			continue
		}
		detail.Count++
	}
	detail.LongNotional = longN
	detail.ShortNotional = shortN
	switch side {
	case LiquidationSideLong:
		detail.Notional = longN
	case LiquidationSideShort:
		detail.Notional = shortN
	default:
		detail.Notional = longN + shortN
	}
	if a.TargetPrice <= 0 {
		return false, detail.Notional, detail
	}
	return detail.Notional+1e-9 >= a.TargetPrice, detail.Notional, detail
}

func validateLiqAlertSpec(kind AlertKind, condition string, target, rangePct float64) error {
	cond := strings.ToLower(strings.TrimSpace(condition))
	switch kind {
	case AlertKindLiqFeed:
		if cond != "" && cond != "down" && cond != "gap" {
			return fmt.Errorf("%w: liquidation_feed condition must be empty, down, or gap", ErrInvalidArgument)
		}
		if target < 0 || math.IsNaN(target) || math.IsInf(target, 0) {
			return fmt.Errorf("%w: min down seconds must be >= 0", ErrInvalidArgument)
		}
		if target > 0 && target < MinLiqFeedAlertSeconds {
			return fmt.Errorf("%w: min down seconds must be at least %d", ErrInvalidArgument, MinLiqFeedAlertSeconds)
		}
		if target > MaxLiqFeedAlertSeconds {
			return fmt.Errorf("%w: min down seconds must be at most %d", ErrInvalidArgument, MaxLiqFeedAlertSeconds)
		}
	case AlertKindLiqCascade:
		switch cond {
		case "", CascadeGradeCascade, CascadeGradeExtreme, CascadeGradeElevated:
		default:
			return fmt.Errorf("%w: liquidation_cascade condition must be cascade, extreme, or elevated", ErrInvalidArgument)
		}
		if target != 0 && (math.IsNaN(target) || math.IsInf(target, 0)) {
			return fmt.Errorf("%w: unused targetPrice must be 0 for liquidation_cascade", ErrInvalidArgument)
		}
	case AlertKindLiqNotional:
		switch cond {
		case "", "both", "total", LiquidationSideLong, LiquidationSideShort:
		default:
			return fmt.Errorf("%w: liquidation_notional condition must be long, short, or both", ErrInvalidArgument)
		}
		if target <= 0 || math.IsNaN(target) || math.IsInf(target, 0) {
			return fmt.Errorf("%w: notional threshold must be a positive USDT amount", ErrInvalidArgument)
		}
		if _, _, err := ParseLiqNotionalWindow(rangePct, ""); err != nil {
			return err
		}
	}
	return nil
}
