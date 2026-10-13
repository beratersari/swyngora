package domain

import (
	"sort"
	"strings"
	"time"
)

// OpenLotRead is one buy that still has remaining quantity.
// Situation is nil when that fill stored no read. Newest marks the buy that
// entrySituation and a situation alert still follow.
type OpenLotRead struct {
	ID            string
	Quantity      float64
	OpenedAt      time.Time
	SourceTradeID string
	Situation     *CoinSituation
	Newest        bool
}

// OpenLotChange is one remaining buy, optionally compared with a later read.
// Comparison is nil when that later read did not load.
type OpenLotChange struct {
	OpenLotRead
	Comparison *SituationComparison
}

// OpenLotReads lists remaining buys, newest first.
// A closed or flat lot is omitted. An equity book returns nil.
// byTrade is keyed by source-trade id. A missing fill leaves Situation nil
// and keeps the lot's own open time.
func OpenLotReads(ex Exchange, lots []TaxLot, byTrade map[string]Trade) []OpenLotRead {
	if IsEquityExchange(ex) {
		return nil
	}
	open := make([]TaxLot, 0, len(lots))
	for _, lot := range lots {
		if lot.Open() {
			open = append(open, lot)
		}
	}
	if len(open) == 0 {
		return nil
	}
	sort.Slice(open, func(i, j int) bool {
		return laterOpenLot(open[i], open[j])
	})
	out := make([]OpenLotRead, 0, len(open))
	for i, lot := range open {
		row := OpenLotRead{
			ID:            lot.ID,
			Quantity:      lot.Quantity,
			OpenedAt:      lot.OpenedAt.UTC(),
			SourceTradeID: strings.TrimSpace(lot.SourceTradeID),
			Newest:        i == 0,
		}
		if tr, ok := byTrade[row.SourceTradeID]; ok {
			if !tr.CreatedAt.IsZero() {
				row.OpenedAt = tr.CreatedAt.UTC()
			}
			row.Situation = tr.Situation
		}
		out = append(out, row)
	}
	return out
}

// EntryFromOpenReads is the newest remaining buy's stored read and fill time.
// A newest buy that stored no read returns a blank read and a zero time,
// even when an older buy stored one.
func EntryFromOpenReads(reads []OpenLotRead) (*CoinSituation, time.Time) {
	for _, read := range reads {
		if !read.Newest {
			continue
		}
		if read.Situation == nil {
			return nil, time.Time{}
		}
		return read.Situation, read.OpenedAt
	}
	return nil, time.Time{}
}

// OpenLotChanges copies remaining buys without a later-read comparison.
func OpenLotChanges(reads []OpenLotRead) []OpenLotChange {
	if len(reads) == 0 {
		return nil
	}
	out := make([]OpenLotChange, len(reads))
	for i := range reads {
		out[i].OpenLotRead = reads[i]
	}
	return out
}

// WithOpenLotComparisons compares each remaining buy with the same later read.
// A buy that stored no read gets an empty comparison, which is not a new opinion.
func WithOpenLotComparisons(reads []OpenLotRead, live *CoinSituation) []OpenLotChange {
	if len(reads) == 0 {
		return nil
	}
	out := make([]OpenLotChange, len(reads))
	for i := range reads {
		cmp := CompareCoinSituation(reads[i].Situation, live)
		out[i] = OpenLotChange{OpenLotRead: reads[i], Comparison: &cmp}
	}
	return out
}
