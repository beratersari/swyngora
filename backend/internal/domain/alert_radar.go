package domain

import (
	"fmt"
	"math"
	"strings"
)

const (
	// AlertKindSetupRadar fires when a coin's current tape looks like
	// the setup before that coin's own important past moves.
	AlertKindSetupRadar AlertKind = "setup_radar"

	// AlertRadarAny / AlertRadarUp / AlertRadarDown filter by after-path lean.
	AlertRadarAny  AlertCondition = "any"
	AlertRadarUp   AlertCondition = "up"
	AlertRadarDown AlertCondition = "down"

	// DefaultSetupRadarMinSimilarity is used when targetPrice is 0.
	DefaultSetupRadarMinSimilarity = 60.0
	MinSetupRadarSimilarity        = 40.0
	MaxSetupRadarSimilarity        = 100.0
)

// IsSetupRadarAlert reports kind=setup_radar.
func IsSetupRadarAlert(k AlertKind) bool {
	k, ok := NormalizeAlertKind(string(k))
	return ok && k == AlertKindSetupRadar
}

// SetupRadarMinSimilarity is the closeness floor (targetPrice, 0–100).
func SetupRadarMinSimilarity(a PriceAlert) float64 {
	v := a.TargetPrice
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		v = DefaultSetupRadarMinSimilarity
	}
	if v < MinSetupRadarSimilarity {
		v = MinSetupRadarSimilarity
	}
	if v > MaxSetupRadarSimilarity {
		v = MaxSetupRadarSimilarity
	}
	return v
}

// SetupRadarLean is any (default), up, or down.
func SetupRadarLean(a PriceAlert) string {
	switch strings.ToLower(strings.TrimSpace(string(a.Condition))) {
	case CVDDirUp:
		return CVDDirUp
	case CVDDirDown:
		return CVDDirDown
	default:
		return string(AlertRadarAny)
	}
}

// SetupRadarAlertDetail is the coin (or hottest scan hit) that crossed the floor.
type SetupRadarAlertDetail struct {
	Exchange   string
	Symbol     string
	Similarity float64
	Threshold  float64
	Lean       string
	Matches    int
	AfterStory string
	WhySimilar string
	Summary    string
	Symbols    []string
	Compared   []AroundSimilarFieldScore
}

func setupRadarHitPasses(a PriceAlert, h ForensicsScanHit) bool {
	if h.Similarity+1e-9 < SetupRadarMinSimilarity(a) {
		return false
	}
	want := SetupRadarLean(a)
	if want == string(AlertRadarAny) {
		return true
	}
	return h.Lean == want
}

func setupRadarDetailFromHit(a PriceAlert, h ForensicsScanHit) SetupRadarAlertDetail {
	d := SetupRadarAlertDetail{
		Exchange:   h.Exchange,
		Symbol:     h.Symbol,
		Similarity: h.Similarity,
		Threshold:  SetupRadarMinSimilarity(a),
		Lean:       h.Lean,
		Matches:    h.Matches,
		AfterStory: h.AfterStory,
		WhySimilar: h.WhySimilar,
		Summary:    h.Summary,
		Compared:   append([]AroundSimilarFieldScore(nil), h.Compared...),
	}
	if d.Exchange == "" {
		d.Exchange = string(a.Exchange)
	}
	if d.Symbol == "" {
		d.Symbol = a.Symbol
	}
	if d.Symbol != "" {
		d.Symbols = []string{d.Symbol}
	}
	return d
}

// SetupRadarObservationFromSimilar checks one coin's current-vs-past compare.
func SetupRadarObservationFromSimilar(a PriceAlert, rep *AroundSimilarReport) (bool, float64, SetupRadarAlertDetail) {
	empty := SetupRadarAlertDetail{
		Exchange:  string(a.Exchange),
		Symbol:    a.Symbol,
		Threshold: SetupRadarMinSimilarity(a),
	}
	hit, ok := ForensicsScanHitFromSimilar(rep)
	if !ok {
		return false, 0, empty
	}
	detail := setupRadarDetailFromHit(a, hit)
	if !setupRadarHitPasses(a, hit) {
		return false, hit.Similarity, detail
	}
	return true, hit.Similarity, detail
}

// SetupRadarObservationFromScan checks a multi-coin Now scan (symbol=all).
// Met when any hit passes the similarity floor and lean filter. New coins
// while the radar is still hot do not change met — the edge machine stays quiet
// until zero hits pass.
func SetupRadarObservationFromScan(a PriceAlert, scan *ForensicsScanReport) (bool, float64, SetupRadarAlertDetail) {
	detail := SetupRadarAlertDetail{
		Exchange:  string(a.Exchange),
		Symbol:    LiqAlertSymbolAll,
		Threshold: SetupRadarMinSimilarity(a),
	}
	if scan == nil {
		return false, 0, detail
	}
	var best *ForensicsScanHit
	for i := range scan.Hits {
		h := &scan.Hits[i]
		if !setupRadarHitPasses(a, *h) {
			continue
		}
		if h.Symbol != "" {
			detail.Symbols = append(detail.Symbols, h.Symbol)
		}
		if best == nil || h.Similarity > best.Similarity {
			best = h
		}
	}
	if best == nil {
		return false, 0, detail
	}
	detail.Exchange = best.Exchange
	if detail.Exchange == "" {
		detail.Exchange = scan.Exchange
	}
	if detail.Exchange == "" {
		detail.Exchange = string(a.Exchange)
	}
	detail.Symbol = best.Symbol
	detail.Similarity = best.Similarity
	detail.Lean = best.Lean
	detail.Matches = best.Matches
	detail.AfterStory = best.AfterStory
	detail.Summary = best.Summary
	return true, best.Similarity, detail
}

func validateRadarAlertSpec(condition string, target float64) error {
	cond := strings.ToLower(strings.TrimSpace(condition))
	switch cond {
	case "", string(AlertRadarAny), CVDDirUp, CVDDirDown:
	default:
		return fmt.Errorf("%w: setup_radar condition must be any, up, or down", ErrInvalidArgument)
	}
	if target < 0 || math.IsNaN(target) || math.IsInf(target, 0) {
		return fmt.Errorf("%w: min similarity must be >= 0", ErrInvalidArgument)
	}
	if target > 0 && target < MinSetupRadarSimilarity {
		return fmt.Errorf("%w: min similarity must be at least %.0f", ErrInvalidArgument, MinSetupRadarSimilarity)
	}
	if target > MaxSetupRadarSimilarity {
		return fmt.Errorf("%w: min similarity must be at most %.0f", ErrInvalidArgument, MaxSetupRadarSimilarity)
	}
	return nil
}
