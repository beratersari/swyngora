package domain

import (
	"fmt"
	"math"
	"strings"
)

const customTapeMinAfter = 2

// CustomSignalTapeQuery is the similar/forensics window that matches a swing bar.
type CustomSignalTapeQuery struct {
	Window       string // current tape length (same range as the signal)
	MoveInterval string // past-move bar size (15m or 1h)
	Horizon      string // after-path to score (same range as the signal)
}

// CustomSignalTapeQueryFor picks tape windows from the swing primary interval.
// A 4h swing uses a 4h current tape and 4h after-path — not the 1h forensics default.
func CustomSignalTapeQueryFor(interval string) CustomSignalTapeQuery {
	switch CandleInterval(strings.ToLower(strings.TrimSpace(interval))) {
	case Interval1m, Interval3m, Interval5m, Interval15m:
		return CustomSignalTapeQuery{Window: "15m", MoveInterval: "15m", Horizon: AroundSimilarHorizon15m}
	case Interval30m, Interval1h:
		return CustomSignalTapeQuery{Window: "1h", MoveInterval: "1h", Horizon: AroundSimilarHorizon1h}
	default:
		return CustomSignalTapeQuery{Window: "4h", MoveInterval: "1h", Horizon: AroundSimilarHorizon4h}
	}
}

// CustomSignalTapeStats is how many similar setups were complete enough to use.
type CustomSignalTapeStats struct {
	Setups int // considered
	Used   int // complete enough to count as evidence
	Thin   int // had missing OI / takers / book / options
}

// ClassifyCustomTapeMatches splits setups into usable evidence vs thin tape.
func ClassifyCustomTapeMatches(matches []AroundSimilarHit) (usable []AroundSimilarHit, stats CustomSignalTapeStats, strength float64) {
	stats.Setups = len(matches)
	var mass float64
	for _, m := range matches {
		q, _, ok := customTapeHitQuality(m)
		if !ok {
			stats.Thin++
			continue
		}
		usable = append(usable, m)
		mass += q
	}
	stats.Used = len(usable)
	if stats.Used > 0 {
		strength = mass / float64(stats.Used)
	}
	return usable, stats, strength
}

func tapeStatsNote(s CustomSignalTapeStats) string {
	if s.Setups <= 0 {
		return ""
	}
	if s.Thin <= 0 {
		return fmt.Sprintf("Used %d of %d setups", s.Used, s.Setups)
	}
	return fmt.Sprintf("Used %d of %d setups; %d had missing tape", s.Used, s.Setups, s.Thin)
}

// SimilarSetupFactorRaw maps similar-past after-paths onto a long custom signal.
// Only complete setups count as evidence. horizon must match the signal range.
func SimilarSetupFactorRaw(rep *AroundSimilarReport, horizon string) CustomSignalRaw {
	if rep == nil || len(rep.Matches) == 0 {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "no similar past setups"}
	}
	usable, stats, qual := ClassifyCustomTapeMatches(rep.Matches)
	note := tapeStatsNote(stats)
	if stats.Used == 0 {
		detail := "no complete similar setups"
		if note != "" {
			detail = note + " — " + detail
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	up, down, when := similarAfterCountsFromHits(usable, horizon)
	total := up + down
	if total < customTapeMinAfter {
		detail := "not enough similar after-tape"
		if note != "" {
			detail = note + ". " + detail
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	lean := ForensicsScanLean(up, down)
	score, detail, ok := tapeLeanScore(lean, up, down, when, qual)
	if !ok {
		detail = "similar after-path is mixed"
		if note != "" {
			detail = note + ". " + detail
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	if note != "" {
		detail = note + ". " + detail
	}
	return CustomSignalRaw{Score: score, Status: CustomFactorUsed, Detail: detail}
}

// ForensicsFactorRaw maps the current tape's match to past pre-move setups
// onto a long custom signal. After-path and quality use every complete setup,
// not only the first hit.
func ForensicsFactorRaw(rep *AroundSimilarReport, horizon string) CustomSignalRaw {
	if rep == nil || len(rep.Matches) == 0 {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "current tape does not match a past pre-move setup"}
	}
	usable, stats, qual := ClassifyCustomTapeMatches(rep.Matches)
	note := tapeStatsNote(stats)
	if stats.Used == 0 {
		detail := "no complete similar setups"
		if note != "" {
			detail = note + " — " + detail
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	up, down, when := similarAfterCountsFromHits(usable, horizon)
	total := up + down
	lean := ForensicsScanLean(up, down)
	if total < customTapeMinAfter && lean != CVDDirUp && lean != CVDDirDown {
		detail := "forensics after-path is unknown"
		if note != "" {
			detail = note + ". " + detail
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	if lean != CVDDirUp && lean != CVDDirDown {
		detail := "forensics after-path is mixed"
		if note != "" {
			detail = note + ". " + detail
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	sim := usable[0].Similarity / 100
	if sim <= 0 {
		sim = 0.4
	}
	if sim > 1 {
		sim = 1
	}
	score, detail, ok := tapeLeanScore(lean, up, down, when, sim*qual)
	if !ok {
		detail = "forensics after-path is mixed"
		if note != "" {
			detail = note + ". " + detail
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	detail = fmt.Sprintf("Current tape is %.0f similar to a past pre-move setup. %s", usable[0].Similarity, detail)
	if note != "" {
		detail = note + ". " + detail
	}
	return CustomSignalRaw{Score: score, Status: CustomFactorUsed, Detail: detail}
}

// customTapeHitQuality uses existing similar coverage / missing fields so
// stale or absent OI, takers, book, or options are not treated as full evidence.
func customTapeHitQuality(hit AroundSimilarHit) (strength float64, note string, ok bool) {
	if len(hit.Compared) == 0 && hit.Coverage <= 0 {
		return 1, "", true
	}
	cov := hit.Coverage
	if cov <= 0 && len(hit.Compared) > 0 {
		var selected, used float64
		for _, c := range hit.Compared {
			selected += c.Weight
			if c.Used {
				used += c.Weight
			}
		}
		if selected > 0 {
			cov = 100 * used / selected
		}
	}
	if cov > 0 && cov < DefaultAroundSimilarMinCoverage {
		return 0, "setup lacked enough selected data", false
	}
	missing := make([]string, 0, 4)
	seen := map[string]bool{}
	for _, name := range hit.Missing {
		if !customTapeField(name) || seen[name] {
			continue
		}
		seen[name] = true
		missing = append(missing, name)
	}
	for _, c := range hit.Compared {
		if c.Used || !customTapeField(c.Name) || seen[c.Name] {
			continue
		}
		seen[c.Name] = true
		missing = append(missing, c.Name)
	}
	if len(missing) >= 3 {
		return 0, "too much tape missing (" + joinList(missing) + ")", false
	}
	strength = 1
	if cov > 0 {
		strength = cov / 100
	}
	if len(missing) > 0 {
		strength *= 1 - 0.2*float64(len(missing))
		note = "missing " + joinList(missing)
	}
	if strength <= 0 {
		return 0, note, false
	}
	return strength, note, true
}

func customTapeField(name string) bool {
	switch name {
	case AroundSimilarFieldTakers, AroundSimilarFieldOI, AroundSimilarFieldBook, AroundSimilarFieldOptions:
		return true
	default:
		return false
	}
}

// similarAfterCountsFromHits uses the signal-range horizon on complete
// setups only. It does not fall back to the report-wide AfterHorizons mix
// (that mix still includes thin tape).
func similarAfterCountsFromHits(hits []AroundSimilarHit, horizon string) (up, down int, when string) {
	want := strings.ToLower(strings.TrimSpace(horizon))
	if want == "" {
		want = AroundSimilarHorizon4h
	}
	for _, m := range hits {
		dir, _, ok := aroundSimilarHorizonAfter(m, want)
		if !ok {
			continue
		}
		switch dir {
		case CVDDirUp:
			up++
		case CVDDirDown:
			down++
		}
	}
	return up, down, want
}

// tapeLeanScore is 50 ± up to 45 from after-path lean. strength 0–1 scales
// how far from 50 (forensics uses similarity). Mixed/unknown is not a score.
func tapeLeanScore(lean string, up, down int, when string, strength float64) (float64, string, bool) {
	if lean != CVDDirUp && lean != CVDDirDown {
		return 50, "", false
	}
	if strength <= 0 {
		strength = 1
	}
	if strength > 1 {
		strength = 1
	}
	total := up + down
	conf := 1.0
	if total > 0 {
		conf = math.Min(1, float64(total)/5)
	}
	sign := 1.0
	if lean == CVDDirDown {
		sign = -1
	}
	score := clampScore(50 + 45*sign*strength*conf)
	whenBit := "afterward"
	if when != "" {
		whenBit = "over the next " + when
	}
	if lean == CVDDirUp {
		return score, fmt.Sprintf("Similar past setups more often rose %s (%d up / %d down) — supports a long custom signal", whenBit, up, down), true
	}
	return score, fmt.Sprintf("Similar past setups more often fell %s (%d up / %d down) — goes against a long custom signal", whenBit, up, down), true
}
