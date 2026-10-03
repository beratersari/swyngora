package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	UnusualBandUnknown     = "unknown"
	UnusualBandExtremeHigh = "extreme_high"
	UnusualBandHigh        = "high"
	UnusualBandTypical     = "typical"
	UnusualBandLow         = "low"
	UnusualBandExtremeLow  = "extreme_low"

	UnusualReasonFewSamples      = "few_samples"
	UnusualReasonNoStoredHistory = "no_stored_history"
	UnusualReasonNoBook          = "no_book"
	UnusualReasonNoSample        = "no_sample"

	UnusualMetricFunding      = "funding"
	UnusualMetricOpenInterest = "open_interest"
	UnusualMetricLiquidations = "liquidations"
	UnusualMetricCVD          = "cvd"
	UnusualMetricSpread       = "spread"
	UnusualMetricPrintSize    = "print_size"

	UnusualLabelPredicted     = "predicted"
	UnusualLabelSettled       = "settled"
	UnusualLabel24hChange     = "24h_change"
	UnusualLabelCompletedHour = "completed_hour"
	UnusualLabelCompletedDay  = "completed_day"
	UnusualLabelLatestHour    = "latest_hour"

	UnusualUnitRate           = "rate"
	UnusualUnitPctChange      = "pct_change"
	UnusualUnitNotional       = "notional"
	UnusualUnitSignedNotional = "signed_notional"
	UnusualUnitSpreadPct      = "spread_pct"

	// UnusualMinSamples is the smallest baseline for 8-hour funding prints,
	// non-overlapping 24h open-interest changes, and completed CVD days.
	UnusualMinSamples = 20
	// UnusualMinHourlySamples is the smallest baseline for hourly series
	// (completed liquidation hours and average spread).
	UnusualMinHourlySamples = 24

	UnusualFundingFresh = 30 * time.Minute
	UnusualOIWindow     = 24 * time.Hour
	UnusualOISlack      = 2 * time.Hour
	// UnusualSpreadLookback is the query window for stored spot-book spread.
	// ORDERBOOK_HISTORY_RETENTION defaults to 30 days. A shorter store reports
	// the span it actually has.
	UnusualSpreadLookback = 30 * 24 * time.Hour

	// UnusualNote is the English note on the API payload. The product UI
	// builds its own sentence from localized copy.
	UnusualNote = "Each figure is compared with this coin's own stored history on that venue. The span is how long those samples actually cover. Each venue is scored on its own. Informational only — not financial advice."
)

// UnusualSample is one aggregated bucket (hour or day) from a store.
type UnusualSample struct {
	At    time.Time
	Value float64
}

// UnusualReading is one metric versus its own past samples.
// The current point is not part of Samples. Percentile is the midrank of
// the current value in that past, rounded to the nearest integer.
type UnusualReading struct {
	Metric     string        `json:"metric"`
	Band       string        `json:"band"`
	Reason     string        `json:"reason,omitempty"`
	Current    float64       `json:"current"`
	Median     float64       `json:"median"`
	Percentile float64       `json:"percentile"`
	Samples    int           `json:"samples"`
	MinSamples int           `json:"minSamples"`
	Span       time.Duration `json:"-"`
	SpanHours  float64       `json:"spanHours"`
	From       time.Time     `json:"from,omitempty"`
	To         time.Time     `json:"to,omitempty"`
	CurrentAt  time.Time     `json:"currentAt,omitempty"`
	Unit       string        `json:"unit,omitempty"`
	Label      string        `json:"label,omitempty"`
	Summary    string        `json:"summary"`
}

// UnusualVenue is one futures venue. Readings are not merged across venues.
type UnusualVenue struct {
	Exchange Exchange         `json:"exchange"`
	Readings []UnusualReading `json:"readings"`
}

// UnusualReport is the stored-history context for one coin.
type UnusualReport struct {
	Symbol   string         `json:"symbol"`
	Exchange string         `json:"exchange"`
	AsOf     time.Time      `json:"asOf"`
	Venues   []UnusualVenue `json:"venues"`
	Note     string         `json:"note"`
}

// FuturesUnusualReader loads aggregated futures history.
// ListSnapshots and ListLiquidations cap at MaxFuturesHistoryLimit and, for
// open interest, the newest rows are often 5-minute samples, so a percentile
// must not be computed from those lists.
type FuturesUnusualReader interface {
	HourlyOpenInterest(ctx context.Context, exchange, symbol string, from time.Time) ([]UnusualSample, error)
	SettledFunding(ctx context.Context, exchange, symbol string, from time.Time) ([]UnusualSample, error)
	LatestPredictedFunding(ctx context.Context, exchange, symbol string) (UnusualSample, bool, error)
	HourlyLiquidationNotional(ctx context.Context, exchange, symbol string, from time.Time) ([]UnusualSample, error)
}

// CVDUnusualReader loads daily signed taker notional (buy minus sell).
type CVDUnusualReader interface {
	DailyCVD(ctx context.Context, exchange, symbol string, from time.Time) ([]UnusualSample, error)
}

// SpreadUnusualReader loads hourly average top-of-book spread percent
// without the ladder payload.
type SpreadUnusualReader interface {
	HourlySpreadPct(ctx context.Context, exchange, symbol string, from time.Time) ([]UnusualSample, error)
}

// UnusualFunding scores the fresh predicted rate, or the latest settled
// print, against stored settlements only.
func UnusualFunding(now time.Time, settled []UnusualSample, predicted *UnusualSample) UnusualReading {
	now = now.UTC()
	past := groupSamples(settled, time.Hour, false)
	noun := "Settled funding"
	label := UnusualLabelSettled
	var current float64
	var currentAt time.Time
	var base []UnusualSample
	if predicted != nil && finite(predicted.Value) && !predicted.At.IsZero() {
		age := now.Sub(predicted.At.UTC())
		if age >= -time.Minute && age <= UnusualFundingFresh {
			current = predicted.Value
			currentAt = predicted.At.UTC()
			base = past
			noun = "Predicted funding"
			label = UnusualLabelPredicted
		}
	}
	if currentAt.IsZero() {
		if len(past) == 0 {
			return unknownReading(UnusualMetricFunding, UnusualReasonNoSample, UnusualMinSamples,
				"No stored funding settlements for this pair yet, so whether the rate is unusual is unknown.")
		}
		last := past[len(past)-1]
		current = last.Value
		currentAt = last.At
		base = past[:len(past)-1]
	}
	r := scoreAgainstPast(current, currentAt, base, UnusualMinSamples)
	r.Metric = UnusualMetricFunding
	r.Unit = UnusualUnitRate
	r.Label = label
	r.Summary = compareSentence(noun, "is", formatFundingRate(current), "stored settlements", r)
	return r
}

// UnusualOpenInterest scores the latest non-overlapping 24h percent change
// against earlier 24h changes. Each window needs an hourly point within
// UnusualOISlack of its start, and a zero or negative start is skipped.
func UnusualOpenInterest(points []UnusualSample) UnusualReading {
	hourly := groupSamples(points, time.Hour, false)
	windows := oiChangeWindows(hourly)
	if len(windows) == 0 {
		return unknownReading(UnusualMetricOpenInterest, UnusualReasonNoSample, UnusualMinSamples,
			"No stored hourly open interest covers a full 24 hour change, so whether that change is unusual is unknown.")
	}
	cur := windows[0]
	pastWins := windows[1:]
	past := make([]UnusualSample, len(pastWins))
	for i, w := range pastWins {
		past[i] = UnusualSample{At: w.End, Value: w.Pct}
	}
	r := scoreAgainstPast(cur.Pct, cur.End, past, UnusualMinSamples)
	if len(pastWins) > 0 {
		oldest := pastWins[len(pastWins)-1]
		newest := pastWins[0]
		r.From = oldest.Start
		r.To = newest.End
		r.Span = r.To.Sub(r.From)
		r.SpanHours = r.Span.Hours()
	}
	r.Metric = UnusualMetricOpenInterest
	r.Unit = UnusualUnitPctChange
	r.Label = UnusualLabel24hChange
	r.Summary = compareSentence("The latest 24h open-interest change", "is", FormatSignedPct(cur.Pct)+"%", "earlier 24h changes", r)
	return r
}

// UnusualLiquidations scores the last completed clock hour against earlier
// hours from the first stored print through the last. Quiet hours inside
// that span count as zero. The hour still in progress is left out.
func UnusualLiquidations(now time.Time, hourly []UnusualSample) UnusualReading {
	now = now.UTC()
	grouped := groupSamples(hourly, time.Hour, true)
	if len(grouped) == 0 {
		return unknownReading(UnusualMetricLiquidations, UnusualReasonNoSample, UnusualMinHourlySamples,
			"No stored liquidation prints cover a completed hour for this pair, so whether that hour is unusual is unknown.")
	}
	first := grouped[0].At
	last := grouped[len(grouped)-1].At
	hourNow := now.Truncate(time.Hour)
	end := last
	if !end.Before(hourNow) {
		end = hourNow.Add(-time.Hour)
	}
	if end.Before(first) {
		return unknownReading(UnusualMetricLiquidations, UnusualReasonNoSample, UnusualMinHourlySamples,
			"No stored liquidation prints cover a completed hour for this pair, so whether that hour is unusual is unknown.")
	}
	values := map[int64]float64{}
	for _, p := range grouped {
		if p.At.Before(first) || p.At.After(end) {
			continue
		}
		values[p.At.UnixMilli()] = p.Value
	}
	var series []UnusualSample
	for t := first; !t.After(end); t = t.Add(time.Hour) {
		series = append(series, UnusualSample{At: t, Value: values[t.UnixMilli()]})
	}
	cur := series[len(series)-1]
	r := scoreAgainstPast(cur.Value, cur.At, series[:len(series)-1], UnusualMinHourlySamples)
	r.Metric = UnusualMetricLiquidations
	r.Unit = UnusualUnitNotional
	r.Label = UnusualLabelCompletedHour
	r.Summary = compareSentence("Last completed hour liquidations", "are", "$"+compactUSD(cur.Value), "completed hours", r)
	return r
}

// UnusualCVD scores the last completed UTC day against earlier stored days.
// The day still in progress is left out. Days with no stored bars are left
// out too: a missing day means the feed was not recorded, not that flow was zero.
func UnusualCVD(now time.Time, daily []UnusualSample) UnusualReading {
	now = now.UTC()
	grouped := groupSamples(daily, 24*time.Hour, true)
	today := now.Truncate(24 * time.Hour)
	done := grouped[:0:0]
	for _, p := range grouped {
		if p.At.Before(today) {
			done = append(done, p)
		}
	}
	if len(done) == 0 {
		return unknownReading(UnusualMetricCVD, UnusualReasonNoSample, UnusualMinSamples,
			"No stored completed UTC day of aggressive flow for this pair yet, so whether that day is unusual is unknown.")
	}
	cur := done[len(done)-1]
	r := scoreAgainstPast(cur.Value, cur.At, done[:len(done)-1], UnusualMinSamples)
	r.Metric = UnusualMetricCVD
	r.Unit = UnusualUnitSignedNotional
	r.Label = UnusualLabelCompletedDay
	r.Summary = compareSentence("Last completed day's net aggressive flow", "is", signedCompactUSD(cur.Value), "earlier days", r)
	return r
}

// UnusualSpread scores the latest hourly average spread against earlier hours.
func UnusualSpread(hourly []UnusualSample) UnusualReading {
	grouped := groupSamples(hourly, time.Hour, false)
	if len(grouped) == 0 {
		return unknownReading(UnusualMetricSpread, UnusualReasonNoSample, UnusualMinHourlySamples,
			"No stored hourly spread for this pair yet, so whether the spread is unusual is unknown.")
	}
	cur := grouped[len(grouped)-1]
	r := scoreAgainstPast(cur.Value, cur.At, grouped[:len(grouped)-1], UnusualMinHourlySamples)
	r.Metric = UnusualMetricSpread
	r.Unit = UnusualUnitSpreadPct
	r.Label = UnusualLabelLatestHour
	r.Summary = compareSentence("The latest hour's average spread", "is", fmt.Sprintf("%.4f%%", cur.Value), "earlier hours", r)
	return r
}

// UnusualPrintSize is unknown. Aggressive print size is only the last
// 30 minutes (current 15 minutes versus the 15 minutes before). There is
// no stored series to score against.
func UnusualPrintSize() UnusualReading {
	return unknownReading(UnusualMetricPrintSize, UnusualReasonNoStoredHistory, 0,
		"Print size is only compared with the previous 15 minutes. There is no stored history long enough to say whether it is unusual.")
}

// UnusualNoBook is the spread reading for a venue that has no spot book.
func UnusualNoBook() UnusualReading {
	return unknownReading(UnusualMetricSpread, UnusualReasonNoBook, 0,
		"This venue has no spot book in this product, so spread versus its own history is unknown.")
}

type oiWindow struct {
	Start time.Time
	End   time.Time
	Pct   float64
}

func oiChangeWindows(hourly []UnusualSample) []oiWindow {
	if len(hourly) == 0 {
		return nil
	}
	byHour := map[int64]float64{}
	hours := make([]int64, 0, len(hourly))
	for _, p := range hourly {
		key := p.At.UnixMilli()
		if _, ok := byHour[key]; !ok {
			hours = append(hours, key)
		}
		byHour[key] = p.Value
	}
	end := hourly[len(hourly)-1]
	var out []oiWindow
	seen := map[int64]struct{}{}
	for len(out) < 120 {
		key := end.At.UnixMilli()
		if _, ok := seen[key]; ok {
			break
		}
		seen[key] = struct{}{}
		startAt, startVal, ok := nearestHour(hours, byHour, end.At.Add(-UnusualOIWindow), UnusualOISlack, end.At)
		if !ok {
			break
		}
		if startVal <= 0 {
			prev, ok := previousHour(hours, byHour, startAt)
			if !ok || !prev.Before(end.At) {
				break
			}
			end = UnusualSample{At: prev, Value: byHour[prev.UnixMilli()]}
			continue
		}
		out = append(out, oiWindow{
			Start: startAt,
			End:   end.At,
			Pct:   (end.Value - startVal) / startVal * 100,
		})
		end = UnusualSample{At: startAt, Value: startVal}
	}
	return out
}

func nearestHour(hours []int64, byHour map[int64]float64, boundary time.Time, slack time.Duration, before time.Time) (time.Time, float64, bool) {
	var best time.Time
	var bestVal float64
	bestDist := time.Duration(math.MaxInt64)
	found := false
	for _, h := range hours {
		t := time.UnixMilli(h).UTC()
		if !t.Before(before) {
			continue
		}
		d := t.Sub(boundary)
		if d < 0 {
			d = -d
		}
		if d > slack {
			continue
		}
		if !found || d < bestDist || (d == bestDist && t.After(best)) {
			best = t
			bestVal = byHour[h]
			bestDist = d
			found = true
		}
	}
	return best, bestVal, found
}

func previousHour(hours []int64, byHour map[int64]float64, before time.Time) (time.Time, bool) {
	var best time.Time
	found := false
	for _, h := range hours {
		t := time.UnixMilli(h).UTC()
		if !t.Before(before) {
			continue
		}
		if !found || t.After(best) {
			best = t
			found = true
		}
	}
	if !found {
		return time.Time{}, false
	}
	if _, ok := byHour[best.UnixMilli()]; !ok {
		return time.Time{}, false
	}
	return best, true
}

func scoreAgainstPast(current float64, currentAt time.Time, past []UnusualSample, minSamples int) UnusualReading {
	clean := make([]UnusualSample, 0, len(past))
	for _, p := range past {
		if p.At.IsZero() || !finite(p.Value) {
			continue
		}
		clean = append(clean, p)
	}
	sort.Slice(clean, func(i, j int) bool { return clean[i].At.Before(clean[j].At) })
	r := UnusualReading{
		Band:       UnusualBandUnknown,
		Current:    current,
		CurrentAt:  currentAt.UTC(),
		Samples:    len(clean),
		MinSamples: minSamples,
	}
	if len(clean) > 0 {
		r.From = clean[0].At
		r.To = clean[len(clean)-1].At
		r.Span = r.To.Sub(r.From)
		r.SpanHours = r.Span.Hours()
		vals := make([]float64, len(clean))
		for i, p := range clean {
			vals[i] = p.Value
		}
		r.Median = medianFloat(vals)
	}
	if len(clean) < minSamples {
		r.Reason = UnusualReasonFewSamples
		return r
	}
	vals := make([]float64, len(clean))
	for i, p := range clean {
		vals[i] = p.Value
	}
	r.Percentile = midrankPercentile(current, vals)
	r.Band = bandForPercentile(r.Percentile)
	r.Reason = ""
	return r
}

func midrankPercentile(current float64, past []float64) float64 {
	n := len(past)
	if n == 0 {
		return 0
	}
	below, equal := 0, 0
	for _, v := range past {
		switch {
		case v < current:
			below++
		case v == current:
			equal++
		}
	}
	pct := (float64(below) + 0.5*float64(equal)) / float64(n) * 100
	return math.Round(pct)
}

func bandForPercentile(p float64) string {
	switch {
	case p >= 98:
		return UnusualBandExtremeHigh
	case p <= 2:
		return UnusualBandExtremeLow
	case p >= 90:
		return UnusualBandHigh
	case p <= 10:
		return UnusualBandLow
	default:
		return UnusualBandTypical
	}
}

func groupSamples(in []UnusualSample, step time.Duration, sum bool) []UnusualSample {
	if step <= 0 {
		step = time.Hour
	}
	type cell struct {
		at  time.Time
		sum float64
		n   int
	}
	var order []int64
	cells := map[int64]*cell{}
	for _, p := range in {
		if p.At.IsZero() || !finite(p.Value) {
			continue
		}
		at := p.At.UTC().Truncate(step)
		key := at.UnixMilli()
		c := cells[key]
		if c == nil {
			c = &cell{at: at}
			cells[key] = c
			order = append(order, key)
		}
		c.sum += p.Value
		c.n++
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	out := make([]UnusualSample, 0, len(order))
	for _, key := range order {
		c := cells[key]
		v := c.sum
		if !sum && c.n > 0 {
			v = c.sum / float64(c.n)
		}
		out = append(out, UnusualSample{At: c.at, Value: v})
	}
	return out
}

func unknownReading(metric, reason string, minSamples int, summary string) UnusualReading {
	return UnusualReading{
		Metric: metric, Band: UnusualBandUnknown, Reason: reason,
		MinSamples: minSamples, Summary: summary,
	}
}

func compareSentence(noun, verb, value, versus string, r UnusualReading) string {
	if r.Band == UnusualBandUnknown {
		if r.Samples == 0 {
			return fmt.Sprintf("%s %s %s. No earlier samples are stored, and %d are needed to call it usual or unusual.",
				noun, verb, value, r.MinSamples)
		}
		return fmt.Sprintf("%s %s %s. %d stored samples over %s are below the %d needed to call it usual or unusual.",
			noun, verb, value, r.Samples, spanPhrase(r.Span), r.MinSamples)
	}
	return fmt.Sprintf("%s %s %s, %s versus %d %s over %s (percentile %.0f).",
		noun, verb, value, bandWord(r.Band), r.Samples, versus, spanPhrase(r.Span), r.Percentile)
}

func bandWord(band string) string {
	switch band {
	case UnusualBandExtremeHigh:
		return "extremely high"
	case UnusualBandHigh:
		return "high"
	case UnusualBandLow:
		return "low"
	case UnusualBandExtremeLow:
		return "extremely low"
	default:
		return "typical"
	}
}

func spanPhrase(d time.Duration) string {
	if d < time.Hour {
		return "under an hour"
	}
	if d < 36*time.Hour {
		h := int(math.Round(d.Hours()))
		if h < 1 {
			h = 1
		}
		if h == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", h)
	}
	days := int(math.Round(d.Hours() / 24))
	if days < 2 {
		days = 2
	}
	return fmt.Sprintf("%d days", days)
}

func formatFundingRate(rate float64) string {
	return fmt.Sprintf("%.4f%%", rate*100)
}

func signedCompactUSD(v float64) string {
	if v < 0 {
		return "-$" + compactUSD(-v)
	}
	if v > 0 {
		return "+$" + compactUSD(v)
	}
	return "$0"
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// MarshalJSON omits unset times. encoding/json omitempty keeps a zero
// time.Time because it is a struct, and that zero date is not a sample.
func (r UnusualReading) MarshalJSON() ([]byte, error) {
	type reading struct {
		Metric     string     `json:"metric"`
		Band       string     `json:"band"`
		Reason     string     `json:"reason,omitempty"`
		Current    float64    `json:"current"`
		Median     float64    `json:"median"`
		Percentile float64    `json:"percentile"`
		Samples    int        `json:"samples"`
		MinSamples int        `json:"minSamples"`
		SpanHours  float64    `json:"spanHours"`
		From       *time.Time `json:"from,omitempty"`
		To         *time.Time `json:"to,omitempty"`
		CurrentAt  *time.Time `json:"currentAt,omitempty"`
		Unit       string     `json:"unit,omitempty"`
		Label      string     `json:"label,omitempty"`
		Summary    string     `json:"summary"`
	}
	out := reading{
		Metric: r.Metric, Band: r.Band, Reason: r.Reason,
		Current: r.Current, Median: r.Median, Percentile: r.Percentile,
		Samples: r.Samples, MinSamples: r.MinSamples, SpanHours: r.SpanHours,
		Unit: r.Unit, Label: r.Label, Summary: r.Summary,
	}
	if t := utcPtr(r.From); t != nil {
		out.From = t
	}
	if t := utcPtr(r.To); t != nil {
		out.To = t
	}
	if t := utcPtr(r.CurrentAt); t != nil {
		out.CurrentAt = t
	}
	return json.Marshal(out)
}

func utcPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := t.UTC()
	return &u
}
