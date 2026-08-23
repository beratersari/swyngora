package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	AbsorptionKindNone = ""
	AbsorptionKindBid  = "bid" // bids absorbing market sells — price holds up
	AbsorptionKindAsk  = "ask" // asks absorbing market buys — price holds down

	AbsorptionGradeNone     = ""
	AbsorptionGradeWeak     = "weak"
	AbsorptionGradeModerate = "moderate"
	AbsorptionGradeStrong   = "strong"
	AbsorptionGradeExtreme  = "extreme"

	AbsorptionResultNone   = ""
	AbsorptionResultHeld   = "held"   // price barely moved
	AbsorptionResultPushed = "pushed" // price moved against the aggressive side

	absorptionMinImbalanceBar    = 0.12
	absorptionMinImbalanceWindow = 0.10
	absorptionBarFlatPct         = 0.08
	MaxAbsorptionPoints          = 320
	MaxAbsorptionEpisodes        = 8
)

// AbsorptionWindows are the buy/sell vs price lookbacks.
var AbsorptionWindows = []struct {
	ID      string
	Dur     time.Duration
	FlatPct float64
}{
	{CVDWindow15m, 15 * time.Minute, 0.20},
	{CVDWindow1h, time.Hour, 0.35},
	{CVDWindow4h, 4 * time.Hour, 0.60},
	{CVDWindow24h, 24 * time.Hour, 1.20},
}

// AbsorptionPoint is one 5-minute bar of aggressive flow versus price.
type AbsorptionPoint struct {
	Time           time.Time
	Price          float64
	PriceChangePct float64
	BuyNotional    float64
	SellNotional   float64
	Delta          float64
	Kind           string // bid | ask | ""
	Absorber       string // buy | sell | ""
	Result         string // held | pushed | ""
	Score          int    // 0–100
	Grade          string
}

// AbsorptionWindowStat is flow versus price over one lookback.
type AbsorptionWindowStat struct {
	Window         string
	BuyNotional    float64
	SellNotional   float64
	Delta          float64
	Volume         float64
	PriceFrom      float64
	PriceTo        float64
	PriceChangePct float64
	Kind           string
	Absorber       string
	Result         string
	Score          int
	Grade          string
	Title          string
	Summary        string
	Complete       bool
}

// AbsorptionEpisode is a consecutive run of the same absorption kind.
type AbsorptionEpisode struct {
	Kind            string
	Absorber        string
	Result          string
	Score           int
	Grade           string
	Bars            int
	Since           time.Time
	Until           time.Time
	Duration        string
	DurationSeconds int
	BuyNotional     float64
	SellNotional    float64
	Delta           float64
	PriceFrom       float64
	PriceTo         float64
	PriceChangePct  float64
	Active          bool
	Title           string
	Summary         string
}

// AbsorptionVenue is one venue's absorption read.
type AbsorptionVenue struct {
	Exchange    Exchange
	Symbol      string
	Market      string // futures | spot
	Points      []AbsorptionPoint
	Windows     []AbsorptionWindowStat
	Current     *AbsorptionEpisode
	Episodes    []AbsorptionEpisode
	LastPrice   float64
	OverlapFrom *time.Time
	OverlapTo   *time.Time
	Summary     string
	Error       string
	Complete    bool
}

// AbsorptionReport is the API result.
type AbsorptionReport struct {
	Symbol       string
	Exchange     string
	AsOf         time.Time
	Venues       []AbsorptionVenue
	Combined     *AbsorptionVenue
	SpotVenues   []AbsorptionVenue
	SpotCombined *AbsorptionVenue
	Summary      string
	Note         string
}

// ClassifyAbsorption labels one interval of aggressive flow versus price.
// PriceChangePct is percent. FlatPct is how little price must move to count as "held".
func ClassifyAbsorption(buy, sell, pricePct, flatPct float64) AbsorptionPoint {
	out := AbsorptionPoint{
		BuyNotional: buy, SellNotional: sell, Delta: buy - sell,
		PriceChangePct: pricePct,
	}
	tot := buy + sell
	if tot <= 0 || flatPct <= 0 {
		return out
	}
	imb := math.Abs(out.Delta) / tot
	minImb := absorptionMinImbalanceBar
	if flatPct >= 0.15 {
		minImb = absorptionMinImbalanceWindow
	}
	if imb < minImb {
		return out
	}
	if out.Delta < 0 {
		// More market sells. Absorption if price did not fall with them.
		if pricePct < -flatPct {
			return out
		}
		out.Kind = AbsorptionKindBid
		out.Absorber = TakerSideBuy
		if pricePct > flatPct {
			out.Result = AbsorptionResultPushed
		} else {
			out.Result = AbsorptionResultHeld
		}
	} else {
		// More market buys. Absorption if price did not rise with them.
		if pricePct > flatPct {
			return out
		}
		out.Kind = AbsorptionKindAsk
		out.Absorber = TakerSideSell
		if pricePct < -flatPct {
			out.Result = AbsorptionResultPushed
		} else {
			out.Result = AbsorptionResultHeld
		}
	}
	out.Score = absorptionScore(imb, pricePct, flatPct, out.Result)
	out.Grade = AbsorptionGrade(out.Score)
	return out
}

// AbsorptionGrade maps a 0–100 score to a word.
func AbsorptionGrade(score int) string {
	switch {
	case score >= 85:
		return AbsorptionGradeExtreme
	case score >= 65:
		return AbsorptionGradeStrong
	case score >= 40:
		return AbsorptionGradeModerate
	case score >= 1:
		return AbsorptionGradeWeak
	default:
		return AbsorptionGradeNone
	}
}

// BuildAbsorptionSeries compares aggressive buy/sell to price over 5m bars.
func BuildAbsorptionSeries(ex Exchange, symbol string, buckets []TakerBucket, prices []CVDPrice, now time.Time, started time.Time) AbsorptionVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	cut := now.Add(-DefaultCVDLookback)
	bars := ResampleTakerBuckets(buckets, CVDBar)
	out := AbsorptionVenue{
		Exchange: ex, Symbol: symbol, Market: CVDMarketFutures,
		Points: []AbsorptionPoint{}, Windows: []AbsorptionWindowStat{},
	}
	var prev float64
	for _, b := range bars {
		if b.Start.Before(cut) {
			continue
		}
		px := priceAt(prices, b.Start)
		chg := 0.0
		if prev > 0 && px > 0 {
			chg = (px - prev) / prev * 100
		}
		pt := ClassifyAbsorption(b.BuyNotional, b.SellNotional, chg, absorptionBarFlatPct)
		pt.Time = b.Start
		pt.Price = px
		out.Points = append(out.Points, pt)
		if px > 0 {
			prev = px
		}
	}
	if len(out.Points) > MaxAbsorptionPoints {
		out.Points = out.Points[len(out.Points)-MaxAbsorptionPoints:]
	}
	if n := len(out.Points); n > 0 {
		out.LastPrice = out.Points[n-1].Price
	}
	for _, w := range AbsorptionWindows {
		out.Windows = append(out.Windows, summarizeAbsorptionWindow(out.Points, w.ID, now.Add(-w.Dur), now, started, w.FlatPct))
	}
	out.Complete = cvdCoverageComplete(started, cut)
	out.Episodes = absorptionEpisodes(out.Points, now)
	if n := len(out.Episodes); n > 0 {
		cur := out.Episodes[n-1]
		out.Current = &cur
	}
	out.Summary = ExplainAbsorptionVenue(out)
	return out
}

// CombineAbsorptionVenues adds overlapping 5m buy/sell and re-scores absorption.
func CombineAbsorptionVenues(symbol string, venues []AbsorptionVenue, prices []CVDPrice, now time.Time) *AbsorptionVenue {
	symbol = NormalizeLiquidationSymbol(symbol)
	now = now.UTC()
	out := &AbsorptionVenue{
		Exchange: "all", Symbol: symbol, Market: CVDMarketFutures,
		Points: []AbsorptionPoint{}, Windows: []AbsorptionWindowStat{},
	}
	var usable []AbsorptionVenue
	for _, v := range venues {
		if v.Error != "" || len(v.Points) == 0 {
			continue
		}
		usable = append(usable, v)
	}
	if len(usable) < 2 {
		out.Windows = emptyAbsorptionWindows(now, time.Time{})
		out.Summary = "Combined absorption only uses the time range both Binance and Bybit have. One venue is still filling in."
		return out
	}
	sort.Slice(usable, func(i, j int) bool {
		return string(usable[i].Exchange) < string(usable[j].Exchange)
	})
	if usable[0].Market != "" {
		out.Market = usable[0].Market
	}
	// Reuse CVD overlap on a thin adapter.
	from, to, ok := overlapAbsorptionRange(usable)
	if !ok {
		out.Windows = emptyAbsorptionWindows(now, time.Time{})
		out.Summary = "Combined absorption needs overlapping 5-minute bars on both venues."
		return out
	}
	out.OverlapFrom = &from
	out.OverlapTo = &to
	indexes := make([]map[int64]AbsorptionPoint, len(usable))
	for i, v := range usable {
		m := make(map[int64]AbsorptionPoint, len(v.Points))
		for _, p := range v.Points {
			m[TruncateToBucket(p.Time, CVDBar).UnixMilli()] = p
		}
		indexes[i] = m
	}
	var prev float64
	for at := from; !at.After(to); at = at.Add(CVDBar) {
		ms := at.UnixMilli()
		pt := AbsorptionPoint{Time: at, Price: priceAt(prices, at)}
		for i := range usable {
			src := indexes[i][ms]
			pt.BuyNotional += src.BuyNotional
			pt.SellNotional += src.SellNotional
		}
		pt.Delta = pt.BuyNotional - pt.SellNotional
		if prev > 0 && pt.Price > 0 {
			pt.PriceChangePct = (pt.Price - prev) / prev * 100
		}
		got := ClassifyAbsorption(pt.BuyNotional, pt.SellNotional, pt.PriceChangePct, absorptionBarFlatPct)
		pt.Kind, pt.Absorber, pt.Result, pt.Score, pt.Grade = got.Kind, got.Absorber, got.Result, got.Score, got.Grade
		out.Points = append(out.Points, pt)
		if pt.Price > 0 {
			prev = pt.Price
		}
	}
	if len(out.Points) > MaxAbsorptionPoints {
		out.Points = out.Points[len(out.Points)-MaxAbsorptionPoints:]
	}
	if n := len(out.Points); n > 0 {
		out.LastPrice = out.Points[n-1].Price
	}
	for _, w := range AbsorptionWindows {
		out.Windows = append(out.Windows, summarizeAbsorptionWindow(out.Points, w.ID, now.Add(-w.Dur), now, from, w.FlatPct))
	}
	allComplete := true
	for _, v := range usable {
		if !v.Complete {
			allComplete = false
			break
		}
	}
	out.Complete = allComplete
	out.Episodes = absorptionEpisodes(out.Points, now)
	if n := len(out.Episodes); n > 0 {
		cur := out.Episodes[n-1]
		out.Current = &cur
	}
	out.Summary = ExplainAbsorptionVenue(*out)
	if !out.Complete {
		out.Summary = strings.TrimSpace(absorptionOverlapNote(from, now) + " " + out.Summary)
	}
	return out
}

// ExplainAbsorptionVenue writes a 4h-first line, then the current run.
func ExplainAbsorptionVenue(v AbsorptionVenue) string {
	if v.Error != "" {
		return v.Error
	}
	var w4 AbsorptionWindowStat
	for _, w := range v.Windows {
		if w.Window == CVDWindow4h {
			w4 = w
			break
		}
	}
	if w4.Window == "" && len(v.Windows) > 0 {
		w4 = v.Windows[0]
	}
	name := prettyBase(v.Symbol)
	if w4.Window == "" {
		return name + ": not enough aggressive volume yet to judge absorption."
	}
	head := fmt.Sprintf("%s %s: %s", name, w4.Window, w4.Summary)
	if v.Current != nil && v.Current.Kind != "" && v.Current.Summary != "" {
		head += " " + v.Current.Summary
	}
	return head
}

// ExplainAbsorptionReport prefers combined, else the first venue.
func ExplainAbsorptionReport(r AbsorptionReport) string {
	if r.Combined != nil && r.Combined.Summary != "" && r.Combined.Error == "" {
		return "Futures " + r.Combined.Summary
	}
	for _, v := range r.Venues {
		if v.Summary != "" && v.Error == "" {
			return "Futures " + string(v.Exchange) + ": " + v.Summary
		}
	}
	if r.SpotCombined != nil && r.SpotCombined.Summary != "" && r.SpotCombined.Error == "" {
		return "Spot " + r.SpotCombined.Summary
	}
	return "No absorption read yet."
}

func absorptionScore(imbalance, pricePct, flatPct float64, result string) int {
	if imbalance <= 0 || flatPct <= 0 {
		return 0
	}
	imb := (imbalance - 0.10) / 0.40
	if imb < 0 {
		imb = 0
	}
	if imb > 1 {
		imb = 1
	}
	// Held: flatter = stronger. Pushed (price against the flow) is stronger still.
	var priceScore float64
	if result == AbsorptionResultPushed {
		extra := (math.Abs(pricePct) - flatPct) / (flatPct * 3)
		if extra < 0 {
			extra = 0
		}
		if extra > 1 {
			extra = 1
		}
		priceScore = 45 + 10*extra
	} else {
		hold := 1 - math.Min(1, math.Abs(pricePct)/flatPct)
		if hold < 0 {
			hold = 0
		}
		priceScore = 40 * hold
	}
	score := 50*imb + priceScore
	if score < 1 {
		score = 1
	}
	if score > 100 {
		score = 100
	}
	return int(math.Round(score))
}

func summarizeAbsorptionWindow(points []AbsorptionPoint, window string, from, now, started time.Time, flatPct float64) AbsorptionWindowStat {
	out := AbsorptionWindowStat{Window: window}
	var first, last *AbsorptionPoint
	var buy, sell float64
	for i := range points {
		p := &points[i]
		if p.Time.Before(from) {
			continue
		}
		if first == nil {
			first = p
		}
		last = p
		buy += p.BuyNotional
		sell += p.SellNotional
	}
	out.BuyNotional, out.SellNotional = buy, sell
	out.Delta = buy - sell
	out.Volume = buy + sell
	out.Complete = cvdCoverageComplete(started, from)
	if first == nil || last == nil {
		out.Title = "no data"
		out.Summary = "Not enough bars in this window."
		return out
	}
	out.PriceFrom = first.Price
	out.PriceTo = last.Price
	if first.Price > 0 && last.Price > 0 {
		out.PriceChangePct = (last.Price - first.Price) / first.Price * 100
	}
	got := ClassifyAbsorption(buy, sell, out.PriceChangePct, flatPct)
	out.Kind, out.Absorber, out.Result, out.Score, out.Grade = got.Kind, got.Absorber, got.Result, got.Score, got.Grade
	out.Title = absorptionTitle(out.Kind, out.Grade, out.Result)
	out.Summary = explainAbsorptionWindow(out)
	return out
}

func explainAbsorptionWindow(w AbsorptionWindowStat) string {
	flow := "flow was balanced"
	if w.Delta > 0 {
		flow = "market buys " + formatQty(w.BuyNotional) + " vs sells " + formatQty(w.SellNotional)
	} else if w.Delta < 0 {
		flow = "market sells " + formatQty(w.SellNotional) + " vs buys " + formatQty(w.BuyNotional)
	}
	px := fmt.Sprintf("price %s%%", FormatSignedPct(w.PriceChangePct))
	if w.Kind == "" {
		if w.Volume <= 0 {
			return "Not enough aggressive volume in this window."
		}
		return fmt.Sprintf("%s, %s — not absorption (price moved with the aggressive side, or flow was mixed).", flow, px)
	}
	return fmt.Sprintf("%s, %s — %s (%s, score %d).", flow, px, absorptionPhrase(w.Kind, w.Result), w.Grade, w.Score)
}

func absorptionTitle(kind, grade, result string) string {
	if kind == "" {
		return "no absorption"
	}
	head := absorptionPhrase(kind, result)
	if grade != "" {
		return grade + " " + head
	}
	return head
}

func absorptionPhrase(kind, result string) string {
	switch kind {
	case AbsorptionKindBid:
		if result == AbsorptionResultPushed {
			return "bids absorbing sells and lifting price"
		}
		return "bids absorbing market sells"
	case AbsorptionKindAsk:
		if result == AbsorptionResultPushed {
			return "asks absorbing buys and pressing price"
		}
		return "asks absorbing market buys"
	default:
		return "no absorption"
	}
}

func absorptionEpisodes(points []AbsorptionPoint, now time.Time) []AbsorptionEpisode {
	var runs []AbsorptionEpisode
	start := -1
	var kind string
	flush := func(end int) {
		if start < 0 || end < start || kind == "" {
			return
		}
		ep := episodeFromPoints(points[start:end+1], now)
		if ep.Bars > 0 {
			runs = append(runs, ep)
		}
	}
	for i, p := range points {
		if p.Kind == "" {
			if start >= 0 {
				flush(i - 1)
				start, kind = -1, ""
			}
			continue
		}
		if start >= 0 && p.Kind != kind {
			flush(i - 1)
			start, kind = -1, ""
		}
		if start < 0 {
			start, kind = i, p.Kind
		}
	}
	if start >= 0 {
		flush(len(points) - 1)
	}
	if len(runs) > MaxAbsorptionEpisodes {
		runs = runs[len(runs)-MaxAbsorptionEpisodes:]
	}
	return runs
}

func episodeFromPoints(pts []AbsorptionPoint, now time.Time) AbsorptionEpisode {
	out := AbsorptionEpisode{}
	if len(pts) == 0 {
		return out
	}
	out.Kind = pts[0].Kind
	out.Absorber = pts[0].Absorber
	out.Bars = len(pts)
	out.Since = pts[0].Time
	out.Until = pts[len(pts)-1].Time
	out.DurationSeconds = int(out.Until.Sub(out.Since)/time.Second) + int(CVDBar/time.Second)
	out.Duration = formatCVDDuration(time.Duration(out.DurationSeconds) * time.Second)
	best := 0
	pushed := 0
	for _, p := range pts {
		out.BuyNotional += p.BuyNotional
		out.SellNotional += p.SellNotional
		if p.Score > best {
			best = p.Score
		}
		if p.Result == AbsorptionResultPushed {
			pushed++
		}
	}
	out.Delta = out.BuyNotional - out.SellNotional
	out.PriceFrom = pts[0].Price
	out.PriceTo = pts[len(pts)-1].Price
	if out.PriceFrom > 0 && out.PriceTo > 0 {
		out.PriceChangePct = (out.PriceTo - out.PriceFrom) / out.PriceFrom * 100
	}
	if pushed*2 >= len(pts) {
		out.Result = AbsorptionResultPushed
	} else {
		out.Result = AbsorptionResultHeld
	}
	// Re-score the whole run with the 15m flat band so a long held tape ranks.
	got := ClassifyAbsorption(out.BuyNotional, out.SellNotional, out.PriceChangePct, 0.20)
	if got.Score > best {
		out.Score = got.Score
	} else {
		out.Score = best
	}
	if got.Kind == out.Kind && got.Result != "" {
		out.Result = got.Result
	}
	out.Grade = AbsorptionGrade(out.Score)
	out.Active = !pts[len(pts)-1].Time.Before(now.Add(-2 * CVDBar))
	out.Title = absorptionTitle(out.Kind, out.Grade, out.Result)
	out.Summary = fmt.Sprintf("%s for %s (%d bars, score %d): market buys %s / sells %s, price %s%%.",
		absorptionPhrase(out.Kind, out.Result), out.Duration, out.Bars, out.Score,
		formatQty(out.BuyNotional), formatQty(out.SellNotional),
		FormatSignedPct(out.PriceChangePct))
	if out.Active {
		out.Summary = "Now: " + out.Summary
	} else {
		out.Summary = "Latest run: " + out.Summary
	}
	return out
}

func emptyAbsorptionWindows(now, started time.Time) []AbsorptionWindowStat {
	out := make([]AbsorptionWindowStat, 0, len(AbsorptionWindows))
	for _, w := range AbsorptionWindows {
		out = append(out, summarizeAbsorptionWindow(nil, w.ID, now.Add(-w.Dur), now, started, w.FlatPct))
	}
	return out
}

func absorptionOverlapNote(from, now time.Time) string {
	if from.IsZero() {
		return "Combined absorption is not complete (need overlapping Binance and Bybit history)."
	}
	d := now.Sub(from)
	if d < 90*time.Minute {
		return fmt.Sprintf("Combined uses ~%.0f minutes where both venues have bars (Bybit still filling) — not complete.", d.Minutes())
	}
	return fmt.Sprintf("Combined uses the last %.0f hours where both venues have bars — not complete.", d.Hours())
}

func overlapAbsorptionRange(venues []AbsorptionVenue) (from, to time.Time, ok bool) {
	thin := make([]CVDVenueSeries, 0, len(venues))
	for _, v := range venues {
		s := CVDVenueSeries{Points: make([]CVDPoint, len(v.Points))}
		for i, p := range v.Points {
			s.Points[i] = CVDPoint{Time: p.Time}
		}
		thin = append(thin, s)
	}
	return overlapCVDRange(thin)
}
