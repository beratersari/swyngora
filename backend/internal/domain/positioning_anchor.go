package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const positioningAnchorNote = "Open interest and funding restart at the clicked candle: each later sample minus the last sample at or before that candle. A venue with no sample by then is named and left off. Liquidation notional is the running sum from that candle, short up and long down. A print at or before the candle starts the sum at zero. A venue whose first print is later is late, and the line starts at that print. A venue with no prints in the window is named and left off. A missing venue is not zero. Informational only — not financial advice."

// PositioningAnchorPoint is one rebased sample.
type PositioningAnchorPoint struct {
	Time  time.Time `json:"time"`
	Value string    `json:"value"`
}

// PositioningAnchorSeries is one venue restarted at the clicked candle.
type PositioningAnchorSeries struct {
	Exchange string                   `json:"exchange"`
	Complete bool                     `json:"complete"`
	Change   string                   `json:"change"`
	Points   []PositioningAnchorPoint `json:"points"`
}

// PositioningAnchor is open interest, funding, and liquidation notional
// restarted at one candle for each futures venue.
type PositioningAnchor struct {
	Symbol              string                    `json:"symbol"`
	Exchange            string                    `json:"exchange"`
	Anchor              time.Time                 `json:"anchor"`
	Bucket              string                    `json:"bucket"`
	OpenInterest        []PositioningAnchorSeries `json:"openInterest"`
	Funding             []PositioningAnchorSeries `json:"funding"`
	Liquidations        []PositioningAnchorSeries `json:"liquidations"`
	MissingOpenInterest []string                  `json:"missingOpenInterest"`
	MissingFunding      []string                  `json:"missingFunding"`
	MissingLiquidations []string                  `json:"missingLiquidations"`
	LateLiquidations    []string                  `json:"lateLiquidations"`
	Note                string                    `json:"note"`
}

// PositioningAnchorInput is stored futures history around a clicked candle.
type PositioningAnchorInput struct {
	Symbol       string
	Exchange     string
	Anchor       time.Time
	Bucket       time.Duration
	OpenInterest []FuturesSnapshot
	Funding      []FuturesSnapshot
	Liquidations []LiquidationEvent
}

type anchorLevel struct {
	at        time.Time
	value     float64
	predicted bool
}

// BuildPositioningAnchor rebases stored positioning series at one candle.
// Open interest and funding need a sample at or before the candle.
// Liquidation notional sums prints from the candle forward.
func BuildPositioningAnchor(in PositioningAnchorInput) *PositioningAnchor {
	anchor := in.Anchor.UTC()
	ex := strings.ToLower(strings.TrimSpace(in.Exchange))
	if ex == "" {
		ex = "all"
	}
	label, err := FuturesBucketLabel(in.Bucket)
	if err != nil {
		label = ""
	}
	out := &PositioningAnchor{
		Symbol:              NormalizeLiquidationSymbol(in.Symbol),
		Exchange:            ex,
		Anchor:              anchor,
		Bucket:              label,
		OpenInterest:        []PositioningAnchorSeries{},
		Funding:             []PositioningAnchorSeries{},
		Liquidations:        []PositioningAnchorSeries{},
		MissingOpenInterest: []string{},
		MissingFunding:      []string{},
		MissingLiquidations: []string{},
		LateLiquidations:    []string{},
		Note:                positioningAnchorNote,
	}
	expected := ExpectedFuturesVenues(ex)
	oi := levelsByVenue(in.OpenInterest, func(row FuturesSnapshot) (float64, bool) {
		return row.Value, true
	})
	funding := levelsByVenue(preferSettledFunding(in.Funding), func(row FuturesSnapshot) (float64, bool) {
		return row.FundingRate * 100, true
	})
	liq := liqByVenue(in.Liquidations)
	for _, venue := range expected {
		name := string(venue)
		if series, ok := anchorLevelSeries(name, oi[venue], anchor, 2); ok {
			out.OpenInterest = append(out.OpenInterest, series)
		} else {
			out.MissingOpenInterest = append(out.MissingOpenInterest, name)
		}
		if series, ok := anchorLevelSeries(name, funding[venue], anchor, 6); ok {
			out.Funding = append(out.Funding, series)
		} else {
			out.MissingFunding = append(out.MissingFunding, name)
		}
		series, kind := anchorLiquidationSeries(name, liq[venue], anchor)
		switch kind {
		case "ready":
			out.Liquidations = append(out.Liquidations, series)
		case "late":
			out.Liquidations = append(out.Liquidations, series)
			out.LateLiquidations = append(out.LateLiquidations, name)
		default:
			out.MissingLiquidations = append(out.MissingLiquidations, name)
		}
	}
	return out
}

func levelsByVenue(rows []FuturesSnapshot, value func(FuturesSnapshot) (float64, bool)) map[Exchange][]anchorLevel {
	out := map[Exchange][]anchorLevel{}
	for _, row := range rows {
		if row.Exchange == "" || row.SampledAt.IsZero() {
			continue
		}
		v, ok := value(row)
		if !ok {
			continue
		}
		out[row.Exchange] = append(out[row.Exchange], anchorLevel{
			at: row.SampledAt.UTC(), value: v, predicted: row.Predicted,
		})
	}
	return out
}

func preferSettledFunding(rows []FuturesSnapshot) []FuturesSnapshot {
	settled := map[Exchange]bool{}
	for _, row := range rows {
		if row.Exchange != "" && !row.Predicted {
			settled[row.Exchange] = true
		}
	}
	if len(settled) == 0 {
		return rows
	}
	out := make([]FuturesSnapshot, 0, len(rows))
	for _, row := range rows {
		if settled[row.Exchange] && row.Predicted {
			continue
		}
		out = append(out, row)
	}
	return out
}

func anchorLevelSeries(exchange string, points []anchorLevel, anchor time.Time, decimals int) (PositioningAnchorSeries, bool) {
	if len(points) == 0 {
		return PositioningAnchorSeries{}, false
	}
	sort.SliceStable(points, func(i, j int) bool {
		if points[i].at.Equal(points[j].at) {
			return points[i].predicted && !points[j].predicted
		}
		return points[i].at.Before(points[j].at)
	})
	collapsed := make([]anchorLevel, 0, len(points))
	for _, point := range points {
		if n := len(collapsed); n > 0 && collapsed[n-1].at.Equal(point.at) {
			collapsed[n-1] = point
			continue
		}
		collapsed = append(collapsed, point)
	}
	baseline := -1
	for i, point := range collapsed {
		if !point.at.After(anchor) {
			baseline = i
		}
	}
	if baseline < 0 {
		return PositioningAnchorSeries{}, false
	}
	base := collapsed[baseline].value
	plotted := make([]PositioningAnchorPoint, 0, len(collapsed))
	for _, point := range collapsed[baseline:] {
		if point.at.Before(anchor) {
			continue
		}
		plotted = append(plotted, PositioningAnchorPoint{
			Time: point.at, Value: formatAnchorNumber(point.value-base, decimals),
		})
	}
	if len(plotted) == 0 || plotted[0].Time.After(anchor) {
		plotted = append([]PositioningAnchorPoint{{
			Time: anchor, Value: "0",
		}}, plotted...)
	}
	return PositioningAnchorSeries{
		Exchange: exchange,
		Complete: true,
		Change:   plotted[len(plotted)-1].Value,
		Points:   plotted,
	}, true
}

func liqByVenue(rows []LiquidationEvent) map[Exchange][]LiquidationEvent {
	out := map[Exchange][]LiquidationEvent{}
	for _, row := range rows {
		if row.Exchange == "" || row.Time.IsZero() {
			continue
		}
		out[row.Exchange] = append(out[row.Exchange], row)
	}
	return out
}

func anchorLiquidationSeries(exchange string, rows []LiquidationEvent, anchor time.Time) (PositioningAnchorSeries, string) {
	if len(rows) == 0 {
		return PositioningAnchorSeries{}, "missing"
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Time.Before(rows[j].Time) })
	reached := false
	after := make([]LiquidationEvent, 0, len(rows))
	for _, row := range rows {
		at := row.Time.UTC()
		if at.Before(anchor) {
			reached = true
			continue
		}
		after = append(after, row)
	}
	if !reached && len(after) == 0 {
		return PositioningAnchorSeries{}, "missing"
	}
	points := make([]PositioningAnchorPoint, 0, len(after)+1)
	sum := 0.0
	if reached && (len(after) == 0 || after[0].Time.UTC().After(anchor)) {
		points = append(points, PositioningAnchorPoint{Time: anchor, Value: "0"})
	}
	for _, row := range after {
		sum += signedLiquidationNotional(row.Side, row.Notional)
		points = append(points, PositioningAnchorPoint{
			Time: row.Time.UTC(), Value: formatAnchorNumber(sum, 2),
		})
	}
	if len(points) == 0 {
		return PositioningAnchorSeries{}, "missing"
	}
	kind := "late"
	if reached {
		kind = "ready"
	}
	return PositioningAnchorSeries{
		Exchange: exchange,
		Complete: reached,
		Change:   points[len(points)-1].Value,
		Points:   points,
	}, kind
}

func signedLiquidationNotional(side string, notional float64) float64 {
	n := notional
	if n < 0 {
		n = -n
	}
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "long":
		return -n
	case "short":
		return n
	default:
		return 0
	}
}

func formatAnchorNumber(v float64, decimals int) string {
	return formatFixed(v, decimals)
}

// PositioningAnchorBucket resolves the chart step for a window.
func PositioningAnchorBucket(raw string, from, to time.Time) (time.Duration, error) {
	if strings.TrimSpace(raw) != "" {
		return ParseFuturesBucket(raw)
	}
	if to.IsZero() || !to.After(from) {
		return 5 * time.Minute, nil
	}
	return FuturesPaneBucketForSpan(to.Sub(from)), nil
}

// ValidatePositioningAnchor rejects an empty click time.
func ValidatePositioningAnchor(anchor time.Time) error {
	if anchor.IsZero() {
		return fmt.Errorf("%w: anchor is required", ErrInvalidArgument)
	}
	return nil
}
