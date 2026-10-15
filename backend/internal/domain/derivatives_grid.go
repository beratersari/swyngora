package domain

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	GridMetricFunding      = "funding"
	GridMetricOI           = "oi"
	GridMetricCVD          = "cvd"
	GridMetricLiquidations = "liquidations"

	GridRange24h = "24h"
	GridRange7d  = "7d"

	// A closed hour needs three quarters of its minutes. A closed day needs
	// four fifths. The open bucket uses the same fraction of the time elapsed.
	gridHourMinFrac = 0.75
	gridDayMinFrac  = 0.80
	// gridEdgeSlack is how far the first and last stored minute may sit from
	// the bucket edges and still count as covering them.
	gridEdgeSlack = 2 * time.Minute
	// gridOpenMinAge keeps a bucket that just opened from looking finished.
	gridOpenMinAge = 2 * time.Minute
)

// GridSpec is one time-grid request after parsing.
type GridSpec struct {
	Metric string
	Range  string
	Step   time.Duration
}

// ParseDerivativesGrid accepts funding, oi, cvd, or liquidations over 24h or 7d.
func ParseDerivativesGrid(metric, rangeID string) (GridSpec, error) {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case GridMetricFunding, "funding_rate", "funding-rate":
		metric = GridMetricFunding
	case GridMetricOI, "open_interest", "open-interest", "openinterest":
		metric = GridMetricOI
	case GridMetricCVD, "day_cvd", "day-cvd":
		metric = GridMetricCVD
	case GridMetricLiquidations, "liquidation", "liq", "liq_heat", "liquidation_heat":
		metric = GridMetricLiquidations
	case "":
		return GridSpec{}, fmt.Errorf("%w: metric is required", ErrInvalidArgument)
	default:
		return GridSpec{}, fmt.Errorf("%w: metric must be funding, oi, cvd, or liquidations", ErrInvalidArgument)
	}
	switch strings.ToLower(strings.TrimSpace(rangeID)) {
	case "", GridRange24h, "1d", "day":
		return GridSpec{Metric: metric, Range: GridRange24h, Step: time.Hour}, nil
	case GridRange7d, "7day", "week":
		return GridSpec{Metric: metric, Range: GridRange7d, Step: 24 * time.Hour}, nil
	default:
		return GridSpec{}, fmt.Errorf("%w: range must be 24h or 7d", ErrInvalidArgument)
	}
}

// GridColumns are the bucket opens, oldest first, including the bucket that
// contains now. 24h is 24 clock hours. 7d is 7 UTC days.
func GridColumns(spec GridSpec, now time.Time) []time.Time {
	now = now.UTC()
	if spec.Step <= 0 {
		return nil
	}
	open := now.Truncate(spec.Step)
	n := 24
	if spec.Step >= 24*time.Hour {
		n = 7
	}
	cols := make([]time.Time, n)
	start := open.Add(-time.Duration(n-1) * spec.Step)
	for i := 0; i < n; i++ {
		cols[i] = start.Add(time.Duration(i) * spec.Step)
	}
	return cols
}

// GridSnapshot is the last funding or open-interest sample in one bucket.
type GridSnapshot struct {
	Metric      string
	Exchange    Exchange
	Symbol      string
	Bucket      time.Time
	SampledAt   time.Time
	Predicted   bool
	Value       float64
	FundingRate float64
}

// GridTakerSum is aggressive buy and sell notional in one bucket.
type GridTakerSum struct {
	Exchange Exchange
	Symbol   string
	Bucket   time.Time
	Buy      float64
	Sell     float64
	Minutes  int
	First    time.Time
	Last     time.Time
}

// GridLiquidationSum is stored force-order notional in one bucket.
type GridLiquidationSum struct {
	Exchange Exchange
	Symbol   string
	Bucket   time.Time
	Notional float64
}

// GridRead is the archive query for many symbols at once.
type GridRead struct {
	Metric  string
	Symbols []string
	From    time.Time
	To      time.Time
	Step    time.Duration
}

// DerivativesGridStore reads bucketed archive rows for the time grid.
type DerivativesGridStore interface {
	ListGridSnapshots(ctx context.Context, q GridRead) ([]GridSnapshot, error)
	ListGridTakerSums(ctx context.Context, q GridRead) ([]GridTakerSum, error)
	ListGridLiquidations(ctx context.Context, q GridRead) ([]GridLiquidationSum, error)
	ListLiquidationCoverage(ctx context.Context) ([]LiquidationCoverage, error)
}

// GridVenue is one exchange inside a time-grid cell. Value is set only when
// that exchange has the column (HasValue). A missing exchange is not a zero.
// A watched liquidation hour with no print is a zero, so HasValue stays true.
// Value uses omitempty; HTTP and MCP maps send a pointer when HasValue is set
// so a real zero is not dropped.
type GridVenue struct {
	Exchange string  `json:"exchange"`
	Value    float64 `json:"value,omitempty"`
	Text     string  `json:"text,omitempty"`
	Definite bool    `json:"definite"`
	HasValue bool    `json:"-"`
}

// GridCell is one coin at one bucket. Value is omitted from JSON when the
// cell is not definite, so a blank is not a zero. Venues still lists each
// expected exchange, including the ones that made the combined cell blank.
type GridCell struct {
	Time     time.Time   `json:"time"`
	Value    float64     `json:"value,omitempty"`
	Text     string      `json:"text,omitempty"`
	Heat     float64     `json:"heat,omitempty"`
	HasHeat  bool        `json:"-"`
	Definite bool        `json:"definite"`
	Missing  []string    `json:"missing,omitempty"`
	Weight   string      `json:"weight,omitempty"`
	Venues   []GridVenue `json:"venues,omitempty"`
}

// GridRow is one coin across the columns.
type GridRow struct {
	Symbol string     `json:"symbol"`
	Base   string     `json:"base,omitempty"`
	Cells  []GridCell `json:"cells"`
}

// DerivativesGrid is coins down the side and time across.
type DerivativesGrid struct {
	Exchange string      `json:"exchange"`
	Metric   string      `json:"metric"`
	Range    string      `json:"range"`
	StepSec  int         `json:"stepSec"`
	AsOf     time.Time   `json:"asOf"`
	Columns  []time.Time `json:"columns"`
	Rows     []GridRow   `json:"rows"`
	Universe int         `json:"universe"`
	Note     string      `json:"note,omitempty"`
}

const derivativesGridNote = "Coins down the side, time across, from stored samples. 24h is one column per clock hour. 7d is one column per UTC day. Funding is the latest settled rate in the cell, or the latest predicted rate when that hour or day has no settlement. Open interest is the percent change from the previous cell. CVD is aggressive buy minus sell. Liquidations are stored force-order notional; a watched hour with no print is a real zero. A cell stays blank until every expected venue has a sample for that column. A missing venue is named and is not counted as zero. Each cell lists every expected exchange and that exchange's own number when it has the column. The open column also stays blank when its newest sample is older than five minutes, or when the tape does not reach both edges of the column. Not financial advice."

// GridBuildInput is everything the pure builder needs.
type GridBuildInput struct {
	Exchange     string
	Spec         GridSpec
	Now          time.Time
	Symbols      []string
	Snapshots    []GridSnapshot
	Taker        []GridTakerSum
	Liquidations []GridLiquidationSum
	Coverage     []LiquidationCoverage
}

// BuildDerivativesGrid folds archive buckets into one row per coin.
func BuildDerivativesGrid(in GridBuildInput) *DerivativesGrid {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	spec := in.Spec
	cols := GridColumns(spec, now)
	expected := ExpectedFuturesVenues(in.Exchange)
	rows := make([]GridRow, 0, len(in.Symbols))
	for _, raw := range in.Symbols {
		symbol := NormalizeLiquidationSymbol(raw)
		if symbol == "" {
			continue
		}
		base, _ := SplitBaseQuote(ExchangeBinance, symbol)
		row := GridRow{Symbol: symbol, Base: base, Cells: make([]GridCell, len(cols))}
		for i, col := range cols {
			row.Cells[i] = gridCell(spec, symbol, col, expected, in, now)
		}
		if spec.Metric == GridMetricLiquidations {
			attachGridHeat(row.Cells)
		}
		rows = append(rows, row)
	}
	ex := strings.ToLower(strings.TrimSpace(in.Exchange))
	if ex == "" {
		ex = "all"
	}
	note := derivativesGridNote
	if IsMarkOnlyVenue(Exchange(ex)) {
		note += " Hyperliquid publishes mark, open interest, and hourly funding only. CVD and liquidation columns stay blank and name that venue. They are not zeros."
	}
	return &DerivativesGrid{
		Exchange: ex,
		Metric:   spec.Metric,
		Range:    spec.Range,
		StepSec:  int(spec.Step.Seconds()),
		AsOf:     now,
		Columns:  cols,
		Rows:     rows,
		Universe: len(rows),
		Note:     note,
	}
}

func gridCell(spec GridSpec, symbol string, col time.Time, expected []Exchange, in GridBuildInput, now time.Time) GridCell {
	cell := GridCell{Time: col}
	switch spec.Metric {
	case GridMetricFunding:
		gridFundingCell(&cell, symbol, col, spec.Step, expected, in.Snapshots, now)
	case GridMetricOI:
		gridOICell(&cell, symbol, col, spec.Step, expected, in.Snapshots, now)
	case GridMetricCVD:
		gridCVDCell(&cell, symbol, col, spec.Step, expected, in.Taker, now)
	case GridMetricLiquidations:
		gridLiqCell(&cell, symbol, col, spec.Step, expected, in, now)
	}
	return cell
}

type gridPoint struct {
	at    time.Time
	value float64
	rate  float64
	pred  bool
	ok    bool
}

func gridFundingCell(cell *GridCell, symbol string, col time.Time, step time.Duration, expected []Exchange, rows []GridSnapshot, now time.Time) {
	var rates, weights []float64
	var weightName string
	allOI := true
	cell.Venues = make([]GridVenue, 0, len(expected))
	for _, ex := range expected {
		pt := gridPickSnapshot(rows, symbol, ex, col, GridMetricFunding)
		venue := GridVenue{Exchange: string(ex)}
		if !gridSampleFresh(pt, col, step, now) {
			cell.Missing = append(cell.Missing, string(ex))
			allOI = false
			cell.Venues = append(cell.Venues, venue)
			continue
		}
		venue.Definite = true
		venue.HasValue = true
		venue.Value = pt.rate
		venue.Text = formatGridFunding(pt.rate)
		cell.Venues = append(cell.Venues, venue)
		oi := gridPickSnapshot(rows, symbol, ex, col, GridMetricOI)
		if !gridSampleFresh(oi, col, step, now) || oi.value <= 0 {
			allOI = false
		} else {
			weights = append(weights, oi.value)
		}
		rates = append(rates, pt.rate)
	}
	if len(cell.Missing) > 0 || len(rates) != len(expected) {
		return
	}
	rate := 0.0
	weightName = "equal"
	if allOI && len(weights) == len(rates) {
		weightName = "oi"
		var sum, w float64
		for i, r := range rates {
			sum += r * weights[i]
			w += weights[i]
		}
		if w > 0 {
			rate = sum / w
		}
	} else {
		for _, r := range rates {
			rate += r
		}
		rate /= float64(len(rates))
	}
	cell.Definite = true
	cell.Value = rate
	cell.Text = formatGridFunding(rate)
	cell.Weight = weightName
	cell.Missing = nil
}

func gridOICell(cell *GridCell, symbol string, col time.Time, step time.Duration, expected []Exchange, rows []GridSnapshot, now time.Time) {
	prevCol := col.Add(-step)
	var nowSum, prevSum float64
	cell.Venues = make([]GridVenue, 0, len(expected))
	for _, ex := range expected {
		cur := gridPickSnapshot(rows, symbol, ex, col, GridMetricOI)
		prev := gridPickSnapshot(rows, symbol, ex, prevCol, GridMetricOI)
		venue := GridVenue{Exchange: string(ex)}
		if !gridSampleFresh(cur, col, step, now) || !gridSampleFresh(prev, prevCol, step, now) || prev.value <= 0 {
			cell.Missing = append(cell.Missing, string(ex))
			cell.Venues = append(cell.Venues, venue)
			continue
		}
		pct := (cur.value - prev.value) / prev.value * 100
		venue.Definite = true
		venue.HasValue = true
		venue.Value = pct
		venue.Text = signedPct(pct)
		cell.Venues = append(cell.Venues, venue)
		nowSum += cur.value
		prevSum += prev.value
	}
	if len(cell.Missing) > 0 || prevSum <= 0 {
		return
	}
	pct := (nowSum - prevSum) / prevSum * 100
	cell.Definite = true
	cell.Value = pct
	cell.Text = signedPct(pct)
	cell.Missing = nil
}

func gridCVDCell(cell *GridCell, symbol string, col time.Time, step time.Duration, expected []Exchange, rows []GridTakerSum, now time.Time) {
	var delta float64
	cell.Venues = make([]GridVenue, 0, len(expected))
	for _, ex := range expected {
		sum, ok := gridTaker(rows, symbol, ex, col)
		venue := GridVenue{Exchange: string(ex)}
		if !ok || !takerBucketReady(sum, col, step, now) {
			cell.Missing = append(cell.Missing, string(ex))
			cell.Venues = append(cell.Venues, venue)
			continue
		}
		part := sum.Buy - sum.Sell
		venue.Definite = true
		venue.HasValue = true
		venue.Value = part
		venue.Text = signedGridUSD(part)
		cell.Venues = append(cell.Venues, venue)
		delta += part
	}
	if len(cell.Missing) > 0 {
		return
	}
	cell.Definite = true
	cell.Value = delta
	cell.Text = signedGridUSD(delta)
	cell.Missing = nil
}

func gridLiqCell(cell *GridCell, symbol string, col time.Time, step time.Duration, expected []Exchange, in GridBuildInput, now time.Time) {
	var total float64
	cell.Venues = make([]GridVenue, 0, len(expected))
	for _, ex := range expected {
		venue := GridVenue{Exchange: string(ex)}
		if !gridLiqWatched(in.Coverage, ex, symbol, col, step, now) {
			cell.Missing = append(cell.Missing, string(ex))
			cell.Venues = append(cell.Venues, venue)
			continue
		}
		n := gridLiqNotional(in.Liquidations, symbol, ex, col)
		venue.Definite = true
		venue.HasValue = true
		venue.Value = n
		venue.Text = compactUSD(n)
		cell.Venues = append(cell.Venues, venue)
		total += n
	}
	if len(cell.Missing) > 0 {
		return
	}
	cell.Definite = true
	cell.Value = total
	cell.Text = compactUSD(total)
	cell.Missing = nil
}

func sameGridMetric(row, want string) bool {
	switch want {
	case GridMetricOI:
		return row == GridMetricOI || row == FuturesMetricOpenInterest
	case GridMetricFunding:
		return row == GridMetricFunding || row == "funding_rate"
	default:
		return row == want
	}
}

func gridPickSnapshot(rows []GridSnapshot, symbol string, ex Exchange, bucket time.Time, metric string) gridPoint {
	var settled, latest gridPoint
	for _, row := range rows {
		if !sameGridMetric(row.Metric, metric) || row.Symbol != symbol || row.Exchange != ex || !row.Bucket.Equal(bucket) {
			continue
		}
		pt := gridPoint{at: row.SampledAt, value: row.Value, rate: row.FundingRate, pred: row.Predicted, ok: true}
		if metric == GridMetricFunding && !row.Predicted {
			if !settled.ok || !row.SampledAt.Before(settled.at) {
				settled = pt
			}
			continue
		}
		if !latest.ok || !row.SampledAt.Before(latest.at) {
			latest = pt
		}
	}
	if metric == GridMetricFunding && settled.ok {
		return settled
	}
	return latest
}

func gridSampleFresh(pt gridPoint, bucket time.Time, step time.Duration, now time.Time) bool {
	if !pt.ok || pt.at.IsZero() {
		return false
	}
	end := bucket.Add(step)
	if now.Before(end) && now.Sub(pt.at) > VenueDerivativesMaxAge {
		return false
	}
	return true
}

func gridTaker(rows []GridTakerSum, symbol string, ex Exchange, bucket time.Time) (GridTakerSum, bool) {
	for _, row := range rows {
		if row.Symbol == symbol && row.Exchange == ex && row.Bucket.Equal(bucket) {
			return row, true
		}
	}
	return GridTakerSum{}, false
}

func takerBucketReady(sum GridTakerSum, bucket time.Time, step time.Duration, now time.Time) bool {
	if sum.Minutes <= 0 || sum.First.IsZero() || sum.Last.IsZero() {
		return false
	}
	end := bucket.Add(step)
	observed := end
	if now.Before(end) {
		observed = now
	}
	span := observed.Sub(bucket)
	if span < gridOpenMinAge {
		return false
	}
	frac := gridHourMinFrac
	if step >= 24*time.Hour {
		frac = gridDayMinFrac
	}
	need := int(math.Ceil(span.Minutes() * frac))
	if need < 1 {
		need = 1
	}
	if sum.Minutes < need {
		return false
	}
	if sum.First.After(bucket.Add(gridEdgeSlack)) {
		return false
	}
	if sum.Last.Before(observed.Add(-gridEdgeSlack)) {
		return false
	}
	return true
}

func gridLiqNotional(rows []GridLiquidationSum, symbol string, ex Exchange, bucket time.Time) float64 {
	var n float64
	for _, row := range rows {
		if row.Symbol == symbol && row.Exchange == ex && row.Bucket.Equal(bucket) {
			n += row.Notional
		}
	}
	return n
}

func gridLiqWatched(rows []LiquidationCoverage, ex Exchange, symbol string, bucket time.Time, step time.Duration, now time.Time) bool {
	cov, ok := gridCoverage(rows, ex, symbol)
	if !ok || cov.FirstWatch.IsZero() || cov.FirstWatch.After(bucket) {
		return false
	}
	end := bucket.Add(step)
	if now.Before(end) {
		seen := cov.LastSeen
		if seen.IsZero() {
			seen = cov.LastEvent
		}
		if seen.IsZero() || now.Sub(seen) > liquidationFeedStall {
			return false
		}
	}
	for _, gap := range cov.Gaps {
		gEnd := gap.To
		if gEnd.IsZero() {
			gEnd = end
		}
		if gap.From.Before(end) && gEnd.After(bucket) {
			return false
		}
	}
	return true
}

func gridCoverage(rows []LiquidationCoverage, ex Exchange, symbol string) (LiquidationCoverage, bool) {
	var venue LiquidationCoverage
	venueOK := false
	for _, row := range rows {
		if row.Exchange != ex {
			continue
		}
		if row.Symbol == symbol {
			return row, true
		}
		if row.Symbol == "" {
			venue = row
			venueOK = true
		}
	}
	return venue, venueOK
}

func attachGridHeat(cells []GridCell) {
	vals := make([]float64, 0, len(cells))
	for _, cell := range cells {
		if cell.Definite {
			vals = append(vals, cell.Value)
		}
	}
	if len(vals) < 3 {
		return
	}
	med := medianFloat(vals)
	for i := range cells {
		if !cells[i].Definite {
			continue
		}
		heat := 1.0
		switch {
		case med > 0:
			heat = cells[i].Value / med
		case cells[i].Value > 0:
			heat = 4
		}
		cells[i].Heat = heat
		cells[i].HasHeat = true
	}
}

func formatGridFunding(rate float64) string {
	pct := rate * 100
	s := formatFixed(pct, 4)
	if pct > 0 && !strings.HasPrefix(s, "+") {
		s = "+" + s
	}
	return s + "%"
}

func signedGridUSD(v float64) string {
	if v < 0 {
		return "-" + compactUSD(-v)
	}
	if v > 0 {
		return "+" + compactUSD(v)
	}
	return "0"
}
