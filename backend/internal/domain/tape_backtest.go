package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	TapeConditionCVD          = "cvd"
	TapeConditionLiquidation  = "liquidation"
	TapeConditionBook         = "book"
	TapeConditionFunding      = "funding"
	TapeConditionOpenInterest = "oi"
	TapeConditionGap          = "gap"
	TapeConditionSize         = "size"

	TapeSideBuy    = "buy"
	TapeSideSell   = "sell"
	TapeSideLong   = "long"
	TapeSideShort  = "short"
	TapeSideBid    = "bid"
	TapeSideAsk    = "ask"
	TapeSideUp     = "up"
	TapeSideDown   = "down"
	TapeSideTop    = "top"
	TapeSideCrowd  = "crowd"
	TapeSideWhale  = "whale"
	TapeSideRetail = "retail"

	// TapeCVDMinMinutes is a finished UTC hour. Fewer stored minutes is not a bar.
	TapeCVDMinMinutes = 50
	// TapeCVDHour is the clock the CVD share is scored on.
	TapeCVDHour = time.Hour

	TapeShareDefault = 0.65
	TapeShareMin     = 0.55
	TapeShareMax     = 0.90

	TapeLiqBucket          = 5 * time.Minute
	TapeLiqMultipleDefault = 4.0
	TapeLiqMultipleMin     = 2.0
	TapeLiqMultipleMax     = 12.0
	// TapeLiqMinBursts is how many 5-minute prints are required before a median exists.
	TapeLiqMinBursts = 5

	TapeBookStep = 15 * time.Minute

	// TapeFundingDefault is the settled funding rate, in percent (0.01 = 0.01%).
	TapeFundingDefault = 0.01
	TapeFundingMin     = 0.005
	TapeFundingMax     = 0.20

	// TapeOIDefault is the 24-hour open-interest change, in percent.
	TapeOIDefault = 5.0
	TapeOIMin     = 1.0
	TapeOIMax     = 50.0
	// TapeOISlack is how far a stored sample may sit from the hour, or from 24 hours earlier.
	TapeOISlack = 90 * time.Minute

	// TapeGapDefault is top-trader long share minus the crowd, in percentage points.
	TapeGapDefault = 3.0
	TapeGapMin     = 1.0
	TapeGapMax     = 20.0

	// TapeSizeDefault is the share of an hour's aggressive notional from prints of at least $100,000.
	TapeSizeDefault    = 0.40
	TapeSizeMin        = 0.15
	TapeSizeMax        = 0.80
	TapeSizeMinMinutes = TapeCVDMinMinutes

	TapeBacktestDefaultLookback = 7 * 24 * time.Hour
	TapeBacktestMaxLookback     = 30 * 24 * time.Hour
	TapeBacktestSignalLimit     = 100

	tapeHorizon1h  = "1h"
	tapeHorizon4h  = "4h"
	tapeHorizon24h = "24h"
)

// TapeBacktestQuery is one replay of stored tape for a single coin.
type TapeBacktestQuery struct {
	Symbol    string
	Exchange  string
	Condition string
	Side      string
	Threshold float64
	From      time.Time
	To        time.Time
	Now       time.Time
	Expected  []Exchange
}

// TapeBacktestInput is the stored series the replay scores. Prices come from candles.
type TapeBacktestInput struct {
	Query             TapeBacktestQuery
	Buckets           []TakerBucket
	Liquidations      []LiquidationEvent
	Books             []BookHistorySnapshot
	Snapshots         []FuturesSnapshot
	Candles           []Candle
	CandleStep        time.Duration
	PriceExchange     string
	VenueErrors       map[string]string
	LiquidationCapped bool
	BookCapped        bool
	FuturesCapped     bool
}

// TapeBacktestHorizon is one forward-return column.
type TapeBacktestHorizon struct {
	Horizon     string   `json:"horizon"`
	SampleCount int      `json:"sampleCount"`
	Average     *float64 `json:"average,omitempty"`
	Median      *float64 `json:"median,omitempty"`
	WinRate     *float64 `json:"winRate,omitempty"`
	// WorstDrop is the lowest spot close versus the entry on the way to this horizon.
	// Zero means price did not close below the entry. It stays empty until the horizon candle has closed.
	WorstDrop *float64 `json:"worstDrop,omitempty"`
}

// TapeBacktestSignal is one cross. Value is a 0–1 share, or a liquidation multiple.
type TapeBacktestSignal struct {
	At        time.Time `json:"at"`
	BarStart  time.Time `json:"barStart"`
	Side      string    `json:"side"`
	Value     float64   `json:"value"`
	Close     float64   `json:"close,omitempty"`
	Return1h  *float64  `json:"return1h,omitempty"`
	Return4h  *float64  `json:"return4h,omitempty"`
	Return24h *float64  `json:"return24h,omitempty"`
	Drop1h    *float64  `json:"drop1h,omitempty"`
	Drop4h    *float64  `json:"drop4h,omitempty"`
	Drop24h   *float64  `json:"drop24h,omitempty"`
}

// TapeBacktestVenue is one exchange, or the combined overlap.
type TapeBacktestVenue struct {
	Exchange    string                `json:"exchange"`
	SignalCount int                   `json:"signalCount"`
	Bars        int                   `json:"bars"`
	From        time.Time             `json:"from,omitempty"`
	To          time.Time             `json:"to,omitempty"`
	Horizons    []TapeBacktestHorizon `json:"horizons"`
	Signals     []TapeBacktestSignal  `json:"signals"`
	Error       string                `json:"error,omitempty"`
	Note        string                `json:"note,omitempty"`
}

// TapeBacktestReport is the synchronous replay. Combined is omitted when any
// expected venue has no complete bar. Definite is false in that case.
type TapeBacktestReport struct {
	Symbol        string              `json:"symbol"`
	Exchange      string              `json:"exchange"`
	Condition     string              `json:"condition"`
	Side          string              `json:"side"`
	Threshold     float64             `json:"threshold"`
	From          time.Time           `json:"from"`
	To            time.Time           `json:"to"`
	AsOf          time.Time           `json:"asOf"`
	PriceExchange string              `json:"priceExchange,omitempty"`
	Definite      bool                `json:"definite"`
	Missing       []string            `json:"missing"`
	Venues        []TapeBacktestVenue `json:"venues"`
	Combined      *TapeBacktestVenue  `json:"combined,omitempty"`
	Summary       string              `json:"summary"`
	Note          string              `json:"note"`
}

type shareBar struct {
	Start   time.Time
	End     time.Time
	Buy     float64
	Sell    float64
	Minutes int
}

type liqBar struct {
	Start    time.Time
	End      time.Time
	Notional float64
}

// levelBar is one finished UTC hour of a signed reading.
// Now and Then are the open-interest notionals behind Value when the reading is a 24h change.
type levelBar struct {
	Start time.Time
	End   time.Time
	Value float64
	Now   float64
	Then  float64
}

// ParseTapeBacktestQuery checks the public replay arguments.
// An empty condition is CVD. An empty window is the last 7 days through now.
func ParseTapeBacktestQuery(exchange, symbol, condition, side string, threshold float64, from, to, now time.Time) (TapeBacktestQuery, error) {
	sym, err := ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return TapeBacktestQuery{}, err
	}
	cond, err := parseTapeCondition(condition)
	if err != nil {
		return TapeBacktestQuery{}, err
	}
	sd, err := parseTapeSide(cond, side)
	if err != nil {
		return TapeBacktestQuery{}, err
	}
	th, err := parseTapeThreshold(cond, threshold)
	if err != nil {
		return TapeBacktestQuery{}, err
	}
	ex, expected, err := parseTapeExchange(cond, exchange)
	if err != nil {
		return TapeBacktestQuery{}, err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	if to.IsZero() || to.After(now) {
		to = now
	}
	to = to.UTC()
	if from.IsZero() {
		from = to.Add(-TapeBacktestDefaultLookback)
	}
	from = from.UTC()
	if !from.Before(to) {
		return TapeBacktestQuery{}, fmt.Errorf("%w: from must be before to", ErrInvalidArgument)
	}
	if to.Sub(from) > TapeBacktestMaxLookback {
		return TapeBacktestQuery{}, fmt.Errorf("%w: range must be %s or less", ErrInvalidArgument, TapeBacktestMaxLookback)
	}
	return TapeBacktestQuery{
		Symbol: sym, Exchange: ex, Condition: cond, Side: sd, Threshold: th,
		From: from, To: to, Now: now, Expected: expected,
	}, nil
}

// BuildTapeBacktest scores crosses and the 1h / 4h / 24h move that followed.
// A cross needs the previous bar on the same clock. Staying above the line
// does not fire again. A missing candle leaves that return blank.
func BuildTapeBacktest(in TapeBacktestInput) (*TapeBacktestReport, error) {
	q := in.Query
	if q.Symbol == "" || q.Condition == "" || len(q.Expected) == 0 {
		return nil, fmt.Errorf("%w: tape backtest query is incomplete", ErrInvalidArgument)
	}
	if in.CandleStep <= 0 {
		in.CandleStep = 5 * time.Minute
	}
	until := q.To.UTC()
	if !q.Now.IsZero() && q.Now.Before(until) {
		until = q.Now.UTC()
	}
	closes := tapeCloses(in.Candles, in.CandleStep, q.Now)
	rep := &TapeBacktestReport{
		Symbol: q.Symbol, Exchange: q.Exchange, Condition: q.Condition, Side: q.Side,
		Threshold: q.Threshold, From: q.From.UTC(), To: q.To.UTC(), AsOf: q.Now.UTC(),
		PriceExchange: in.PriceExchange, Missing: []string{}, Venues: []TapeBacktestVenue{},
	}
	var shareSets map[string][]shareBar
	var liqSets map[string][]liqBar
	var levelSets map[string][]levelBar
	switch q.Condition {
	case TapeConditionCVD:
		shareSets = map[string][]shareBar{}
		for _, ex := range q.Expected {
			bars, minutes := cvdHourBars(in.Buckets, ex, q.From, until)
			shareSets[string(ex)] = bars
			rep.Venues = append(rep.Venues, tapeShareVenue(string(ex), q, bars, minutes, in.VenueErrors[string(ex)], closes, in.CandleStep))
		}
	case TapeConditionSize:
		shareSets = map[string][]shareBar{}
		for _, ex := range q.Expected {
			bars, minutes := sizeHourBars(in.Buckets, ex, q.From, until)
			shareSets[string(ex)] = bars
			rep.Venues = append(rep.Venues, tapeShareVenue(string(ex), q, bars, minutes, in.VenueErrors[string(ex)], closes, in.CandleStep))
		}
	case TapeConditionLiquidation:
		liqSets = map[string][]liqBar{}
		for _, ex := range q.Expected {
			bars := liquidationBars(in.Liquidations, ex, q.Side, q.From, until)
			liqSets[string(ex)] = bars
			rep.Venues = append(rep.Venues, tapeLiqVenue(string(ex), q, bars, in.VenueErrors[string(ex)], closes, in.CandleStep))
		}
	case TapeConditionBook:
		shareSets = map[string][]shareBar{}
		for _, ex := range q.Expected {
			bars, samples := bookShareBars(in.Books, ex, q.From, until)
			shareSets[string(ex)] = bars
			rep.Venues = append(rep.Venues, tapeShareVenue(string(ex), q, bars, samples, in.VenueErrors[string(ex)], closes, in.CandleStep))
		}
	case TapeConditionFunding, TapeConditionOpenInterest, TapeConditionGap:
		levelSets = map[string][]levelBar{}
		for _, ex := range q.Expected {
			bars, note := tapeLevelBars(q.Condition, in.Snapshots, ex, q.From, until)
			errText := in.VenueErrors[string(ex)]
			if errText == "" {
				errText = note
			}
			if errText != "" {
				bars = nil
			}
			levelSets[string(ex)] = bars
			rep.Venues = append(rep.Venues, tapeLevelVenue(string(ex), q, bars, errText, closes, in.CandleStep))
		}
	default:
		return nil, fmt.Errorf("%w: unknown tape condition %q", ErrInvalidArgument, q.Condition)
	}
	for _, v := range rep.Venues {
		if v.Bars == 0 || v.Error != "" {
			rep.Missing = append(rep.Missing, v.Exchange)
		}
	}
	if q.Exchange == "all" && len(rep.Missing) == 0 {
		var combined TapeBacktestVenue
		switch q.Condition {
		case TapeConditionLiquidation:
			combined = tapeLiqVenue("all", q, combineLiqBars(liqSets, q.Expected), "", closes, in.CandleStep)
		case TapeConditionFunding, TapeConditionGap:
			combined = tapeLevelVenue("all", q, combineLevelBars(levelSets, q.Expected, "avg"), "", closes, in.CandleStep)
		case TapeConditionOpenInterest:
			combined = tapeLevelVenue("all", q, combineLevelBars(levelSets, q.Expected, "oi"), "", closes, in.CandleStep)
		default:
			combined = tapeShareVenue("all", q, combineShareBars(shareSets, q.Expected), 0, "", closes, in.CandleStep)
		}
		combined.Exchange = "all"
		rep.Combined = &combined
		rep.Definite = true
	} else if q.Exchange != "all" {
		rep.Definite = len(rep.Missing) == 0
	}
	rep.Note = tapeBacktestNote(in)
	rep.Summary = tapeBacktestSummary(rep)
	return rep, nil
}

func tapeShareVenue(name string, q TapeBacktestQuery, bars []shareBar, raw int, errText string, closes map[int64]float64, step time.Duration) TapeBacktestVenue {
	out := TapeBacktestVenue{
		Exchange: name, Bars: len(bars), Signals: []TapeBacktestSignal{}, Error: errText,
		Horizons: emptyTapeHorizons(),
	}
	if errText != "" {
		out.Bars = 0
		out.Note = errText
		return out
	}
	if len(bars) == 0 {
		if raw > 0 {
			out.Note = "Stored rows did not finish a bar."
		}
		return out
	}
	out.From = bars[0].Start
	out.To = bars[len(bars)-1].End
	stepBar := TapeCVDHour
	if q.Condition == TapeConditionBook {
		stepBar = TapeBookStep
	}
	sigs := make([]TapeBacktestSignal, 0)
	for _, hit := range crossShare(bars, q.Side, q.Threshold, stepBar) {
		sig := TapeBacktestSignal{At: hit.At, BarStart: hit.Start, Side: q.Side, Value: roundTo(hit.Value, 4)}
		attachTapeReturns(&sig, closes, step)
		sigs = append(sigs, sig)
	}
	out.SignalCount = len(sigs)
	out.Horizons = tapeHorizons(sigs)
	if len(sigs) > TapeBacktestSignalLimit {
		sigs = sigs[len(sigs)-TapeBacktestSignalLimit:]
		out.Note = fmt.Sprintf("Showing the newest %d of %d crosses.", TapeBacktestSignalLimit, out.SignalCount)
	}
	out.Signals = sigs
	return out
}

func tapeLiqVenue(name string, q TapeBacktestQuery, bars []liqBar, errText string, closes map[int64]float64, step time.Duration) TapeBacktestVenue {
	out := TapeBacktestVenue{
		Exchange: name, Bars: len(bars), Signals: []TapeBacktestSignal{}, Error: errText,
		Horizons: emptyTapeHorizons(),
	}
	if errText != "" {
		out.Bars = 0
		out.Note = errText
		return out
	}
	if len(bars) == 0 {
		return out
	}
	out.From = bars[0].Start
	out.To = bars[len(bars)-1].End
	positive := make([]float64, 0)
	for _, b := range bars {
		if b.Notional > 0 {
			positive = append(positive, b.Notional)
		}
	}
	if len(positive) < TapeLiqMinBursts {
		out.Note = fmt.Sprintf("Need at least %d printed 5-minute buckets to judge a typical burst.", TapeLiqMinBursts)
		out.Bars = len(positive)
		return out
	}
	med := medianFloat(positive)
	if med <= 0 {
		out.Note = "Typical 5-minute notional is zero."
		return out
	}
	level := med * q.Threshold
	sigs := make([]TapeBacktestSignal, 0)
	for i := 1; i < len(bars); i++ {
		if bars[i].Notional >= level && bars[i-1].Notional < level {
			mult := bars[i].Notional / med
			sig := TapeBacktestSignal{
				At: bars[i].End, BarStart: bars[i].Start, Side: q.Side, Value: roundTo(mult, 2),
			}
			attachTapeReturns(&sig, closes, step)
			sigs = append(sigs, sig)
		}
	}
	out.Bars = len(positive)
	out.SignalCount = len(sigs)
	out.Horizons = tapeHorizons(sigs)
	if len(sigs) > TapeBacktestSignalLimit {
		sigs = sigs[len(sigs)-TapeBacktestSignalLimit:]
		out.Note = fmt.Sprintf("Showing the newest %d of %d crosses.", TapeBacktestSignalLimit, out.SignalCount)
	}
	out.Signals = sigs
	return out
}

func tapeLevelVenue(name string, q TapeBacktestQuery, bars []levelBar, errText string, closes map[int64]float64, step time.Duration) TapeBacktestVenue {
	out := TapeBacktestVenue{
		Exchange: name, Bars: len(bars), Signals: []TapeBacktestSignal{}, Error: errText,
		Horizons: emptyTapeHorizons(),
	}
	if errText != "" {
		out.Bars = 0
		out.Note = errText
		return out
	}
	if len(bars) == 0 {
		return out
	}
	out.From = bars[0].Start
	out.To = bars[len(bars)-1].End
	sigs := make([]TapeBacktestSignal, 0)
	for _, hit := range crossLevel(bars, q.Side, q.Threshold, q.Condition == TapeConditionFunding) {
		if hit.At.Before(q.From) || hit.At.After(q.To) {
			continue
		}
		sig := TapeBacktestSignal{At: hit.At, BarStart: hit.Start, Side: q.Side, Value: roundTo(hit.Value, 4)}
		attachTapeReturns(&sig, closes, step)
		sigs = append(sigs, sig)
	}
	out.SignalCount = len(sigs)
	out.Horizons = tapeHorizons(sigs)
	if len(sigs) > TapeBacktestSignalLimit {
		sigs = sigs[len(sigs)-TapeBacktestSignalLimit:]
		out.Note = fmt.Sprintf("Showing the newest %d of %d crosses.", TapeBacktestSignalLimit, out.SignalCount)
	}
	out.Signals = sigs
	return out
}

func crossLevel(bars []levelBar, side string, threshold float64, consecutive bool) []shareHit {
	low := side == TapeSideShort || side == TapeSideDown || side == TapeSideCrowd
	hit := func(prev, b levelBar) (shareHit, bool) {
		if low {
			if b.Value <= -threshold && prev.Value > -threshold {
				return shareHit{Start: b.Start, At: b.End, Value: b.Value}, true
			}
			return shareHit{}, false
		}
		if b.Value >= threshold && prev.Value < threshold {
			return shareHit{Start: b.Start, At: b.End, Value: b.Value}, true
		}
		return shareHit{}, false
	}
	out := make([]shareHit, 0)
	if consecutive {
		for i := 1; i < len(bars); i++ {
			if row, ok := hit(bars[i-1], bars[i]); ok {
				out = append(out, row)
			}
		}
		return out
	}
	byStart := make(map[int64]levelBar, len(bars))
	for _, b := range bars {
		byStart[b.Start.UTC().UnixMilli()] = b
	}
	for _, b := range bars {
		prev, ok := byStart[b.Start.Add(-time.Hour).UTC().UnixMilli()]
		if !ok {
			continue
		}
		if row, ok := hit(prev, b); ok {
			out = append(out, row)
		}
	}
	return out
}

func tapeLevelBars(condition string, rows []FuturesSnapshot, ex Exchange, from, until time.Time) ([]levelBar, string) {
	switch condition {
	case TapeConditionFunding:
		return fundingPrintBars(rows, ex, from, until), ""
	case TapeConditionOpenInterest:
		return oiChangeBars(rows, ex, from, until), ""
	default:
		bars, unsupported := gapHourBars(rows, ex, from, until)
		if unsupported {
			return nil, "This venue does not publish a top-trader ratio."
		}
		return bars, ""
	}
}

func fundingPrintBars(rows []FuturesSnapshot, ex Exchange, from, until time.Time) []levelBar {
	type last struct {
		at  time.Time
		pct float64
	}
	// One settled print per UTC hour. Venues often settle 8 hours apart, so the
	// cross compares consecutive prints rather than the empty hours between them.
	hours := map[int64]last{}
	for _, row := range rows {
		if row.Predicted || row.SampledAt.IsZero() {
			continue
		}
		if row.Exchange != "" && row.Exchange != ex {
			continue
		}
		at := row.SampledAt.UTC()
		if at.After(until) || at.Before(from.Add(-24*time.Hour)) {
			continue
		}
		start := TruncateToBucket(at, time.Hour)
		cur, ok := hours[start.UnixMilli()]
		if ok && !at.After(cur.at) {
			continue
		}
		hours[start.UnixMilli()] = last{at: at, pct: row.FundingRate * 100}
	}
	out := make([]levelBar, 0, len(hours))
	for ms, row := range hours {
		start := time.UnixMilli(ms).UTC()
		out = append(out, levelBar{Start: start, End: row.at, Value: row.pct})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].End.Before(out[j].End) })
	return out
}

func oiChangeBars(rows []FuturesSnapshot, ex Exchange, from, until time.Time) []levelBar {
	pts := make([]tapePoint, 0)
	for _, row := range rows {
		if row.SampledAt.IsZero() || row.Value <= 0 {
			continue
		}
		if row.Exchange != "" && row.Exchange != ex {
			continue
		}
		pts = append(pts, tapePoint{at: row.SampledAt.UTC(), value: row.Value})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].at.Before(pts[j].at) })
	out := make([]levelBar, 0)
	start := TruncateToBucket(from, time.Hour).Add(-time.Hour)
	for ; !start.Add(time.Hour).After(until); start = start.Add(time.Hour) {
		end := start.Add(time.Hour)
		now, okNow := tapeSampleAt(pts, end)
		then, okThen := tapeSampleAt(pts, end.Add(-24*time.Hour))
		if !okNow || !okThen || then <= 0 {
			continue
		}
		out = append(out, levelBar{
			Start: start, End: end, Now: now, Then: then,
			Value: (now - then) / then * 100,
		})
	}
	return out
}

type tapePoint struct {
	at    time.Time
	value float64
}

func tapeSampleAt(pts []tapePoint, target time.Time) (float64, bool) {
	var value float64
	found := false
	for _, p := range pts {
		if p.at.After(target) {
			break
		}
		if target.Sub(p.at) > TapeOISlack {
			continue
		}
		value = p.value
		found = true
	}
	return value, found
}

func gapHourBars(rows []FuturesSnapshot, ex Exchange, from, until time.Time) ([]levelBar, bool) {
	type last struct {
		at  time.Time
		gap float64
	}
	hours := map[int64]last{}
	saw := false
	top := false
	for _, row := range rows {
		if row.SampledAt.IsZero() {
			continue
		}
		if row.Exchange != "" && row.Exchange != ex {
			continue
		}
		saw = true
		crowd := row.LongShare
		lead := row.SizeLongShare
		if lead <= 0 {
			lead = row.TopAccLongShare
		}
		if crowd <= 0 || lead <= 0 {
			continue
		}
		top = true
		at := row.SampledAt.UTC()
		start := TruncateToBucket(at, time.Hour)
		end := start.Add(time.Hour)
		if end.After(until) || start.Before(from.Add(-time.Hour)) {
			continue
		}
		cur, ok := hours[start.UnixMilli()]
		if ok && !at.After(cur.at) {
			continue
		}
		hours[start.UnixMilli()] = last{at: at, gap: (lead - crowd) * 100}
	}
	if saw && !top {
		return nil, true
	}
	out := make([]levelBar, 0, len(hours))
	for ms, row := range hours {
		start := time.UnixMilli(ms).UTC()
		out = append(out, levelBar{Start: start, End: start.Add(time.Hour), Value: row.gap})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, false
}

func combineLevelBars(sets map[string][]levelBar, expected []Exchange, mode string) []levelBar {
	idx := make(map[string]map[int64]levelBar, len(expected))
	seen := map[int64]struct{}{}
	starts := make([]int64, 0)
	for _, ex := range expected {
		m := make(map[int64]levelBar, len(sets[string(ex)]))
		for _, b := range sets[string(ex)] {
			ms := b.Start.UTC().UnixMilli()
			m[ms] = b
			if _, ok := seen[ms]; !ok {
				seen[ms] = struct{}{}
				starts = append(starts, ms)
			}
		}
		idx[string(ex)] = m
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	out := make([]levelBar, 0)
	for _, ms := range starts {
		var sum, now, then float64
		var start, end time.Time
		n := 0
		ok := true
		for _, ex := range expected {
			b, found := idx[string(ex)][ms]
			if !found {
				ok = false
				break
			}
			sum += b.Value
			now += b.Now
			then += b.Then
			start, end = b.Start, b.End
			n++
		}
		if !ok || n == 0 {
			continue
		}
		bar := levelBar{Start: start, End: end, Value: sum / float64(n), Now: now, Then: then}
		if mode == "oi" {
			if then <= 0 {
				continue
			}
			bar.Value = (now - then) / then * 100
		}
		out = append(out, bar)
	}
	return out
}

type shareHit struct {
	Start time.Time
	At    time.Time
	Value float64
}

func crossShare(bars []shareBar, side string, threshold float64, step time.Duration) []shareHit {
	byStart := make(map[int64]shareBar, len(bars))
	for _, b := range bars {
		byStart[b.Start.UTC().UnixMilli()] = b
	}
	out := make([]shareHit, 0)
	for _, b := range bars {
		prev, ok := byStart[b.Start.Add(-step).UTC().UnixMilli()]
		if !ok {
			continue
		}
		cur := b.share(side)
		old := prev.share(side)
		if cur >= threshold && old < threshold {
			out = append(out, shareHit{Start: b.Start, At: b.End, Value: cur})
		}
	}
	return out
}

func (b shareBar) share(side string) float64 {
	tot := b.Buy + b.Sell
	if tot <= 0 {
		return 0
	}
	if side == TapeSideSell || side == TapeSideAsk || side == TapeSideRetail {
		return b.Sell / tot
	}
	return b.Buy / tot
}

func cvdHourBars(buckets []TakerBucket, ex Exchange, from, until time.Time) ([]shareBar, int) {
	type cell struct{ buy, sell float64 }
	hours := map[int64]map[int64]*cell{}
	raw := 0
	for _, b := range buckets {
		if b.Start.IsZero() || (b.BuyNotional <= 0 && b.SellNotional <= 0) {
			continue
		}
		if b.Exchange != "" && b.Exchange != ex {
			continue
		}
		start := b.Start.UTC()
		if start.Before(from) || start.After(until) {
			continue
		}
		hour := TruncateToBucket(start, TapeCVDHour)
		end := hour.Add(TapeCVDHour)
		if hour.Before(from) || end.After(until) {
			continue
		}
		raw++
		minute := TruncateToBucket(start, time.Minute).UnixMilli()
		slot := hours[hour.UnixMilli()]
		if slot == nil {
			slot = map[int64]*cell{}
			hours[hour.UnixMilli()] = slot
		}
		c := slot[minute]
		if c == nil {
			c = &cell{}
			slot[minute] = c
		}
		c.buy += b.BuyNotional
		c.sell += b.SellNotional
	}
	out := make([]shareBar, 0, len(hours))
	for ms, slot := range hours {
		if len(slot) < TapeCVDMinMinutes {
			continue
		}
		bar := shareBar{Start: time.UnixMilli(ms).UTC(), End: time.UnixMilli(ms).UTC().Add(TapeCVDHour), Minutes: len(slot)}
		for _, c := range slot {
			bar.Buy += c.buy
			bar.Sell += c.sell
		}
		if bar.Buy+bar.Sell <= 0 {
			continue
		}
		out = append(out, bar)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, raw
}

func sizeHourBars(buckets []TakerBucket, ex Exchange, from, until time.Time) ([]shareBar, int) {
	type cell struct{ whale, retail float64 }
	hours := map[int64]map[int64]*cell{}
	raw := 0
	for _, b := range buckets {
		if !b.SizeKnown || b.Start.IsZero() {
			continue
		}
		whale := b.WhaleBuy + b.WhaleSell
		retail := b.RetailBuy + b.RetailSell
		if whale+retail <= 0 {
			continue
		}
		if b.Exchange != "" && b.Exchange != ex {
			continue
		}
		start := b.Start.UTC()
		if start.Before(from) || start.After(until) {
			continue
		}
		hour := TruncateToBucket(start, TapeCVDHour)
		end := hour.Add(TapeCVDHour)
		if hour.Before(from) || end.After(until) {
			continue
		}
		raw++
		minute := TruncateToBucket(start, time.Minute).UnixMilli()
		slot := hours[hour.UnixMilli()]
		if slot == nil {
			slot = map[int64]*cell{}
			hours[hour.UnixMilli()] = slot
		}
		c := slot[minute]
		if c == nil {
			c = &cell{}
			slot[minute] = c
		}
		c.whale += whale
		c.retail += retail
	}
	out := make([]shareBar, 0, len(hours))
	for ms, slot := range hours {
		if len(slot) < TapeSizeMinMinutes {
			continue
		}
		bar := shareBar{Start: time.UnixMilli(ms).UTC(), End: time.UnixMilli(ms).UTC().Add(TapeCVDHour), Minutes: len(slot)}
		for _, c := range slot {
			bar.Buy += c.whale
			bar.Sell += c.retail
		}
		if bar.Buy+bar.Sell <= 0 {
			continue
		}
		out = append(out, bar)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, raw
}

func bookShareBars(snaps []BookHistorySnapshot, ex Exchange, from, until time.Time) ([]shareBar, int) {
	type last struct {
		at  time.Time
		bid float64
		ask float64
	}
	blocks := map[int64]last{}
	raw := 0
	for _, s := range snaps {
		if s.SampledAt.IsZero() {
			continue
		}
		if s.Exchange != "" && s.Exchange != ex {
			continue
		}
		at := s.SampledAt.UTC()
		if at.Before(from) || at.After(until) {
			continue
		}
		bid, ask := s.BidNotional, s.AskNotional
		if bid+ask <= 0 {
			continue
		}
		raw++
		start := TruncateToBucket(at, TapeBookStep)
		end := start.Add(TapeBookStep)
		if end.After(until) {
			continue
		}
		cur, ok := blocks[start.UnixMilli()]
		if ok && at.Before(cur.at) {
			continue
		}
		blocks[start.UnixMilli()] = last{at: at, bid: bid, ask: ask}
	}
	out := make([]shareBar, 0, len(blocks))
	for ms, row := range blocks {
		start := time.UnixMilli(ms).UTC()
		out = append(out, shareBar{
			Start: start, End: start.Add(TapeBookStep), Buy: row.bid, Sell: row.ask, Minutes: 1,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out, raw
}

func liquidationBars(events []LiquidationEvent, ex Exchange, side string, from, until time.Time) []liqBar {
	acc := map[int64]float64{}
	var first, last time.Time
	for _, e := range events {
		if e.Time.IsZero() || e.Side != side || e.Notional <= 0 {
			continue
		}
		if e.Exchange != "" && e.Exchange != ex {
			continue
		}
		at := e.Time.UTC()
		if at.Before(from) || at.After(until) {
			continue
		}
		start := TruncateToBucket(at, TapeLiqBucket)
		end := start.Add(TapeLiqBucket)
		if end.After(until) {
			continue
		}
		acc[start.UnixMilli()] += e.Notional
		if first.IsZero() || start.Before(first) {
			first = start
		}
		if last.IsZero() || start.After(last) {
			last = start
		}
	}
	if first.IsZero() {
		return nil
	}
	out := make([]liqBar, 0)
	for t := first; !t.After(last); t = t.Add(TapeLiqBucket) {
		out = append(out, liqBar{Start: t, End: t.Add(TapeLiqBucket), Notional: acc[t.UnixMilli()]})
	}
	return out
}

func combineShareBars(sets map[string][]shareBar, expected []Exchange) []shareBar {
	idx := make(map[string]map[int64]shareBar, len(expected))
	seen := map[int64]struct{}{}
	starts := make([]int64, 0)
	for _, ex := range expected {
		m := make(map[int64]shareBar, len(sets[string(ex)]))
		for _, b := range sets[string(ex)] {
			ms := b.Start.UTC().UnixMilli()
			m[ms] = b
			if _, ok := seen[ms]; !ok {
				seen[ms] = struct{}{}
				starts = append(starts, ms)
			}
		}
		idx[string(ex)] = m
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	out := make([]shareBar, 0)
	for _, ms := range starts {
		var buy, sell float64
		var start, end time.Time
		ok := true
		for _, ex := range expected {
			b, found := idx[string(ex)][ms]
			if !found {
				ok = false
				break
			}
			buy += b.Buy
			sell += b.Sell
			start, end = b.Start, b.End
		}
		if !ok || buy+sell <= 0 {
			continue
		}
		out = append(out, shareBar{Start: start, End: end, Buy: buy, Sell: sell, Minutes: 1})
	}
	return out
}

func combineLiqBars(sets map[string][]liqBar, expected []Exchange) []liqBar {
	type span struct {
		from, to time.Time
		idx      map[int64]float64
		ok       bool
	}
	rows := make([]span, 0, len(expected))
	var from, to time.Time
	for _, ex := range expected {
		bars := sets[string(ex)]
		if len(bars) == 0 {
			return nil
		}
		sp := span{from: bars[0].Start, to: bars[len(bars)-1].Start, idx: map[int64]float64{}, ok: true}
		for _, b := range bars {
			sp.idx[b.Start.UTC().UnixMilli()] = b.Notional
		}
		rows = append(rows, sp)
		if from.IsZero() || sp.from.After(from) {
			from = sp.from
		}
		if to.IsZero() || sp.to.Before(to) {
			to = sp.to
		}
	}
	if from.IsZero() || to.Before(from) {
		return nil
	}
	out := make([]liqBar, 0)
	for t := from; !t.After(to); t = t.Add(TapeLiqBucket) {
		var sum float64
		for _, sp := range rows {
			sum += sp.idx[t.UnixMilli()]
		}
		out = append(out, liqBar{Start: t, End: t.Add(TapeLiqBucket), Notional: sum})
	}
	return out
}

func tapeCloses(candles []Candle, step time.Duration, now time.Time) map[int64]float64 {
	out := map[int64]float64{}
	if step <= 0 {
		return out
	}
	for _, c := range candles {
		if c.OpenTime.IsZero() {
			continue
		}
		open := TruncateToBucket(c.OpenTime.UTC(), step)
		closeAt := open.Add(step)
		if !now.IsZero() && closeAt.After(now) {
			continue
		}
		px, err := parseClose(c.Close)
		if err != nil || px <= 0 {
			continue
		}
		out[open.UTC().UnixMilli()] = px
	}
	return out
}

func attachTapeReturns(sig *TapeBacktestSignal, closes map[int64]float64, step time.Duration) {
	entry, ok := tapeCloseAt(closes, sig.At, step)
	if !ok {
		return
	}
	sig.Close = entry
	sig.Return1h = tapeForward(closes, sig.At, time.Hour, step, entry)
	sig.Return4h = tapeForward(closes, sig.At, 4*time.Hour, step, entry)
	sig.Return24h = tapeForward(closes, sig.At, 24*time.Hour, step, entry)
	if sig.Return1h != nil {
		sig.Drop1h = tapePathDrop(closes, sig.At, time.Hour, step, entry)
	}
	if sig.Return4h != nil {
		sig.Drop4h = tapePathDrop(closes, sig.At, 4*time.Hour, step, entry)
	}
	if sig.Return24h != nil {
		sig.Drop24h = tapePathDrop(closes, sig.At, 24*time.Hour, step, entry)
	}
}

func tapePathDrop(closes map[int64]float64, at time.Time, horizon, step time.Duration, entry float64) *float64 {
	if entry <= 0 || step <= 0 {
		return nil
	}
	end := at.Add(horizon)
	minPct := 0.0
	found := false
	for t := at.Add(step); !t.After(end); t = t.Add(step) {
		px, ok := tapeCloseAt(closes, t, step)
		if !ok || px <= 0 {
			continue
		}
		pct := (px - entry) / entry * 100
		if !found || pct < minPct {
			minPct = pct
			found = true
		}
	}
	if !found {
		return nil
	}
	if minPct > 0 {
		minPct = 0
	}
	v := roundTo(minPct, 4)
	return &v
}

func tapeCloseAt(closes map[int64]float64, at time.Time, step time.Duration) (float64, bool) {
	open := TruncateToBucket(at.UTC().Add(-step), step)
	px, ok := closes[open.UnixMilli()]
	return px, ok && px > 0
}

func tapeForward(closes map[int64]float64, at time.Time, horizon, step time.Duration, entry float64) *float64 {
	px, ok := tapeCloseAt(closes, at.Add(horizon), step)
	if !ok || entry <= 0 {
		return nil
	}
	pct := roundTo((px-entry)/entry*100, 4)
	return &pct
}

func emptyTapeHorizons() []TapeBacktestHorizon {
	return []TapeBacktestHorizon{
		{Horizon: tapeHorizon1h},
		{Horizon: tapeHorizon4h},
		{Horizon: tapeHorizon24h},
	}
}

func tapeHorizons(sigs []TapeBacktestSignal) []TapeBacktestHorizon {
	one := make([]float64, 0, len(sigs))
	four := make([]float64, 0, len(sigs))
	day := make([]float64, 0, len(sigs))
	drop1 := make([]float64, 0, len(sigs))
	drop4 := make([]float64, 0, len(sigs))
	drop24 := make([]float64, 0, len(sigs))
	for _, s := range sigs {
		if s.Return1h != nil && !math.IsNaN(*s.Return1h) {
			one = append(one, *s.Return1h)
		}
		if s.Return4h != nil && !math.IsNaN(*s.Return4h) {
			four = append(four, *s.Return4h)
		}
		if s.Return24h != nil && !math.IsNaN(*s.Return24h) {
			day = append(day, *s.Return24h)
		}
		if s.Drop1h != nil && !math.IsNaN(*s.Drop1h) {
			drop1 = append(drop1, *s.Drop1h)
		}
		if s.Drop4h != nil && !math.IsNaN(*s.Drop4h) {
			drop4 = append(drop4, *s.Drop4h)
		}
		if s.Drop24h != nil && !math.IsNaN(*s.Drop24h) {
			drop24 = append(drop24, *s.Drop24h)
		}
	}
	return []TapeBacktestHorizon{
		tapeHorizonStats(tapeHorizon1h, one, drop1),
		tapeHorizonStats(tapeHorizon4h, four, drop4),
		tapeHorizonStats(tapeHorizon24h, day, drop24),
	}
}

func tapeHorizonStats(id string, xs, drops []float64) TapeBacktestHorizon {
	out := TapeBacktestHorizon{Horizon: id, SampleCount: len(xs)}
	if len(xs) == 0 {
		return out
	}
	sum := 0.0
	wins := 0
	for _, v := range xs {
		sum += v
		if v > 0 {
			wins++
		}
	}
	avg := roundTo(sum/float64(len(xs)), 4)
	med := roundTo(medianFloat(xs), 4)
	win := roundTo(float64(wins)/float64(len(xs))*100, 2)
	out.Average = &avg
	out.Median = &med
	out.WinRate = &win
	if len(drops) > 0 {
		worst := drops[0]
		for _, d := range drops[1:] {
			if d < worst {
				worst = d
			}
		}
		w := roundTo(worst, 4)
		out.WorstDrop = &w
	}
	return out
}

func tapeBacktestSummary(rep *TapeBacktestReport) string {
	if rep == nil {
		return ""
	}
	n := 0
	for _, v := range rep.Venues {
		n += v.SignalCount
	}
	label := tapeSideLabel(rep.Condition, rep.Side)
	if n == 0 && (rep.Combined == nil || rep.Combined.SignalCount == 0) {
		if len(rep.Missing) > 0 && rep.Exchange == "all" {
			return fmt.Sprintf("No %s cross in this window. Combined is withheld because %s has no complete bar.", label, strings.Join(rep.Missing, ", "))
		}
		return fmt.Sprintf("No %s cross in this window.", label)
	}
	parts := make([]string, 0, len(rep.Venues)+1)
	if n > 0 {
		parts = append(parts, fmt.Sprintf("%d %s crosses", n, label))
	}
	if rep.Combined != nil {
		parts = append(parts, fmt.Sprintf("combined %d", rep.Combined.SignalCount))
	} else if rep.Exchange == "all" && len(rep.Missing) > 0 {
		parts = append(parts, "combined withheld ("+strings.Join(rep.Missing, ", ")+")")
	}
	return strings.Join(parts, ". ") + "."
}

func tapeSideLabel(condition, side string) string {
	switch condition {
	case TapeConditionLiquidation:
		if side == TapeSideShort {
			return "short-liquidation"
		}
		return "long-liquidation"
	case TapeConditionBook:
		if side == TapeSideAsk {
			return "ask-share"
		}
		return "bid-share"
	case TapeConditionFunding:
		if side == TapeSideShort {
			return "shorts-pay funding"
		}
		return "longs-pay funding"
	case TapeConditionOpenInterest:
		if side == TapeSideDown {
			return "open-interest drop"
		}
		return "open-interest rise"
	case TapeConditionGap:
		if side == TapeSideCrowd {
			return "crowd-richer gap"
		}
		return "top-richer gap"
	case TapeConditionSize:
		if side == TapeSideRetail {
			return "small-print share"
		}
		return "large-print share"
	default:
		if side == TapeSideSell {
			return "sell-share"
		}
		return "buy-share"
	}
}

func tapeBacktestNote(in TapeBacktestInput) string {
	var b strings.Builder
	b.WriteString("Replays stored tape. A CVD cross is a finished UTC hour with at least 50 minute buckets whose buy or sell share reaches the threshold after the previous hour finished below it. A liquidation burst is measured force-order prints: one 5-minute bucket at least the chosen multiple of that side's own median printed bucket, after the previous bucket was below. It is not an estimated leverage level. A book cross is the last spot ladder in a 15-minute block (Binance, Coinbase, Bybit; top of the stored book, not the perpetual book) whose bid or ask share reaches the threshold after the previous block was below. A funding cross is a settled print, in percent, after the previous hour was on the other side of the line. Predicted funding is left out. An open-interest cross is the percent change from a sample about 24 hours earlier. A missing sample is left out and is not zero. A crowd gap is top-trader long share minus account long share, in percentage points. Bybit does not publish a top-trader ratio, so a crowd-gap replay uses Binance and OKX. A large-print cross is a finished UTC hour where prints of at least $100,000 are at least the chosen share of aggressive notional. Minutes stored before print size was known are left out. Identical clip repeats are not stored, so this is the size share, not a count of matching prints. Combined uses only blocks every expected venue recorded. Forward returns are the spot 5-minute close at the signal, then 1 hour, 4 hours, and 24 hours later. Worst drop is the lowest close versus the entry on the way to that horizon. Zero means price did not close below the entry. A horizon stays blank until that candle has closed. ")
	if in.PriceExchange != "" {
		fmt.Fprintf(&b, "Prices are %s spot closes. ", in.PriceExchange)
	}
	if len(in.Candles) == 0 {
		b.WriteString("No spot candles were available, so forward returns are blank. ")
	}
	if in.LiquidationCapped {
		b.WriteString("This window hit the newest 2,500 liquidation prints, so older prints are not in the replay. ")
	} else if in.Query.Condition == TapeConditionLiquidation {
		b.WriteString("Liquidation history used here is capped at the newest 2,500 prints. ")
	}
	if in.BookCapped {
		b.WriteString("This window hit the newest 500 book snapshots, so the replay is only those latest samples. ")
	} else if in.Query.Condition == TapeConditionBook {
		b.WriteString("Book history used here is capped at the newest 500 snapshots. ")
	}
	if in.FuturesCapped {
		b.WriteString("This window hit the stored futures-sample cap, so older funding, open interest, or crowd samples are absent. ")
	}
	b.WriteString("Informational only — not financial advice.")
	return b.String()
}

func parseTapeCondition(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", TapeConditionCVD:
		return TapeConditionCVD, nil
	case TapeConditionLiquidation, "liquidations", "liq":
		return TapeConditionLiquidation, nil
	case TapeConditionBook, "imbalance":
		return TapeConditionBook, nil
	case TapeConditionFunding, "funding-rate", "funding_rate":
		return TapeConditionFunding, nil
	case TapeConditionOpenInterest, "open_interest", "open-interest":
		return TapeConditionOpenInterest, nil
	case TapeConditionGap, "crowd":
		return TapeConditionGap, nil
	case TapeConditionSize, "print-size", "print_size":
		return TapeConditionSize, nil
	default:
		return "", fmt.Errorf("%w: condition must be cvd, liquidation, book, funding, oi, gap, or size", ErrInvalidArgument)
	}
}

func parseTapeSide(condition, raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch condition {
	case TapeConditionCVD:
		if s == "" || s == TapeSideBuy {
			return TapeSideBuy, nil
		}
		if s == TapeSideSell {
			return TapeSideSell, nil
		}
		return "", fmt.Errorf("%w: cvd side must be buy or sell", ErrInvalidArgument)
	case TapeConditionLiquidation:
		if s == "" || s == TapeSideLong {
			return TapeSideLong, nil
		}
		if s == TapeSideShort {
			return TapeSideShort, nil
		}
		return "", fmt.Errorf("%w: liquidation side must be long or short", ErrInvalidArgument)
	case TapeConditionFunding:
		if s == "" || s == TapeSideLong {
			return TapeSideLong, nil
		}
		if s == TapeSideShort {
			return TapeSideShort, nil
		}
		return "", fmt.Errorf("%w: funding side must be long or short", ErrInvalidArgument)
	case TapeConditionOpenInterest:
		if s == "" || s == TapeSideUp {
			return TapeSideUp, nil
		}
		if s == TapeSideDown {
			return TapeSideDown, nil
		}
		return "", fmt.Errorf("%w: open-interest side must be up or down", ErrInvalidArgument)
	case TapeConditionGap:
		if s == "" || s == TapeSideTop {
			return TapeSideTop, nil
		}
		if s == TapeSideCrowd {
			return TapeSideCrowd, nil
		}
		return "", fmt.Errorf("%w: crowd-gap side must be top or crowd", ErrInvalidArgument)
	case TapeConditionSize:
		if s == "" || s == TapeSideWhale {
			return TapeSideWhale, nil
		}
		if s == TapeSideRetail {
			return TapeSideRetail, nil
		}
		return "", fmt.Errorf("%w: size side must be whale or retail", ErrInvalidArgument)
	default:
		if s == "" || s == TapeSideBid {
			return TapeSideBid, nil
		}
		if s == TapeSideAsk {
			return TapeSideAsk, nil
		}
		return "", fmt.Errorf("%w: book side must be bid or ask", ErrInvalidArgument)
	}
}

func parseTapeThreshold(condition string, raw float64) (float64, error) {
	if math.IsNaN(raw) || raw < 0 {
		return 0, fmt.Errorf("%w: threshold must be positive", ErrInvalidArgument)
	}
	switch condition {
	case TapeConditionLiquidation:
		if raw == 0 {
			return TapeLiqMultipleDefault, nil
		}
		if raw < TapeLiqMultipleMin || raw > TapeLiqMultipleMax {
			return 0, fmt.Errorf("%w: liquidation multiple must be between 2 and 12", ErrInvalidArgument)
		}
		return raw, nil
	case TapeConditionFunding:
		if raw == 0 {
			return TapeFundingDefault, nil
		}
		if raw < TapeFundingMin || raw > TapeFundingMax {
			return 0, fmt.Errorf("%w: funding threshold must be between 0.005 and 0.20 percent", ErrInvalidArgument)
		}
		return raw, nil
	case TapeConditionOpenInterest:
		if raw == 0 {
			return TapeOIDefault, nil
		}
		if raw < TapeOIMin || raw > TapeOIMax {
			return 0, fmt.Errorf("%w: open-interest threshold must be between 1 and 50 percent", ErrInvalidArgument)
		}
		return raw, nil
	case TapeConditionGap:
		if raw == 0 {
			return TapeGapDefault, nil
		}
		if raw < TapeGapMin || raw > TapeGapMax {
			return 0, fmt.Errorf("%w: crowd-gap threshold must be between 1 and 20 percentage points", ErrInvalidArgument)
		}
		return raw, nil
	case TapeConditionSize:
		if raw == 0 {
			return TapeSizeDefault, nil
		}
		if raw > 1 && raw <= 100 {
			raw = raw / 100
		}
		if raw < TapeSizeMin || raw > TapeSizeMax {
			return 0, fmt.Errorf("%w: large-print share must be between 0.15 and 0.80", ErrInvalidArgument)
		}
		return raw, nil
	default:
		if raw == 0 {
			return TapeShareDefault, nil
		}
		if raw > 1 && raw <= 100 {
			raw = raw / 100
		}
		if raw < TapeShareMin || raw > TapeShareMax {
			return 0, fmt.Errorf("%w: share threshold must be between 0.55 and 0.90", ErrInvalidArgument)
		}
		return raw, nil
	}
}

func parseTapeExchange(condition, raw string) (string, []Exchange, error) {
	if condition == TapeConditionGap {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" || s == "all" || s == "combined" {
			return "all", []Exchange{ExchangeBinance, ExchangeOKX}, nil
		}
		switch Exchange(s) {
		case ExchangeBinance, ExchangeOKX:
			return s, []Exchange{Exchange(s)}, nil
		default:
			return "", nil, fmt.Errorf("%w: crowd gap uses binance, okx, or all", ErrInvalidArgument)
		}
	}
	if condition == TapeConditionBook {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" || s == "all" || s == "combined" {
			return "all", []Exchange{ExchangeBinance, ExchangeCoinbase, ExchangeBybit}, nil
		}
		switch Exchange(s) {
		case ExchangeBinance, ExchangeCoinbase, ExchangeBybit:
			return s, []Exchange{Exchange(s)}, nil
		default:
			return "", nil, fmt.Errorf("%w: book exchange must be binance, coinbase, bybit, or all", ErrInvalidArgument)
		}
	}
	ex, err := ParseLiquidationExchange(raw)
	if err != nil {
		return "", nil, err
	}
	if ex == "all" {
		return "all", FuturesVenues(), nil
	}
	return ex, []Exchange{Exchange(ex)}, nil
}

func roundTo(v float64, places int) float64 {
	if places < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
