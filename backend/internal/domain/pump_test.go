package domain

import (
	"testing"
	"time"
)

func candle(t0 time.Time, o, h, l, c, vol string) Candle {
	return Candle{
		OpenTime: t0, CloseTime: t0.Add(time.Hour),
		Open: o, High: h, Low: l, Close: c, Volume: vol,
	}
}

func TestDetectPumpEvents_CloseReturnUp(t *testing.T) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	// flat then +10% bar
	candles := []Candle{
		candle(base, "100", "101", "99", "100", "10"),
		candle(base.Add(time.Hour), "100", "111", "100", "110", "50"),
	}
	ev, err := DetectPumpEvents(candles, PumpDetectOptions{
		MinReturnPct: 5,
		WindowBars:   1,
		Mode:         PumpModeCloseReturn,
		Direction:    PumpDirectionUp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 {
		t.Fatalf("want 1 event, got %d", len(ev))
	}
	if ev[0].ReturnPct < 9.9 || ev[0].ReturnPct > 10.1 {
		t.Fatalf("return=%v", ev[0].ReturnPct)
	}
}

func TestDetectPumpEvents_VolumeFilter(t *testing.T) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		candle(base, "100", "100", "100", "100", "100"),
		candle(base.Add(time.Hour), "100", "120", "100", "120", "10"), // +20% but low vol
	}
	ev, err := DetectPumpEvents(candles, PumpDetectOptions{
		MinReturnPct:   5,
		WindowBars:     1,
		MinVolumeRatio: 2, // need 2x median
		Direction:      PumpDirectionUp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 0 {
		t.Fatalf("expected volume filter to drop event, got %d", len(ev))
	}
}

func TestDetectPumpEvents_Dump(t *testing.T) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	candles := []Candle{
		candle(base, "100", "100", "100", "100", "10"),
		candle(base.Add(time.Hour), "100", "100", "80", "85", "10"),
	}
	ev, err := DetectPumpEvents(candles, PumpDetectOptions{
		MinReturnPct: 10,
		Direction:    PumpDirectionDown,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 1 || ev[0].ReturnPct > -10 {
		t.Fatalf("%+v", ev)
	}
}

func TestBestPumpEvent(t *testing.T) {
	t.Parallel()
	if _, ok := BestPumpEvent(nil); ok {
		t.Fatal("empty")
	}
	got, ok := BestPumpEvent([]PumpEvent{
		{ReturnPct: 4, VolumeRatio: 1, OpenTime: time.Unix(1, 0).UTC()},
		{ReturnPct: -12, VolumeRatio: 3, OpenTime: time.Unix(2, 0).UTC()},
		{ReturnPct: 8, VolumeRatio: 2, OpenTime: time.Unix(3, 0).UTC()},
	})
	if !ok || got.ReturnPct != -12 || got.VolumeRatio != 3 {
		t.Fatalf("%+v ok=%v", got, ok)
	}
}

func TestBarsForLookbackHours(t *testing.T) {
	n, err := BarsForLookbackHours(Interval1h, 48)
	if err != nil || n != 48 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	n, err = BarsForLookbackHours(Interval15m, 6)
	if err != nil || n != 24 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
