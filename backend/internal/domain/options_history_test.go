package domain

import (
	"testing"
	"time"
)

func TestOptionsSampleFromSnapshot(t *testing.T) {
	t.Parallel()
	asOf := time.Date(2026, 3, 1, 12, 7, 0, 0, time.UTC)
	snap := &OptionsSnapshot{
		Asset: OptionsAssetBTC, AsOf: asOf,
		OpenInterest: 100, OpenInterestUSD: 1e7,
		PutCallOI: 1.2, HasPutCallOI: true,
		DVOL: 50, HasDVOL: true,
		ATM:        &OptionsATM{IV: 51},
		IndexPrice: 100000,
		Gamma: &OptionsGamma{
			NetUSD: -8e8, Regime: GammaRegimeShort, HasFlip: true,
			Flip: 97200, FlipSide: GammaFlipAbove,
		},
	}
	got, ok := OptionsSampleFromSnapshot(snap)
	if !ok || got.Asset != "BTC" || got.OpenInterest != 100 || !got.HasAtmIV {
		t.Fatalf("%+v ok=%v", got, ok)
	}
	if !got.HasGamma || got.GammaNetUSD != -8e8 || got.GammaRegime != GammaRegimeShort {
		t.Fatalf("gamma sample %+v", got)
	}
	if !got.HasGammaFlip || got.GammaFlip != 97200 || got.GammaFlipSide != GammaFlipAbove {
		t.Fatalf("flip sample %+v", got)
	}
	if got.SampledAt.Minute() != 5 {
		t.Fatalf("bucket=%v", got.SampledAt)
	}
}

func TestBuildOptionsHistory_WindowsAndPoints(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	snap := &OptionsSnapshot{
		Asset: "BTC", AsOf: now,
		OpenInterestUSD: 120, OpenInterest: 12, HasPutCallOI: true, PutCallOI: 1.4,
		DVOL: 52, HasDVOL: true,
	}
	samples := []OptionsSample{
		{Asset: "BTC", SampledAt: now.Add(-2 * time.Hour), OpenInterestUSD: 100, OpenInterest: 10, HasPutCallOI: true, PutCallOI: 1.0, HasDVOL: true, DVOL: 48},
		{Asset: "BTC", SampledAt: now.Add(-25 * time.Hour), OpenInterestUSD: 80, OpenInterest: 8, HasPutCallOI: true, PutCallOI: 0.8, HasDVOL: true, DVOL: 45},
	}
	dvol := []OptionsDVOLPoint{
		{Time: now.Add(-7 * 24 * time.Hour), Value: 40},
		{Time: now.Add(-24 * time.Hour), Value: 46},
		{Time: now, Value: 52},
	}
	got := BuildOptionsHistory(snap, samples, dvol, now)
	if got == nil || len(got.Points) < 3 {
		t.Fatalf("points=%+v", got)
	}
	var h1, h24, d7 *OptionsHistoryWindow
	for i := range got.Windows {
		switch got.Windows[i].Window {
		case OptionsHistoryWindow1h:
			h1 = &got.Windows[i]
		case OptionsHistoryWindow24h:
			h24 = &got.Windows[i]
		case OptionsHistoryWindow7d:
			d7 = &got.Windows[i]
		}
	}
	if h1 == nil || h1.OI.Complete {
		t.Fatalf("1h should lack a 1h-old OI sample: %+v", h1)
	}
	if h24 == nil || !h24.OI.Complete || h24.OI.ChangePct < 40 || h24.OI.ChangePct > 60 {
		t.Fatalf("24h oi %+v", h24)
	}
	if h24.PutCall.Current != 1.4 || h24.DVOL.Current != 52 {
		t.Fatalf("24h pcr/dvol %+v", h24)
	}
	if d7 == nil || !d7.DVOL.Complete || d7.DVOL.Past != 40 {
		t.Fatalf("7d dvol %+v", d7)
	}
}

func TestBuildOptionsHistory_GammaAndFlip(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC)
	snap := &OptionsSnapshot{
		Asset: "BTC", AsOf: now, IndexPrice: 101000,
		Gamma: &OptionsGamma{
			NetUSD: -200, Regime: GammaRegimeShort, HasFlip: true,
			Flip: 98000, FlipSide: GammaFlipAbove,
		},
	}
	samples := []OptionsSample{
		{
			Asset: "BTC", SampledAt: now.Add(-25 * time.Hour),
			HasGamma: true, GammaNetUSD: 400, GammaRegime: GammaRegimeLong,
			HasGammaFlip: true, GammaFlip: 102000, GammaFlipSide: GammaFlipBelow,
			HasIndex: true, IndexPrice: 100000,
		},
	}
	got := BuildOptionsHistory(snap, samples, nil, now)
	if got == nil {
		t.Fatal("nil history")
	}
	var h24 *OptionsHistoryWindow
	for i := range got.Windows {
		if got.Windows[i].Window == OptionsHistoryWindow24h {
			h24 = &got.Windows[i]
		}
	}
	if h24 == nil || !h24.Gamma.Complete || h24.Gamma.Current != -200 || h24.Gamma.Past != 400 {
		t.Fatalf("24h gamma %+v", h24)
	}
	if !h24.HasRegimeChange || h24.Regime != GammaRegimeShort || h24.RegimeWas != GammaRegimeLong {
		t.Fatalf("regime %+v", h24)
	}
	if !h24.Flip.Complete || !h24.HasFlipSideChange || h24.FlipSide != GammaFlipAbove {
		t.Fatalf("flip %+v", h24)
	}
	var sawGamma bool
	for _, p := range got.Points {
		if p.HasGamma {
			sawGamma = true
			break
		}
	}
	if !sawGamma {
		t.Fatalf("points missing gamma %+v", got.Points)
	}
}

func TestBuildOptionsHistory_Empty(t *testing.T) {
	t.Parallel()
	if got := BuildOptionsHistory(nil, nil, nil, time.Time{}); got != nil {
		t.Fatalf("%+v", got)
	}
}

func TestDownsampleOptionsHistoryKeepsEnds(t *testing.T) {
	t.Parallel()
	in := make([]OptionsHistoryPoint, 0, 21)
	t0 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 21; i++ {
		in = append(in, OptionsHistoryPoint{Time: t0.Add(time.Duration(i) * time.Hour), HasDVOL: true, DVOL: float64(i)})
	}
	got := downsampleOptionsHistory(in, 5)
	if len(got) != 5 || !got[0].Time.Equal(in[0].Time) || !got[4].Time.Equal(in[20].Time) {
		t.Fatalf("%+v", got)
	}
}
