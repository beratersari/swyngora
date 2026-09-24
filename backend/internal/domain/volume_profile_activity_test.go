package domain

import (
	"testing"
	"time"
)

func TestAttachVolumeProfileActivity_SamePriceRows(t *testing.T) {
	from := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	bars := []VolumeProfileBar{
		{Time: from, Low: 100, High: 100.4, Close: 100.2, Volume: 10, BuyVolume: 4, SellVolume: 6, BuySellKnown: true},
		{Time: from.Add(5 * time.Minute), Low: 102, High: 102.4, Close: 102.2, Volume: 100, BuyVolume: 80, SellVolume: 20, BuySellKnown: true},
	}
	got := BuildVolumeProfile(ExchangeBinance, "BTCUSDT", bars, 1, 102.2, from, from.Add(10*time.Minute), Interval5m)
	if got.Error != "" {
		t.Fatal(got.Error)
	}
	oi := []VolumeProfileOiPoint{
		{Time: from, Value: 1_000_000},
		{Time: from.Add(5 * time.Minute), Value: 1_300_000},  // +300k while 102 traded
		{Time: from.Add(10 * time.Minute), Value: 1_100_000}, // -200k while 102 still the only later bar? second step overlaps second bar only if bar covers it
	}
	// Second bar opens at from+5m and covers [from+5m, from+10m). The second OI step is (from+5m, from+10m], so it overlaps that bar.
	liqs := []LiquidationEvent{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong, Price: 102.1, Notional: 80_000, Time: from.Add(6 * time.Minute)},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideShort, Price: 100.2, Notional: 25_000, Time: from.Add(time.Minute)},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: LiquidationSideLong, Price: 90, Notional: 9_000, Time: from.Add(time.Minute)},
	}
	AttachVolumeProfileActivity(&got, bars, oi, liqs)
	if !got.OiKnown || !got.LiqKnown {
		t.Fatalf("flags oi=%v liq=%v", got.OiKnown, got.LiqKnown)
	}
	var at100, at102 VolumeProfileBin
	for _, b := range got.Bins {
		switch {
		case b.Price == 100:
			at100 = b
		case b.Price == 102:
			at102 = b
		}
	}
	if at102.Volume <= at100.Volume {
		t.Fatalf("volume should still cluster at 102: %+v %+v", at100, at102)
	}
	if at100.OiOpened < 300_000-1 || at102.OiClosed < 200_000-1 {
		t.Fatalf("oi 100 opened=%v 102 closed=%v", at100.OiOpened, at102.OiClosed)
	}
	if at102.OiOpened > 1 || at100.OiClosed > 1 {
		t.Fatalf("oi crossed rows 100=%+v 102=%+v", at100, at102)
	}
	if at102.LongLiq != 80_000 || at100.ShortLiq != 25_000 {
		t.Fatalf("liqs 102=%+v 100=%+v", at102, at100)
	}
	if got.LongLiq != 80_000 {
		t.Fatalf("outside print must stay off the ladder, long=%v", got.LongLiq)
	}
	if got.Summary == "" || !containsAll(got.Summary, "opened") || !containsAll(got.Summary, "closed") || !containsAll(got.Summary, "long") || !containsAll(got.Summary, "short") {
		t.Fatalf("summary %q", got.Summary)
	}
}

func TestCombineVolumeProfiles_SumsOiAndLiquidations(t *testing.T) {
	from := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	mk := func(ex Exchange, oiOpen, longLiq float64) VolumeProfileVenue {
		bars := []VolumeProfileBar{{Time: from, Low: 100, High: 100.4, Volume: 50, BuyVolume: 30, SellVolume: 20, BuySellKnown: true}}
		v := BuildVolumeProfile(ex, "BTCUSDT", bars, 1, 100.2, from, from.Add(time.Hour), Interval5m)
		v.Bins[0].OiOpened = oiOpen
		v.OiOpened = oiOpen
		v.OiKnown = true
		v.Bins[0].LongLiq = longLiq
		v.LongLiq = longLiq
		v.LiqKnown = true
		return v
	}
	got := CombineVolumeProfiles("BTCUSDT", []VolumeProfileVenue{
		mk(ExchangeBinance, 1_000_000, 40_000),
		mk(ExchangeBybit, 250_000, 10_000),
	}, 1, 100.2, from, from.Add(time.Hour), Interval5m)
	if got == nil || !got.OiKnown || !got.LiqKnown {
		t.Fatalf("%+v", got)
	}
	if len(got.Bins) != 1 || got.Bins[0].OiOpened != 1_250_000 || got.Bins[0].LongLiq != 50_000 {
		t.Fatalf("combined bin %+v", got.Bins)
	}
	if got.OiOpened != 1_250_000 || got.LongLiq != 50_000 {
		t.Fatalf("combined totals oi=%v liq=%v", got.OiOpened, got.LongLiq)
	}
}


