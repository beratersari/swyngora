package domain

import (
	"errors"
	"math"
	"testing"
)

func TestApplyBuy(t *testing.T) {
	cash, qty, avg, err := ApplyBuy(10000, 1, 100, 0, 0)
	if err != nil || math.Abs(cash-9900) > 1e-9 || qty != 1 || avg != 100 {
		t.Fatalf("cash=%v qty=%v avg=%v err=%v", cash, qty, avg, err)
	}
	cash, qty, avg, err = ApplyBuy(cash, 1, 200, qty, avg)
	if err != nil || math.Abs(avg-150) > 1e-9 || qty != 2 {
		t.Fatalf("avg=%v qty=%v err=%v", avg, qty, err)
	}
	_, _, _, err = ApplyBuy(10, 1, 100, 0, 0)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("want insufficient cash: %v", err)
	}
}

func TestApplySell(t *testing.T) {
	cash, qty, real, err := ApplySell(0, 1, 120, 2, 100)
	if err != nil || math.Abs(cash-120) > 1e-9 || qty != 1 || math.Abs(real-20) > 1e-9 {
		t.Fatalf("cash=%v qty=%v real=%v err=%v", cash, qty, real, err)
	}
	_, _, _, err = ApplySell(0, 5, 100, 2, 100)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("want insufficient qty: %v", err)
	}
}

func TestUnrealizedPnL(t *testing.T) {
	u := UnrealizedPnL(2, 100, 110)
	if math.Abs(u-20) > 1e-9 {
		t.Fatalf("%v", u)
	}
}

func TestPendingOrderTriggered(t *testing.T) {
	if !PendingOrderTriggered(PendingLimitBuy, 100, 99) {
		t.Fatal("limit buy should trigger at or below")
	}
	if PendingOrderTriggered(PendingLimitBuy, 100, 101) {
		t.Fatal("limit buy should not trigger above")
	}
	if !PendingOrderTriggered(PendingLimitSell, 100, 101) {
		t.Fatal("limit sell should trigger at or above")
	}
	if PendingOrderTriggered(PendingLimitSell, 100, 99) {
		t.Fatal("limit sell should not trigger below")
	}
	if !PendingOrderTriggered(PendingStopLoss, 90, 89) {
		t.Fatal("stop loss should trigger at or below")
	}
	if PendingOrderTriggered(PendingStopLoss, 90, 91) {
		t.Fatal("stop loss should not trigger above")
	}
}

func TestSideForPendingType(t *testing.T) {
	if SideForPendingType(PendingLimitBuy) != TradeSideBuy {
		t.Fatal("buy")
	}
	if SideForPendingType(PendingLimitSell) != TradeSideSell || SideForPendingType(PendingStopLoss) != TradeSideSell {
		t.Fatal("sell")
	}
}