package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// AlertKindLiqLevel fires when last price or later prints hit a predicted
	// 10x/25x/50x/100x band that meets a minimum later-print confidence.
	AlertKindLiqLevel AlertKind = "liquidation_level"

	AlertImportanceHigh   = "high"
	AlertImportanceMedium = "medium"
	AlertImportanceLow    = "low"

	// DefaultLiqLevelMinConfidence skips bands that keep missing later prints.
	DefaultLiqLevelMinConfidence = 0.35
	LiqLevelFairMinConfidence    = 0.45
	LiqLevelHighMinConfidence    = 0.80
)

// IsLiqLevelAlert reports kind=liquidation_level.
func IsLiqLevelAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && k == AlertKindLiqLevel
}

// LevelAlertBand is one predicted pocket used for alerts.
type LevelAlertBand struct {
	Leverage    int
	Side        string
	Price       float64
	Lo          float64
	Hi          float64
	Confidence  float64
	Reliability string
	Predicted   float64
}

// LiqLevelAlertDetail is the pocket that crossed the alert.
type LiqLevelAlertDetail struct {
	Exchange      string
	Symbol        string
	Side          string
	Leverage      int
	Price         float64
	LastPrice     float64
	Confidence    float64
	Reliability   string
	Importance    string
	Reason        string // price_in_band | prints_in_band
	Realized      float64
	MinConfidence float64
}

// LiqLevelHit is the predicted pocket a notional wave landed in.
type LiqLevelHit struct {
	Leverage    int
	Side        string
	Price       float64
	Confidence  float64
	Reliability string
	Realized    float64
}

// IsLiquidationAlert is feed, cascade, notional, or level.
func IsLiquidationAlert(k AlertKind) bool {
	return IsLiqFeedAlert(k) || IsLiqCascadeAlert(k) || IsLiqNotionalAlert(k) || IsLiqLevelAlert(k)
}

// LiqLevelMinConfidence is the floor a band must clear to fire.
// targetPrice in (0,1] is an explicit floor; otherwise condition selects it.
func LiqLevelMinConfidence(a PriceAlert) float64 {
	if a.TargetPrice > 0 && a.TargetPrice <= 1 && !math.IsNaN(a.TargetPrice) && !math.IsInf(a.TargetPrice, 0) {
		return a.TargetPrice
	}
	switch strings.ToLower(strings.TrimSpace(string(a.Condition))) {
	case LiquidationLevelsReliabilityHigh:
		return LiqLevelHighMinConfidence
	case LiquidationLevelsReliabilityFair:
		return LiqLevelFairMinConfidence
	default:
		return DefaultLiqLevelMinConfidence
	}
}

// LiqLevelMinReliability is high, fair, or empty (any at/above the floor).
func LiqLevelMinReliability(a PriceAlert) string {
	switch strings.ToLower(strings.TrimSpace(string(a.Condition))) {
	case LiquidationLevelsReliabilityHigh:
		return LiquidationLevelsReliabilityHigh
	case LiquidationLevelsReliabilityFair:
		return LiquidationLevelsReliabilityFair
	default:
		return ""
	}
}

// LevelAlertImportance maps confidence to high / medium / low.
func LevelAlertImportance(confidence float64) string {
	switch {
	case confidence >= 0.75:
		return AlertImportanceHigh
	case confidence >= 0.45:
		return AlertImportanceMedium
	default:
		return AlertImportanceLow
	}
}

// BuildLevelAlertBands places 10x/25x/50x/100x pockets at last and stamps
// later-print confidence from review buckets.
func BuildLevelAlertBands(last float64, buckets []LiquidationLevelsReviewBucket) []LevelAlertBand {
	if last <= 0 {
		return nil
	}
	pred := PredictDisplayLevelBands(last, 1_000_000, 0.5, 0.5, 0)
	table := reviewConfidenceTable(buckets)
	rel := map[[2]int]string{}
	for _, b := range buckets {
		side := 0
		if b.Side == LiquidationSideShort {
			side = 1
		}
		rel[[2]int{b.Leverage, side}] = b.Reliability
	}
	out := make([]LevelAlertBand, 0, len(pred))
	for _, p := range pred {
		sideN := 0
		if p.Side == LiquidationSideShort {
			sideN = 1
		}
		conf := confidenceFor(table, p.Leverage, p.Side)
		r := rel[[2]int{p.Leverage, sideN}]
		if r == "" {
			r = levelReliability(0, 0, 0)
		}
		out = append(out, LevelAlertBand{
			Leverage: p.Leverage, Side: p.Side, Price: p.Price, Lo: p.Lo, Hi: p.Hi,
			Confidence: conf, Reliability: r, Predicted: p.Predicted,
		})
	}
	return out
}

func reliabilityAtLeast(got, min string) bool {
	if min == "" {
		return true
	}
	rank := map[string]int{
		LiquidationLevelsReliabilityLow:     1,
		LiquidationLevelsReliabilityUnknown: 1,
		LiquidationLevelsReliabilityFair:    2,
		LiquidationLevelsReliabilityHigh:    3,
	}
	return rank[got] >= rank[min]
}

// LevelAlertObservation is true when last or later prints sit in a predicted
// band that clears the alert's min confidence / reliability.
func LevelAlertObservation(a PriceAlert, last float64, bands []LevelAlertBand, events []LiquidationEvent, now time.Time) (bool, float64, LiqLevelAlertDetail) {
	minC := LiqLevelMinConfidence(a)
	minRel := LiqLevelMinReliability(a)
	detail := LiqLevelAlertDetail{
		Exchange:      string(a.Exchange),
		Symbol:        a.Symbol,
		LastPrice:     last,
		MinConfidence: minC,
	}
	if last <= 0 || len(bands) == 0 {
		return false, 0, detail
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	from := now.Add(-DefaultLiqNotionalWindow)
	var best LevelAlertBand
	var bestRealized float64
	var bestReason string
	found := false
	for _, b := range bands {
		if b.Confidence+1e-9 < minC {
			continue
		}
		if !reliabilityAtLeast(b.Reliability, minRel) {
			continue
		}
		inPrice := last >= b.Lo && last <= b.Hi
		realized, _ := realizedLiqInBand([][]LiquidationEvent{events}, from, now, b.Lo, b.Hi, b.Side)
		if !inPrice && realized <= 0 {
			continue
		}
		if found && b.Confidence <= best.Confidence {
			continue
		}
		best = b
		bestRealized = realized
		if inPrice {
			bestReason = "price_in_band"
		} else {
			bestReason = "prints_in_band"
		}
		found = true
	}
	if !found {
		return false, 0, detail
	}
	detail.Side = best.Side
	detail.Leverage = best.Leverage
	detail.Price = best.Price
	detail.Confidence = best.Confidence
	detail.Reliability = best.Reliability
	detail.Importance = LevelAlertImportance(best.Confidence)
	detail.Reason = bestReason
	detail.Realized = bestRealized
	return true, best.Confidence, detail
}

// AnnotateNotionalWithLevels tags a dollar-wave alert with the predicted
// pocket the prints landed in. High-confidence pockets are high importance;
// waves that miss the map stay low.
func AnnotateNotionalWithLevels(detail *LiqNotionalAlertDetail, last float64, events []LiquidationEvent, bands []LevelAlertBand, now time.Time) {
	if detail == nil {
		return
	}
	detail.Importance = AlertImportanceLow
	if last <= 0 || len(bands) == 0 || len(events) == 0 {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	win, _, _ := ParseLiqNotionalWindow(0, detail.Window)
	if win <= 0 {
		win = DefaultLiqNotionalWindow
	}
	from := now.Add(-win)
	var best LiqLevelHit
	found := false
	for _, b := range bands {
		if detail.Side != "" && detail.Side != "both" && b.Side != detail.Side {
			continue
		}
		realized, _ := realizedLiqInBand([][]LiquidationEvent{events}, from, now, b.Lo, b.Hi, b.Side)
		if realized <= 0 {
			continue
		}
		if found && b.Confidence <= best.Confidence {
			continue
		}
		best = LiqLevelHit{
			Leverage: b.Leverage, Side: b.Side, Price: b.Price,
			Confidence: b.Confidence, Reliability: b.Reliability, Realized: realized,
		}
		found = true
	}
	if !found {
		return
	}
	detail.Level = &best
	detail.Importance = LevelAlertImportance(best.Confidence)
}

func validateLiqLevelSpec(condition string, target float64) error {
	cond := strings.ToLower(strings.TrimSpace(condition))
	switch cond {
	case "", "any", LiquidationLevelsReliabilityHigh, LiquidationLevelsReliabilityFair:
	default:
		return fmt.Errorf("%w: liquidation_level condition must be any, high, or fair", ErrInvalidArgument)
	}
	if target < 0 || target > 1 || math.IsNaN(target) || math.IsInf(target, 0) {
		return fmt.Errorf("%w: min confidence must be between 0 and 1", ErrInvalidArgument)
	}
	return nil
}
