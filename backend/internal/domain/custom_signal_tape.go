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

// SimilarSetupFactorRaw maps similar-past after-paths onto a long custom signal.
// Up lean supports the long mix; down lean goes against it. Mixed or thin
// after-tape is omitted (not scored as 50). horizon must match the signal range.
func SimilarSetupFactorRaw(rep *AroundSimilarReport, horizon string) CustomSignalRaw {
	if rep == nil || len(rep.Matches) == 0 {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "no similar past setups"}
	}
	up, down, when := similarAfterCounts(rep, horizon)
	total := up + down
	if total < customTapeMinAfter {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "not enough similar after-tape"}
	}
	qual, qnote, ok := customTapeQuality(rep)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: qnote}
	}
	lean := ForensicsScanLean(up, down)
	score, detail, ok := tapeLeanScore(lean, up, down, when, qual)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "similar after-path is mixed"}
	}
	if qnote != "" {
		detail = qnote + ". " + detail
	}
	return CustomSignalRaw{Score: score, Status: CustomFactorUsed, Detail: detail}
}

// ForensicsFactorRaw maps the current tape's match to past pre-move setups
// onto a long custom signal. Similarity scales how hard the lean counts.
// Mixed or unmatched tape is omitted. After-path uses horizon, not the 1h desk default.
func ForensicsFactorRaw(rep *AroundSimilarReport, horizon string) CustomSignalRaw {
	hit, ok := ForensicsScanHitFromSimilar(rep)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "current tape does not match a past pre-move setup"}
	}
	up, down, when := similarAfterCounts(rep, horizon)
	total := up + down
	lean := ForensicsScanLean(up, down)
	if total < customTapeMinAfter && lean != CVDDirUp && lean != CVDDirDown {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "forensics after-path is unknown"}
	}
	if lean != CVDDirUp && lean != CVDDirDown {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "forensics after-path is mixed"}
	}
	qual, qnote, ok := customTapeQuality(rep)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: qnote}
	}
	sim := hit.Similarity / 100
	if sim <= 0 {
		sim = 0.4
	}
	if sim > 1 {
		sim = 1
	}
	score, detail, ok := tapeLeanScore(lean, up, down, when, sim*qual)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "forensics after-path is mixed"}
	}
	detail = fmt.Sprintf("Current tape is %.0f similar to a past pre-move setup. %s", hit.Similarity, detail)
	if qnote != "" {
		detail = qnote + ". " + detail
	}
	return CustomSignalRaw{Score: score, Status: CustomFactorUsed, Detail: detail}
}

// customTapeQuality uses existing similar coverage / missing fields so stale
// or absent OI, takers, book, or options are not treated as full evidence.
func customTapeQuality(rep *AroundSimilarReport) (strength float64, note string, ok bool) {
	if rep == nil || len(rep.Matches) == 0 {
		return 0, "no similar past setups", false
	}
	hit := rep.Matches[0]
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
		return 0, "closest setup lacked enough selected data", false
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

// similarAfterCounts uses the signal-range horizon only. It does not fall
// back to the forensics 1h desk pick.
func similarAfterCounts(rep *AroundSimilarReport, horizon string) (up, down int, when string) {
	want := strings.ToLower(strings.TrimSpace(horizon))
	if want == "" {
		want = AroundSimilarHorizon4h
	}
	if rep == nil {
		return 0, 0, want
	}
	for _, hz := range rep.AfterHorizons {
		if strings.ToLower(hz.Horizon) != want {
			continue
		}
		if hz.Sample > 0 || hz.Events > 0 {
			return hz.Up, hz.Down, hz.Horizon
		}
	}
	return 0, 0, want
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
