package domain

import (
	"errors"
	"math"
	"testing"
	"time"
)

func testLot(id string, qty, px float64, opened time.Time) TaxLot {
	return TaxLot{ID: id, Quantity: qty, OriginalQuantity: qty, Price: px, OpenedAt: opened}
}

func TestConsumeLots_FIFOPartialAndRemaining(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	open := []TaxLot{
		testLot("a", 1, 100, t0),
		testLot("b", 1, 200, t1),
	}
	fills, updated, realized, err := ConsumeLots(open, 1.5, 180, LotMethodFIFO, t1.Add(time.Hour), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(fills) != 2 {
		t.Fatalf("fills=%d", len(fills))
	}
	if fills[0].LotID != "a" || math.Abs(fills[0].Quantity-1) > 1e-12 {
		t.Fatalf("fill0=%+v", fills[0])
	}
	if fills[1].LotID != "b" || math.Abs(fills[1].Quantity-0.5) > 1e-12 {
		t.Fatalf("fill1=%+v", fills[1])
	}
	// (180-100)*1 + (180-200)*0.5 = 80 - 10 = 70
	if math.Abs(realized-70) > 1e-9 {
		t.Fatalf("realized=%v", realized)
	}
	var remB *TaxLot
	for i := range updated {
		if updated[i].ID == "a" {
			if updated[i].Open() || updated[i].Quantity != 0 || updated[i].ClosedAt == nil {
				t.Fatalf("lot a should be closed: %+v", updated[i])
			}
		}
		if updated[i].ID == "b" {
			remB = &updated[i]
		}
	}
	if remB == nil || math.Abs(remB.Quantity-0.5) > 1e-12 || remB.ClosedAt != nil {
		t.Fatalf("remaining b=%+v", remB)
	}
	if math.Abs(AvgCostFromLots(updated)-200) > 1e-9 {
		t.Fatalf("avg=%v", AvgCostFromLots(updated))
	}
}

func TestConsumeLots_LIFODifferentPnL(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	open := []TaxLot{
		testLot("a", 1, 100, t0),
		testLot("b", 1, 200, t0.Add(time.Hour)),
	}
	_, updated, realized, err := ConsumeLots(open, 1, 180, LotMethodLIFO, t0.Add(2*time.Hour), 0)
	if err != nil {
		t.Fatal(err)
	}
	// newest first: sell the 200 lot → (180-200)*1 = -20
	if math.Abs(realized-(-20)) > 1e-9 {
		t.Fatalf("realized=%v", realized)
	}
	merged := MergeLotUpdates(open, updated)
	if math.Abs(AvgCostFromLots(merged)-100) > 1e-9 {
		t.Fatalf("remaining avg=%v lots=%+v", AvgCostFromLots(merged), merged)
	}
}

func TestConsumeLots_Insufficient(t *testing.T) {
	t0 := time.Now().UTC()
	_, _, _, err := ConsumeLots([]TaxLot{testLot("a", 0.5, 10, t0)}, 1, 12, LotMethodFIFO, t0, 0)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestConsumeLots_RealizedAfterFee(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	open := []TaxLot{testLot("a", 1, 100, t0)}
	_, _, realized, err := ConsumeLots(open, 1, 110, LotMethodFIFO, t0, 0.01) // 1% fee
	if err != nil {
		t.Fatal(err)
	}
	// net 110*0.99=108.9 → pnl 8.9
	if math.Abs(realized-8.9) > 1e-9 {
		t.Fatalf("realized=%v", realized)
	}
}

func TestNormalizeLotMethod(t *testing.T) {
	m, err := NormalizeLotMethod("")
	if err != nil || m != LotMethodFIFO {
		t.Fatalf("%v %v", m, err)
	}
	m, err = NormalizeLotMethod("LIFO")
	if err != nil || m != LotMethodLIFO {
		t.Fatalf("%v %v", m, err)
	}
	if _, err := NormalizeLotMethod("hifo"); err == nil {
		t.Fatal("expected error")
	}
}
