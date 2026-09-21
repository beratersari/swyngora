package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	FearGreedExtremeFear  = "extreme_fear"
	FearGreedFear         = "fear"
	FearGreedNeutral      = "neutral"
	FearGreedGreed        = "greed"
	FearGreedExtremeGreed = "extreme_greed"

	FearGreedTowardGreed   = "greed"
	FearGreedTowardFear    = "fear"
	FearGreedTowardNeutral = "neutral"

	FearGreedFactorBreadth      = "breadth"
	FearGreedFactorMomentum     = "momentum"
	FearGreedFactorVolatility   = "volatility"
	FearGreedFactorFunding      = "funding"
	FearGreedFactorPositioning  = "positioning"
	FearGreedFactorLiquidations = "liquidations"
	FearGreedFactorBook         = "book"
	FearGreedFactorTaker        = "taker"

	// FearGreedMinFactors is how many inputs must be present before a score is published.
	FearGreedMinFactors = 3
	// FearGreedFactorCount is the full input set.
	FearGreedFactorCount = 8
	// FearGreedSampleEvery is the history bucket.
	FearGreedSampleEvery = 15 * time.Minute
	// FearGreedHistoryKeep is how far back the API returns stored scores.
	FearGreedHistoryKeep = 7 * 24 * time.Hour
	fearGreedHistoryMax  = 700

	fearGreedNote = "Calculated from Swyngora's own tape (24h breadth, BTC move, volatility versus usual, OI-weighted funding, account long/short, 24h liquidations, perp bid share, and aggressive taker flow). It is not the Binance or CoinGlass index. Missing inputs are left out and the rest are reweighted. Informational only — not financial advice."
)

// FearGreedWindows are past-score lookbacks.
var FearGreedWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{SnapshotWindow1h, time.Hour},
	{SnapshotWindow4h, 4 * time.Hour},
	{SnapshotWindow24h, 24 * time.Hour},
	{"7d", 7 * 24 * time.Hour},
}

// FearGreedInputs is the raw market tape used to score fear and greed.
// Each Has* flag is false when that feed was missing; it is not treated as neutral.
type FearGreedInputs struct {
	BreadthUpPct   float64 // 0–100, share of followed coins up over 24h
	HasBreadth     bool
	BTCChangePct   float64 // BTC 24h percent change
	HasMomentum    bool
	VolRatio       float64 // BTC 24h range ÷ its usual range
	HasVol         bool
	FundingRatePct float64 // displayed percent, +0.01 means longs pay 0.01%
	FundingPayer   string  // long | short | none
	HasFunding     bool
	LongPct        float64 // 0–100 account long share
	HasLong        bool
	LiqLong        float64
	LiqShort       float64
	HasLiq         bool
	BidPct         float64 // 0–100 perp bid share within 1% of mid
	HasBook        bool
	TakerBuyPct    float64 // 0–100 aggressive buy share
	TakerWindow    string
	HasTaker       bool
}

// FearGreedFactor is one input's reading and how many points it pulled the score.
type FearGreedFactor struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Detail   string  `json:"detail"`
	Reading  int     `json:"reading"` // 0 fear … 100 greed
	Weight   int     `json:"weight"`  // nominal weight before missing inputs are dropped
	SharePct float64 `json:"sharePct"`
	Pull     float64 `json:"pull"` // points added to a neutral 50
	Toward   string  `json:"toward"`
}

// FearGreedSample is one stored score.
type FearGreedSample struct {
	Time    time.Time `json:"time"`
	Score   int       `json:"score"`
	Level   string    `json:"level"`
	TopID   string    `json:"topId,omitempty"`
	TopPull float64   `json:"topPull,omitempty"`
}

// FearGreedWindow is the live score versus a stored score.
type FearGreedWindow struct {
	Window     string    `json:"window"`
	PastScore  int       `json:"pastScore,omitempty"`
	PastLevel  string    `json:"pastLevel,omitempty"`
	Score      int       `json:"score"`
	Level      string    `json:"level"`
	Change     int       `json:"change"`
	HasPast    bool      `json:"hasPast"`
	Complete   bool      `json:"complete"`
	SampleTime time.Time `json:"sampleTime,omitempty"`
}

// FearGreedReport is the current index plus what moved it and the stored path.
type FearGreedReport struct {
	Score    int               `json:"score"`
	Level    string            `json:"level"`
	Label    string            `json:"label"`
	AsOf     time.Time         `json:"asOf"`
	Used     int               `json:"used"`
	Total    int               `json:"total"`
	Complete bool              `json:"complete"`
	Top      *FearGreedFactor  `json:"top,omitempty"`
	Factors  []FearGreedFactor `json:"factors"`
	Missing  []string          `json:"missing,omitempty"`
	History  []FearGreedSample `json:"history"`
	Windows  []FearGreedWindow `json:"windows"`
	Summary  string            `json:"summary"`
	Note     string            `json:"note"`
}

// FearGreedStore keeps past index prints.
type FearGreedStore interface {
	SaveFearGreed(ctx context.Context, sample FearGreedSample) error
	ListFearGreed(ctx context.Context, since time.Time) ([]FearGreedSample, error)
	PurgeFearGreedBefore(ctx context.Context, before time.Time) (int, error)
}

type fearGreedSpec struct {
	id     string
	name   string
	weight int
}

var fearGreedSpecs = []fearGreedSpec{
	{FearGreedFactorBreadth, "Breadth", 18},
	{FearGreedFactorMomentum, "BTC move", 16},
	{FearGreedFactorVolatility, "Volatility", 14},
	{FearGreedFactorFunding, "Funding", 14},
	{FearGreedFactorPositioning, "Long/short", 12},
	{FearGreedFactorLiquidations, "Liquidations", 12},
	{FearGreedFactorBook, "Perp book", 8},
	{FearGreedFactorTaker, "Taker flow", 6},
}

// FearGreedSampleTime floors a timestamp onto the history bucket.
func FearGreedSampleTime(t time.Time) time.Time {
	if t.IsZero() {
		t = time.Now()
	}
	return t.UTC().Truncate(FearGreedSampleEvery)
}

// FearGreedLevel maps a 0–100 score onto a named band.
// 0–24 extreme fear, 25–44 fear, 45–55 neutral, 56–74 greed, 75–100 extreme greed.
func FearGreedLevel(score int) (id, label string) {
	switch {
	case score <= 24:
		return FearGreedExtremeFear, "Extreme fear"
	case score <= 44:
		return FearGreedFear, "Fear"
	case score <= 55:
		return FearGreedNeutral, "Neutral"
	case score <= 74:
		return FearGreedGreed, "Greed"
	default:
		return FearGreedExtremeGreed, "Extreme greed"
	}
}

// BuildFearGreed scores the tape. Missing inputs are dropped and the remaining
// weights are renormalized. The factor with the largest pull from 50 moved the score the most.
func BuildFearGreed(in FearGreedInputs, now time.Time) *FearGreedReport {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	type row struct {
		spec    fearGreedSpec
		reading float64
		detail  string
		ok      bool
	}
	rows := make([]row, 0, len(fearGreedSpecs))
	for _, spec := range fearGreedSpecs {
		reading, detail, ok := fearGreedReading(spec.id, in)
		rows = append(rows, row{spec: spec, reading: reading, detail: detail, ok: ok})
	}
	sumW := 0
	used := 0
	for _, r := range rows {
		if r.ok {
			sumW += r.spec.weight
			used++
		}
	}
	out := &FearGreedReport{
		AsOf:    now,
		Used:    used,
		Total:   FearGreedFactorCount,
		Factors: []FearGreedFactor{},
		History: []FearGreedSample{},
		Windows: []FearGreedWindow{},
		Note:    fearGreedNote,
	}
	if used == 0 || sumW == 0 {
		out.Summary = "Not enough market data to score fear and greed."
		return out
	}
	type scored struct {
		factor FearGreedFactor
		abs    float64
	}
	scoredRows := make([]scored, 0, used)
	var weighted float64
	for _, r := range rows {
		if !r.ok {
			out.Missing = append(out.Missing, r.spec.id)
			continue
		}
		share := float64(r.spec.weight) / float64(sumW)
		pull := share * (r.reading - 50)
		weighted += share * r.reading
		reading := int(math.Round(clampFearGreed(r.reading)))
		f := FearGreedFactor{
			ID:       r.spec.id,
			Name:     r.spec.name,
			Detail:   r.detail,
			Reading:  reading,
			Weight:   r.spec.weight,
			SharePct: math.Round(share*1000) / 10,
			Pull:     math.Round(pull*10) / 10,
			Toward:   fearGreedToward(pull),
		}
		scoredRows = append(scoredRows, scored{factor: f, abs: math.Abs(pull)})
	}
	sort.SliceStable(scoredRows, func(i, j int) bool {
		if scoredRows[i].abs != scoredRows[j].abs {
			return scoredRows[i].abs > scoredRows[j].abs
		}
		if scoredRows[i].factor.Weight != scoredRows[j].factor.Weight {
			return scoredRows[i].factor.Weight > scoredRows[j].factor.Weight
		}
		return scoredRows[i].factor.ID < scoredRows[j].factor.ID
	})
	out.Factors = make([]FearGreedFactor, len(scoredRows))
	for i := range scoredRows {
		out.Factors[i] = scoredRows[i].factor
	}
	score := int(math.Round(clampFearGreed(weighted)))
	level, label := FearGreedLevel(score)
	out.Score = score
	out.Level = level
	out.Label = label
	out.Complete = used == FearGreedFactorCount
	if len(out.Factors) > 0 {
		top := out.Factors[0]
		out.Top = &top
	}
	out.Summary = explainFearGreed(out)
	return out
}

// AttachFearGreedHistory fills windows and the stored series from samples.
// The live score stays the one BuildFearGreed already set.
func AttachFearGreedHistory(rep *FearGreedReport, samples []FearGreedSample) {
	if rep == nil {
		return
	}
	now := rep.AsOf.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	clean := make([]FearGreedSample, 0, len(samples))
	for _, s := range samples {
		if s.Time.IsZero() || s.Score < 0 || s.Score > 100 {
			continue
		}
		s.Time = s.Time.UTC()
		if s.Level == "" {
			s.Level, _ = FearGreedLevel(s.Score)
		}
		clean = append(clean, s)
	}
	sort.SliceStable(clean, func(i, j int) bool { return clean[i].Time.Before(clean[j].Time) })
	rep.Windows = make([]FearGreedWindow, 0, len(FearGreedWindows))
	for _, spec := range FearGreedWindows {
		w := FearGreedWindow{Window: spec.ID, Score: rep.Score, Level: rep.Level}
		past, ok := fearGreedPast(clean, now.Add(-spec.Dur))
		if ok && fearGreedWindowFresh(now, past.Time, spec.Dur) {
			w.HasPast = true
			w.Complete = true
			w.PastScore = past.Score
			w.PastLevel = past.Level
			w.Change = rep.Score - past.Score
			w.SampleTime = past.Time
		}
		rep.Windows = append(rep.Windows, w)
	}
	cutoff := now.Add(-FearGreedHistoryKeep)
	hist := make([]FearGreedSample, 0, len(clean))
	for _, s := range clean {
		if s.Time.Before(cutoff) || s.Time.After(now.Add(time.Minute)) {
			continue
		}
		hist = append(hist, s)
	}
	if len(hist) > fearGreedHistoryMax {
		hist = hist[len(hist)-fearGreedHistoryMax:]
	}
	if hist == nil {
		hist = []FearGreedSample{}
	}
	rep.History = hist
}

func explainFearGreed(rep *FearGreedReport) string {
	if rep == nil || rep.Label == "" {
		return "Not enough market data to score fear and greed."
	}
	base := fmt.Sprintf("%s %d.", rep.Label, rep.Score)
	if rep.Top == nil || rep.Top.Toward == FearGreedTowardNeutral {
		return base + " Inputs are close to even."
	}
	return fmt.Sprintf("%s %s moved the score the most (%s toward %s).",
		base, rep.Top.Name, formatFearGreedPull(rep.Top.Pull), rep.Top.Toward)
}

func formatFearGreedPull(v float64) string {
	return formatFearGreedSigned(v, 1)
}

func formatFearGreedRate(v float64) string {
	return formatFearGreedSigned(v, 4)
}

func formatFearGreedSigned(v float64, dec int) string {
	s := formatFixed(v, dec)
	if v > 0 && s != "0" {
		return "+" + s
	}
	return s
}

func fearGreedToward(pull float64) string {
	switch {
	case pull >= 0.5:
		return FearGreedTowardGreed
	case pull <= -0.5:
		return FearGreedTowardFear
	default:
		return FearGreedTowardNeutral
	}
}

func fearGreedReading(id string, in FearGreedInputs) (float64, string, bool) {
	switch id {
	case FearGreedFactorBreadth:
		if !in.HasBreadth || math.IsNaN(in.BreadthUpPct) {
			return 0, "", false
		}
		v := clampFearGreed(in.BreadthUpPct)
		return v, fmt.Sprintf("%s%% of followed coins are up over 24h", formatFixed(in.BreadthUpPct, 1)), true
	case FearGreedFactorMomentum:
		if !in.HasMomentum || math.IsNaN(in.BTCChangePct) {
			return 0, "", false
		}
		v := clampFearGreed(50 + in.BTCChangePct/8*50)
		return v, fmt.Sprintf("BTC 24h %s%%", FormatSignedPct(in.BTCChangePct)), true
	case FearGreedFactorVolatility:
		if !in.HasVol || in.VolRatio <= 0 || math.IsNaN(in.VolRatio) {
			return 0, "", false
		}
		v := clampFearGreed(50 + (1-in.VolRatio)*50)
		return v, fmt.Sprintf("BTC 24h range is %s× its usual", formatFixed(in.VolRatio, 2)), true
	case FearGreedFactorFunding:
		if !in.HasFunding || math.IsNaN(in.FundingRatePct) {
			return 0, "", false
		}
		v := clampFearGreed(50 + in.FundingRatePct/0.05*50)
		payer := in.FundingPayer
		if payer == "" {
			payer = FundingPayer(in.FundingRatePct)
		}
		who := "flat"
		switch payer {
		case "long":
			who = "longs pay"
		case "short":
			who = "shorts pay"
		}
		return v, fmt.Sprintf("OI-weighted funding %s%% (%s)", formatFearGreedRate(in.FundingRatePct), who), true
	case FearGreedFactorPositioning:
		if !in.HasLong || math.IsNaN(in.LongPct) {
			return 0, "", false
		}
		v := clampFearGreed(50 + (in.LongPct-50)*2.5)
		return v, fmt.Sprintf("%s%% of accounts are long", formatFixed(in.LongPct, 1)), true
	case FearGreedFactorLiquidations:
		tot := in.LiqLong + in.LiqShort
		if !in.HasLiq || tot <= 0 || math.IsNaN(tot) {
			return 0, "", false
		}
		shortShare := in.LiqShort / tot * 100
		longShare := 100 - shortShare
		return clampFearGreed(shortShare), fmt.Sprintf("24h liquidations are %s%% longs and %s%% shorts", formatFixed(longShare, 0), formatFixed(shortShare, 0)), true
	case FearGreedFactorBook:
		if !in.HasBook || math.IsNaN(in.BidPct) {
			return 0, "", false
		}
		v := clampFearGreed(50 + (in.BidPct-50)*2)
		return v, fmt.Sprintf("Bids are %s%% of perp size within 1%% of mid", formatFixed(in.BidPct, 1)), true
	case FearGreedFactorTaker:
		if !in.HasTaker || math.IsNaN(in.TakerBuyPct) {
			return 0, "", false
		}
		v := clampFearGreed(50 + (in.TakerBuyPct-50)*2)
		win := in.TakerWindow
		if win == "" {
			win = "1h"
		}
		return v, fmt.Sprintf("%s aggressive buys are %s%% of futures volume", win, formatFixed(in.TakerBuyPct, 1)), true
	default:
		return 0, "", false
	}
}

func clampFearGreed(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 50
	}
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func fearGreedPast(samples []FearGreedSample, target time.Time) (FearGreedSample, bool) {
	var best FearGreedSample
	ok := false
	for _, s := range samples {
		if s.Time.After(target) {
			continue
		}
		if !ok || s.Time.After(best.Time) {
			best = s
			ok = true
		}
	}
	return best, ok
}

func fearGreedWindowFresh(now, sample time.Time, dur time.Duration) bool {
	age := now.Sub(sample)
	if age < 0 {
		return false
	}
	slack := dur / 2
	if slack < 45*time.Minute {
		slack = 45 * time.Minute
	}
	return age <= dur+slack
}
