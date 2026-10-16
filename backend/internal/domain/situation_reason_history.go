package domain

// ReasonHistory keeps the still and left groups for each selected clause at the
// kind the live read says now. Clauses follow SituationClauseIDs. A selected
// clause that is blank on the live read is omitted. A live kind with no closed
// lot comes back with empty still and left, so the caller can say there is no
// past close. An unset reason, an empty reason, or a nil live read returns nil.
// The buckets are the ones BuildSituationLeave already counted. Rows are not one total.
func ReasonHistory(leave SituationLeave, reason SituationReason, live *CoinSituation) []SituationLeaveRow {
	if !reason.Set || len(reason.Clauses) == 0 || live == nil {
		return nil
	}
	wanted := make(map[string]bool, len(reason.Clauses))
	for _, id := range reason.Clauses {
		wanted[id] = true
	}
	liveKind := make(map[string]string, len(SituationClauseIDs))
	for _, kind := range situationStoredKinds(live) {
		liveKind[kind.Clause] = kind.Kind
	}
	byKey := make(map[situationKindKey]SituationLeaveRow, len(leave.Rows))
	for _, row := range leave.Rows {
		byKey[situationKindKey{clause: row.Clause, kind: row.Kind}] = row
	}
	var out []SituationLeaveRow
	for _, id := range SituationClauseIDs {
		if !wanted[id] {
			continue
		}
		kind := liveKind[id]
		if kind == "" {
			continue
		}
		row, ok := byKey[situationKindKey{clause: id, kind: kind}]
		if !ok {
			row = SituationLeaveRow{
				Clause: id,
				Kind:   kind,
				Still:  emptySituationLeaveBucket(),
				Left:   emptySituationLeaveBucket(),
			}
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
