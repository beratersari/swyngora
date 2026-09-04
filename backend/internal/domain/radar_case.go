package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	DefaultRadarCooldown   = 4 * time.Hour
	DefaultRadarSince      = "7d"
	DefaultRadarListLimit  = 50
	MaxRadarListLimit      = 200
	RadarFlatReturnPct     = 0.05
	RadarCaseDisclaimer    = "Saved Setup Radar hits and the price path after each one. Usefulness is how often an up/down lean matched the later move. Informational only — not financial advice."
)

// RadarAfter is what price did after a saved setup, for one horizon.
type RadarAfter struct {
	Horizon   string
	DueAt     time.Time
	FilledAt  *time.Time
	ReturnPct float64
	Direction string // up | down | flat
	Price     float64
	Useful    *bool // lean matched direction when lean is up/down
}

// RadarCase is one Setup Radar hit kept so later moves can be measured.
type RadarCase struct {
	ID         string
	Exchange   string
	Symbol     string
	FoundAt    time.Time
	FoundPrice float64
	Similarity float64
	Coverage   float64
	Lean       string
	WhySimilar string
	AfterStory string
	Summary    string
	Lookback   string
	Window     string
	Compared   []AroundSimilarFieldScore
	After      []RadarAfter
}

// RadarHorizonStat is how useful saved setups were over one after-window.
type RadarHorizonStat struct {
	Horizon    string
	Sample     int
	LeanRight  int
	LeanWrong  int
	Up         int
	Down       int
	Flat       int
	MedianPct  float64
	AveragePct float64
	UsefulPct  float64
}

// RadarReview is saved Setup Radar hits plus how useful they were.
type RadarReview struct {
	Exchange string
	Symbol   string
	Since    string
	From     time.Time
	Limit    int
	AsOf     time.Time
	Cases    []RadarCase
	Horizons []RadarHorizonStat
	Summary  string
	Note     string
}

// RadarListQuery filters saved cases (newest first).
type RadarListQuery struct {
	Exchange string
	Symbol   string
	Since    time.Time
	Limit    int
}

// RadarCasePort persists Setup Radar hits and after-fills.
type RadarCasePort interface {
	InsertIfQuiet(ctx context.Context, c RadarCase, cooldown time.Duration) (*RadarCase, bool, error)
	List(ctx context.Context, q RadarListQuery) ([]RadarCase, error)
	ListDueAfters(ctx context.Context, now time.Time, limit int) ([]RadarAfterDue, error)
	FillAfter(ctx context.Context, caseID, horizon string, fill RadarAfter) error
	SetFoundPrice(ctx context.Context, caseID string, price float64) error
}

// RadarAfterDue is an unfilled horizon whose due time has passed.
type RadarAfterDue struct {
	CaseID     string
	Exchange   string
	Symbol     string
	FoundAt    time.Time
	FoundPrice float64
	Lean       string
	Horizon    string
	DueAt      time.Time
}

// NewRadarCase snapshots a live scan hit at foundAt.
func NewRadarCase(hit ForensicsScanHit, foundAt time.Time, price float64, lookback, window string) RadarCase {
	if foundAt.IsZero() {
		foundAt = time.Now().UTC()
	} else {
		foundAt = foundAt.UTC()
	}
	ex := strings.ToLower(strings.TrimSpace(hit.Exchange))
	if ex == "" {
		ex = "all"
	}
	c := RadarCase{
		Exchange:   ex,
		Symbol:     strings.ToUpper(strings.TrimSpace(hit.Symbol)),
		FoundAt:    foundAt,
		FoundPrice: price,
		Similarity: hit.Similarity,
		Coverage:   hit.Coverage,
		Lean:       hit.Lean,
		WhySimilar: hit.WhySimilar,
		AfterStory: hit.AfterStory,
		Summary:    hit.Summary,
		Lookback:   lookback,
		Window:     window,
		Compared:   append([]AroundSimilarFieldScore(nil), hit.Compared...),
	}
	for _, h := range DefaultAroundSimilarHorizons() {
		c.After = append(c.After, RadarAfter{
			Horizon: h.ID,
			DueAt:   foundAt.Add(h.Dur),
		})
	}
	return c
}

// ClampRadarListLimit keeps review pages bounded.
func ClampRadarListLimit(n int) int {
	if n <= 0 {
		return DefaultRadarListLimit
	}
	if n > MaxRadarListLimit {
		return MaxRadarListLimit
	}
	return n
}

// ParseRadarSince accepts 24h, 7d, 30d, or RFC3339. Empty is 7d.
func ParseRadarSince(raw string, now time.Time) (time.Time, string, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultRadarSince
	}
	switch s {
	case "24h", "1d":
		return now.Add(-24 * time.Hour), "24h", nil
	case "7d":
		return now.Add(-7 * 24 * time.Hour), "7d", nil
	case "30d":
		return now.Add(-30 * 24 * time.Hour), "30d", nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	}
	if err != nil {
		return time.Time{}, "", fmt.Errorf("%w: since must be 24h, 7d, 30d, or RFC3339", ErrInvalidArgument)
	}
	return t.UTC(), t.UTC().Format(time.RFC3339), nil
}

// RadarAfterDirection is up / down / flat from a percent return.
func RadarAfterDirection(ret float64) string {
	if math.IsNaN(ret) || math.IsInf(ret, 0) {
		return "flat"
	}
	if ret > RadarFlatReturnPct {
		return CVDDirUp
	}
	if ret < -RadarFlatReturnPct {
		return CVDDirDown
	}
	return "flat"
}

// FillRadarAfter computes return and whether the lean was useful.
func FillRadarAfter(foundPrice, afterPrice float64, lean string, now time.Time) RadarAfter {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	var ret float64
	if foundPrice > 0 && afterPrice > 0 {
		ret = (afterPrice - foundPrice) / foundPrice * 100
	}
	dir := RadarAfterDirection(ret)
	out := RadarAfter{
		FilledAt:  &now,
		ReturnPct: ret,
		Direction: dir,
		Price:     afterPrice,
	}
	if scored, hit := RadarLeanUseful(lean, dir); scored {
		out.Useful = &hit
	}
	return out
}

// RadarLeanUseful is scored only for up/down leans. Flat is a miss.
func RadarLeanUseful(lean, dir string) (scored, hit bool) {
	switch lean {
	case CVDDirUp, CVDDirDown:
		return true, dir == lean
	default:
		return false, false
	}
}

// AfterPriceFromCandles is the last close at or before due.
func AfterPriceFromCandles(bars []Candle, due time.Time) (float64, bool) {
	if due.IsZero() {
		return 0, false
	}
	due = due.UTC()
	var best float64
	ok := false
	for _, c := range bars {
		t := c.CloseTime
		if t.IsZero() {
			t = c.OpenTime
		}
		if t.IsZero() || t.After(due.Add(time.Minute)) {
			continue
		}
		px, err := strconvParseFloat(c.Close)
		if err != nil || px <= 0 {
			continue
		}
		best = px
		ok = true
	}
	return best, ok
}

func strconvParseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	if err != nil || f <= 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("bad price")
	}
	return f, nil
}

// FinishRadarReview ranks cases (already newest-first from store) and writes stats.
func FinishRadarReview(r *RadarReview) {
	if r == nil {
		return
	}
	if r.Note == "" {
		r.Note = RadarCaseDisclaimer
	}
	if r.Cases == nil {
		r.Cases = []RadarCase{}
	}
	r.Horizons = SummarizeRadarHorizons(r.Cases)
	r.Summary = ExplainRadarReview(*r)
}

// SummarizeRadarHorizons folds filled after-windows into usefulness stats.
func SummarizeRadarHorizons(cases []RadarCase) []RadarHorizonStat {
	order := DefaultAroundSimilarHorizons()
	by := map[string]*RadarHorizonStat{}
	rets := map[string][]float64{}
	for _, h := range order {
		by[h.ID] = &RadarHorizonStat{Horizon: h.ID}
	}
	for _, c := range cases {
		for _, a := range c.After {
			if a.FilledAt == nil {
				continue
			}
			st := by[a.Horizon]
			if st == nil {
				st = &RadarHorizonStat{Horizon: a.Horizon}
				by[a.Horizon] = st
			}
			st.Sample++
			rets[a.Horizon] = append(rets[a.Horizon], a.ReturnPct)
			switch a.Direction {
			case CVDDirUp:
				st.Up++
			case CVDDirDown:
				st.Down++
			default:
				st.Flat++
			}
			if scored, hit := RadarLeanUseful(c.Lean, a.Direction); scored {
				if hit {
					st.LeanRight++
				} else {
					st.LeanWrong++
				}
			}
		}
	}
	out := make([]RadarHorizonStat, 0, len(order))
	for _, h := range order {
		st := by[h.ID]
		if st == nil {
			continue
		}
		vals := rets[h.ID]
		if n := len(vals); n > 0 {
			sum := 0.0
			for _, v := range vals {
				sum += v
			}
			st.AveragePct = sum / float64(n)
			st.MedianPct = medianFloat(vals)
		}
		if tot := st.LeanRight + st.LeanWrong; tot > 0 {
			st.UsefulPct = 100 * float64(st.LeanRight) / float64(tot)
		}
		out = append(out, *st)
	}
	return out
}

// ExplainRadarReview says how useful saved setups were.
func ExplainRadarReview(r RadarReview) string {
	if len(r.Cases) == 0 {
		return "No saved Setup Radar hits in this window yet. Open Now or wait for the background scan."
	}
	head := fmt.Sprintf("Saved %d setup(s)", len(r.Cases))
	if r.Since != "" {
		head += " over " + r.Since
	}
	head += "."
	var bits []string
	for _, h := range r.Horizons {
		tot := h.LeanRight + h.LeanWrong
		if h.Sample == 0 {
			continue
		}
		if tot == 0 {
			bits = append(bits, fmt.Sprintf("after %s: %d filled, no up/down lean to score", h.Horizon, h.Sample))
			continue
		}
		bits = append(bits, fmt.Sprintf("after %s: %d of %d directional calls matched (%.0f%%), median %s%%",
			h.Horizon, h.LeanRight, tot, h.UsefulPct, FormatSignedPct(h.MedianPct)))
	}
	if len(bits) == 0 {
		return head + " After-windows are still filling."
	}
	return head + " " + strings.Join(bits, ". ") + "."
}
