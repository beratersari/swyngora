package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	FundingPressureLookback  = 24 * time.Hour
	FundingPressureMaxPoints = 288 // 24h of 5m samples
	FundingPressureMaxMoves  = 8
	fundingPressureBucket    = 5 * time.Minute

	// ~0.002 of displayed ratePct (0.00002 decimal) to call a window steeper/flatter.
	fundingPressureWindowMove = 0.00002
	// Larger jump to call out a steepen/flatten event on the history.
	fundingPressureEventMove = 0.00008

	FundingPressureFlipped = "flipped"
	FundingPressureSteeper = "steeper"
	FundingPressureFlatter = "flatter"
	FundingPressureStable  = "stable"
)

// FundingPressureWindows is 1h / 4h / 24h of OI-weighted funding.
var FundingPressureWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{SnapshotWindow1h, time.Hour},
	{SnapshotWindow4h, 4 * time.Hour},
	{SnapshotWindow24h, 24 * time.Hour},
}

// FundingPressureVenue is one venue's contribution to combined pressure.
type FundingPressureVenue struct {
	Exchange          string
	Rate              string
	RatePct           string
	Payer             string
	OpenInterestValue string
	WeightPct         string // share of combined OI
}

// FundingPressure is predicted funding weighted by USDT open interest.
// Binance 8B at +0.01% and Bybit 1B at −0.01% is not a wash — the bigger book dominates.
type FundingPressure struct {
	Rate              string
	RatePct           string
	Payer             string
	OpenInterestValue string
	VenueCount        int
	DominantVenue     string
	Venues            []FundingPressureVenue
	NextFundingTime   time.Time
	IntervalHours     int
	Summary           string
	Time              time.Time
	rate              float64
}

// FundingPressurePoint is combined pressure at one sample time.
type FundingPressurePoint struct {
	Time              time.Time
	Rate              string
	RatePct           string
	Payer             string
	OpenInterestValue string
	VenueCount        int
	DominantVenue     string
}

// FundingPressureWindow is live pressure vs a lookback.
type FundingPressureWindow struct {
	Window      string
	PastRate    string
	PastRatePct string
	PastPayer   string
	Change      string // signed decimal
	ChangePct   string // signed displayed percent points
	Payer       string
	Kind        string // flipped | steeper | flatter | stable
	Complete    bool
	SampleTime  time.Time
}

// FundingPressureMove is when combined pressure flipped side or got much steeper/flatter.
type FundingPressureMove struct {
	Time      time.Time
	Kind      string
	FromPayer string
	ToPayer   string
	RatePct   string
	ChangePct string
	Summary   string
}

type fundingWeight struct {
	Exchange Exchange
	Rate     float64
	OI       float64
}

type fundingPressureRaw struct {
	Time   time.Time
	Rate   float64
	OI     float64
	Venues []fundingWeight
}

// AttachFundingPressure sets OI-weighted combined pressure from live venue prints.
func AttachFundingPressure(snap *FundingSnapshot, fund []*FundingSeries, oi []*OpenInterestSeries, now time.Time) {
	if snap == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	oiBy := map[Exchange]float64{}
	for _, s := range oi {
		if s == nil || s.Current.Value <= 0 {
			continue
		}
		oiBy[s.Exchange] = s.Current.Value
	}
	var weights []fundingWeight
	var next time.Time
	var interval int
	var nextOI float64
	for _, s := range fund {
		if s == nil {
			continue
		}
		v, ok := oiBy[s.Exchange]
		if !ok {
			continue
		}
		weights = append(weights, fundingWeight{Exchange: s.Exchange, Rate: s.Current.Rate, OI: v})
		if v > nextOI {
			nextOI = v
			next = s.NextFundingTime
			interval = s.IntervalHours
		}
	}
	raw, ok := blendFundingPressure(weights, now)
	if !ok {
		return
	}
	p := formatFundingPressure(raw)
	if !next.IsZero() {
		p.NextFundingTime = next.UTC()
	}
	p.IntervalHours = interval
	snap.Pressure = &p
}

// AttachFundingPressureHistory fills history, windows, and moves from stored
// predicted funding + OI samples (same 5m buckets as futures history).
func AttachFundingPressureHistory(snap *FundingSnapshot, funding, oi []FuturesSnapshot, now time.Time) {
	if snap == nil {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	pts := buildFundingPressureHistory(funding, oi, snap.Exchange, now)
	if snap.Pressure != nil {
		live := fundingPressureRaw{
			Time: now,
			Rate: snap.Pressure.rate,
			OI:   parseQty(snap.Pressure.OpenInterestValue),
		}
		if len(pts) == 0 || pts[len(pts)-1].Time.Before(now.Add(-fundingPressureBucket/2)) {
			pts = append(pts, live)
		} else {
			pts[len(pts)-1].Rate = live.Rate
			if live.OI > 0 {
				pts[len(pts)-1].OI = live.OI
			}
			pts[len(pts)-1].Time = now
		}
	}
	snap.PressureHistory = formatFundingPressurePoints(pts)
	liveRate := 0.0
	livePayer := ""
	if snap.Pressure != nil {
		liveRate = snap.Pressure.rate
		livePayer = snap.Pressure.Payer
	} else if len(pts) > 0 {
		last := pts[len(pts)-1]
		liveRate = last.Rate
		livePayer = FundingPayer(last.Rate)
	}
	snap.PressureWindows = fundingPressureWindows(pts, now, liveRate, livePayer)
	snap.PressureMoves = detectFundingPressureMoves(pts)
}

func blendFundingPressure(weights []fundingWeight, at time.Time) (fundingPressureRaw, bool) {
	var num, den float64
	used := make([]fundingWeight, 0, len(weights))
	for _, w := range weights {
		if w.OI <= 0 || math.IsNaN(w.Rate) || math.IsInf(w.Rate, 0) {
			continue
		}
		used = append(used, w)
		num += w.Rate * w.OI
		den += w.OI
	}
	if den <= 0 || len(used) == 0 {
		return fundingPressureRaw{}, false
	}
	sort.Slice(used, func(i, j int) bool {
		if used[i].OI == used[j].OI {
			return string(used[i].Exchange) < string(used[j].Exchange)
		}
		return used[i].OI > used[j].OI
	})
	return fundingPressureRaw{Time: at.UTC(), Rate: num / den, OI: den, Venues: used}, true
}

func formatFundingPressure(raw fundingPressureRaw) FundingPressure {
	dec, pct := FormatFundingRate(raw.Rate)
	payer := FundingPayer(raw.Rate)
	out := FundingPressure{
		Rate:              dec,
		RatePct:           pct,
		Payer:             payer,
		OpenInterestValue: formatQty(raw.OI),
		VenueCount:        len(raw.Venues),
		Venues:            make([]FundingPressureVenue, 0, len(raw.Venues)),
		Summary:           fundingPressureSummary(raw),
		Time:              raw.Time.UTC(),
		rate:              raw.Rate,
	}
	if raw.Time.IsZero() {
		out.Time = time.Time{}
	}
	if len(raw.Venues) > 0 {
		out.DominantVenue = string(raw.Venues[0].Exchange)
	}
	for _, w := range raw.Venues {
		d, p := FormatFundingRate(w.Rate)
		share := 0.0
		if raw.OI > 0 {
			share = w.OI / raw.OI * 100
		}
		out.Venues = append(out.Venues, FundingPressureVenue{
			Exchange:          string(w.Exchange),
			Rate:              d,
			RatePct:           p,
			Payer:             FundingPayer(w.Rate),
			OpenInterestValue: formatQty(w.OI),
			WeightPct:         formatFixed(share, 1),
		})
	}
	return out
}

func formatFundingPressurePoints(pts []fundingPressureRaw) []FundingPressurePoint {
	if len(pts) == 0 {
		return nil
	}
	out := make([]FundingPressurePoint, 0, len(pts))
	for _, p := range pts {
		dec, pct := FormatFundingRate(p.Rate)
		row := FundingPressurePoint{
			Time:              p.Time.UTC(),
			Rate:              dec,
			RatePct:           pct,
			Payer:             FundingPayer(p.Rate),
			OpenInterestValue: formatQty(p.OI),
			VenueCount:        len(p.Venues),
		}
		if p.Time.IsZero() {
			row.Time = time.Time{}
		}
		if len(p.Venues) > 0 {
			row.DominantVenue = string(p.Venues[0].Exchange)
		}
		out = append(out, row)
	}
	return out
}

func fundingPressureSummary(raw fundingPressureRaw) string {
	if len(raw.Venues) == 0 {
		return ""
	}
	_, pct := FormatFundingRate(raw.Rate)
	pay := fundingPayPhrase(FundingPayer(raw.Rate))
	if len(raw.Venues) == 1 {
		w := raw.Venues[0]
		return fmt.Sprintf("%s is the whole book at %s%% (%s).", fundingExName(w.Exchange), pct, pay)
	}
	parts := make([]string, 0, len(raw.Venues))
	for _, w := range raw.Venues {
		_, vp := FormatFundingRate(w.Rate)
		share := 0.0
		if raw.OI > 0 {
			share = w.OI / raw.OI * 100
		}
		parts = append(parts, fmt.Sprintf("%s %s%% of OI at %s%%", fundingExName(w.Exchange), formatFixed(share, 1), vp))
	}
	return fmt.Sprintf("%s. Combined pressure %s%% (%s).", strings.Join(parts, "; "), pct, pay)
}

func fundingExName(ex Exchange) string {
	if ex == ExchangeOKX {
		return "OKX"
	}
	return titleExchange(ex)
}

func fundingPayPhrase(payer string) string {
	switch payer {
	case "long":
		return "longs pay"
	case "short":
		return "shorts pay"
	default:
		return "flat"
	}
}

func buildFundingPressureHistory(funding, oi []FuturesSnapshot, exchange string, now time.Time) []fundingPressureRaw {
	from := now.Add(-FundingPressureLookback)
	type key struct {
		ex Exchange
		t  int64
	}
	rates := map[key]float64{}
	ois := map[key]float64{}
	buckets := map[int64]struct{}{}
	wantEx := func(ex Exchange) bool {
		if exchange == "" || exchange == "all" {
			return IsFuturesVenue(ex)
		}
		return string(ex) == exchange
	}
	for _, r := range funding {
		if !r.Predicted || !wantEx(r.Exchange) {
			continue
		}
		at := r.SampledAt.UTC()
		if at.IsZero() || at.Before(from) || at.After(now) {
			continue
		}
		t := TruncateToBucket(at, fundingPressureBucket)
		rates[key{r.Exchange, t.Unix()}] = r.FundingRate
		buckets[t.Unix()] = struct{}{}
	}
	for _, r := range oi {
		if !wantEx(r.Exchange) || r.Value <= 0 {
			continue
		}
		at := r.SampledAt.UTC()
		if at.IsZero() || at.Before(from) || at.After(now) {
			continue
		}
		t := TruncateToBucket(at, fundingPressureBucket)
		ois[key{r.Exchange, t.Unix()}] = r.Value
	}
	times := make([]int64, 0, len(buckets))
	for t := range buckets {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	out := make([]fundingPressureRaw, 0, len(times))
	prevSec := int64(fundingPressureBucket / time.Second)
	for _, ts := range times {
		var weights []fundingWeight
		for _, ex := range FuturesVenues() {
			if !wantEx(ex) {
				continue
			}
			rate, hasR := rates[key{ex, ts}]
			oiV, hasO := ois[key{ex, ts}]
			if !hasO {
				oiV, hasO = ois[key{ex, ts - prevSec}]
			}
			if !hasR || !hasO {
				continue
			}
			weights = append(weights, fundingWeight{Exchange: ex, Rate: rate, OI: oiV})
		}
		raw, ok := blendFundingPressure(weights, time.Unix(ts, 0).UTC())
		if ok {
			out = append(out, raw)
		}
	}
	if len(out) > FundingPressureMaxPoints {
		out = out[len(out)-FundingPressureMaxPoints:]
	}
	return out
}

func findFundingPressureSample(hist []fundingPressureRaw, target time.Time, slack time.Duration) (fundingPressureRaw, bool) {
	var best fundingPressureRaw
	found := false
	for _, p := range hist {
		if p.Time.IsZero() || p.Time.After(target) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	if !found {
		return fundingPressureRaw{}, false
	}
	return best, !best.Time.Before(target.Add(-slack))
}

func fundingPressureWindows(hist []fundingPressureRaw, now time.Time, liveRate float64, livePayer string) []FundingPressureWindow {
	out := make([]FundingPressureWindow, 0, len(FundingPressureWindows))
	for _, w := range FundingPressureWindows {
		row := FundingPressureWindow{Window: w.ID, Payer: livePayer}
		past, complete := findFundingPressureSample(hist, now.Add(-w.Dur), OpenInterestSampleSlack(w.Dur))
		if past.Time.IsZero() {
			out = append(out, row)
			continue
		}
		dec, pct := FormatFundingRate(past.Rate)
		dDec, dPct := formatSignedFunding(liveRate - past.Rate)
		row.PastRate = dec
		row.PastRatePct = pct
		row.PastPayer = FundingPayer(past.Rate)
		row.Change = dDec
		row.ChangePct = dPct
		row.Kind = fundingPressureKind(past.Rate, liveRate, fundingPressureWindowMove)
		row.Complete = complete
		row.SampleTime = past.Time.UTC()
		out = append(out, row)
	}
	return out
}

func detectFundingPressureMoves(hist []fundingPressureRaw) []FundingPressureMove {
	if len(hist) < 2 {
		return nil
	}
	var out []FundingPressureMove
	for i := 1; i < len(hist); i++ {
		prev, cur := hist[i-1], hist[i]
		kind := fundingPressureKind(prev.Rate, cur.Rate, fundingPressureEventMove)
		if kind == FundingPressureStable {
			continue
		}
		_, pct := FormatFundingRate(cur.Rate)
		_, dPct := formatSignedFunding(cur.Rate - prev.Rate)
		from, to := FundingPayer(prev.Rate), FundingPayer(cur.Rate)
		out = append(out, FundingPressureMove{
			Time:      cur.Time.UTC(),
			Kind:      kind,
			FromPayer: from,
			ToPayer:   to,
			RatePct:   pct,
			ChangePct: dPct,
			Summary:   fundingPressureMoveSummary(kind, from, to, pct),
		})
	}
	if len(out) > FundingPressureMaxMoves {
		out = out[len(out)-FundingPressureMaxMoves:]
	}
	return out
}

func fundingPressureKind(past, now, thresh float64) string {
	pastP, nowP := FundingPayer(past), FundingPayer(now)
	if pastP != nowP && pastP != "none" && nowP != "none" {
		return FundingPressureFlipped
	}
	dAbs := math.Abs(now) - math.Abs(past)
	switch {
	case dAbs > thresh:
		return FundingPressureSteeper
	case dAbs < -thresh:
		return FundingPressureFlatter
	default:
		return FundingPressureStable
	}
}

func fundingPressureMoveSummary(kind, from, to, ratePct string) string {
	switch kind {
	case FundingPressureFlipped:
		return fmt.Sprintf("Funding pressure flipped from %s to %s.", fundingPayPhrase(from), fundingPayPhrase(to))
	case FundingPressureSteeper:
		return fmt.Sprintf("Funding pressure got steeper at %s%% (%s).", ratePct, fundingPayPhrase(to))
	case FundingPressureFlatter:
		return fmt.Sprintf("Funding pressure got flatter at %s%% (%s).", ratePct, fundingPayPhrase(to))
	default:
		return ""
	}
}

func formatSignedFunding(delta float64) (dec, pct string) {
	dec, pct = FormatFundingRate(delta)
	if delta > 1e-12 {
		if !strings.HasPrefix(dec, "+") && !strings.HasPrefix(dec, "-") {
			dec = "+" + dec
		}
		if !strings.HasPrefix(pct, "+") && !strings.HasPrefix(pct, "-") {
			pct = "+" + pct
		}
	}
	return dec, pct
}


