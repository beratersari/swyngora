package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	AroundMoveGradeNotable = "notable"
	AroundMoveGradeStrong  = "strong"
	AroundMoveGradeExtreme = "extreme"

	DefaultAroundMovesLookback = "24h"
	DefaultAroundMovesInterval = "15m"
	DefaultAroundMovesLimit    = 8
	MaxAroundMovesLimit        = 15
	MaxAroundMovesLookback     = 7 * 24 * time.Hour
	aroundMoveFlatPct          = 0.15
)

// AroundMoveLookbacks are accepted history windows for finding moves.
var AroundMoveLookbacks = []struct {
	ID  string
	Dur time.Duration
}{
	{"4h", 4 * time.Hour},
	{"12h", 12 * time.Hour},
	{"24h", 24 * time.Hour},
	{"3d", 3 * 24 * time.Hour},
	{"7d", 7 * 24 * time.Hour},
}

// AroundMove is one strong up or down stretch in the tape.
type AroundMove struct {
	At        time.Time
	Until     time.Time
	Direction string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	ReturnPct float64
	Volume    float64
	Bars      int
	Grade     string
	During    string
	Title     string
	Summary   string
}

// AroundMoveHit is a detected move plus the around-the-move tape.
type AroundMoveHit struct {
	AroundMove
	Around *AroundReport
}

// AroundMovesReport is the API result.
type AroundMovesReport struct {
	Symbol       string
	Exchange     string
	Lookback     string
	Interval     string
	Direction    string
	MinReturnPct float64
	From         time.Time
	To           time.Time
	AsOf         time.Time
	Moves        []AroundMoveHit
	Summary      string
	Note         string
}

// ParseAroundLookback accepts 4h / 12h / 24h / 3d / 7d (empty = 24h).
func ParseAroundLookback(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultAroundMovesLookback
	}
	for _, w := range AroundMoveLookbacks {
		if w.ID == s {
			return w.ID, w.Dur, nil
		}
	}
	return "", 0, fmt.Errorf("%w: lookback must be 4h, 12h, 24h, 3d, or 7d", ErrInvalidArgument)
}

// ParseAroundMovesInterval accepts 15m or 1h (empty = 15m).
func ParseAroundMovesInterval(raw string) (string, time.Duration, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		s = DefaultAroundMovesInterval
	}
	switch s {
	case "15m":
		return s, 15 * time.Minute, nil
	case "1h":
		return s, time.Hour, nil
	default:
		return "", 0, fmt.Errorf("%w: interval must be 15m or 1h", ErrInvalidArgument)
	}
}

// ParseAroundMovesDirection accepts up / down / both (empty = both).
func ParseAroundMovesDirection(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return string(PumpDirectionBoth), nil
	}
	switch PumpDirection(s) {
	case PumpDirectionUp, PumpDirectionDown, PumpDirectionBoth:
		return s, nil
	default:
		return "", fmt.Errorf("%w: direction must be up, down, or both", ErrInvalidArgument)
	}
}

// ClampAroundMovesLimit bounds how many moves we keep.
func ClampAroundMovesLimit(n int) int {
	if n <= 0 {
		return DefaultAroundMovesLimit
	}
	if n > MaxAroundMovesLimit {
		return MaxAroundMovesLimit
	}
	return n
}

// DefaultAroundMovesMinPct is the floor when the caller does not set one.
func DefaultAroundMovesMinPct(interval string) float64 {
	if interval == "1h" {
		return 2.5
	}
	return 1.5
}

// AroundDuringFor maps a move length onto the around during window.
func AroundDuringFor(d time.Duration) string {
	switch {
	case d <= 7*time.Minute:
		return AroundDuring5m
	case d <= 22*time.Minute:
		return AroundDuring15m
	case d <= 45*time.Minute:
		return AroundDuring30m
	default:
		return AroundDuring1h
	}
}

// FindImportantMoves walks same-direction legs and keeps the strongest ones.
func FindImportantMoves(bars []AroundBar, minPct float64, direction string, limit int) []AroundMove {
	if minPct <= 0 {
		minPct = DefaultAroundMovesMinPct("15m")
	}
	dir, err := ParseAroundMovesDirection(direction)
	if err != nil {
		dir = string(PumpDirectionBoth)
	}
	limit = ClampAroundMovesLimit(limit)
	if len(bars) == 0 {
		return nil
	}
	sorted := append([]AroundBar(nil), bars...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Time.Before(sorted[j].Time) })

	legs := splitAroundMoveLegs(sorted)
	out := make([]AroundMove, 0, len(legs))
	for _, leg := range legs {
		if !aroundMovePasses(leg.ReturnPct, minPct, dir) {
			continue
		}
		annotateAroundMove(&leg)
		out = append(out, leg)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := absFloat(out[i].ReturnPct), absFloat(out[j].ReturnPct)
		if ai == aj {
			return out[i].At.After(out[j].At)
		}
		return ai > aj
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ExplainAroundMovesReport writes a ranked one-liner.
func ExplainAroundMovesReport(r AroundMovesReport) string {
	name := prettyBase(r.Symbol)
	if len(r.Moves) == 0 {
		return fmt.Sprintf("%s: no %s move of at least %s%% in the last %s.",
			name, aroundMovesDirWord(r.Direction), formatFixed(r.MinReturnPct, 1), r.Lookback)
	}
	top := r.Moves[0]
	head := fmt.Sprintf("%s %s: %d important move(s). Largest %s%% at %s",
		name, r.Lookback, len(r.Moves), FormatSignedPct(top.ReturnPct), top.At.UTC().Format("15:04"))
	if top.Around != nil && top.Around.Summary != "" {
		head += ". " + top.Around.Summary
		return head
	}
	return head + "."
}

func splitAroundMoveLegs(bars []AroundBar) []AroundMove {
	var out []AroundMove
	var cur []AroundBar
	var curSign int
	flush := func() {
		if len(cur) == 0 {
			return
		}
		if m, ok := aroundMoveFromBars(cur); ok {
			out = append(out, m)
		}
		cur = nil
		curSign = 0
	}
	var prevClose float64
	for _, b := range bars {
		if b.Close <= 0 && b.Open <= 0 {
			continue
		}
		ret := 0.0
		if prevClose > 0 && b.Close > 0 {
			ret = (b.Close - prevClose) / prevClose * 100
		} else if b.Open > 0 && b.Close > 0 {
			ret = (b.Close - b.Open) / b.Open * 100
		}
		if b.Close > 0 {
			prevClose = b.Close
		}
		sign := 0
		switch {
		case ret > aroundMoveFlatPct:
			sign = 1
		case ret < -aroundMoveFlatPct:
			sign = -1
		}
		if sign == 0 {
			if len(cur) > 0 {
				cur = append(cur, b)
			}
			continue
		}
		if len(cur) == 0 || sign == curSign {
			if len(cur) == 0 {
				curSign = sign
			}
			cur = append(cur, b)
			continue
		}
		flush()
		curSign = sign
		cur = []AroundBar{b}
	}
	flush()
	return out
}

func aroundMoveFromBars(bars []AroundBar) (AroundMove, bool) {
	if len(bars) == 0 {
		return AroundMove{}, false
	}
	open := bars[0].Open
	if open <= 0 {
		open = bars[0].Close
	}
	closePx := bars[len(bars)-1].Close
	if closePx <= 0 {
		closePx = bars[len(bars)-1].Open
	}
	if open <= 0 || closePx <= 0 {
		return AroundMove{}, false
	}
	out := AroundMove{
		At: bars[0].Time.UTC(), Until: bars[len(bars)-1].Time.UTC(),
		Open: open, Close: closePx, Bars: len(bars),
	}
	for _, b := range bars {
		if b.High > out.High {
			out.High = b.High
		}
		if out.Low == 0 || (b.Low > 0 && b.Low < out.Low) {
			out.Low = b.Low
		}
		out.Volume += b.Volume
	}
	out.ReturnPct = (closePx - open) / open * 100
	out.Direction = CVDDirFlat
	if out.ReturnPct > aroundMoveFlatPct {
		out.Direction = CVDDirUp
	} else if out.ReturnPct < -aroundMoveFlatPct {
		out.Direction = CVDDirDown
	} else {
		return AroundMove{}, false
	}
	span := out.Until.Sub(out.At)
	if span < 0 {
		span = 0
	}
	// Include the last bar's width so a single 15m print is 15m, not 0.
	if guess := aroundMoveBarStep(bars); guess > 0 {
		span += guess
	}
	out.During = AroundDuringFor(span)
	return out, true
}

func aroundMoveBarStep(bars []AroundBar) time.Duration {
	if len(bars) >= 2 {
		d := bars[1].Time.Sub(bars[0].Time)
		if d > 0 {
			return d
		}
	}
	return 15 * time.Minute
}

func annotateAroundMove(m *AroundMove) {
	if m == nil {
		return
	}
	abs := absFloat(m.ReturnPct)
	switch {
	case abs >= 6:
		m.Grade = AroundMoveGradeExtreme
	case abs >= 3:
		m.Grade = AroundMoveGradeStrong
	default:
		m.Grade = AroundMoveGradeNotable
	}
	side := "rise"
	if m.Direction == CVDDirDown {
		side = "drop"
	}
	m.Title = fmt.Sprintf("%s %s %s%%", m.Grade, side, FormatSignedPct(m.ReturnPct))
	m.Summary = fmt.Sprintf("%s %s from %s to %s (%s → %s) over %d bar(s).",
		m.Grade, side, m.At.UTC().Format("15:04"), m.Until.UTC().Format("15:04"),
		formatQty(m.Open), formatQty(m.Close), m.Bars)
}

func aroundMovePasses(ret, minPct float64, direction string) bool {
	switch direction {
	case string(PumpDirectionUp):
		return ret >= minPct
	case string(PumpDirectionDown):
		return ret <= -minPct
	default:
		return absFloat(ret) >= minPct
	}
}

func aroundMovesDirWord(direction string) string {
	switch direction {
	case string(PumpDirectionUp):
		return "up"
	case string(PumpDirectionDown):
		return "down"
	default:
		return "up or down"
	}
}
