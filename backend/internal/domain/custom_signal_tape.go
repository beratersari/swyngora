package domain

import (
	"fmt"
	"math"
)

const customTapeMinAfter = 2

// SimilarSetupFactorRaw maps similar-past after-paths onto a long custom signal.
// Up lean supports the long mix; down lean goes against it. Mixed or thin
// after-tape is omitted (not scored as 50).
func SimilarSetupFactorRaw(rep *AroundSimilarReport) CustomSignalRaw {
	if rep == nil || len(rep.Matches) == 0 {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "no similar past setups"}
	}
	up, down, when := similarAfterCounts(rep)
	total := up + down
	if total < customTapeMinAfter {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "not enough similar after-tape"}
	}
	lean := ForensicsScanLean(up, down)
	score, detail, ok := tapeLeanScore(lean, up, down, when, 1)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "similar after-path is mixed"}
	}
	return CustomSignalRaw{Score: score, Status: CustomFactorUsed, Detail: detail}
}

// ForensicsFactorRaw maps the current tape's match to past pre-move setups
// onto a long custom signal. Similarity scales how hard the lean counts.
// Mixed or unmatched tape is omitted.
func ForensicsFactorRaw(rep *AroundSimilarReport) CustomSignalRaw {
	hit, ok := ForensicsScanHitFromSimilar(rep)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "current tape does not match a past pre-move setup"}
	}
	up, down, when := similarAfterCounts(rep)
	total := up + down
	lean := hit.Lean
	if lean == "" {
		lean = ForensicsScanLean(up, down)
	}
	if total < customTapeMinAfter && lean != CVDDirUp && lean != CVDDirDown {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "forensics after-path is unknown"}
	}
	if lean != CVDDirUp && lean != CVDDirDown {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "forensics after-path is mixed"}
	}
	sim := hit.Similarity / 100
	if sim <= 0 {
		sim = 0.4
	}
	if sim > 1 {
		sim = 1
	}
	score, detail, ok := tapeLeanScore(lean, up, down, when, sim)
	if !ok {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "forensics after-path is mixed"}
	}
	detail = fmt.Sprintf("Current tape is %.0f similar to a past pre-move setup. %s", hit.Similarity, detail)
	return CustomSignalRaw{Score: score, Status: CustomFactorUsed, Detail: detail}
}

func similarAfterCounts(rep *AroundSimilarReport) (up, down int, when string) {
	if rep == nil {
		return 0, 0, ""
	}
	up, down = rep.UpAfter, rep.DownAfter
	if hz := pickForensicsWhyHorizon(rep.AfterHorizons); hz.Sample > 0 || hz.Events > 0 {
		up, down = hz.Up, hz.Down
		when = hz.Horizon
	}
	return up, down, when
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
