package domain

import (
	"math"
	"slices"
	"sort"
)

// SituationRecordNote tells a caller how to read the rows.
// The same close can appear under several clauses, so the row sums are not one total.
const SituationRecordNote = "Each close is counted once inside a clause. The same close can appear under several clauses, so the rows are not a single total. A crypto buy with no stored read is skipped. Equities are left out. Open positions are not included. A profitable close has realized profit above zero. Informational only — not financial advice."

// SituationClose is one sold slice of a lot, with the read stored on the buy that opened it.
// Situation is nil when that buy stored no read.
type SituationClose struct {
	Exchange    Exchange
	RealizedPnL float64
	Situation   *CoinSituation
}

// SituationRecordRow is one entry kind that appeared on at least one counted close.
// Count is the number of closes that stored this kind. Profitable is how many of
// those had realized profit above zero. RealizedPnL is their sum.
type SituationRecordRow struct {
	Clause      string  `json:"clause"`
	Kind        string  `json:"kind"`
	Count       int     `json:"count"`
	Profitable  int     `json:"profitable"`
	RealizedPnL float64 `json:"realizedPnL"`
}

// SituationRecord is this paper book's closed crypto lots, grouped by the read on the buy.
// Counted is the number of closes that stored at least one clause. Each of those
// closes is one here, even when it appears on several rows. Skipped is the number
// of crypto closes whose buy stored no read.
type SituationRecord struct {
	Counted int                  `json:"counted"`
	Skipped int                  `json:"skipped"`
	Rows    []SituationRecordRow `json:"rows"`
	Note    string               `json:"note"`
}

type situationKindKey struct {
	clause string
	kind   string
}

// BuildSituationRecord groups closed slices by the kinds stored on each buy.
// An equity close is left out. A crypto close with no stored clause is skipped.
// A non-finite profit is left out and is not counted as skipped.
// A close with several stored clauses is counted once and added to each of those rows.
func BuildSituationRecord(closes []SituationClose) SituationRecord {
	out := SituationRecord{
		Rows: []SituationRecordRow{},
		Note: SituationRecordNote,
	}
	rows := map[situationKindKey]*SituationRecordRow{}
	for _, close := range closes {
		if IsEquityExchange(close.Exchange) {
			continue
		}
		if !situationPnLFinite(close.RealizedPnL) {
			continue
		}
		kinds := situationStoredKinds(close.Situation)
		if len(kinds) == 0 {
			out.Skipped++
			continue
		}
		out.Counted++
		win := close.RealizedPnL > PositionEpsilon
		for _, kind := range kinds {
			key := situationKindKey{clause: kind.Clause, kind: kind.Kind}
			row := rows[key]
			if row == nil {
				row = &SituationRecordRow{Clause: kind.Clause, Kind: kind.Kind}
				rows[key] = row
			}
			row.Count++
			row.RealizedPnL += close.RealizedPnL
			if win {
				row.Profitable++
			}
		}
	}
	out.Rows = make([]SituationRecordRow, 0, len(rows))
	for _, row := range rows {
		out.Rows = append(out.Rows, *row)
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		ci, ki := situationRecordRank(out.Rows[i].Clause, out.Rows[i].Kind)
		cj, kj := situationRecordRank(out.Rows[j].Clause, out.Rows[j].Kind)
		if ci != cj {
			return ci < cj
		}
		if ki != kj {
			return ki < kj
		}
		return out.Rows[i].Kind < out.Rows[j].Kind
	})
	return out
}

type situationStoredKind struct {
	Clause string
	Kind   string
}

func situationStoredKinds(s *CoinSituation) []situationStoredKind {
	if s == nil {
		return nil
	}
	var out []situationStoredKind
	add := func(clause, kind string) {
		if kind == "" {
			return
		}
		out = append(out, situationStoredKind{Clause: clause, Kind: kind})
	}
	add("priceCvd", kindOf(s.PriceCVD))
	add("openInterest", flowOf(s.OpenInterest))
	add("liquidation", sideOfLiq(s.Liquidation))
	add("prints", sideOfPrints(s.Prints))
	add("funding", payerOf(s.Funding))
	add("crowd", leanOf(s.Crowd))
	add("size", sizeKindOf(s.Size))
	return out
}

func situationPnLFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

var situationRecordClauseOrder = []string{
	"priceCvd", "openInterest", "liquidation", "prints", "funding", "crowd", "size",
}

func situationRecordKindOrder(clause string) []string {
	switch clause {
	case "priceCvd":
		return []string{SituationCVDPriceUp, SituationCVDPriceDown, SituationCVDAgrees}
	case "openInterest":
		return []string{SituationOIOpened, SituationOIClosed, SituationOIFlat}
	case "liquidation":
		return []string{SituationLiqLong, SituationLiqShort, SituationLiqNone}
	case "prints":
		return []string{SituationPrintLong, SituationPrintShort, SituationPrintEven, SituationPrintQuiet}
	case "funding":
		return []string{"long", "short", "none"}
	case "crowd":
		return []string{LongShortGapLeanTopLong, LongShortGapLeanTopShort, LongShortGapLeanEven}
	case "size":
		return []string{
			SituationSizeWhaleBuyRetailSell,
			SituationSizeWhaleSellRetailBuy,
			SituationSizeBothBuy,
			SituationSizeBothSell,
			SituationSizeWhaleBuy,
			SituationSizeWhaleSell,
			SituationSizeRetailBuy,
			SituationSizeRetailSell,
			SituationSizeFlat,
		}
	default:
		return nil
	}
}

func situationRecordRank(clause, kind string) (int, int) {
	ci := slices.Index(situationRecordClauseOrder, clause)
	if ci < 0 {
		ci = len(situationRecordClauseOrder)
	}
	order := situationRecordKindOrder(clause)
	ki := slices.Index(order, kind)
	if ki < 0 {
		ki = len(order) + 1
	}
	return ci, ki
}
