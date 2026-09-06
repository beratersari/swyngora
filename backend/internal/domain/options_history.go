package domain

import (
	"context"
	"math"
	"sort"
	"time"
)

const (
	OptionsHistoryWindow1h  = SnapshotWindow1h
	OptionsHistoryWindow4h  = SnapshotWindow4h
	OptionsHistoryWindow24h = SnapshotWindow24h
	OptionsHistoryWindow7d  = "7d"

	OptionsHistoryLookback     = 7 * 24 * time.Hour
	OptionsHistoryMaxPoints    = 200
	OptionsHistorySampleBucket = 5 * time.Minute
	OptionsHistoryListLimit    = 2500
)

// OptionsHistoryWindows is 1h / 4h / 24h / 7d.
var OptionsHistoryWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{OptionsHistoryWindow1h, time.Hour},
	{OptionsHistoryWindow4h, 4 * time.Hour},
	{OptionsHistoryWindow24h, 24 * time.Hour},
	{OptionsHistoryWindow7d, 7 * 24 * time.Hour},
}

// OptionsSample is one persisted desk print (OI, put/call, IV).
type OptionsSample struct {
	Asset           string
	SampledAt       time.Time
	OpenInterest    float64
	OpenInterestUSD float64
	CallOI          float64
	PutOI           float64
	PutCallOI       float64
	HasPutCallOI    bool
	DVOL            float64
	HasDVOL         bool
	AtmIV           float64
	HasAtmIV        bool
}

// OptionsDVOLPoint is one Deribit DVOL close.
type OptionsDVOLPoint struct {
	Time  time.Time
	Value float64
}

// OptionsHistoryPoint is one chart sample. Missing fields are omitted on the wire.
type OptionsHistoryPoint struct {
	Time            time.Time
	OpenInterest    float64
	OpenInterestUSD float64
	HasOI           bool
	PutCallOI       float64
	HasPutCallOI    bool
	DVOL            float64
	HasDVOL         bool
	AtmIV           float64
	HasAtmIV        bool
}

// OptionsHistoryWindow is current vs a lookback for OI, PCR, and IV.
type OptionsHistoryWindow struct {
	Window   string
	OI       SnapshotChange
	PutCall  SnapshotChange
	DVOL     SnapshotChange
	AtmIV    SnapshotChange
	Complete bool
}

// OptionsHistory is how the options desk moved over the last hours and days.
type OptionsHistory struct {
	Windows []OptionsHistoryWindow
	Points  []OptionsHistoryPoint
	From    time.Time
	To      time.Time
	Note    string
}

// OptionsHistoryStore persists desk samples (BTC/ETH).
type OptionsHistoryStore interface {
	InsertSample(ctx context.Context, rec OptionsSample) (inserted bool, err error)
	ListSamples(ctx context.Context, asset string, from, to time.Time, limit int) ([]OptionsSample, error)
	PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, error)
}

// OptionsDVOLHistoryPort loads Deribit DVOL candles (optional on the chain port).
type OptionsDVOLHistoryPort interface {
	GetOptionsDVOLHistory(ctx context.Context, asset string, from, to time.Time) ([]OptionsDVOLPoint, error)
}

// OptionsSampleFromSnapshot copies headline numbers into a 5-minute bucket.
func OptionsSampleFromSnapshot(snap *OptionsSnapshot) (OptionsSample, bool) {
	if snap == nil {
		return OptionsSample{}, false
	}
	at := snap.AsOf.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	out := OptionsSample{
		Asset:           snap.Asset,
		SampledAt:       TruncateToBucket(at, OptionsHistorySampleBucket),
		OpenInterest:    snap.OpenInterest,
		OpenInterestUSD: snap.OpenInterestUSD,
		CallOI:          snap.CallOI,
		PutOI:           snap.PutOI,
		PutCallOI:       snap.PutCallOI,
		HasPutCallOI:    snap.HasPutCallOI,
		DVOL:            snap.DVOL,
		HasDVOL:         snap.HasDVOL,
	}
	if snap.ATM != nil && snap.ATM.IV > 0 {
		out.AtmIV = snap.ATM.IV
		out.HasAtmIV = true
	}
	if out.OpenInterest <= 0 && !out.HasDVOL && !out.HasPutCallOI && !out.HasAtmIV {
		return OptionsSample{}, false
	}
	return out, true
}

// BuildOptionsHistory folds local samples + DVOL candles into windows and a chart series.
func BuildOptionsHistory(snap *OptionsSnapshot, samples []OptionsSample, dvol []OptionsDVOLPoint, now time.Time) *OptionsHistory {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	from := now.Add(-OptionsHistoryLookback)
	out := &OptionsHistory{
		Windows: make([]OptionsHistoryWindow, 0, len(OptionsHistoryWindows)),
		Points:  nil,
		From:    from,
		To:      now,
		Note:    "Open interest and put/call are sampled while the desk runs (5-minute buckets). Implied vol also uses Deribit DVOL history. Informational only — not financial advice.",
	}
	merged := mergeOptionsHistory(snap, samples, dvol, from, now)
	out.Points = downsampleOptionsHistory(merged, OptionsHistoryMaxPoints)
	latest := optionsHistoryLatest(merged)
	for _, spec := range OptionsHistoryWindows {
		past := optionsHistoryAtOrBefore(merged, now.Add(-spec.Dur), optionsHistorySlack(spec.Dur))
		win := OptionsHistoryWindow{Window: spec.ID}
		if latest.HasOI && past.HasOI {
			cur, old := optionsOIValue(latest), optionsOIValue(past)
			if cur > 0 || old > 0 {
				win.OI = ChangeFromValues(spec.ID, cur, old, old > 0)
			}
		}
		if latest.HasPutCallOI && past.HasPutCallOI {
			win.PutCall = ChangeFromValues(spec.ID, latest.PutCallOI, past.PutCallOI, true)
		}
		if latest.HasDVOL && past.HasDVOL {
			win.DVOL = ChangeFromValues(spec.ID, latest.DVOL, past.DVOL, true)
		}
		if latest.HasAtmIV && past.HasAtmIV {
			win.AtmIV = ChangeFromValues(spec.ID, latest.AtmIV, past.AtmIV, true)
		}
		win.Complete = win.OI.Complete || win.PutCall.Complete || win.DVOL.Complete || win.AtmIV.Complete
		out.Windows = append(out.Windows, win)
	}
	if snap != nil {
		snap.Moves = DetectOptionsMoves(merged, now)
	}
	if len(out.Points) == 0 && !anyOptionsWindowComplete(out.Windows) {
		return nil
	}
	return out
}

func optionsOIValue(p OptionsHistoryPoint) float64 {
	if p.OpenInterestUSD > 0 {
		return p.OpenInterestUSD
	}
	return p.OpenInterest
}

func mergeOptionsHistory(snap *OptionsSnapshot, samples []OptionsSample, dvol []OptionsDVOLPoint, from, to time.Time) []OptionsHistoryPoint {
	byTs := map[int64]*OptionsHistoryPoint{}
	put := func(t time.Time) *OptionsHistoryPoint {
		t = t.UTC().Truncate(time.Minute)
		if t.Before(from) || t.After(to.Add(time.Minute)) {
			return nil
		}
		key := t.Unix()
		p := byTs[key]
		if p == nil {
			p = &OptionsHistoryPoint{Time: t}
			byTs[key] = p
		}
		return p
	}
	for _, s := range samples {
		p := put(s.SampledAt)
		if p == nil {
			continue
		}
		if s.OpenInterest > 0 || s.OpenInterestUSD > 0 {
			p.OpenInterest = s.OpenInterest
			p.OpenInterestUSD = s.OpenInterestUSD
			p.HasOI = true
		}
		if s.HasPutCallOI {
			p.PutCallOI = s.PutCallOI
			p.HasPutCallOI = true
		}
		if s.HasDVOL && s.DVOL > 0 {
			p.DVOL = s.DVOL
			p.HasDVOL = true
		}
		if s.HasAtmIV && s.AtmIV > 0 {
			p.AtmIV = s.AtmIV
			p.HasAtmIV = true
		}
	}
	for _, d := range dvol {
		p := put(d.Time)
		if p == nil || d.Value <= 0 {
			continue
		}
		p.DVOL = d.Value
		p.HasDVOL = true
	}
	if snap != nil {
		p := put(snap.AsOf)
		if p == nil {
			p = put(to)
		}
		if p != nil {
			if snap.OpenInterest > 0 || snap.OpenInterestUSD > 0 {
				p.OpenInterest = snap.OpenInterest
				p.OpenInterestUSD = snap.OpenInterestUSD
				p.HasOI = true
			}
			if snap.HasPutCallOI {
				p.PutCallOI = snap.PutCallOI
				p.HasPutCallOI = true
			}
			if snap.HasDVOL && snap.DVOL > 0 {
				p.DVOL = snap.DVOL
				p.HasDVOL = true
			}
			if snap.ATM != nil && snap.ATM.IV > 0 {
				p.AtmIV = snap.ATM.IV
				p.HasAtmIV = true
			}
		}
	}
	out := make([]OptionsHistoryPoint, 0, len(byTs))
	for _, p := range byTs {
		if p.HasOI || p.HasPutCallOI || p.HasDVOL || p.HasAtmIV {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	forwardFillOptionsHistory(out)
	return out
}

func forwardFillOptionsHistory(in []OptionsHistoryPoint) {
	var oi, pcr, atm OptionsHistoryPoint
	for i := range in {
		if in[i].HasOI {
			oi = in[i]
		} else if oi.HasOI {
			in[i].OpenInterest = oi.OpenInterest
			in[i].OpenInterestUSD = oi.OpenInterestUSD
			in[i].HasOI = true
		}
		if in[i].HasPutCallOI {
			pcr = in[i]
		} else if pcr.HasPutCallOI {
			in[i].PutCallOI = pcr.PutCallOI
			in[i].HasPutCallOI = true
		}
		if in[i].HasAtmIV {
			atm = in[i]
		} else if atm.HasAtmIV {
			in[i].AtmIV = atm.AtmIV
			in[i].HasAtmIV = true
		}
	}
}

func downsampleOptionsHistory(in []OptionsHistoryPoint, max int) []OptionsHistoryPoint {
	if max <= 0 || len(in) <= max {
		return in
	}
	out := make([]OptionsHistoryPoint, 0, max)
	step := float64(len(in)-1) / float64(max-1)
	last := -1
	for i := 0; i < max; i++ {
		idx := int(math.Round(float64(i) * step))
		if idx <= last {
			idx = last + 1
		}
		if idx >= len(in) {
			idx = len(in) - 1
		}
		out = append(out, in[idx])
		last = idx
	}
	if last != len(in)-1 {
		out[len(out)-1] = in[len(in)-1]
	}
	return out
}

func optionsHistoryLatest(in []OptionsHistoryPoint) OptionsHistoryPoint {
	if len(in) == 0 {
		return OptionsHistoryPoint{}
	}
	return in[len(in)-1]
}

func optionsHistorySlack(window time.Duration) time.Duration {
	s := window / 4
	if s < 15*time.Minute {
		s = 15 * time.Minute
	}
	if s > 6*time.Hour {
		s = 6 * time.Hour
	}
	return s
}

func optionsHistoryAtOrBefore(in []OptionsHistoryPoint, at time.Time, slack time.Duration) OptionsHistoryPoint {
	var out OptionsHistoryPoint
	found := false
	for i := range in {
		if in[i].Time.After(at) {
			break
		}
		out = in[i]
		found = true
	}
	if !found || (slack > 0 && at.Sub(out.Time) > slack) {
		return OptionsHistoryPoint{}
	}
	return out
}

func anyOptionsWindowComplete(wins []OptionsHistoryWindow) bool {
	for _, w := range wins {
		if w.Complete {
			return true
		}
	}
	return false
}
