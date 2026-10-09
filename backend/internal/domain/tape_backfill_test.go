package domain

import (
	"testing"
	"time"
)

func TestTakerBucketsFromPrints_SumsClosedMinutes(t *testing.T) {
	minute := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	got := TakerBucketsFromPrints([]TakerPrint{
		{Exchange: ExchangeBinance, Symbol: "btcusdt", Side: TakerSideBuy, Notional: 100, Time: minute.Add(10 * time.Second)},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideSell, Notional: 40, Time: minute.Add(20 * time.Second)},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideBuy, Notional: 5, Time: minute.Add(time.Minute + time.Second)},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideBuy, Notional: 0, Time: minute},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: TakerSideBuy, Notional: 7, Time: minute},
	})
	if len(got) != 3 {
		t.Fatalf("buckets %+v", got)
	}
	if got[0].Exchange != ExchangeBinance || got[0].BuyNotional != 100 || got[0].SellNotional != 40 || !got[0].Start.Equal(minute) {
		t.Fatalf("first %+v", got[0])
	}
	if got[1].Exchange != ExchangeBybit || got[1].BuyNotional != 7 {
		t.Fatalf("bybit %+v", got[1])
	}
	if !got[2].Start.Equal(minute.Add(time.Minute)) || got[2].BuyNotional != 5 {
		t.Fatalf("next %+v", got[2])
	}
	for _, b := range got {
		if !b.SizeKnown || b.WhaleBuy != 0 || b.WhaleSell != 0 {
			t.Fatalf("small prints are retail %+v", b)
		}
	}
	if got[0].RetailBuy != 100 || got[0].RetailSell != 40 {
		t.Fatalf("retail %+v", got[0])
	}
}

func TestTakerBucketsFromPrints_SplitsWhaleAndRetail(t *testing.T) {
	minute := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	got := TakerBucketsFromPrints([]TakerPrint{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideBuy, Notional: DefaultWhaleMinNotional, Time: minute},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideSell, Notional: DefaultWhaleMinNotional - 1, Time: minute.Add(time.Second)},
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideSell, Notional: DefaultWhaleMinNotional, Time: minute.Add(2 * time.Second)},
	})
	if len(got) != 1 || !got[0].SizeKnown {
		t.Fatalf("%+v", got)
	}
	b := got[0]
	if b.WhaleBuy != DefaultWhaleMinNotional || b.WhaleSell != DefaultWhaleMinNotional {
		t.Fatalf("whale %+v", b)
	}
	if b.RetailSell != DefaultWhaleMinNotional-1 || b.RetailBuy != 0 {
		t.Fatalf("retail %+v", b)
	}
	if b.BuyNotional != b.WhaleBuy+b.RetailBuy || b.SellNotional != b.WhaleSell+b.RetailSell {
		t.Fatalf("split must partition the minute %+v", b)
	}
}

func TestTapeBackfillEnd_StopsAtStoredMinute(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 30, 0, time.UTC)
	live := now.Add(-BinanceTapeLiveGap).Truncate(time.Minute)
	if got := TapeBackfillEnd(now, time.Time{}, time.Time{}, false, false); !got.Equal(live) {
		t.Fatalf("live edge %s", got)
	}
	earliest := live.Add(-2 * time.Hour)
	if got := TapeBackfillEnd(now, earliest.Add(20*time.Second), time.Time{}, true, false); !got.Equal(earliest.Truncate(time.Minute)) {
		t.Fatalf("earliest %s", got)
	}
	scan := earliest.Add(-time.Hour)
	if got := TapeBackfillEnd(now, earliest, scan, true, true); !got.Equal(scan) {
		t.Fatalf("scan %s", got)
	}
	// A scan newer than the stored minute must not pull the end forward into stored rows.
	if got := TapeBackfillEnd(now, earliest, live, true, true); !got.Equal(earliest.Truncate(time.Minute)) {
		t.Fatalf("stored wins %s", got)
	}
}
