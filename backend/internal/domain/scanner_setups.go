package domain

import (
	"sort"
	"strings"
	"time"
)

// Scanner confluence grouping (indicator-scanner hits, not the swing engine).
const (
	ScannerConfluenceWindow = 24 * time.Hour
	ScannerSetupGradeA      = "A"
	ScannerSetupGradeB      = "B"
	ScannerSetupGradeC      = "C"
)

// ScannerSetup is a pair+interval with at least two distinct factor types
// (trend / momentum / volume) inside the confluence window.
type ScannerSetup struct {
	Key       string
	Exchange  Exchange
	Symbol    string
	Interval  string
	Factors   []ScannerRuleType
	Score     int
	Grade     string
	SameBar   bool
	LatestAt  time.Time
	Summaries []string
}

// GradeFromScore maps distinct-factor count to A/B/C (A = 3/3, B = 2/3).
func GradeFromScore(score int) string {
	if score >= 3 {
		return ScannerSetupGradeA
	}
	if score >= 2 {
		return ScannerSetupGradeB
	}
	return ScannerSetupGradeC
}

// BuildScannerSetups groups live scanner hits into confluence setups.
// Score = distinct factor types in the window. Single-factor groups are omitted.
// now/window zero values use time.Now and ScannerConfluenceWindow.
func BuildScannerSetups(results []ScannerResult, now time.Time, window time.Duration) []ScannerSetup {
	if len(results) == 0 {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if window <= 0 {
		window = ScannerConfluenceWindow
	}
	cutoff := now.Add(-window)

	recent := make([]ScannerResult, 0, len(results))
	for _, r := range results {
		ms := resultTime(r)
		if ms.IsZero() || !ms.Before(cutoff) {
			recent = append(recent, r)
		}
	}

	groups := map[string][]ScannerResult{}
	order := make([]string, 0)
	for _, r := range recent {
		key := string(r.Exchange) + "|" + r.Symbol + "|" + r.Interval
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], r)
	}

	out := make([]ScannerSetup, 0, len(groups))
	for _, key := range order {
		hits := groups[key]
		factors := uniqueScannerTypes(hits)
		score := len(factors)
		if score < 2 {
			continue
		}
		parts := strings.SplitN(key, "|", 3)
		exchange, symbol, interval := Exchange(""), "", ""
		if len(parts) == 3 {
			exchange, symbol, interval = Exchange(parts[0]), parts[1], parts[2]
		}
		summaries := make([]string, 0, 4)
		for _, h := range hits {
			if h.Summary == "" {
				continue
			}
			summaries = append(summaries, h.Summary)
			if len(summaries) == 4 {
				break
			}
		}
		out = append(out, ScannerSetup{
			Key:       key,
			Exchange:  exchange,
			Symbol:    symbol,
			Interval:  interval,
			Factors:   factors,
			Score:     score,
			Grade:     GradeFromScore(score),
			SameBar:   hasSameBarConfluence(hits),
			LatestAt:  latestResultTime(hits),
			Summaries: summaries,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].SameBar != out[j].SameBar {
			return out[i].SameBar
		}
		return out[i].LatestAt.After(out[j].LatestAt)
	})
	return out
}

// CountHitsSince counts results with a parseable match time at or after since.
func CountHitsSince(results []ScannerResult, since time.Time) int {
	if len(results) == 0 {
		return 0
	}
	n := 0
	for _, r := range results {
		ms := resultTime(r)
		if !ms.IsZero() && !ms.Before(since) {
			n++
		}
	}
	return n
}

func resultTime(r ScannerResult) time.Time {
	if !r.MatchedAt.IsZero() {
		return r.MatchedAt
	}
	if t, err := time.Parse(time.RFC3339Nano, r.MarketDataKey); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, r.MarketDataKey); err == nil {
		return t
	}
	return time.Time{}
}

func uniqueScannerTypes(hits []ScannerResult) []ScannerRuleType {
	set := map[ScannerRuleType]struct{}{}
	for _, h := range hits {
		switch h.RuleType {
		case ScannerRuleRSI, ScannerRuleMACrossover, ScannerRuleVolumeIncrease:
			set[h.RuleType] = struct{}{}
		}
	}
	order := []ScannerRuleType{ScannerRuleMACrossover, ScannerRuleRSI, ScannerRuleVolumeIncrease}
	out := make([]ScannerRuleType, 0, len(order))
	for _, t := range order {
		if _, ok := set[t]; ok {
			out = append(out, t)
		}
	}
	return out
}

func latestResultTime(hits []ScannerResult) time.Time {
	var best time.Time
	for _, h := range hits {
		ms := resultTime(h)
		if ms.After(best) {
			best = ms
		}
	}
	return best
}

func hasSameBarConfluence(hits []ScannerResult) bool {
	byBar := map[string]map[ScannerRuleType]struct{}{}
	for _, h := range hits {
		key := h.MarketDataKey
		if key == "" {
			if !h.MatchedAt.IsZero() {
				key = h.MatchedAt.UTC().Format(time.RFC3339Nano)
			} else {
				key = h.ID
			}
		}
		set := byBar[key]
		if set == nil {
			set = map[ScannerRuleType]struct{}{}
			byBar[key] = set
		}
		set[h.RuleType] = struct{}{}
		if len(set) >= 2 {
			return true
		}
	}
	return false
}
