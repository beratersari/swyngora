package domain

import (
	"errors"
	"math"
	"testing"
	"time"
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

func TestTrailingStopAdvanceAndTrigger(t *testing.T) {
	// 5% trail, peak 100 → stop 95
	if p := TrailStopPrice(100, 0.05, TrailTypePercent); math.Abs(p-95) > 1e-9 {
		t.Fatalf("percent stop=%v", p)
	}
	if p := TrailStopPrice(100, 10, TrailTypeOffset); math.Abs(p-90) > 1e-9 {
		t.Fatalf("offset stop=%v", p)
	}
	// Ratchet up only
	peak, stop, moved := AdvanceTrailingStop(100, 120, 0.05, TrailTypePercent)
	if !moved || math.Abs(peak-120) > 1e-9 || math.Abs(stop-114) > 1e-9 {
		t.Fatalf("up peak=%v stop=%v moved=%v", peak, stop, moved)
	}
	// Pullback does not lower peak/stop
	peak2, stop2, moved2 := AdvanceTrailingStop(120, 110, 0.05, TrailTypePercent)
	if moved2 || peak2 != 120 || math.Abs(stop2-114) > 1e-9 {
		t.Fatalf("pullback peak=%v stop=%v moved=%v", peak2, stop2, moved2)
	}
	// Gap through stop still triggers
	if !PendingOrderTriggered(PendingTrailingStop, 114, 100) {
		t.Fatal("gap through stop should trigger")
	}
	if PendingOrderTriggered(PendingTrailingStop, 114, 115) {
		t.Fatal("above stop should not trigger")
	}
}

func TestValidateBracketPrices(t *testing.T) {
	if err := ValidateBracketPrices(100, 120, 90); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBracketPrices(100, 90, 80); err == nil {
		t.Fatal("tp must be above entry")
	}
	if err := ValidateBracketPrices(100, 120, 110); err == nil {
		t.Fatal("sl must be below entry")
	}
}

func TestCanAmendPendingOrder(t *testing.T) {
	ok := PendingOrder{
		Type: PendingLimitBuy, Status: PendingStatusOpen, TimeInForce: TimeInForceGTC,
	}
	if err := CanAmendPendingOrder(ok); err != nil {
		t.Fatalf("limit_buy gtc: %v", err)
	}
	sl := ok
	sl.Type = PendingStopLoss
	if err := CanAmendPendingOrder(sl); err != nil {
		t.Fatalf("stop_loss: %v", err)
	}
	filled := ok
	filled.Status = PendingStatusFilled
	if err := CanAmendPendingOrder(filled); !errors.Is(err, ErrConflict) {
		t.Fatalf("filled want conflict: %v", err)
	}
	trail := ok
	trail.Type = PendingTrailingStop
	if err := CanAmendPendingOrder(trail); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("trailing want invalid: %v", err)
	}
	ioc := ok
	ioc.TimeInForce = TimeInForceIOC
	if err := CanAmendPendingOrder(ioc); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("ioc want invalid: %v", err)
	}
	oco := ok
	oco.OCOGroupID = "g1"
	if err := CanAmendPendingOrder(oco); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("oco want invalid: %v", err)
	}
	br := ok
	br.BracketID = "b1"
	br.BracketRole = BracketRoleEntry
	if err := CanAmendPendingOrder(br); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("bracket want invalid: %v", err)
	}
}

func TestAmendOriginalQuantityAndMaxRemaining(t *testing.T) {
	if q := AmendOriginalQuantity(1, 0.5); math.Abs(q-1.5) > 1e-12 {
		t.Fatalf("qty=%v", q)
	}
	if err := ValidateAmendRemaining(0); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("zero remaining: %v", err)
	}
	if err := ValidateAmendTriggerPrice(0); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("zero trigger: %v", err)
	}
	maxBuy := MaxAmendRemaining(TradeSideBuy, 100, 250, 0)
	if math.Abs(maxBuy-2.5) > 1e-12 {
		t.Fatalf("max buy=%v", maxBuy)
	}
	maxSell := MaxAmendRemaining(TradeSideSell, 0, 0, 1.25)
	if math.Abs(maxSell-1.25) > 1e-12 {
		t.Fatalf("max sell=%v", maxSell)
	}
}

func TestValidateOCOPrices(t *testing.T) {
	if err := ValidateOCOPrices(120, 90); err != nil {
		t.Fatal(err)
	}
	if err := ValidateOCOPrices(90, 120); err == nil {
		t.Fatal("want tp > sl")
	}
	if err := ValidateOCOPrices(0, 90); err == nil {
		t.Fatal("want positive")
	}
}

func TestOCOWinnerForTick(t *testing.T) {
	tp := &PendingOrder{Type: PendingLimitSell, TriggerPrice: 120, Status: PendingStatusOpen}
	sl := &PendingOrder{Type: PendingStopLoss, TriggerPrice: 90, Status: PendingStatusOpen}
	if OCOWinnerForTick(tp, sl, 100) != nil {
		t.Fatal("neither triggered")
	}
	if w := OCOWinnerForTick(tp, sl, 125); w != tp {
		t.Fatal("tp only")
	}
	if w := OCOWinnerForTick(tp, sl, 80); w != sl {
		t.Fatal("sl only")
	}
	// Gap through both: stop wins (single fill)
	if w := OCOWinnerForTick(tp, sl, 50); w != sl {
		// 50 triggers SL (last<=90) and also... limit sell triggers last>=120? 50 does not trigger TP.
		t.Fatalf("sl at 50: got %v", w)
	}
	// Both: use price that hits both - for long OCO, TP is above and SL below entry, so one tick
	// cannot hit both unless gap. Simulate both flags by using price above TP and... can't hit SL.
	// Force: temporarily set TP trigger very low so both fire.
	tpBoth := &PendingOrder{Type: PendingLimitSell, TriggerPrice: 100, Status: PendingStatusOpen}
	slBoth := &PendingOrder{Type: PendingStopLoss, TriggerPrice: 110, Status: PendingStatusOpen}
	// last=105: limit sell 100 triggers (>=100), stop 110 triggers (<=110)
	if w := OCOWinnerForTick(tpBoth, slBoth, 105); w != slBoth {
		t.Fatalf("both triggered want stop, got %+v", w)
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

func TestTimeInForceAndExpiry(t *testing.T) {
	tif, err := NormalizeTimeInForce("")
	if err != nil || tif != TimeInForceGTC {
		t.Fatalf("%v %v", tif, err)
	}
	if _, err := NormalizeTimeInForce("day"); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
	exp := time.Now().UTC().Add(-time.Minute)
	o := PendingOrder{ExpiresAt: &exp}
	if !PendingOrderExpired(o, time.Now().UTC()) {
		t.Fatal("expected expired")
	}
	future := time.Now().UTC().Add(time.Hour)
	o.ExpiresAt = &future
	if PendingOrderExpired(o, time.Now().UTC()) {
		t.Fatal("not yet expired")
	}
}

func TestAvailableAndReservations(t *testing.T) {
	if math.Abs(BuyReserveCash(2, 100, TradingCost{})-200) > 1e-9 {
		t.Fatal("reserve")
	}
	if math.Abs(AvailableCash(1000, 250)-750) > 1e-9 {
		t.Fatal("avail cash")
	}
	if math.Abs(AvailablePosition(5, 2)-3) > 1e-9 {
		t.Fatal("avail pos")
	}
	// Cheaper fill than limit: reserved cash can cover full remaining
	if math.Abs(MaxBuyFillQty(2, 200, 90, 0)-2) > 1e-9 {
		t.Fatal("max buy full")
	}
	// Cap by cash at higher fill price within reservation
	if math.Abs(MaxBuyFillQty(3, 100, 50, 0)-2) > 1e-9 {
		t.Fatalf("max buy partial got %v", MaxBuyFillQty(3, 100, 50, 0))
	}
	if math.Abs(ClampFillQty(10, 0, 3)-3) > 1e-9 {
		t.Fatal("clamp max")
	}
	if math.Abs(ClampFillQty(10, 4, 0)-4) > 1e-9 {
		t.Fatal("clamp requested")
	}
	if math.Abs(AfterBuyFillReservation(1, 100, TradingCost{})-100) > 1e-9 {
		t.Fatal("after buy res")
	}
	if AfterSellFillReservation(0) != 0 {
		t.Fatal("after sell res")
	}
}