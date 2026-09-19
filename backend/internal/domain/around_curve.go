package domain

import (
	"fmt"
	"math"
	"time"
)

// AroundCurve is the listed near-vs-far futures curve at the close of a phase.
type AroundCurve struct {
	Exchange  Exchange
	Shape     string
	Slope     string
	Spread    float64
	SpreadPct float64
	NearLabel string
	FarLabel  string
	NearMark  float64
	FarMark   float64
	At        time.Time
	Complete  bool
	Summary   string
}

// AroundCurveAt samples near vs far mark klines at or before `at`.
func AroundCurveAt(hist []BasisCurvePoint, live BasisCurve, ex Exchange, at time.Time) *AroundCurve {
	if at.IsZero() || !live.Complete || live.Near == nil || live.Far == nil {
		return nil
	}
	pt, ok := findCurveSample(hist, at.UTC(), curveHistorySlack(time.Hour))
	if !ok || pt.NearMark <= 0 || pt.FarMark <= 0 {
		return nil
	}
	out := AroundCurve{
		Exchange:  ex,
		Shape:     pt.Shape,
		Slope:     pt.Slope,
		Spread:    pt.Spread,
		SpreadPct: pt.SpreadPct,
		NearLabel: live.Near.Label,
		FarLabel:  live.Far.Label,
		NearMark:  pt.NearMark,
		FarMark:   pt.FarMark,
		At:        pt.Time.UTC(),
		Complete:  true,
	}
	out.Summary = explainAroundCurve(out)
	if out.Summary == "" {
		return nil
	}
	return &out
}

func explainAroundCurve(c AroundCurve) string {
	near, far := c.NearLabel, c.FarLabel
	if near == "" {
		near = "near"
	}
	if far == "" {
		far = "far"
	}
	gap := FormatSignedPct(math.Abs(c.SpreadPct))
	switch c.Shape {
	case BasisCurveContango:
		return fmt.Sprintf("Futures curve was in contango: far (%s) was %s%% richer than near (%s).", far, gap, near)
	case BasisCurveBackwardation:
		return fmt.Sprintf("Futures curve was in backwardation: near (%s) was %s%% richer than far (%s).", near, gap, far)
	case BasisKindFlat:
		return fmt.Sprintf("Futures curve was flat: near (%s) and far (%s) were about in line.", near, far)
	default:
		return ""
	}
}
