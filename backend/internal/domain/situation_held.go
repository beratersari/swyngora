package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// situationHeldStep is the stored minute the clocks walk. Futures samples
	// and CVD bars are both five minutes.
	situationHeldStep = 5 * time.Minute
	// situationHeldGap is the longest step that still counts as one series.
	// One missed five-minute sample stays in the run. A wider hole means the
	// archive does not reach the change, and the clock stays blank.
	situationHeldGap = 10 * time.Minute
)

// HeldSample is one stored minute of a clause.
// An empty Kind means that minute cannot be read. A hole inside the current
// run leaves the clock blank. It is not a new opinion.
type HeldSample struct {
	At   time.Time
	Kind string
}

// SituationPrintMinute is one venue's force-order notional inside a five-minute bucket.
type SituationPrintMinute struct {
	At       time.Time
	Exchange Exchange
	Long     float64
	Short    float64
}

// SituationHeldInput is the stored series already loaded for the live read.
// The estimated liquidation pocket is not among them.
type SituationHeldInput struct {
	Now          time.Time
	Symbol       string
	CVD          []CVDPoint
	OpenInterest []FuturesSnapshot
	Funding      []FundingPressurePoint
	Crowd        []LongShortVenueSnap
	Prints       []SituationPrintMinute
	Coverage     []LiquidationCoverage
}

// HeldSince is the first stored minute of kind.
// It is zero when the newest minute is not that kind, when an unreadable
// minute sits inside the run, when the minutes jump by more than situationHeldGap,
// or when the run reaches the oldest minute without a different kind.
func HeldSince(samples []HeldSample, kind string) time.Time {
	if kind == "" || len(samples) == 0 {
		return time.Time{}
	}
	rows := make([]HeldSample, 0, len(samples))
	for _, sample := range samples {
		if sample.At.IsZero() {
			continue
		}
		rows = append(rows, HeldSample{At: sample.At.UTC(), Kind: sample.Kind})
	}
	if len(rows) == 0 {
		return time.Time{}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].At.Equal(rows[j].At) {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].At.Before(rows[j].At)
	})
	compact := rows[:0]
	for _, sample := range rows {
		if len(compact) > 0 && compact[len(compact)-1].At.Equal(sample.At) {
			compact[len(compact)-1] = sample
			continue
		}
		compact = append(compact, sample)
	}
	rows = compact
	last := rows[len(rows)-1]
	if last.Kind != kind {
		return time.Time{}
	}
	start := last.At
	for i := len(rows) - 2; i >= 0; i-- {
		if rows[i+1].At.Sub(rows[i].At) > situationHeldGap {
			return time.Time{}
		}
		if rows[i].Kind == "" {
			return time.Time{}
		}
		if rows[i].Kind != kind {
			return start
		}
		start = rows[i].At
	}
	return time.Time{}
}

// StampSituationHeld sets Since on the clauses whose stored minutes show a change.
// A clause that is blank stays blank. The liquidation model is left alone.
func StampSituationHeld(sit *CoinSituation, in SituationHeldInput) {
	if sit == nil {
		return
	}
	if sit.PriceCVD != nil {
		sit.PriceCVD.Since = HeldSince(priceCVDHeldSamples(in.CVD), sit.PriceCVD.Kind)
	}
	if sit.Size != nil {
		sit.Size.Since = HeldSince(sizeHeldSamples(in.CVD), sit.Size.Kind)
	}
	if sit.OpenInterest != nil {
		sit.OpenInterest.Since = HeldSince(openInterestHeldSamples(in.OpenInterest, in.Now), sit.OpenInterest.Flow)
	}
	if sit.Funding != nil {
		sit.Funding.Since = HeldSince(fundingHeldSamples(in.Funding), sit.Funding.Payer)
	}
	if sit.Crowd != nil {
		sit.Crowd.Since = HeldSince(crowdHeldSamples(in.Crowd), sit.Crowd.Lean)
	}
	if sit.Prints != nil {
		sit.Prints.Since = HeldSince(printHeldSamples(in.Prints, in.Coverage, in.Symbol, in.Now), sit.Prints.Side)
	}
}

// WithoutSituationHeld returns a copy with the clocks cleared.
// The live read keeps its Since. A stored fill keeps the kind only.
func WithoutSituationHeld(sit *CoinSituation) *CoinSituation {
	if sit == nil {
		return nil
	}
	out := *sit
	if sit.PriceCVD != nil {
		c := *sit.PriceCVD
		c.Since = time.Time{}
		out.PriceCVD = &c
	}
	if sit.OpenInterest != nil {
		c := *sit.OpenInterest
		c.Since = time.Time{}
		out.OpenInterest = &c
	}
	if sit.Liquidation != nil {
		c := *sit.Liquidation
		c.Since = time.Time{}
		out.Liquidation = &c
	}
	if sit.Prints != nil {
		c := *sit.Prints
		c.Since = time.Time{}
		out.Prints = &c
	}
	if sit.Funding != nil {
		c := *sit.Funding
		c.Since = time.Time{}
		out.Funding = &c
	}
	if sit.Crowd != nil {
		c := *sit.Crowd
		c.Since = time.Time{}
		out.Crowd = &c
	}
	if sit.Size != nil {
		c := *sit.Size
		c.Since = time.Time{}
		out.Size = &c
	}
	return &out
}

// ClearSituationHeld drops clocks on this value.
// The stored read keeps the kind. The live card asks again.
func ClearSituationHeld(sit *CoinSituation) {
	if sit == nil {
		return
	}
	if sit.PriceCVD != nil {
		sit.PriceCVD.Since = time.Time{}
	}
	if sit.OpenInterest != nil {
		sit.OpenInterest.Since = time.Time{}
	}
	if sit.Liquidation != nil {
		sit.Liquidation.Since = time.Time{}
	}
	if sit.Prints != nil {
		sit.Prints.Since = time.Time{}
	}
	if sit.Funding != nil {
		sit.Funding.Since = time.Time{}
	}
	if sit.Crowd != nil {
		sit.Crowd.Since = time.Time{}
	}
	if sit.Size != nil {
		sit.Size.Since = time.Time{}
	}
}

// PrintMinutes folds force-order events into five-minute venue buckets.
// A side other than short counts as long, matching the stored sum.
func PrintMinutes(events []LiquidationEvent) []SituationPrintMinute {
	type key struct {
		ex Exchange
		at int64
	}
	by := map[key]SituationPrintMinute{}
	for _, event := range events {
		if event.Time.IsZero() || event.Notional <= 0 || event.Exchange == "" {
			continue
		}
		at := TruncateToBucket(event.Time.UTC(), situationHeldStep)
		k := key{ex: event.Exchange, at: at.Unix()}
		row := by[k]
		row.At = at
		row.Exchange = event.Exchange
		if event.Side == LiquidationSideShort {
			row.Short += event.Notional
		} else {
			row.Long += event.Notional
		}
		by[k] = row
	}
	out := make([]SituationPrintMinute, 0, len(by))
	for _, row := range by {
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].At.Equal(out[j].At) {
			return out[i].Exchange < out[j].Exchange
		}
		return out[i].At.Before(out[j].At)
	})
	return out
}

func priceCVDHeldSamples(points []CVDPoint) []HeldSample {
	if len(points) == 0 {
		return nil
	}
	out := make([]HeldSample, 0, len(points))
	for i := range points {
		prefix := points[:i+1]
		last := prefix[len(prefix)-1]
		series := CVDVenueSeries{Complete: true, Points: prefix, LastPrice: last.Price}
		series.Divergence = summarizeCVDDivergence(prefix, nil)
		kind := ""
		if got, ok, _ := situationPriceCVD(&CVDReport{Combined: &series}); ok {
			kind = got
		}
		out = append(out, HeldSample{At: last.Time, Kind: kind})
	}
	return out
}

func sizeHeldSamples(points []CVDPoint) []HeldSample {
	out := make([]HeldSample, 0, len(points))
	for _, point := range points {
		if point.Time.IsZero() {
			continue
		}
		kind := ""
		if point.SizeKnown {
			kind = situationSizeKind(point.WhaleDelta, point.RetailDelta)
		}
		out = append(out, HeldSample{At: point.Time, Kind: kind})
	}
	return out
}

func openInterestHeldSamples(rows []FuturesSnapshot, now time.Time) []HeldSample {
	byEx := map[Exchange][]readingOI{}
	var earliest, latest time.Time
	for _, row := range rows {
		if row.SampledAt.IsZero() || !isFuturesSituationVenue(row.Exchange) || row.Contracts <= 0 {
			continue
		}
		at := TruncateToBucket(row.SampledAt.UTC(), situationHeldStep)
		byEx[row.Exchange] = append(byEx[row.Exchange], readingOI{at: at, sampled: row.SampledAt.UTC(), contracts: row.Contracts})
		if earliest.IsZero() || at.Before(earliest) {
			earliest = at
		}
		if latest.IsZero() || at.After(latest) {
			latest = at
		}
	}
	for ex, list := range byEx {
		sort.Slice(list, func(i, j int) bool {
			if list[i].at.Equal(list[j].at) {
				return list[i].sampled.Before(list[j].sampled)
			}
			return list[i].at.Before(list[j].at)
		})
		compact := list[:0]
		for _, row := range list {
			if len(compact) > 0 && compact[len(compact)-1].at.Equal(row.at) {
				compact[len(compact)-1] = row
				continue
			}
			compact = append(compact, row)
		}
		byEx[ex] = compact
	}
	if earliest.IsZero() {
		return nil
	}
	if !now.IsZero() && now.UTC().After(latest) {
		latest = TruncateToBucket(now.UTC(), situationHeldStep)
	}
	slack := OpenInterestSampleSlack(time.Hour)
	start := earliest.Add(time.Hour)
	var out []HeldSample
	for t := start; !t.After(latest); t = t.Add(situationHeldStep) {
		kind := ""
		var cur, past float64
		complete := true
		for _, ex := range FuturesVenues() {
			curPt, curOK := oiReadingAt(byEx[ex], t, slack)
			pastPt, pastOK := oiReadingAt(byEx[ex], t.Add(-time.Hour), slack)
			_, pctOK := oiPctChange(curPt.contracts, pastPt.contracts)
			if !curOK || !pastOK || !pctOK {
				complete = false
				break
			}
			cur += curPt.contracts
			past += pastPt.contracts
		}
		if complete {
			pct, pctOK := oiPctChange(cur, past)
			if pctOK {
				kind = situationFlowFromDirection(OpenInterestDirection(pct))
			}
		}
		out = append(out, HeldSample{At: t, Kind: kind})
	}
	return out
}

func oiReadingAt(rows []readingOI, target time.Time, slack time.Duration) (readingOI, bool) {
	var best readingOI
	found := false
	for _, row := range rows {
		if row.at.After(target) {
			break
		}
		best = row
		found = true
	}
	if !found || best.at.Before(target.Add(-slack)) {
		return readingOI{}, false
	}
	return best, true
}

type readingOI struct {
	at        time.Time
	sampled   time.Time
	contracts float64
}

func situationFlowFromDirection(direction string) string {
	switch direction {
	case "up":
		return SituationOIOpened
	case "down":
		return SituationOIClosed
	default:
		return SituationOIFlat
	}
}

func fundingHeldSamples(points []FundingPressurePoint) []HeldSample {
	out := make([]HeldSample, 0, len(points))
	need := len(FuturesVenues())
	for _, point := range points {
		if point.Time.IsZero() {
			continue
		}
		kind := ""
		if point.VenueCount >= need && point.Payer != "" {
			kind = point.Payer
		}
		out = append(out, HeldSample{At: point.Time, Kind: kind})
	}
	return out
}

func crowdHeldSamples(venues []LongShortVenueSnap) []HeldSample {
	binance := crowdGapByBucket(venues, ExchangeBinance)
	okx := crowdGapByBucket(venues, ExchangeOKX)
	if len(binance) == 0 && len(okx) == 0 {
		return nil
	}
	seen := map[int64]struct{}{}
	for at := range binance {
		seen[at] = struct{}{}
	}
	for at := range okx {
		seen[at] = struct{}{}
	}
	times := make([]int64, 0, len(seen))
	for at := range seen {
		times = append(times, at)
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	out := make([]HeldSample, 0, len(times))
	for _, at := range times {
		b, bok := binance[at]
		o, ook := okx[at]
		kind := ""
		if bok && ook {
			kind = LongShortGapLean((b + o) / 2)
		}
		out = append(out, HeldSample{At: time.Unix(at, 0).UTC(), Kind: kind})
	}
	return out
}

func crowdGapByBucket(venues []LongShortVenueSnap, ex Exchange) map[int64]float64 {
	out := map[int64]float64{}
	for _, venue := range venues {
		if Exchange(strings.ToLower(strings.TrimSpace(venue.Exchange))) != ex {
			continue
		}
		for _, point := range venue.GapHistory {
			gap, ok := parseSituationGap(point.GapPct)
			if !ok || point.Time.IsZero() {
				continue
			}
			out[TruncateToBucket(point.Time.UTC(), situationHeldStep).Unix()] = gap
		}
		if venue.Gap != nil && !venue.Gap.Time.IsZero() {
			gap, ok := parseSituationGap(venue.Gap.GapPct)
			if ok {
				out[TruncateToBucket(venue.Gap.Time.UTC(), situationHeldStep).Unix()] = gap
			}
		}
	}
	return out
}

func parseSituationGap(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "+")
	s = strings.ReplaceAll(s, "−", "-")
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n != n {
		return 0, false
	}
	return n, true
}

func printHeldSamples(rows []SituationPrintMinute, coverage []LiquidationCoverage, symbol string, now time.Time) []HeldSample {
	if now.IsZero() || len(rows) == 0 {
		return nil
	}
	now = now.UTC()
	type side struct {
		long  float64
		short float64
	}
	byEx := map[Exchange][]struct {
		at          time.Time
		long, short float64
	}{}
	for _, row := range rows {
		if row.At.IsZero() || !isFuturesSituationVenue(row.Exchange) {
			continue
		}
		byEx[row.Exchange] = append(byEx[row.Exchange], struct {
			at          time.Time
			long, short float64
		}{at: row.At.UTC(), long: row.Long, short: row.Short})
	}
	for ex, list := range byEx {
		sort.Slice(list, func(i, j int) bool { return list[i].at.Before(list[j].at) })
		byEx[ex] = list
	}
	start := now.Add(-24 * time.Hour)
	var out []HeldSample
	for t := start; !t.After(now); t = t.Add(situationHeldStep) {
		windowFrom := t.Add(-24 * time.Hour)
		kind := ""
		recorded := true
		var longN, shortN float64
		for _, ex := range FuturesVenues() {
			if !situationPrintWindowRecorded(printCoverage(ex, symbol, coverage), windowFrom, t) {
				recorded = false
				break
			}
			longN, shortN = addPrintWindow(byEx[ex], windowFrom, t, longN, shortN)
		}
		if !recorded {
			if len(out) == 0 {
				continue
			}
			out = append(out, HeldSample{At: t})
			continue
		}
		kind = situationPrintSide(longN, shortN)
		out = append(out, HeldSample{At: t, Kind: kind})
	}
	if len(out) == 0 || out[len(out)-1].At.Equal(now) {
		return out
	}
	// The live clause is read at now, which may sit between two buckets.
	windowFrom := now.Add(-24 * time.Hour)
	recorded := true
	var longN, shortN float64
	for _, ex := range FuturesVenues() {
		if !situationPrintWindowRecorded(printCoverage(ex, symbol, coverage), windowFrom, now) {
			recorded = false
			break
		}
		longN, shortN = addPrintWindow(byEx[ex], windowFrom, now, longN, shortN)
	}
	if !recorded {
		return out
	}
	out = append(out, HeldSample{At: now, Kind: situationPrintSide(longN, shortN)})
	return out
}

func addPrintWindow(rows []struct {
	at          time.Time
	long, short float64
}, from, until time.Time, longN, shortN float64) (float64, float64) {
	for _, row := range rows {
		if row.at.Before(from) || !row.at.Before(until) {
			continue
		}
		longN += row.long
		shortN += row.short
	}
	return longN, shortN
}

func isFuturesSituationVenue(ex Exchange) bool {
	for _, want := range FuturesVenues() {
		if want == ex {
			return true
		}
	}
	return false
}
