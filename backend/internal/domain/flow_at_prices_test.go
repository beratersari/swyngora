package domain

import (
	"errors"
	"testing"
	"time"
)

func TestParseFlowAtPricesWindow(t *testing.T) {
	id, dur, err := ParseFlowAtPricesWindow("")
	if err != nil || id != FlowAtPricesWindow15m || dur != 15*time.Minute {
		t.Fatalf("default %s %s %v", id, dur, err)
	}
	id, dur, err = ParseFlowAtPricesWindow("5m")
	if err != nil || id != FlowAtPricesWindow5m || dur != 5*time.Minute {
		t.Fatalf("5m %s %s %v", id, dur, err)
	}
	id, dur, err = ParseFlowAtPricesWindow("30d")
	if err != nil || id != "30d" || dur != 30*24*time.Hour {
		t.Fatalf("30d %s %v %v", id, dur, err)
	}
	id, dur, err = ParseFlowAtPricesWindow("1d")
	if err != nil || id != FlowAtPricesWindow1d || dur != 24*time.Hour {
		t.Fatalf("1d %s %s %v", id, dur, err)
	}
	if _, _, err := ParseFlowAtPricesWindow("24h"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("bad window %v", err)
	}
}

func TestResolveFlowAtPricesRange_CustomAndCap(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	from, to, id, err := ResolveFlowAtPricesRange("15m", nil, nil, now)
	if err != nil || id != FlowAtPricesWindow15m || to != now || to.Sub(from) != 15*time.Minute {
		t.Fatalf("%s %s %s %v", id, from, to, err)
	}
	start := now.Add(-10 * time.Minute)
	end := now.Add(-5 * time.Minute)
	from, to, id, err = ResolveFlowAtPricesRange("1h", &start, &end, now)
	if err != nil || id != "custom" || !from.Equal(start) || !to.Equal(end) {
		t.Fatalf("custom %s %s %s %v", id, from, to, err)
	}
	hours := now.Add(-4 * time.Hour)
	from, to, id, err = ResolveFlowAtPricesRange("", &hours, &now, now)
	if err != nil || to.Sub(from) != 4*time.Hour {
		t.Fatalf("4h custom %s %v", to.Sub(from), err)
	}
	tooLong := now.Add(-31 * 24 * time.Hour)
	if _, _, _, err := ResolveFlowAtPricesRange("", &tooLong, &now, now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("cap %v", err)
	}
	after := now.Add(time.Hour)
	if _, _, _, err := ResolveFlowAtPricesRange("", &after, &now, now); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("order %v", err)
	}
}

func TestBuildFlowAtPricesVenue_LevelBuySellDelta(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 15, 0, 0, time.UTC)
	from := now.Add(-15 * time.Minute)
	prints := []TakerPrint{
		{Side: TakerSideBuy, Price: 65520, Notional: 8_000_000, Time: now.Add(-10 * time.Minute)},
		{Side: TakerSideBuy, Price: 65521, Notional: 500_000, Time: now.Add(-9 * time.Minute)},
		{Side: TakerSideSell, Price: 65520, Notional: 2_000_000, Time: now.Add(-8 * time.Minute)},
		{Side: TakerSideSell, Price: 65400, Notional: 1_000_000, Time: now.Add(-7 * time.Minute)},
		{Side: TakerSideBuy, Price: 65600, Notional: 300_000, Time: now.Add(-6 * time.Minute)},
	}
	got := BuildFlowAtPricesVenue(ExchangeBinance, "BTCUSDT", prints, 10, 65550, 0, 0, from, now)
	if got.Error != "" {
		t.Fatal(got.Error)
	}
	if got.POC.Price != 65520 {
		t.Fatalf("poc %v levels=%+v", got.POC.Price, got.Levels)
	}
	row := levelAt(got.Levels, 65520)
	if row.BuyVolume != 8_500_000 || row.SellVolume != 2_000_000 || row.Delta != 6_500_000 {
		t.Fatalf("65520 %+v", row)
	}
	if row.Trades != 3 || row.BuyCount != 2 || row.SellCount != 1 {
		t.Fatalf("counts %+v", row)
	}
	if !row.IsPoc || row.Dominant != TakerSideBuy {
		t.Fatalf("poc/dom %+v", row)
	}
	if got.Levels[0].Price < got.Levels[len(got.Levels)-1].Price {
		t.Fatalf("want high-to-low %+v", pricesOf(got.Levels))
	}
	if got.Delta != 8_500_000+300_000-2_000_000-1_000_000 {
		t.Fatalf("tape delta %v", got.Delta)
	}
}

func TestBuildFlowAtPricesVenue_PriceBand(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 15, 0, 0, time.UTC)
	from := now.Add(-15 * time.Minute)
	prints := []TakerPrint{
		{Side: TakerSideBuy, Price: 65520, Notional: 8_000_000, Time: now.Add(-5 * time.Minute)},
		{Side: TakerSideSell, Price: 64000, Notional: 9_000_000, Time: now.Add(-4 * time.Minute)},
		{Side: TakerSideBuy, Price: 65600, Notional: 100_000, Time: now.Add(-3 * time.Minute)},
	}
	got := BuildFlowAtPricesVenue(ExchangeBinance, "btcusdt", prints, 10, 65500, 65400, 65600, from, now)
	if len(got.Levels) != 2 {
		t.Fatalf("band %+v", got.Levels)
	}
	if levelAt(got.Levels, 64000).Trades != 0 {
		t.Fatalf("64000 should be clipped %+v", got.Levels)
	}
	if got.POC.Price != 65520 {
		t.Fatalf("poc %v", got.POC.Price)
	}
}

func TestBuildFlowAtPricesVenue_Empty(t *testing.T) {
	now := time.Now().UTC()
	got := BuildFlowAtPricesVenue(ExchangeBybit, "ETHUSDT", nil, 1, 0, 0, 0, now.Add(-time.Minute), now)
	if got.Error == "" || len(got.Levels) != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestCombineFlowAtPricesVenues(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 15, 0, 0, time.UTC)
	from := now.Add(-15 * time.Minute)
	bn := BuildFlowAtPricesVenue(ExchangeBinance, "BTCUSDT", []TakerPrint{
		{Side: TakerSideBuy, Price: 65520, Notional: 4_000_000, Time: now.Add(-10 * time.Minute)},
		{Side: TakerSideSell, Price: 65520, Notional: 1_000_000, Time: now.Add(-9 * time.Minute)},
	}, 10, 65520, 0, 0, from, now)
	by := BuildFlowAtPricesVenue(ExchangeBybit, "BTCUSDT", []TakerPrint{
		{Side: TakerSideBuy, Price: 65525, Notional: 1_000_000, Time: now.Add(-8 * time.Minute)},
		{Side: TakerSideSell, Price: 65520, Notional: 500_000, Time: now.Add(-7 * time.Minute)},
	}, 10, 65520, 0, 0, from, now)
	got := CombineFlowAtPricesVenues("BTCUSDT", []FlowAtPricesVenue{bn, by}, 10, 65520, from, now)
	if got == nil {
		t.Fatal("nil")
	}
	row := levelAt(got.Levels, 65520)
	if row.BuyVolume != 5_000_000 || row.SellVolume != 1_500_000 || row.Trades != 4 {
		t.Fatalf("combined 65520 %+v", row)
	}
	if got.Delta != 5_000_000-1_500_000 {
		t.Fatalf("delta %v", got.Delta)
	}
}

func TestExplainFlowAtPricesReport(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 15, 0, 0, time.UTC)
	from := now.Add(-15 * time.Minute)
	v := BuildFlowAtPricesVenue(ExchangeBinance, "BTCUSDT", []TakerPrint{
		{Side: TakerSideBuy, Price: 65520, Notional: 8_000_000, Time: now.Add(-10 * time.Minute)},
		{Side: TakerSideSell, Price: 65520, Notional: 2_000_000, Time: now.Add(-9 * time.Minute)},
	}, 1, 65520, 0, 0, from, now)
	r := FlowAtPricesReport{Symbol: "BTCUSDT", Venues: []FlowAtPricesVenue{v}, Combined: &v}
	got := ExplainFlowAtPricesReport(r)
	if got == "" || got == v.Error {
		t.Fatalf("explain %q", got)
	}
}

func levelAt(levels []FlowAtPricesLevel, price float64) FlowAtPricesLevel {
	for _, lv := range levels {
		if lv.Price == price {
			return lv
		}
	}
	return FlowAtPricesLevel{}
}

func pricesOf(levels []FlowAtPricesLevel) []float64 {
	out := make([]float64, len(levels))
	for i, lv := range levels {
		out[i] = lv.Price
	}
	return out
}
