package domain

import "time"

const (
	RecordedFuturesOI = "futures_oi"
	RecordedBook      = "book"
	RecordedFlow      = "flow"

	HistorySpanNote = "Span is how long this process has stored samples for the coin. The startup archive backfills the liquid USDT perpetuals, up to 36 names with the seed pairs first, and only as far as each public endpoint documents. A missing venue has no row. Informational only."
)

// RecordedSeries is the oldest stored sample for one kind and venue.
type RecordedSeries struct {
	Kind     string    `json:"kind"`
	Exchange Exchange  `json:"exchange"`
	Since    time.Time `json:"since"`
	SpanSec  int       `json:"spanSec"`
}

// HistorySpan says how long each stored series for a coin actually covers.
type HistorySpan struct {
	Symbol string           `json:"symbol"`
	AsOf   time.Time        `json:"asOf"`
	Series []RecordedSeries `json:"series"`
	Note   string           `json:"note"`
}

// BuildHistorySpan copies series that have a start time and sets each span
// from that start to now. Empty input still returns a report.
func BuildHistorySpan(symbol string, now time.Time, series []RecordedSeries) HistorySpan {
	now = now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	out := make([]RecordedSeries, 0, len(series))
	for _, row := range series {
		if row.Since.IsZero() || row.Exchange == "" || row.Kind == "" {
			continue
		}
		since := row.Since.UTC()
		span := int(now.Sub(since).Seconds())
		if span < 0 {
			span = 0
		}
		row.Since = since
		row.SpanSec = span
		out = append(out, row)
	}
	if out == nil {
		out = []RecordedSeries{}
	}
	return HistorySpan{
		Symbol: symbol,
		AsOf:   now,
		Series: out,
		Note:   HistorySpanNote,
	}
}
