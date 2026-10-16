package domain

import (
	"sort"
	"time"
)

// SituationChange is one minute a stored situation kind changed.
// Clause is priceCvd, openInterest, prints, funding, crowd, or size.
// Kind is the new kind. The estimated liquidation pocket is not a change.
type SituationChange struct {
	Clause string    `json:"clause"`
	At     time.Time `json:"at"`
	Kind   string    `json:"kind"`
}

// situationChangeOrder is the clause order on the chart label.
var situationChangeOrder = map[string]int{
	"priceCvd":     0,
	"openInterest": 1,
	"prints":       2,
	"funding":      3,
	"crowd":        4,
	"size":         5,
}

// SituationChanges lists every observed kind change in the series that
// stamps the live clocks. A gap wider than ten minutes, or a minute that
// cannot be read, is not a change. An empty result is nil.
func SituationChanges(in SituationHeldInput) []SituationChange {
	out := make([]SituationChange, 0)
	add := func(clause string, samples []HeldSample) {
		for _, sample := range kindChanges(samples) {
			out = append(out, SituationChange{Clause: clause, At: sample.At, Kind: sample.Kind})
		}
	}
	add("priceCvd", priceCVDHeldSamples(in.CVD))
	add("openInterest", openInterestHeldSamples(in.OpenInterest, in.Now))
	add("prints", printHeldSamples(in.Prints, in.Coverage, in.Symbol, in.Now))
	add("funding", fundingHeldSamples(in.Funding))
	add("crowd", crowdHeldSamples(in.Crowd))
	add("size", sizeHeldSamples(in.CVD))
	sort.Slice(out, func(i, j int) bool {
		if out[i].At.Equal(out[j].At) {
			return situationChangeOrder[out[i].Clause] < situationChangeOrder[out[j].Clause]
		}
		return out[i].At.Before(out[j].At)
	})
	if len(out) == 0 {
		return nil
	}
	return out
}
