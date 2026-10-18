package domain

import (
	"fmt"
	"strings"
)

const (
	// AlertConditionLeaves is a situation alert on the newest open paper buy.
	AlertConditionLeaves AlertCondition = "leaves"
	// AlertConditionMatches is a situation alert on clause kinds the caller selected.
	// It does not need an open trade.
	AlertConditionMatches AlertCondition = "matches"
	// SituationMatchMet means a selected clause had that kind when the alert fired.
	SituationMatchMet = "met"
)

// SituationMatch is the clause kinds a matches alert watches.
// An empty field is not watched. Print dollars, clocks, and anchors are not stored.
type SituationMatch struct {
	PriceCVD     string `json:"priceCvd,omitempty"`
	OpenInterest string `json:"openInterest,omitempty"`
	Liquidation  string `json:"liquidation,omitempty"`
	Prints       string `json:"prints,omitempty"`
	Funding      string `json:"funding,omitempty"`
	Crowd        string `json:"crowd,omitempty"`
	Size         string `json:"size,omitempty"`
}

// IsSituationMatchAlert reports a situation alert that watches selected clause kinds.
func IsSituationMatchAlert(a PriceAlert) bool {
	if !IsSituationAlert(a.Kind) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(a.Condition)), string(AlertConditionMatches))
}

// NormalizeSituationMatch checks that at least one clause is a known kind.
// Empty fields are dropped. Clocks and dollar amounts are not part of the match.
func NormalizeSituationMatch(in *SituationMatch) (*SituationMatch, error) {
	if in == nil {
		return nil, fmt.Errorf("%w: select at least one situation clause", ErrInvalidArgument)
	}
	out := &SituationMatch{
		PriceCVD:     strings.TrimSpace(in.PriceCVD),
		OpenInterest: strings.TrimSpace(in.OpenInterest),
		Liquidation:  strings.TrimSpace(in.Liquidation),
		Prints:       strings.TrimSpace(in.Prints),
		Funding:      strings.TrimSpace(in.Funding),
		Crowd:        strings.TrimSpace(in.Crowd),
		Size:         strings.TrimSpace(in.Size),
	}
	n := 0
	for _, row := range out.rows() {
		if row.kind == "" {
			continue
		}
		if !knownSituationKind(row.clause, row.kind) {
			return nil, fmt.Errorf("%w: %s kind %q is not a situation kind", ErrInvalidArgument, row.clause, row.kind)
		}
		n++
	}
	if n == 0 {
		return nil, fmt.Errorf("%w: select at least one situation clause", ErrInvalidArgument)
	}
	return out, nil
}

// HasClause reports whether any clause kind is set, before validation.
func (m *SituationMatch) HasClause() bool {
	if m == nil {
		return false
	}
	for _, row := range m.rows() {
		if strings.TrimSpace(row.kind) != "" {
			return true
		}
	}
	return false
}

// AsSituation is the selected kinds as a coin read, with no clocks or dollars.
func (m *SituationMatch) AsSituation() *CoinSituation {
	if m == nil {
		return nil
	}
	out := &CoinSituation{}
	if m.PriceCVD != "" {
		out.PriceCVD = &SituationPriceCVD{Kind: m.PriceCVD}
	}
	if m.OpenInterest != "" {
		out.OpenInterest = &SituationOpenInterest{Flow: m.OpenInterest, Window: OpenInterestWindow1h}
	}
	if m.Liquidation != "" {
		out.Liquidation = &SituationLiquidation{Side: m.Liquidation}
	}
	if m.Prints != "" {
		out.Prints = &SituationPrints{Side: m.Prints, Window: "24h"}
	}
	if m.Funding != "" {
		out.Funding = &SituationFunding{Payer: m.Funding}
	}
	if m.Crowd != "" {
		out.Crowd = &SituationCrowd{Lean: m.Crowd}
	}
	if m.Size != "" {
		out.Size = &SituationSize{Kind: m.Size}
	}
	return out
}

type situationMatchRow struct {
	clause string
	kind   string
}

func (m *SituationMatch) rows() []situationMatchRow {
	if m == nil {
		return nil
	}
	return []situationMatchRow{
		{clause: "priceCvd", kind: m.PriceCVD},
		{clause: "openInterest", kind: m.OpenInterest},
		{clause: "liquidation", kind: m.Liquidation},
		{clause: "prints", kind: m.Prints},
		{clause: "funding", kind: m.Funding},
		{clause: "crowd", kind: m.Crowd},
		{clause: "size", kind: m.Size},
	}
}

func knownSituationKind(clause, kind string) bool {
	for _, allowed := range situationRecordKindOrder(clause) {
		if kind == allowed {
			return true
		}
	}
	return false
}

func situationLiveKind(live *CoinSituation, clause string) string {
	switch clause {
	case "priceCvd":
		return kindOf(livePrice(live))
	case "openInterest":
		return flowOf(liveOI(live))
	case "liquidation":
		return sideOfLiq(liveLiq(live))
	case "prints":
		return sideOfPrints(livePrints(live))
	case "funding":
		return payerOf(liveFunding(live))
	case "crowd":
		return leanOf(liveCrowd(live))
	case "size":
		return sizeKindOf(liveSize(live))
	default:
		return ""
	}
}

// SituationMatched reports whether every selected clause is present on the live
// read with that kind. A blank live clause is not a match. A clause that was
// not selected is ignored. Print dollars can change without a new side.
// When matched, comparison.same is true and each clause has status met.
func SituationMatched(want *SituationMatch, live *CoinSituation) (bool, SituationComparison) {
	var out SituationComparison
	if want == nil {
		return false, out
	}
	selected := 0
	for _, row := range want.rows() {
		if row.kind == "" {
			continue
		}
		selected++
		liveKind := situationLiveKind(live, row.clause)
		if liveKind == "" || liveKind != row.kind {
			return false, SituationComparison{}
		}
		out.Clauses = append(out.Clauses, SituationClauseLeft{
			Clause: row.clause,
			Status: SituationMatchMet,
			Entry:  row.kind,
			Live:   liveKind,
		})
	}
	if selected == 0 {
		return false, SituationComparison{}
	}
	out.Same = true
	return true, out
}
