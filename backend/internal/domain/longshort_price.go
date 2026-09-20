package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	LongShortPriceDivTopLongPriceDown = "top_long_price_down"
	LongShortPriceDivTopShortPriceUp  = "top_short_price_up"
	LongShortVsPriceOpposite          = "opposite"
	LongShortVsPriceConfirms          = "confirms"

	// Net move floors so a 0.05% tick is not a divergence.
	longShortPriceFlatPct = 0.30
	longShortTopFlatPP    = 0.50
	longShortPriceMinRun  = 15 * time.Minute
	longShortPriceMaxRun  = 4 * time.Hour
)

// LongShortPriceDivergence is when top-trader long share and price move opposite ways.
// Example: top size +3pp long while price drops 1.5% over 30m.
type LongShortPriceDivergence struct {
	Kind            string // top_long_price_down | top_short_price_up
	Versus          string // top_positions | top_accounts
	Since           time.Time
	LastAt          time.Time
	Duration        string
	DurationSeconds int
	PriceChangePct  string
	TopChangePct    string // signed percentage points of long share
	GapChangePct    string // signed pp of crowd-vs-top gap
	Headline        string
	Detail          string
	Bars            int
}

type longShortPriced struct {
	Time           time.Time
	Versus         string
	Crowd          float64
	Top            float64
	Price          float64
	PriceChangePct float64
	TopChangePP    float64
	GapChangePP    float64
	VsPrice        string
}

// AttachLongShortPrices joins 5m candles onto each series that has a top-trader layer.
func AttachLongShortPrices(series []*LongShortSeries, candles map[Exchange][]Candle) {
	if len(series) == 0 {
		return
	}
	for _, s := range series {
		if s == nil {
			continue
		}
		AttachLongShortPriceDivergence(s, candles[s.Exchange])
	}
}

// ApplyLongShortPrices copies priced history and divergence onto a built snapshot.
func ApplyLongShortPrices(snap *LongShortSnapshot, series []*LongShortSeries) {
	if snap == nil || len(series) == 0 {
		return
	}
	by := map[string]*LongShortSeries{}
	for _, s := range series {
		if s != nil {
			by[string(s.Exchange)] = s
		}
	}
	for i := range snap.Venues {
		s := by[snap.Venues[i].Exchange]
		if s == nil {
			continue
		}
		snap.Venues[i].GapHistory = s.GapHistory
		snap.Venues[i].Divergence = s.Divergence
		if snap.Divergence == nil && s.Divergence != nil {
			d := *s.Divergence
			snap.Divergence = &d
			snap.GapHistory = s.GapHistory
		}
	}
}

// AttachLongShortPriceDivergence fills Divergence and price fields on gap history.
func AttachLongShortPriceDivergence(s *LongShortSeries, candles []Candle) {
	if s == nil {
		return
	}
	s.Divergence = nil
	versus, layer := longShortGapLayer(s)
	if layer == nil || len(candles) == 0 {
		return
	}
	raw := alignLongShortGapPoints(s, layer, versus)
	if len(raw) < 2 {
		return
	}
	px := candleCloseByBucket(candles, longShortGapBucket)
	pts := make([]longShortPriced, 0, len(raw))
	var prev *longShortPriced
	for _, r := range raw {
		closePx, ok := px[r.Time.Unix()]
		if !ok || closePx <= 0 {
			continue
		}
		row := longShortPriced{
			Time: r.Time, Versus: r.Versus, Crowd: r.Crowd, Top: r.Top, Price: closePx,
		}
		if prev != nil && prev.Price > 0 {
			row.PriceChangePct = (closePx - prev.Price) / prev.Price * 100
			row.TopChangePP = (r.Top - prev.Top) * 100
			row.GapChangePP = r.gapPP() - prev.gapPP()
			row.VsPrice = longShortBarVsPrice(row.PriceChangePct, row.TopChangePP)
		}
		pts = append(pts, row)
		cp := row
		prev = &cp
	}
	if len(pts) < 2 {
		return
	}
	enrichLongShortGapHistoryPrices(s, pts)
	div := detectLongShortPriceDivergence(pts)
	if div != nil {
		s.Divergence = div
	}
}

func (r longShortPriced) gapPP() float64 {
	return (r.Top - r.Crowd) * 100
}

func candleCloseByBucket(candles []Candle, bucket time.Duration) map[int64]float64 {
	out := map[int64]float64{}
	for _, c := range candles {
		px := parseQty(c.Close)
		if px <= 0 {
			continue
		}
		t := c.OpenTime
		if t.IsZero() {
			t = c.CloseTime
			if !t.IsZero() && bucket > 0 {
				t = t.Add(-bucket + time.Millisecond)
			}
		}
		if t.IsZero() {
			continue
		}
		out[TruncateToBucket(t.UTC(), bucket).Unix()] = px
	}
	return out
}

func longShortBarVsPrice(pricePct, topPP float64) string {
	if pricePct > 0 && topPP > 0 || pricePct < 0 && topPP < 0 {
		return LongShortVsPriceConfirms
	}
	if pricePct > 0 && topPP < 0 || pricePct < 0 && topPP > 0 {
		return LongShortVsPriceOpposite
	}
	return ""
}

func longShortNetVsPriceKind(pricePct, topPP float64) string {
	if math.Abs(pricePct) < longShortPriceFlatPct || math.Abs(topPP) < longShortTopFlatPP {
		return ""
	}
	if topPP > 0 && pricePct < 0 {
		return LongShortPriceDivTopLongPriceDown
	}
	if topPP < 0 && pricePct > 0 {
		return LongShortPriceDivTopShortPriceUp
	}
	return ""
}

func detectLongShortPriceDivergence(pts []longShortPriced) *LongShortPriceDivergence {
	n := len(pts) - 1
	if n < 1 {
		return nil
	}
	end := pts[n]
	cutoff := end.Time.Add(-longShortPriceMaxRun)
	kind := ""
	start := n
	for i := n - 1; i >= 0; i-- {
		if pts[i].Time.Before(cutoff) {
			break
		}
		pricePct := 0.0
		if pts[i].Price > 0 {
			pricePct = (end.Price - pts[i].Price) / pts[i].Price * 100
		}
		topPP := (end.Top - pts[i].Top) * 100
		k := longShortNetVsPriceKind(pricePct, topPP)
		if k == "" {
			if start == n {
				continue
			}
			break
		}
		if kind == "" {
			kind = k
			start = i
			continue
		}
		if k != kind {
			break
		}
		start = i
	}
	if kind == "" || start >= n {
		return nil
	}
	from, to := pts[start], end
	dur := to.Time.Sub(from.Time)
	if dur < longShortPriceMinRun {
		return nil
	}
	pricePct := 0.0
	if from.Price > 0 {
		pricePct = (to.Price - from.Price) / from.Price * 100
	}
	topPP := (to.Top - from.Top) * 100
	gapPP := to.gapPP() - from.gapPP()
	out := &LongShortPriceDivergence{
		Kind:            kind,
		Versus:          to.Versus,
		Since:           from.Time.UTC(),
		LastAt:          to.Time.UTC(),
		DurationSeconds: int(dur / time.Second),
		Duration:        formatCVDDuration(dur),
		PriceChangePct:  FormatSignedPct(pricePct),
		TopChangePct:    FormatSignedPct(topPP),
		GapChangePct:    FormatSignedPct(gapPP),
		Bars:            n - start,
		Headline:        longShortPriceDivHeadline(kind, pricePct, dur),
		Detail:          longShortPriceDivDetail(kind, topPP, gapPP, from.Time, to.Versus),
	}
	return out
}

func longShortPriceDivHeadline(kind string, pricePct float64, dur time.Duration) string {
	how := formatCVDDuration(dur)
	move := formatFixed(math.Abs(pricePct), 2)
	switch kind {
	case LongShortPriceDivTopLongPriceDown:
		return fmt.Sprintf("Top traders got more long while price dropped %s%% over %s.", move, how)
	case LongShortPriceDivTopShortPriceUp:
		return fmt.Sprintf("Top traders got more short while price rose %s%% over %s.", move, how)
	default:
		return ""
	}
}

func longShortPriceDivDetail(kind string, topPP, gapPP float64, since time.Time, versus string) string {
	who := "Top size"
	if versus == LongShortKindTopAccounts {
		who = "Top-trader accounts"
	}
	side := "long"
	if kind == LongShortPriceDivTopShortPriceUp {
		side = "short"
	}
	out := fmt.Sprintf("%s long share %s pp (%s). Crowd-vs-top gap %s pp. Started %s UTC.",
		who, FormatSignedPct(topPP), side, FormatSignedPct(gapPP), since.UTC().Format("15:04"))
	return out
}

func enrichLongShortGapHistoryPrices(s *LongShortSeries, pts []longShortPriced) {
	if s == nil || len(s.GapHistory) == 0 || len(pts) == 0 {
		return
	}
	by := map[int64]longShortPriced{}
	for _, p := range pts {
		by[p.Time.Unix()] = p
	}
	for i := range s.GapHistory {
		p, ok := by[s.GapHistory[i].Time.Unix()]
		if !ok {
			continue
		}
		s.GapHistory[i].Price = formatQty(p.Price)
		if p.PriceChangePct != 0 {
			s.GapHistory[i].PriceChangePct = FormatSignedPct(p.PriceChangePct)
		}
		if p.TopChangePP != 0 {
			s.GapHistory[i].TopChangePct = FormatSignedPct(p.TopChangePP)
		}
		s.GapHistory[i].VsPrice = p.VsPrice
	}
}
