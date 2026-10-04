package domain

import (
	"strings"
	"testing"
	"time"
)

func TestDetectCascadeVenue_BurstVsTypical(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	// Quiet hour: one small long every 5 minutes.
	for i := 1; i <= 70; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
			Price: 64000, Quantity: 0.01, Notional: 80, Time: now.Add(-time.Duration(i*5) * time.Minute),
		})
	}
	// Last minute: many longs.
	for i := 0; i < 6; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong,
			Price: 64000, Quantity: 1, Notional: 2000, Time: now.Add(-time.Duration(5+i) * time.Second),
		})
	}
	got := DetectCascadeVenue(ExchangeBinance, "BTCUSDT", ev, now)
	if got.Side != LiquidationSideLong {
		t.Fatalf("side %+v", got)
	}
	if cascadeGradeRank(got.Grade) < cascadeGradeRank(CascadeGradeCascade) {
		t.Fatalf("grade %s score %.1f windows %+v", got.Grade, got.Score, got.Windows)
	}
	quiet := DetectCascadeVenue(ExchangeBybit, "BTCUSDT", ev, now)
	if quiet.Grade != CascadeGradeQuiet {
		t.Fatalf("bybit must stay quiet %+v", quiet)
	}
}

func TestDetectCascadeBoth_SameSide(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit, ExchangeOKX} {
		for i := 1; i <= 40; i++ {
			ev = append(ev, LiquidationEvent{
				Exchange: ex, Symbol: "ETHUSDT", Side: LiquidationSideShort,
				Notional: 60, Time: now.Add(-time.Duration(i*6) * time.Minute),
			})
		}
		for i := 0; i < 6; i++ {
			ev = append(ev, LiquidationEvent{
				Exchange: ex, Symbol: "ETHUSDT", Side: LiquidationSideShort,
				Notional: 3000, Time: now.Add(-time.Duration(4+i) * time.Second),
			})
		}
	}
	rep := BuildCascadeReport("ETHUSDT", "all", ev, now)
	if rep.Both == nil || !rep.Both.Agree || rep.Both.Side != LiquidationSideShort {
		t.Fatalf("both %+v venues %+v", rep.Both, rep.Venues)
	}
	if !strings.Contains(rep.Both.Summary, "OKX") {
		t.Fatalf("summary %s", rep.Both.Summary)
	}
}

func TestDetectCascadeBoth_QuietVenueDoesNotAgree(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit} {
		for i := 1; i <= 40; i++ {
			ev = append(ev, LiquidationEvent{
				Exchange: ex, Symbol: "ETHUSDT", Side: LiquidationSideShort,
				Notional: 60, Time: now.Add(-time.Duration(i*6) * time.Minute),
			})
		}
		for i := 0; i < 6; i++ {
			ev = append(ev, LiquidationEvent{
				Exchange: ex, Symbol: "ETHUSDT", Side: LiquidationSideShort,
				Notional: 3000, Time: now.Add(-time.Duration(4+i) * time.Second),
			})
		}
	}
	rep := BuildCascadeReport("ETHUSDT", "all", ev, now)
	if rep.Both == nil || rep.Both.Agree {
		t.Fatalf("both %+v", rep.Both)
	}
	if !strings.Contains(rep.Both.Summary, "OKX") {
		t.Fatalf("summary %s", rep.Both.Summary)
	}
}

func TestDetectCascadeBoth_Rules(t *testing.T) {
	if DetectCascadeBoth(nil) != nil {
		t.Fatal("empty slice")
	}
	agree := DetectCascadeBoth([]CascadeVenue{
		{Exchange: ExchangeBinance, Side: LiquidationSideShort, Grade: CascadeGradeCascade, Score: 40, Hottest: CascadeWindow15m},
		{Exchange: ExchangeBybit, Side: LiquidationSideShort, Grade: CascadeGradeExtreme, Score: 80, Hottest: CascadeWindow1m},
		{Exchange: ExchangeOKX, Side: CascadeSideBoth, Grade: CascadeGradeCascade, Score: 55, Hottest: CascadeWindow5m},
	})
	if agree == nil || !agree.Agree || agree.Side != LiquidationSideShort || agree.Grade != CascadeGradeCascade || agree.Score != 40 || agree.Hottest != CascadeWindow1m {
		t.Fatalf("agree %+v", agree)
	}
	if !strings.Contains(agree.Summary, "Binance, Bybit, and OKX") {
		t.Fatalf("summary %s", agree.Summary)
	}
	missing := DetectCascadeBoth([]CascadeVenue{
		{Exchange: ExchangeBinance, Side: LiquidationSideLong, Grade: CascadeGradeCascade, Score: 50, Hottest: CascadeWindow1m},
		{Exchange: ExchangeBybit, Side: LiquidationSideLong, Grade: CascadeGradeCascade, Score: 50, Hottest: CascadeWindow1m},
	})
	if missing == nil || missing.Agree || !strings.Contains(missing.Summary, "OKX") {
		t.Fatalf("missing %+v", missing)
	}
	partial := DetectCascadeBoth([]CascadeVenue{
		{Exchange: ExchangeBinance, Side: LiquidationSideLong, Grade: CascadeGradeCascade, Score: 50, Hottest: CascadeWindow1m},
		{Exchange: ExchangeBybit, Side: CascadeSideNone, Grade: CascadeGradeQuiet},
		{Exchange: ExchangeOKX, Side: CascadeSideNone, Grade: CascadeGradeQuiet},
	})
	if partial == nil || partial.Agree || !strings.Contains(partial.Summary, "Binance has a long cascade") || !strings.Contains(partial.Summary, "Bybit and OKX are not") {
		t.Fatalf("partial %+v", partial)
	}
	opposite := DetectCascadeBoth([]CascadeVenue{
		{Exchange: ExchangeBinance, Side: LiquidationSideLong, Grade: CascadeGradeCascade, Score: 50, Hottest: CascadeWindow1m},
		{Exchange: ExchangeBybit, Side: LiquidationSideShort, Grade: CascadeGradeCascade, Score: 50, Hottest: CascadeWindow1m},
		{Exchange: ExchangeOKX, Side: CascadeSideNone, Grade: CascadeGradeQuiet},
	})
	if opposite == nil || opposite.Agree || opposite.Side != CascadeSideNone || opposite.Summary != "Venues are bursting on opposite sides." {
		t.Fatalf("opposite %+v", opposite)
	}
	elevated := DetectCascadeBoth([]CascadeVenue{
		{Exchange: ExchangeBinance, Side: LiquidationSideShort, Grade: CascadeGradeElevated, Score: 20, Hottest: CascadeWindow5m},
		{Exchange: ExchangeBybit, Side: LiquidationSideShort, Grade: CascadeGradeElevated, Score: 22, Hottest: CascadeWindow5m},
		{Exchange: ExchangeOKX, Side: LiquidationSideShort, Grade: CascadeGradeCascade, Score: 40, Hottest: CascadeWindow5m},
	})
	if elevated == nil || elevated.Agree || !strings.Contains(elevated.Summary, "not yet a confirmed cascade") {
		t.Fatalf("elevated %+v", elevated)
	}
}

func TestBuildCascadeScan_MarketAndCoins(t *testing.T) {
	now := time.Date(2026, 8, 30, 16, 0, 0, 0, time.UTC)
	var ev []LiquidationEvent
	for i := 1; i <= 40; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: LiquidationSideLong,
			Notional: 50, Time: now.Add(-time.Duration(i*6) * time.Minute),
		})
	}
	for i := 0; i < 6; i++ {
		ev = append(ev, LiquidationEvent{
			Exchange: ExchangeBinance, Symbol: "SOLUSDT", Side: LiquidationSideLong,
			Notional: 2500, Time: now.Add(-time.Duration(3+i) * time.Second),
		})
	}
	ev = append(ev, LiquidationEvent{
		Exchange: ExchangeBinance, Symbol: "AAAUSDT", Side: LiquidationSideShort,
		Notional: 40, Time: now.Add(-2 * time.Hour),
	})
	scan := BuildCascadeScan("all", ev, now)
	if len(scan.Hits) == 0 || scan.Hits[0].Symbol != "SOLUSDT" {
		t.Fatalf("hits %+v", scan.Hits)
	}
	if scan.Market.Symbol != "all" || len(scan.Market.Venues) != len(FuturesVenues()) {
		t.Fatalf("market %+v", scan.Market)
	}
}

func TestEventsSince_AllAndVenue(t *testing.T) {
	b := NewLiquidationBook()
	now := time.Now().UTC()
	b.Record(LiquidationEvent{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong, Notional: 1, Time: now})
	b.Record(LiquidationEvent{Exchange: ExchangeBybit, Symbol: "ETHUSDT", Side: LiquidationSideShort, Notional: 2, Time: now})
	all := b.EventsSince("all", "all", now.Add(-time.Minute))
	if len(all) != 2 {
		t.Fatalf("all %d", len(all))
	}
	bn := b.EventsSince("binance", "", now.Add(-time.Minute))
	if len(bn) != 1 || bn[0].Exchange != ExchangeBinance {
		t.Fatalf("binance %+v", bn)
	}
}
