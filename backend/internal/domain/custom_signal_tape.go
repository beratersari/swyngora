package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
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
	return finishTapeFactorRaw(score, detail, stats, rep)
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
	return finishTapeFactorRaw(score, detail, stats, rep)
}

func finishTapeFactorRaw(score float64, detail string, stats CustomSignalTapeStats, rep *AroundSimilarReport) CustomSignalRaw {
	fresh := CustomSignalTapeFreshFrom(rep)
	if tapeSeriesPresent(fresh) == 0 {
		miss := "current tape is missing"
		if note := tapeStatsNote(stats); note != "" {
			miss = note + " — " + miss
		}
		return CustomSignalRaw{
			Score: 50, Status: CustomFactorMissing, Detail: miss,
			Freshness: CustomFreshnessMissing,
		}
	}
	if n := tapeSeriesPresent(fresh); n < 4 {
		detail = strings.TrimSpace(detail + fmt.Sprintf(". Current tape has %d of 4 series", n))
	}
	return CustomSignalRaw{
		Score: score, Status: CustomFactorUsed, Detail: detail,
		Quality: tapeFactorQuality(stats, fresh), Freshness: tapeFactorFreshness(fresh),
	}
}

// tapeFactorQuality is 0–1 from complete-setup share times current-tape coverage.
// Each of OI / takers / book / options is a quarter. Missing series add 0.
func tapeFactorQuality(stats CustomSignalTapeStats, fresh CustomSignalTapeFresh) float64 {
	if stats.Used <= 0 || stats.Setups <= 0 {
		return 0
	}
	share := float64(stats.Used) / float64(stats.Setups)
	return clamp01(share * tapeFreshQuality(fresh))
}

func tapeSeriesPresent(f CustomSignalTapeFresh) int {
	n := 0
	for _, a := range []CustomSignalTapeAge{f.OI, f.Takers, f.Book, f.Options} {
		if a.Age != "" && a.Age != "missing" {
			n++
		}
	}
	return n
}

// tapeFactorFreshness is missing when no current series exist, old when any
// present print is stale, and fresh only when at least one series is current.
func tapeFactorFreshness(fresh CustomSignalTapeFresh) string {
	if tapeSeriesPresent(fresh) == 0 {
		return CustomFreshnessMissing
	}
	if fresh.OI.Stale || fresh.Takers.Stale || fresh.Book.Stale || fresh.Options.Stale {
		return CustomFreshnessOld
	}
	return CustomFreshnessFresh
}

func tapeFreshQuality(f CustomSignalTapeFresh) float64 {
	rows := []CustomSignalTapeAge{f.OI, f.Takers, f.Book, f.Options}
	var acc float64
	for _, a := range rows {
		if a.Age == "" || a.Age == "missing" {
			continue
		}
		if a.Stale {
			acc += 0.4
		} else {
			acc += 1
		}
	}
	return acc / 4
}

// CustomSignalTapeAge is how old one current-tape series is vs the scan clock.
type CustomSignalTapeAge struct {
	Field  string `json:"field"`
	Age    string `json:"age"`
	AgeSec int    `json:"ageSec,omitempty"`
	At     string `json:"at,omitempty"`
	Stale  bool   `json:"stale,omitempty"`
}

// CustomSignalTapeFresh is OI / taker / book / options age on the current tape.
type CustomSignalTapeFresh struct {
	OI      CustomSignalTapeAge `json:"oi"`
	Takers  CustomSignalTapeAge `json:"takers"`
	Book    CustomSignalTapeAge `json:"book"`
	Options CustomSignalTapeAge `json:"options"`
	Note    string              `json:"note,omitempty"`
}

// CustomSignalTapeFreshFrom reports how old the current tape prints are.
// Missing series say "missing". A print older than its slack is marked stale
// so old OI / takers / book / options cannot silently support the signal.
func CustomSignalTapeFreshFrom(rep *AroundSimilarReport) CustomSignalTapeFresh {
	if rep == nil {
		return CustomSignalTapeFresh{}
	}
	asOf := rep.AsOf
	if asOf.IsZero() {
		asOf = time.Now().UTC()
	}
	cur := rep.Current
	out := CustomSignalTapeFresh{
		Takers:  tapeAgeOf(AroundSimilarFieldTakers, takerSampleAt(cur), asOf, AroundTapeFreshSlack),
		OI:      tapeAgeOf(AroundSimilarFieldOI, oiSampleAt(cur), asOf, AroundTapeFreshSlack),
		Book:    tapeAgeOf(AroundSimilarFieldBook, bookSampleAt(cur), asOf, DefaultBookHistorySlack),
		Options: tapeAgeOf(AroundSimilarFieldOptions, optionsSampleAt(cur), asOf, aroundOptionsSlack),
	}
	out.Note = formatTapeFreshNote(out)
	return out
}

func takerSampleAt(p AroundPhase) time.Time {
	if !p.Flow.LastBar.IsZero() {
		return p.Flow.LastBar
	}
	if p.Flow.BuySellKnown {
		return p.To
	}
	return time.Time{}
}

func oiSampleAt(p AroundPhase) time.Time {
	if p.Futures != nil && p.Futures.HasOI {
		return p.Futures.OIToAt
	}
	return time.Time{}
}

func bookSampleAt(p AroundPhase) time.Time {
	if p.Book != nil && p.Book.Complete {
		return p.Book.SampledTo
	}
	return time.Time{}
}

func optionsSampleAt(p AroundPhase) time.Time {
	if p.Options != nil && p.Options.Complete {
		return p.Options.SampledTo
	}
	return time.Time{}
}

func tapeAgeOf(field string, sampled, asOf time.Time, slack time.Duration) CustomSignalTapeAge {
	out := CustomSignalTapeAge{Field: field, Age: "missing"}
	if sampled.IsZero() {
		return out
	}
	sampled = sampled.UTC()
	asOf = asOf.UTC()
	age := asOf.Sub(sampled)
	if age < 0 {
		age = 0
	}
	out.Age = formatTapeAge(age)
	out.AgeSec = int(age / time.Second)
	out.At = sampled.Format(time.RFC3339)
	if slack > 0 && age > slack {
		out.Stale = true
	}
	return out
}

func formatTapeAge(d time.Duration) string {
	if d < time.Minute {
		sec := int(d / time.Second)
		if sec < 1 {
			sec = 0
		}
		return fmt.Sprintf("%ds", sec)
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%dh", int(d/time.Hour))
}

func formatTapeFreshNote(f CustomSignalTapeFresh) string {
	parts := make([]string, 0, 4)
	for _, a := range []CustomSignalTapeAge{f.OI, f.Takers, f.Book, f.Options} {
		label := a.Field
		if a.Age == "missing" || a.Age == "" {
			parts = append(parts, label+" missing")
			continue
		}
		bit := label + " " + a.Age + " ago"
		if a.Stale {
			bit += " (old)"
		}
		parts = append(parts, bit)
	}
	return strings.Join(parts, "; ")
}

// customTapeHitQuality rejects a setup when any selected OI / taker / book /
// options field is missing. Those setups are not used as evidence.
func customTapeHitQuality(hit AroundSimilarHit) (strength float64, note string, ok bool) {
	if len(hit.Compared) == 0 && hit.Coverage <= 0 && len(hit.Missing) == 0 {
		return 1, "", true
	}
	missing := customTapeMissing(hit)
	if len(missing) > 0 {
		return 0, "missing " + joinList(missing), false
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
	strength = 1
	if cov > 0 {
		strength = cov / 100
	}
	if strength <= 0 {
		return 0, "setup lacked enough selected data", false
	}
	return strength, "", true
}

func customTapeMissing(hit AroundSimilarHit) []string {
	seen := map[string]bool{}
	var missing []string
	add := func(name string) {
		if !customTapeField(name) || seen[name] {
			return
		}
		seen[name] = true
		missing = append(missing, name)
	}
	for _, name := range hit.Missing {
		add(name)
	}
	for _, c := range hit.Compared {
		if !c.Used {
			add(c.Name)
		}
	}
	return missing
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
