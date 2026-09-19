package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	AroundTapeDivPriceUpDeltaDown = "price_up_delta_down"
	AroundTapeDivPriceDownDeltaUp = "price_down_delta_up"
	aroundTapeDominanceBars       = 3
	aroundTapeMinBuySellBars      = 8
)

// AroundTapeRead is a Hyblock-style flow-at-prices read of the before+during tape.
type AroundTapeRead struct {
	POC            float64
	POCVolume      float64
	POCBuy         float64
	POCSell        float64
	Volume         float64
	BuyNotional    float64
	SellNotional   float64
	Delta          float64
	Dominant       string // buy | sell | balanced
	DominantFrom   float64
	DominantSide   string
	DominantAt     time.Time
	DeltaFlip      float64
	DeltaFlipKind  string // negative | positive
	DeltaFlipAt    time.Time
	PriceAfterFlip string // up | down | flat
	Divergence     string
	Summary        string
	Complete       bool
}

// BuildAroundTapeRead walks 1m bars for POC, dominance, and delta-vs-price.
func BuildAroundTapeRead(bars []AroundBar, from, to time.Time) *AroundTapeRead {
	slice := FilterAroundBars(bars, from, to)
	if len(slice) < aroundTapeMinBuySellBars {
		return nil
	}
	known := 0
	for _, b := range slice {
		if b.BuySellKnown && (b.BuyVolume > 0 || b.SellVolume > 0) {
			known++
		}
	}
	if known < aroundTapeMinBuySellBars {
		return nil
	}
	out := AroundTapeRead{Complete: true}
	for _, b := range slice {
		out.Volume += b.Volume
		if b.BuySellKnown {
			out.BuyNotional += b.BuyVolume
			out.SellNotional += b.SellVolume
		}
	}
	out.Delta = out.BuyNotional - out.SellNotional
	out.Dominant = TakerDominant(out.BuyNotional, out.SellNotional)
	if prof := aroundProfileOf(slice, from, to); prof != nil {
		out.POC = prof.POC
		out.POCVolume = prof.POCVolume
	}
	pocBuy, pocSell := pocBuySell(slice, out.POC)
	out.POCBuy, out.POCSell = pocBuy, pocSell
	fillTapeDominance(slice, &out)
	fillTapeDeltaFlip(slice, &out)
	out.Summary = explainAroundTapeRead(out)
	if out.Summary == "" {
		return nil
	}
	return &out
}

func pocBuySell(bars []AroundBar, poc float64) (buy, sell float64) {
	if poc <= 0 {
		return 0, 0
	}
	// Include bars whose range overlaps the POC row (one tick around the node).
	tick := poc * 0.0008
	if tick < 1 {
		tick = 1
	}
	lo, hi := poc, poc+tick
	for _, b := range bars {
		if !b.BuySellKnown || b.High < lo || b.Low > hi {
			continue
		}
		buy += b.BuyVolume
		sell += b.SellVolume
	}
	return buy, sell
}

func fillTapeDominance(bars []AroundBar, out *AroundTapeRead) {
	run := 0
	var runSide string
	var runStart AroundBar
	for _, b := range bars {
		if !b.BuySellKnown {
			run, runSide = 0, ""
			continue
		}
		side := TakerDominant(b.BuyVolume, b.SellVolume)
		if side == TakerSideEven {
			run, runSide = 0, ""
			continue
		}
		if side != runSide {
			runSide = side
			run = 1
			runStart = b
			continue
		}
		run++
		if run == aroundTapeDominanceBars && (out.DominantSide == "" || out.DominantSide != side) {
			out.DominantSide = side
			out.DominantFrom = tapeBarPrice(runStart)
			out.DominantAt = runStart.Time.UTC()
		}
	}
}

func fillTapeDeltaFlip(bars []AroundBar, out *AroundTapeRead) {
	var cvd float64
	var prev float64
	flipped := false
	var flipPrice, flipCVD float64
	var flipAt time.Time
	var flipKind string
	for _, b := range bars {
		if !b.BuySellKnown {
			continue
		}
		cvd += b.BuyVolume - b.SellVolume
		px := tapeBarPrice(b)
		if !flipped && prev != 0 && ((prev > 0 && cvd <= 0) || (prev < 0 && cvd >= 0)) {
			flipped = true
			flipPrice, flipCVD, flipAt = px, cvd, b.Time.UTC()
			if cvd <= 0 {
				flipKind = "negative"
			} else {
				flipKind = "positive"
			}
		}
		prev = cvd
	}
	if !flipped || flipPrice <= 0 {
		return
	}
	out.DeltaFlip = flipPrice
	out.DeltaFlipKind = flipKind
	out.DeltaFlipAt = flipAt
	last := tapeBarPrice(bars[len(bars)-1])
	if last <= 0 || flipPrice <= 0 {
		return
	}
	chg := (last - flipPrice) / flipPrice * 100
	out.PriceAfterFlip = changeDir(chg)
	if flipKind == "negative" && chg > 0.05 {
		out.Divergence = AroundTapeDivPriceUpDeltaDown
	}
	if flipKind == "positive" && chg < -0.05 {
		out.Divergence = AroundTapeDivPriceDownDeltaUp
	}
	_ = flipCVD
}

func tapeBarPrice(b AroundBar) float64 {
	if b.Close > 0 {
		return b.Close
	}
	return TypicalPrice(b.High, b.Low, b.Close)
}

func explainAroundTapeRead(r AroundTapeRead) string {
	var bits []string
	if r.POC > 0 && (r.POCBuy > 0 || r.POCSell > 0) {
		bits = append(bits, fmt.Sprintf("Around %s most volume printed: %s market buys vs %s market sells",
			formatQty(r.POC), formatTapeNotional(r.POCBuy), formatTapeNotional(r.POCSell)))
	} else if r.BuyNotional+r.SellNotional > 0 {
		bits = append(bits, fmt.Sprintf("%s market buys vs %s market sells in the window",
			formatTapeNotional(r.BuyNotional), formatTapeNotional(r.SellNotional)))
	}
	if r.DominantFrom > 0 && (r.DominantSide == TakerSideBuy || r.DominantSide == TakerSideSell) {
		who := "buyer"
		if r.DominantSide == TakerSideSell {
			who = "seller"
		}
		bits = append(bits, fmt.Sprintf("%s dominance started around %s", who, formatQty(r.DominantFrom)))
	}
	if r.DeltaFlip > 0 && r.DeltaFlipKind != "" {
		side := "negative"
		if r.DeltaFlipKind == "positive" {
			side = "positive"
		}
		s := fmt.Sprintf("delta turned %s after %s", side, formatQty(r.DeltaFlip))
		switch r.Divergence {
		case AroundTapeDivPriceUpDeltaDown:
			s += " while price was still going up"
		case AroundTapeDivPriceDownDeltaUp:
			s += " while price was still going down"
		}
		bits = append(bits, s)
	}
	if len(bits) == 0 {
		return ""
	}
	return ensureSentence(joinList(bits))
}

func formatTapeNotional(v float64) string {
	av := math.Abs(v)
	sign := ""
	if v < 0 {
		sign = "-"
		v = av
	}
	switch {
	case av >= 1e9:
		return fmt.Sprintf("%s%.1fB", sign, av/1e9)
	case av >= 1e6:
		return fmt.Sprintf("%s%.1fM", sign, av/1e6)
	case av >= 1e3:
		return fmt.Sprintf("%s%.0fK", sign, av/1e3)
	default:
		return sign + formatFixed(av, 0)
	}
}

func pickAroundTapeRead(venues []AroundVenue) *AroundTapeRead {
	var best *AroundTapeRead
	for i := range venues {
		tr := venues[i].TapeRead
		if tr == nil || !tr.Complete {
			continue
		}
		if best == nil || tr.Volume > best.Volume {
			cp := *tr
			best = &cp
		}
	}
	return best
}
