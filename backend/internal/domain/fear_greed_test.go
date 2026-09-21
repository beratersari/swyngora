package domain

import (
	"testing"
	"time"
)

func TestFearGreedLevelBands(t *testing.T) {
	cases := []struct {
		score int
		id    string
	}{
		{0, FearGreedExtremeFear},
		{24, FearGreedExtremeFear},
		{25, FearGreedFear},
		{44, FearGreedFear},
		{45, FearGreedNeutral},
		{55, FearGreedNeutral},
		{56, FearGreedGreed},
		{74, FearGreedGreed},
		{75, FearGreedExtremeGreed},
		{100, FearGreedExtremeGreed},
	}
	for _, tc := range cases {
		id, label := FearGreedLevel(tc.score)
		if id != tc.id || label == "" {
			t.Fatalf("score %d → %s %q", tc.score, id, label)
		}
	}
}

func TestBuildFearGreed_NeutralWhenInputsAreEven(t *testing.T) {
	got := BuildFearGreed(evenFearGreedInputs(), time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC))
	if got.Score != 50 || got.Level != FearGreedNeutral {
		t.Fatalf("score=%d level=%s summary=%s", got.Score, got.Level, got.Summary)
	}
	if got.Used != FearGreedFactorCount || !got.Complete {
		t.Fatalf("used=%d complete=%v", got.Used, got.Complete)
	}
	if got.Top == nil || got.Top.Toward != FearGreedTowardNeutral {
		t.Fatalf("top %+v", got.Top)
	}
	if got.Summary == "" || got.Note == "" {
		t.Fatal("missing summary or note")
	}
}

func TestBuildFearGreed_FundingPullsTowardGreed(t *testing.T) {
	in := evenFearGreedInputs()
	in.FundingRatePct = 0.05 // +0.05% maps to a reading of 100
	in.HasFunding = true
	got := BuildFearGreed(in, time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC))
	if got.Score <= 50 || got.Level == FearGreedFear || got.Level == FearGreedExtremeFear {
		t.Fatalf("score=%d level=%s", got.Score, got.Level)
	}
	if got.Top == nil || got.Top.ID != FearGreedFactorFunding || got.Top.Toward != FearGreedTowardGreed || got.Top.Pull <= 0 {
		t.Fatalf("top %+v", got.Top)
	}
	if got.Factors[0].ID != FearGreedFactorFunding {
		t.Fatalf("first factor %s", got.Factors[0].ID)
	}
}

func TestBuildFearGreed_DropsMissingAndRenormalizes(t *testing.T) {
	in := FearGreedInputs{
		HasBreadth:   true,
		BreadthUpPct: 100, // reading 100, weight 18
		HasMomentum:  true,
		BTCChangePct: -8, // reading 0, weight 16
	}
	got := BuildFearGreed(in, time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC))
	// (18*100 + 16*0) / 34 = 52.94 → 53
	if got.Score != 53 {
		t.Fatalf("score=%d summary=%s", got.Score, got.Summary)
	}
	if got.Used != 2 || got.Complete || len(got.Missing) != 6 {
		t.Fatalf("used=%d missing=%v", got.Used, got.Missing)
	}
	if got.Top == nil || got.Top.ID != FearGreedFactorBreadth {
		t.Fatalf("top %+v", got.Top)
	}
	if got.Factors[0].SharePct < 52 || got.Factors[0].SharePct > 53 {
		t.Fatalf("share %v", got.Factors[0].SharePct)
	}
}

func TestBuildFearGreed_HighVolatilityIsFear(t *testing.T) {
	in := evenFearGreedInputs()
	in.VolRatio = 2
	got := BuildFearGreed(in, time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC))
	var vol *FearGreedFactor
	for i := range got.Factors {
		if got.Factors[i].ID == FearGreedFactorVolatility {
			vol = &got.Factors[i]
		}
	}
	if vol == nil || vol.Reading != 0 || vol.Toward != FearGreedTowardFear {
		t.Fatalf("vol %+v", vol)
	}
}

func TestBuildFearGreed_LiquidationSides(t *testing.T) {
	longs := BuildFearGreed(FearGreedInputs{HasLiq: true, LiqLong: 80, LiqShort: 20}, time.Now().UTC())
	shorts := BuildFearGreed(FearGreedInputs{HasLiq: true, LiqLong: 20, LiqShort: 80}, time.Now().UTC())
	if longs.Factors[0].Reading != 20 || shorts.Factors[0].Reading != 80 {
		t.Fatalf("long-liq reading %d short-liq reading %d", longs.Factors[0].Reading, shorts.Factors[0].Reading)
	}
	empty := BuildFearGreed(FearGreedInputs{HasLiq: true}, time.Now().UTC())
	if empty.Used != 0 {
		t.Fatalf("empty liq used=%d", empty.Used)
	}
}

func TestAttachFearGreedHistory_WindowAndSeries(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	rep := BuildFearGreed(evenFearGreedInputs(), now)
	rep.Score = 70
	rep.Level = FearGreedGreed
	dayAgo := now.Add(-24 * time.Hour)
	hourAgo := now.Add(-time.Hour)
	samples := []FearGreedSample{
		{Time: dayAgo, Score: 40, Level: FearGreedFear, TopID: FearGreedFactorVolatility},
		{Time: hourAgo, Score: 61, Level: FearGreedGreed},
		{Time: now.Add(-8 * 24 * time.Hour), Score: 10, Level: FearGreedExtremeFear},
	}
	AttachFearGreedHistory(rep, samples)
	var day, hour FearGreedWindow
	for _, w := range rep.Windows {
		switch w.Window {
		case "24h":
			day = w
		case "1h":
			hour = w
		}
	}
	if !day.Complete || day.PastScore != 40 || day.Change != 30 || day.PastLevel != FearGreedFear {
		t.Fatalf("24h %+v", day)
	}
	if !hour.Complete || hour.PastScore != 61 || hour.Change != 9 {
		t.Fatalf("1h %+v", hour)
	}
	if len(rep.History) != 2 {
		t.Fatalf("history %d (week-old print should drop)", len(rep.History))
	}
	if !rep.History[0].Time.Equal(dayAgo) || rep.History[1].Score != 61 {
		t.Fatalf("history %+v", rep.History)
	}
}

func TestFearGreedSampleTimeBuckets(t *testing.T) {
	got := FearGreedSampleTime(time.Date(2026, 9, 21, 12, 14, 0, 0, time.UTC))
	want := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s", got)
	}
}

func evenFearGreedInputs() FearGreedInputs {
	return FearGreedInputs{
		HasBreadth:     true,
		BreadthUpPct:   50,
		HasMomentum:    true,
		BTCChangePct:   0,
		HasVol:         true,
		VolRatio:       1,
		HasFunding:     true,
		FundingRatePct: 0,
		FundingPayer:   "none",
		HasLong:        true,
		LongPct:        50,
		HasLiq:         true,
		LiqLong:        50,
		LiqShort:       50,
		HasBook:        true,
		BidPct:         50,
		HasTaker:       true,
		TakerBuyPct:    50,
		TakerWindow:    "1h",
	}
}
