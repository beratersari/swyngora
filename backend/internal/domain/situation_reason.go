package domain

import (
	"fmt"
	"strings"
)

const (
	SituationClausePriceCVD     = "priceCvd"
	SituationClauseOpenInterest = "openInterest"
	SituationClauseLiquidation  = "liquidation"
	SituationClausePrints       = "prints"
	SituationClauseFunding      = "funding"
	SituationClauseCrowd        = "crowd"
	SituationClauseSize         = "size"

	// Preview situation states. Skipped means this order does not store a reason.
	SituationPreviewReady   = "ready"
	SituationPreviewBlank   = "blank"
	SituationPreviewFailed  = "failed"
	SituationPreviewSkipped = "skipped"
)

// SituationClauseIDs is the order a buy reason is stored and compared.
var SituationClauseIDs = []string{
	SituationClausePriceCVD,
	SituationClauseOpenInterest,
	SituationClauseLiquidation,
	SituationClausePrints,
	SituationClauseFunding,
	SituationClauseCrowd,
	SituationClauseSize,
}

// SituationReason is the set of clauses a buy is for.
// Set false means the caller did not name a set, and the fill keeps the whole read.
// Set true with an empty list stores no read.
type SituationReason struct {
	Set     bool
	Clauses []string
}

// ParseSituationReason checks clause ids. Duplicates are dropped. Order follows SituationClauseIDs.
func ParseSituationReason(raw []string) (SituationReason, error) {
	seen := map[string]bool{}
	for _, item := range raw {
		name := strings.TrimSpace(item)
		if !IsSituationClause(name) {
			return SituationReason{}, fmt.Errorf("%w: reason clause %q", ErrInvalidArgument, name)
		}
		seen[name] = true
	}
	out := make([]string, 0, len(seen))
	for _, id := range SituationClauseIDs {
		if seen[id] {
			out = append(out, id)
		}
	}
	return SituationReason{Set: true, Clauses: out}, nil
}

// IsSituationClause reports whether name is one of the seven coin-read clauses.
func IsSituationClause(name string) bool {
	switch strings.TrimSpace(name) {
	case SituationClausePriceCVD, SituationClauseOpenInterest, SituationClauseLiquidation,
		SituationClausePrints, SituationClauseFunding, SituationClauseCrowd, SituationClauseSize:
		return true
	default:
		return false
	}
}

// HashPart is the idempotency piece for this reason. Empty when the caller did not set one.
func (r SituationReason) HashPart() string {
	if !r.Set {
		return ""
	}
	if len(r.Clauses) == 0 {
		return "-"
	}
	return strings.Join(r.Clauses, ",")
}

// SituationClauseNames lists the clauses that are present on sit, in compare order.
func SituationClauseNames(sit *CoinSituation) []string {
	if sit == nil {
		return nil
	}
	var out []string
	for _, id := range SituationClauseIDs {
		if situationClauseSet(sit, id) {
			out = append(out, id)
		}
	}
	return out
}

// KeepSituationClauses copies only the named clauses.
// A reason that was not set returns sit unchanged.
// A set reason whose clauses are all blank returns nil.
func KeepSituationClauses(sit *CoinSituation, reason SituationReason) *CoinSituation {
	if sit == nil || !reason.Set {
		return sit
	}
	sit = WithoutSituationHeld(sit)
	out := &CoinSituation{Symbol: sit.Symbol, AsOf: sit.AsOf, Note: sit.Note}
	kept := false
	for _, id := range reason.Clauses {
		if !situationClauseSet(sit, id) {
			continue
		}
		copySituationClause(sit, out, id)
		kept = true
	}
	if !kept {
		return nil
	}
	return out
}

func situationClauseSet(sit *CoinSituation, id string) bool {
	if sit == nil {
		return false
	}
	switch id {
	case SituationClausePriceCVD:
		return sit.PriceCVD != nil && sit.PriceCVD.Kind != ""
	case SituationClauseOpenInterest:
		return sit.OpenInterest != nil && sit.OpenInterest.Flow != ""
	case SituationClauseLiquidation:
		return sit.Liquidation != nil && sit.Liquidation.Side != ""
	case SituationClausePrints:
		return sit.Prints != nil && sit.Prints.Side != ""
	case SituationClauseFunding:
		return sit.Funding != nil && sit.Funding.Payer != ""
	case SituationClauseCrowd:
		return sit.Crowd != nil && sit.Crowd.Lean != ""
	case SituationClauseSize:
		return sit.Size != nil && sit.Size.Kind != ""
	default:
		return false
	}
}

func copySituationClause(from, to *CoinSituation, id string) {
	switch id {
	case SituationClausePriceCVD:
		to.PriceCVD = from.PriceCVD
	case SituationClauseOpenInterest:
		to.OpenInterest = from.OpenInterest
	case SituationClauseLiquidation:
		to.Liquidation = from.Liquidation
	case SituationClausePrints:
		to.Prints = from.Prints
	case SituationClauseFunding:
		to.Funding = from.Funding
	case SituationClauseCrowd:
		to.Crowd = from.Crowd
	case SituationClauseSize:
		to.Size = from.Size
	}
}
