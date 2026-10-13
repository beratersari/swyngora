package domain

import "sort"

// SituationLeaveNote tells a caller how to read the still and left groups.
// The same close can appear under several clauses, so the row sums are not one total.
const SituationLeaveNote = "Each row is one kind stored on the buy. still is the closes whose sell read kept that kind. left is the closes whose sell read changed that kind or left it blank. A change in print dollars with the same side stays in still. A sell with no stored read is unreadable and stays out of both groups. The same close can appear on more than one row, so the rows are not one total. skipped is a crypto close whose buy stored no read. Equities are left out. Open positions are not included. A profitable close has realized profit above zero. Informational only — not financial advice."

// SituationLeaveBucket is one side of an entry kind: the sell still matched, or it had left.
// Closes lists those slices, newest first. A zero count is a real empty group.
type SituationLeaveBucket struct {
	Count       int                    `json:"count"`
	Profitable  int                    `json:"profitable"`
	RealizedPnL float64                `json:"realizedPnL"`
	Closes      []SituationRecordClose `json:"closes"`
}

// SituationLeaveRow is one entry kind split by whether the sell still said that kind.
type SituationLeaveRow struct {
	Clause string               `json:"clause"`
	Kind   string               `json:"kind"`
	Still  SituationLeaveBucket `json:"still"`
	Left   SituationLeaveBucket `json:"left"`
}

// SituationLeave splits closed crypto lots by whether each stored entry clause
// still matched the sell. Counted is each comparable close once. Skipped is a
// crypto close whose buy stored no read. Unreadable is a close whose buy stored
// a read and whose sell stored none. A clause the buy did not store stays quiet.
type SituationLeave struct {
	Counted    int                 `json:"counted"`
	Skipped    int                 `json:"skipped"`
	Unreadable int                 `json:"unreadable"`
	Rows       []SituationLeaveRow `json:"rows"`
	Note       string              `json:"note"`
}

// BuildSituationLeave groups sold slices into still and left for each kind stored on the buy.
// An equity close is left out. A non-finite profit is left out and is not counted as skipped.
// A blank sell read stays out of both groups. A close with several stored clauses is counted
// once and added to each of those rows.
func BuildSituationLeave(closes []SituationClose) SituationLeave {
	out := SituationLeave{
		Rows: []SituationLeaveRow{},
		Note: SituationLeaveNote,
	}
	rows := map[situationKindKey]*SituationLeaveRow{}
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
		if len(situationStoredKinds(close.Exit)) == 0 {
			out.Unreadable++
			continue
		}
		out.Counted++
		left := situationLeaveClauses(close.Situation, close.Exit)
		for _, kind := range kinds {
			key := situationKindKey{clause: kind.Clause, kind: kind.Kind}
			row := rows[key]
			if row == nil {
				row = &SituationLeaveRow{
					Clause: kind.Clause,
					Kind:   kind.Kind,
					Still:  emptySituationLeaveBucket(),
					Left:   emptySituationLeaveBucket(),
				}
				rows[key] = row
			}
			bucket := &row.Still
			if _, ok := left[kind.Clause]; ok {
				bucket = &row.Left
			}
			addSituationLeave(bucket, close)
		}
	}
	out.Rows = make([]SituationLeaveRow, 0, len(rows))
	for _, row := range rows {
		sortRecordCloses(row.Still.Closes)
		sortRecordCloses(row.Left.Closes)
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

func emptySituationLeaveBucket() SituationLeaveBucket {
	return SituationLeaveBucket{Closes: []SituationRecordClose{}}
}

func addSituationLeave(bucket *SituationLeaveBucket, close SituationClose) {
	bucket.Count++
	bucket.RealizedPnL += close.RealizedPnL
	if close.RealizedPnL > PositionEpsilon {
		bucket.Profitable++
	}
	bucket.Closes = append(bucket.Closes, situationRecordClose(close))
}

// situationLeaveClauses names the stored entry clauses the sell no longer says.
// A blank clause on the sell counts. A clause the buy did not store stays quiet.
func situationLeaveClauses(entry, exit *CoinSituation) map[string]struct{} {
	cmp := CompareCoinSituation(entry, exit)
	out := make(map[string]struct{}, len(cmp.Clauses))
	for _, clause := range cmp.Clauses {
		out[clause.Clause] = struct{}{}
	}
	return out
}
