package domain

import (
	"testing"
	"time"
)

func TestKindChanges_MarksTheFirstMinuteOfANewKind(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	samples := []HeldSample{
		{At: t0, Kind: "long"},
		{At: t0.Add(5 * time.Minute), Kind: "long"},
		{At: t0.Add(10 * time.Minute), Kind: "short"},
		{At: t0.Add(15 * time.Minute), Kind: "short"},
		{At: t0.Add(20 * time.Minute), Kind: "none"},
	}
	got := kindChanges(samples)
	if len(got) != 2 {
		t.Fatalf("changes %+v", got)
	}
	if !got[0].At.Equal(t0.Add(10*time.Minute)) || got[0].Kind != "short" {
		t.Fatalf("first %+v", got[0])
	}
	if !got[1].At.Equal(t0.Add(20*time.Minute)) || got[1].Kind != "none" {
		t.Fatalf("second %+v", got[1])
	}
	if !HeldSince(samples, "none").Equal(got[1].At) {
		t.Fatalf("since %s change %s", HeldSince(samples, "none"), got[1].At)
	}
}

func TestKindChanges_GapAndBlankAreNotAChange(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	gap := kindChanges([]HeldSample{
		{At: t0, Kind: "long"},
		{At: t0.Add(20 * time.Minute), Kind: "short"},
	})
	if len(gap) != 0 {
		t.Fatalf("gap %+v", gap)
	}
	blank := kindChanges([]HeldSample{
		{At: t0, Kind: "long"},
		{At: t0.Add(5 * time.Minute), Kind: ""},
		{At: t0.Add(10 * time.Minute), Kind: "short"},
	})
	if len(blank) != 0 {
		t.Fatalf("blank %+v", blank)
	}
}

func TestSituationChanges_NamesFundingAndSizeAndSkipsThePocket(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	got := SituationChanges(SituationHeldInput{
		Now: t0.Add(15 * time.Minute),
		Funding: []FundingPressurePoint{
			{Time: t0, VenueCount: len(FuturesVenues()), Payer: "long"},
			{Time: t0.Add(5 * time.Minute), VenueCount: len(FuturesVenues()), Payer: "long"},
			{Time: t0.Add(10 * time.Minute), VenueCount: len(FuturesVenues()), Payer: "short"},
		},
		CVD: []CVDPoint{
			{Time: t0, SizeKnown: true, WhaleDelta: 10, RetailDelta: -10},
			{Time: t0.Add(5 * time.Minute), SizeKnown: true, WhaleDelta: 10, RetailDelta: -10},
			{Time: t0.Add(10 * time.Minute), SizeKnown: true, WhaleDelta: -10, RetailDelta: 10},
		},
	})
	if len(got) != 2 {
		t.Fatalf("changes %+v", got)
	}
	if got[0].Clause != "funding" || got[0].Kind != "short" || !got[0].At.Equal(t0.Add(10*time.Minute)) {
		t.Fatalf("funding %+v", got[0])
	}
	if got[1].Clause != "size" || got[1].Kind != SituationSizeWhaleSellRetailBuy {
		t.Fatalf("size %+v", got[1])
	}
	for _, row := range got {
		if row.Clause == "liquidation" {
			t.Fatalf("pocket marked %+v", row)
		}
	}
}

func TestSituationChanges_QuietSeriesIsNil(t *testing.T) {
	t0 := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	got := SituationChanges(SituationHeldInput{
		Funding: []FundingPressurePoint{
			{Time: t0, VenueCount: len(FuturesVenues()), Payer: "long"},
			{Time: t0.Add(5 * time.Minute), VenueCount: len(FuturesVenues()), Payer: "long"},
		},
	})
	if got != nil {
		t.Fatalf("changes %+v", got)
	}
}
