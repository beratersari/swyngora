package domain

import (
	"math"
	"testing"
	"time"
)

func TestSpreadTarget_SplitsWhatIsLeft(t *testing.T) {
	if math.Abs(SpreadTarget(1, 0, SpreadDefaultSlices)-0.1) > 1e-12 {
		t.Fatalf("first slice %v", SpreadTarget(1, 0, SpreadDefaultSlices))
	}
	if math.Abs(SpreadTarget(0.9, 1, SpreadDefaultSlices)-0.1) > 1e-12 {
		t.Fatalf("second slice %v", SpreadTarget(0.9, 1, SpreadDefaultSlices))
	}
	if math.Abs(SpreadTarget(0.4, 9, SpreadDefaultSlices)-0.4) > 1e-12 {
		t.Fatalf("last slice takes the rest, got %v", SpreadTarget(0.4, 9, SpreadDefaultSlices))
	}
}

func TestNormalizeSpreadSchedule(t *testing.T) {
	def, err := NormalizeSpreadSchedule(0, 0)
	if err != nil || def.SliceCount != 10 || def.Minutes != 10 {
		t.Fatalf("default %+v %v", def, err)
	}
	if def.Duration != 10*time.Minute || def.Interval != 10*time.Minute/9 {
		t.Fatalf("default gap %s duration %s", def.Interval, def.Duration)
	}
	wide, err := NormalizeSpreadSchedule(30, 15)
	if err != nil || wide.Interval != 30*time.Minute/14 || wide.IntervalSec() != 129 {
		t.Fatalf("30m/15 %+v sec %d err %v", wide, wide.IntervalSec(), err)
	}
	short, err := NormalizeSpreadSchedule(5, 5)
	if err != nil || short.Interval != 75*time.Second {
		t.Fatalf("5m/5 %+v %v", short, err)
	}
	if _, err := NormalizeSpreadSchedule(1, 60); err == nil {
		t.Fatal("60 slices in one minute should be too close")
	}
	if _, err := NormalizeSpreadSchedule(0, 1); err == nil {
		t.Fatal("one slice is not a spread")
	}
}

func TestSpreadReservePrice_UsesCapOrOnePercent(t *testing.T) {
	if math.Abs(SpreadReservePrice(100, 0)-101) > 1e-9 {
		t.Fatalf("buffer %v", SpreadReservePrice(100, 0))
	}
	if math.Abs(SpreadReservePrice(100, 0.005)-100.5) > 1e-9 {
		t.Fatalf("cap %v", SpreadReservePrice(100, 0.005))
	}
	lock := SpreadBuyReserve(2, 100, 0, CoinbaseFeeRate)
	if math.Abs(lock-2*101*1.006) > 1e-6 {
		t.Fatalf("lock %v", lock)
	}
}

func TestAllocateSpreadHolds_FollowsFreeQuantity(t *testing.T) {
	holds, ok := AllocateSpreadHolds(1, []SpreadFree{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Free: 1},
		{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Free: 1},
		{Exchange: ExchangeBybit, Symbol: "BTCUSDT", Free: 0},
	})
	if !ok || len(holds) != 2 {
		t.Fatalf("holds %+v ok %v", holds, ok)
	}
	var sum float64
	for _, h := range holds {
		sum += h.Quantity
		if math.Abs(h.Quantity-0.5) > 1e-9 {
			t.Fatalf("share %+v", h)
		}
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("sum %v", sum)
	}
	if _, ok := AllocateSpreadHolds(3, []SpreadFree{{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Free: 1}}); ok {
		t.Fatal("short inventory was accepted")
	}
}

func TestReduceSpreadHoldsAndBuyRelease(t *testing.T) {
	left := ReduceSpreadHolds([]SpreadHold{
		{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quantity: 0.5},
		{Exchange: ExchangeCoinbase, Symbol: "BTC-USD", Quantity: 0.5},
	}, []SpreadHold{{Exchange: ExchangeBinance, Symbol: "BTCUSDT", Quantity: 0.1}})
	if len(left) != 2 || math.Abs(left[0].Quantity-0.4) > 1e-9 {
		t.Fatalf("left %+v", left)
	}
	release := SpreadBuyRelease(100, 1, 0.1, 9)
	if math.Abs(release-10) > 1e-9 {
		t.Fatalf("release share %v", release)
	}
	release = SpreadBuyRelease(100, 1, 0.1, 12)
	if math.Abs(release-12) > 1e-9 {
		t.Fatalf("release spent %v", release)
	}
}
