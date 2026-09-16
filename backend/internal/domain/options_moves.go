package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	OptionsMoveOIJump          = "oi_jump"
	OptionsMoveOIDrop          = "oi_drop"
	OptionsMovePCRJump         = "pcr_jump"
	OptionsMovePCRDrop         = "pcr_drop"
	OptionsMoveIVJump          = "iv_jump"
	OptionsMoveIVDrop          = "iv_drop"
	OptionsMoveGammaToShort    = "gamma_to_short"
	OptionsMoveGammaToLong     = "gamma_to_long"
	OptionsMoveFlipCrossedUp   = "flip_crossed_up"
	OptionsMoveFlipCrossedDown = "flip_crossed_down"

	OptionsMoveTopicOI    = "oi"
	OptionsMoveTopicLean  = "lean"
	OptionsMoveTopicVol   = "vol"
	OptionsMoveTopicGamma = "gamma"

	optionsMoveRatioWatch   = 2.0
	optionsMoveRatioNotable = 3.0
)

type optionsMoveKind struct {
	topic    string
	jumpKind string
	dropKind string
	floor    map[string]float64
	unit     string // pct | pcr | pts
}

var optionsMoveWindows = []struct {
	ID       string
	Dur      time.Duration
	MinPrior int
}{
	{OptionsHistoryWindow1h, time.Hour, 6},
	{OptionsHistoryWindow4h, 4 * time.Hour, 4},
	{OptionsHistoryWindow24h, 24 * time.Hour, 3},
}

var optionsMoveSeries = []optionsMoveKind{
	{
		topic: OptionsMoveTopicOI, jumpKind: OptionsMoveOIJump, dropKind: OptionsMoveOIDrop, unit: "pct",
		floor: map[string]float64{OptionsHistoryWindow1h: 5, OptionsHistoryWindow4h: 8, OptionsHistoryWindow24h: 12},
	},
	{
		topic: OptionsMoveTopicLean, jumpKind: OptionsMovePCRJump, dropKind: OptionsMovePCRDrop, unit: "pcr",
		floor: map[string]float64{OptionsHistoryWindow1h: 0.10, OptionsHistoryWindow4h: 0.15, OptionsHistoryWindow24h: 0.20},
	},
	{
		topic: OptionsMoveTopicVol, jumpKind: OptionsMoveIVJump, dropKind: OptionsMoveIVDrop, unit: "pts",
		floor: map[string]float64{OptionsHistoryWindow1h: 1.5, OptionsHistoryWindow4h: 2.5, OptionsHistoryWindow24h: 4},
	},
}

// OptionsMove is one unusual short-window change in OI, put/call, or IV.
type OptionsMove struct {
	Kind     string
	Topic    string
	Window   string
	Severity string
	Title    string
	Detail   string
	Current  float64
	Typical  float64
	Ratio    float64
	HasNums  bool
}

// OptionsMoves is the unusual-move block on a snapshot (empty when nothing stands out).
type OptionsMoves struct {
	Findings []OptionsMove
	Summary  string
}

// DetectOptionsMoves flags OI / put-call / IV changes much larger than typical.
func DetectOptionsMoves(points []OptionsHistoryPoint, now time.Time) *OptionsMoves {
	if len(points) < 4 {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	var findings []OptionsMove
	for _, ser := range optionsMoveSeries {
		if f := detectOptionsMoveSeries(points, now, ser); f != nil {
			findings = append(findings, *f)
		}
	}
	findings = append(findings, detectOptionsGammaMoves(points, now)...)
	if len(findings) == 0 {
		return nil
	}
	return &OptionsMoves{Findings: findings, Summary: optionsMovesSummary(findings)}
}

func detectOptionsMoveSeries(points []OptionsHistoryPoint, now time.Time, ser optionsMoveKind) *OptionsMove {
	var best *OptionsMove
	for _, w := range optionsMoveWindows {
		floor := ser.floor[w.ID]
		if floor <= 0 {
			continue
		}
		cur, typical, ok := optionsMoveVsTypical(points, now, w.Dur, w.MinPrior, ser.topic)
		if !ok {
			continue
		}
		curAbs := math.Abs(cur)
		if curAbs < floor {
			continue
		}
		ratio := 0.0
		if typical > 1e-9 {
			ratio = curAbs / typical
		} else if curAbs >= floor*1.5 {
			ratio = optionsMoveRatioNotable
		}
		if ratio < optionsMoveRatioWatch {
			continue
		}
		sev := OptionsFindingWatch
		if ratio >= optionsMoveRatioNotable || curAbs >= floor*2 {
			sev = OptionsFindingNotable
		}
		kind := ser.jumpKind
		if cur < 0 {
			kind = ser.dropKind
		}
		cand := OptionsMove{
			Kind: kind, Topic: ser.topic, Window: w.ID, Severity: sev,
			Title:   optionsMoveTitle(kind, w.ID, cur, ser.unit),
			Detail:  optionsMoveDetail(kind, w.ID, cur, typical, ratio, ser.unit),
			Current: cur, Typical: typical, Ratio: ratio, HasNums: true,
		}
		if best == nil || cand.Ratio > best.Ratio || (cand.Ratio == best.Ratio && cand.Severity == OptionsFindingNotable) {
			cp := cand
			best = &cp
		}
	}
	return best
}

func optionsMoveVsTypical(points []OptionsHistoryPoint, now time.Time, dur time.Duration, minPrior int, topic string) (current, typical float64, ok bool) {
	need := minPrior + 1
	changes := make([]float64, 0, need)
	slack := dur
	for i := 0; i < need+4 && len(changes) < need; i++ {
		end := now.Add(-time.Duration(i) * dur)
		start := end.Add(-dur)
		ev, eok := optionsMoveValueAt(points, end, slack, topic)
		sv, sok := optionsMoveValueAt(points, start, slack, topic)
		if !eok || !sok {
			continue
		}
		chg, cok := optionsMoveChange(topic, ev, sv)
		if !cok {
			continue
		}
		changes = append(changes, chg)
	}
	if len(changes) < minPrior+1 {
		return 0, 0, false
	}
	current = changes[0]
	priors := make([]float64, 0, len(changes)-1)
	for _, v := range changes[1:] {
		priors = append(priors, math.Abs(v))
	}
	typical = medianFloat(priors)
	return current, typical, true
}

func optionsMoveValueAt(points []OptionsHistoryPoint, at time.Time, slack time.Duration, topic string) (float64, bool) {
	p := optionsHistoryAtOrBefore(points, at, slack)
	switch topic {
	case OptionsMoveTopicOI:
		if !p.HasOI {
			return 0, false
		}
		v := optionsOIValue(p)
		if v <= 0 {
			return 0, false
		}
		return v, true
	case OptionsMoveTopicLean:
		if !p.HasPutCallOI {
			return 0, false
		}
		return p.PutCallOI, true
	case OptionsMoveTopicVol:
		if !p.HasDVOL || p.DVOL <= 0 {
			return 0, false
		}
		return p.DVOL, true
	default:
		return 0, false
	}
}

func optionsMoveChange(topic string, end, start float64) (float64, bool) {
	switch topic {
	case OptionsMoveTopicOI:
		if start <= 0 {
			return 0, false
		}
		return (end - start) / math.Abs(start) * 100, true
	case OptionsMoveTopicLean, OptionsMoveTopicVol:
		return end - start, true
	default:
		return 0, false
	}
}

func detectOptionsGammaMoves(points []OptionsHistoryPoint, now time.Time) []OptionsMove {
	var regime, flip *OptionsMove
	for _, w := range optionsMoveWindows {
		past := optionsHistoryAtOrBefore(points, now.Add(-w.Dur), optionsHistorySlack(w.Dur))
		latest := optionsHistoryLatest(points)
		if latest.Time.IsZero() || past.Time.IsZero() {
			continue
		}
		sev := OptionsFindingWatch
		if w.ID == OptionsHistoryWindow1h || w.ID == OptionsHistoryWindow4h {
			sev = OptionsFindingNotable
		}
		if regime == nil && latest.HasGamma && past.HasGamma &&
			latest.GammaRegime != "" && past.GammaRegime != "" &&
			latest.GammaRegime != past.GammaRegime &&
			(latest.GammaRegime == GammaRegimeShort || latest.GammaRegime == GammaRegimeLong) {
			kind := OptionsMoveGammaToLong
			title := fmt.Sprintf("Dealer gamma flipped to long (stabler) in %s", w.ID)
			detail := fmt.Sprintf("Regime was %s; now %s. Net GEX %s.", past.GammaRegime, latest.GammaRegime, FormatSignedQty(latest.GammaNetUSD))
			if latest.GammaRegime == GammaRegimeShort {
				kind = OptionsMoveGammaToShort
				title = fmt.Sprintf("Dealer gamma flipped to short (faster) in %s", w.ID)
			}
			regime = &OptionsMove{
				Kind: kind, Topic: OptionsMoveTopicGamma, Window: w.ID, Severity: sev,
				Title: title, Detail: detail,
			}
		}
		if flip == nil && latest.HasGammaFlip && past.HasGammaFlip &&
			latest.GammaFlipSide != "" && past.GammaFlipSide != "" &&
			latest.GammaFlipSide != past.GammaFlipSide &&
			(latest.GammaFlipSide == GammaFlipAbove || latest.GammaFlipSide == GammaFlipBelow) {
			kind := OptionsMoveFlipCrossedUp
			title := fmt.Sprintf("Spot moved above the gamma flip in %s", w.ID)
			detail := fmt.Sprintf("Flip was %s of spot; now %s (flip %.0f, spot %.0f).",
				past.GammaFlipSide, latest.GammaFlipSide, latest.GammaFlip, latest.IndexPrice)
			if latest.GammaFlipSide == GammaFlipBelow {
				kind = OptionsMoveFlipCrossedDown
				title = fmt.Sprintf("Spot moved below the gamma flip in %s", w.ID)
			}
			flip = &OptionsMove{
				Kind: kind, Topic: OptionsMoveTopicGamma, Window: w.ID, Severity: sev,
				Title: title, Detail: detail,
			}
		}
	}
	out := make([]OptionsMove, 0, 2)
	if regime != nil {
		out = append(out, *regime)
	}
	if flip != nil {
		out = append(out, *flip)
	}
	return out
}

func optionsMoveTitle(kind, window string, current float64, unit string) string {
	abs := math.Abs(current)
	switch kind {
	case OptionsMoveOIJump:
		return fmt.Sprintf("Open interest jumped %s in %s — well above typical", formatOptionsMove(abs, unit), window)
	case OptionsMoveOIDrop:
		return fmt.Sprintf("Open interest dropped %s in %s — well above typical", formatOptionsMove(abs, unit), window)
	case OptionsMovePCRJump:
		return fmt.Sprintf("Put/call swung toward puts in %s — larger than usual", window)
	case OptionsMovePCRDrop:
		return fmt.Sprintf("Put/call swung toward calls in %s — larger than usual", window)
	case OptionsMoveIVJump:
		return fmt.Sprintf("Implied vol jumped %s in %s — well above typical", formatOptionsMove(abs, unit), window)
	case OptionsMoveIVDrop:
		return fmt.Sprintf("Implied vol dropped %s in %s — well above typical", formatOptionsMove(abs, unit), window)
	default:
		return "Unusual options move"
	}
}

func optionsMoveDetail(kind, window string, current, typical, ratio float64, unit string) string {
	dir := "up"
	if current < 0 {
		dir = "down"
	}
	what := "Open interest"
	switch kind {
	case OptionsMovePCRJump, OptionsMovePCRDrop:
		what = "Put/call"
	case OptionsMoveIVJump, OptionsMoveIVDrop:
		what = "Implied vol"
	}
	return fmt.Sprintf("%s moved %s %s over %s. Typical same-length move is %s (%.1f× that).",
		what, dir, formatOptionsMove(math.Abs(current), unit), window, formatOptionsMove(typical, unit), ratio)
}

func formatOptionsMove(v float64, unit string) string {
	switch unit {
	case "pct":
		return fmt.Sprintf("%.1f%%", v)
	case "pcr":
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%.1f pts", v)
	}
}

func optionsMovesSummary(findings []OptionsMove) string {
	switch len(findings) {
	case 0:
		return ""
	case 1:
		return findings[0].Title
	default:
		return fmt.Sprintf("%d unusual options moves in the last day.", len(findings))
	}
}
