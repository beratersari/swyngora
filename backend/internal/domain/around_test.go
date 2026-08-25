package domain

import (
	"errors"
	"testing"
	"time"
)

func TestResolveAroundPlan_AdjacentWindows(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	now := at.Add(3 * time.Hour)
	got, err := ResolveAroundPlan("1h", "15m", at, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Window != AroundWindow1h || got.During != AroundDuring15m {
		t.Fatalf("ids %+v", got)
	}
	if !got.BeforeFrom.Equal(at.Add(-time.Hour)) || !got.BeforeTo.Equal(at) {
		t.Fatalf("before %s %s", got.BeforeFrom, got.BeforeTo)
	}
	if !got.DuringFrom.Equal(at) || !got.DuringTo.Equal(at.Add(15*time.Minute)) {
		t.Fatalf("during %s %s", got.DuringFrom, got.DuringTo)
	}
	if !got.AfterFrom.Equal(at.Add(15*time.Minute)) || !got.AfterTo.Equal(at.Add(15*time.Minute+time.Hour)) {
		t.Fatalf("after %s %s", got.AfterFrom, got.AfterTo)
	}
	if got.Clipped {
		t.Fatal("should not clip")
	}
}

func TestResolveAroundPlan_ClipsFutureAfter(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	now := at.Add(20 * time.Minute)
	got, err := ResolveAroundPlan("1h", "15m", at, now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Clipped || !got.AfterTo.Equal(now) {
		t.Fatalf("clip %+v", got)
	}
	if !got.DuringTo.Equal(at.Add(15 * time.Minute)) {
		t.Fatalf("during still complete %s", got.DuringTo)
	}
}

func TestResolveAroundPlan_RejectsFutureAndMissing(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	if _, err := ResolveAroundPlan("1h", "15m", time.Time{}, now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("missing %v", err)
	}
	if _, err := ResolveAroundPlan("1h", "15m", now.Add(time.Minute), now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("future %v", err)
	}
	if _, err := ResolveAroundPlan("3d", "15m", now.Add(-time.Hour), now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("window %v", err)
	}
	if _, err := ResolveAroundPlan("1h", "2h", now.Add(-time.Hour), now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("during %v", err)
	}
}

func TestBuildAroundPhase_PriceVolumeVWAPAndTypical(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	var all []AroundBar
	// Eight quiet prior hours at 1_000 volume, then a 10_000 during bar.
	for i := 9; i >= 1; i-- {
		t0 := at.Add(-time.Duration(i) * time.Hour)
		all = append(all, AroundBar{
			Time: t0, Open: 100, High: 101, Low: 99, Close: 100, Volume: 1_000,
			BuyVolume: 500, SellVolume: 500, BuySellKnown: true,
		})
	}
	all = append(all, AroundBar{
		Time: at, Open: 100, High: 110, Low: 100, Close: 109, Volume: 10_000,
		BuyVolume: 8_000, SellVolume: 2_000, BuySellKnown: true,
	})
	ph := BuildAroundPhase(AroundPhaseDuring, all, all, at, at.Add(time.Hour))
	if !ph.Complete || ph.Price.Open != 100 || ph.Price.Close != 109 {
		t.Fatalf("price %+v", ph.Price)
	}
	if ph.Price.Direction != CVDDirUp || ph.Price.ChangePct < 8.9 || ph.Price.ChangePct > 9.1 {
		t.Fatalf("pct %+v", ph.Price)
	}
	if ph.Flow.Volume != 10_000 || ph.Flow.Dominant != TakerSideBuy {
		t.Fatalf("flow %+v", ph.Flow)
	}
	if ph.Flow.VWAP < 106 || ph.Flow.VWAP > 107 {
		t.Fatalf("vwap %+v", ph.Flow)
	}
	if !ph.Flow.TypicalKnown || ph.Flow.Typical < 900 || ph.Flow.Typical > 1100 {
		t.Fatalf("typical %+v", ph.Flow)
	}
	if ph.Flow.VolumeRatio < 9 || ph.Flow.VolumeGrade != VolumeSurgeExtreme {
		t.Fatalf("ratio %+v", ph.Flow)
	}
	if ph.Summary == "" {
		t.Fatal("summary")
	}
}

func TestCompareAroundPhases_ReversedAfterSpike(t *testing.T) {
	mk := func(phase string, open, close, vol float64, buy, sell float64) AroundPhase {
		ch := close - open
		pct := ch / open * 100
		return AroundPhase{
			Phase: phase, Complete: true,
			Price: AroundPrice{Open: open, Close: close, Change: ch, ChangePct: pct, Direction: changeDir(pct)},
			Flow: AroundFlow{
				Volume: vol, BuyVolume: buy, SellVolume: sell, Delta: buy - sell,
				BuySellKnown: true, Dominant: TakerDominant(buy, sell),
			},
		}
	}
	got := CompareAroundPhases([]AroundPhase{
		mk(AroundPhaseBefore, 100, 100.2, 1_000, 500, 500),
		mk(AroundPhaseDuring, 100.2, 104, 8_000, 6_000, 2_000),
		mk(AroundPhaseAfter, 104, 101, 2_000, 500, 1_500),
	})
	var price, vol, delta AroundChange
	for _, c := range got {
		switch c.Metric {
		case "price":
			price = c
		case "volume":
			vol = c
		case "delta":
			delta = c
		}
	}
	if price.Path != AroundPathReversed {
		t.Fatalf("price path %s (%s)", price.Path, price.Summary)
	}
	if vol.Path != AroundPathFaded {
		t.Fatalf("vol path %s", vol.Path)
	}
	if delta.Path != AroundPathReversed {
		t.Fatalf("delta path %s", delta.Path)
	}
}

func TestCompareAroundPhases_ContinuedMove(t *testing.T) {
	mk := func(phase string, open, close, vol float64) AroundPhase {
		ch := close - open
		pct := ch / open * 100
		return AroundPhase{
			Phase: phase, Complete: true,
			Price: AroundPrice{Open: open, Close: close, Change: ch, ChangePct: pct, Direction: changeDir(pct)},
			Flow:  AroundFlow{Volume: vol},
		}
	}
	got := CompareAroundPhases([]AroundPhase{
		mk(AroundPhaseBefore, 100, 100.1, 1_000),
		mk(AroundPhaseDuring, 100.1, 103, 5_000),
		mk(AroundPhaseAfter, 103, 105, 4_000),
	})
	if got[0].Metric != "price" || got[0].Path != AroundPathContinued {
		t.Fatalf("%+v", got)
	}
}

func TestAroundSweepsToEvents_FiltersByWindow(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	sweeps := []LiquiditySweep{
		{Side: LiquiditySweepSideHigh, Level: 100, PiercedAt: at.Add(-2 * time.Hour), Title: "old"},
		{Side: LiquiditySweepSideHigh, Level: 110, PiercedAt: at.Add(2 * time.Minute), Title: "in", Summary: "swept 110"},
		{Side: LiquiditySweepSideLow, Level: 90, PiercedAt: at.Add(2 * time.Hour), Title: "late"},
	}
	got := AroundSweepsToEvents(sweeps, at, at.Add(15*time.Minute))
	if len(got) != 1 || got[0].Level != 110 || got[0].Kind != AroundEventSweep {
		t.Fatalf("%+v", got)
	}
}

func TestCombineAroundVenues_SumsVolume(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	plan, err := ResolveAroundPlan("1h", "15m", at, at.Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mk := func(ex Exchange, duringVol float64) AroundVenue {
		bars := []AroundBar{
			{Time: at.Add(-30 * time.Minute), Open: 100, High: 101, Low: 99, Close: 100, Volume: 1_000},
			{Time: at, Open: 100, High: 104, Low: 100, Close: 103, Volume: duringVol},
			{Time: at.Add(20 * time.Minute), Open: 103, High: 103, Low: 102, Close: 102.5, Volume: 800},
		}
		return BuildAroundVenue(ex, "BTCUSDT", bars, plan, Interval1m)
	}
	got := CombineAroundVenues("BTCUSDT", []AroundVenue{
		mk(ExchangeBinance, 5_000),
		mk(ExchangeBybit, 3_000),
	}, plan, Interval1m)
	if got == nil || got.Error != "" {
		t.Fatalf("%+v", got)
	}
	var during AroundPhase
	for _, p := range got.Phases {
		if p.Phase == AroundPhaseDuring {
			during = p
		}
	}
	if during.Flow.Volume != 8_000 {
		t.Fatalf("combined during vol %+v", during)
	}
	if got.Summary == "" {
		t.Fatal("summary")
	}
}

func TestExplainAroundReport_PrefersCombined(t *testing.T) {
	r := AroundReport{
		Combined: &AroundVenue{Summary: "BTC moved +2% during the window."},
		Venues:   []AroundVenue{{Summary: "binance only"}},
	}
	if ExplainAroundReport(r) != "Combined: BTC moved +2% during the window." {
		t.Fatalf("%s", ExplainAroundReport(r))
	}
}

func TestLatestFuturesSnapshotAtOrBefore_NeverUsesLaterSample(t *testing.T) {
	at := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	rows := []FuturesSnapshot{
		{SampledAt: at.Add(-20 * time.Minute), Value: 100},
		{SampledAt: at.Add(5 * time.Minute), Value: 200},
	}
	got := LatestFuturesSnapshotAtOrBefore(rows, at)
	if got == nil || got.Value != 100 {
		t.Fatalf("should keep last known before the move %+v", got)
	}
	if NearestFuturesSnapshot(rows, at, 10*time.Minute).Value != 200 {
		t.Fatal("nearest still allowed to pick after (legacy)")
	}
}
