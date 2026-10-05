package domain

import "time"

const (
	RecordedFuturesOI    = "futures_oi"
	RecordedBook         = "book"
	RecordedFlow         = "flow"
	RecordedLiquidations = "liquidations"

	HistorySpanNote = "Span is how long this process has stored samples for the coin. The startup archive backfills the liquid USDT perpetuals, up to 100 names with the seed pairs first, and only as far as each public endpoint documents. Binance USD-M aggTrades are walked into taker minutes and flow-at-prices for that same list, about 7 days, one hour at a time. Bybit and OKX flow stay on the minutes this process recorded. Spot books and flow-at-prices are kept about 30 days. Liquidation prints this process recorded are kept with the futures archive (about 90 days); public refill is still about 7 days. A missing venue has no row. Informational only."
)

// StoredSince is the oldest row for one stored symbol on one venue.
type StoredSince struct {
	Exchange Exchange
	Symbol   string
	Since    time.Time
}

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

// SpanQuerySymbols normalizes a recorder symbol list and drops blanks.
func SpanQuerySymbols(symbols []string) []string {
	out := make([]string, 0, len(symbols))
	seen := map[string]struct{}{}
	for _, raw := range symbols {
		s := NormalizeLiquidationSymbol(raw)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
		if len(out) >= LiquidRecorderLimit {
			break
		}
	}
	return out
}

// AttachRecordedSpans copies each coin's stored span onto the board.
// Span seconds are measured to the board clock. Coins with no rows stay empty.
func AttachRecordedSpans(board *DerivativesBoard, bySymbol map[string][]RecordedSeries) {
	if board == nil || len(bySymbol) == 0 {
		return
	}
	now := board.AsOf
	for i := range board.Coins {
		span := BuildHistorySpan(board.Coins[i].Symbol, now, bySymbol[board.Coins[i].Symbol])
		if len(span.Series) == 0 {
			continue
		}
		board.Coins[i].Recorded = span.Series
	}
}
