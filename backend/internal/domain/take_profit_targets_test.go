package domain

import (
	"math"
	"testing"
)

func TestNormalizeTakeProfitTargetsScaleOut(t *testing.T) {
	got, err := NormalizeTakeProfitTargets([]TakeProfitTarget{
		{Percent: 50, OffsetPercent: 15},
		{Percent: 25, OffsetPercent: 5},
		{Percent: 25, OffsetPercent: 10},
	}, 0, 1, 100, 90, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("targets: %+v", got)
	}
	wantPx := []float64{105, 110, 115}
	wantQty := []float64{0.25, 0.25, 0.5}
	for i := range got {
		if math.Abs(got[i].Price-wantPx[i]) > 1e-9 || math.Abs(got[i].Quantity-wantQty[i]) > 1e-9 {
			t.Fatalf("target %d: %+v", i, got[i])
		}
	}
}

func TestNormalizeTakeProfitTargetsLegacyPrice(t *testing.T) {
	got, err := NormalizeTakeProfitTargets(nil, 120, 2, 100, 90, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Percent != 100 || got[0].Price != 120 || got[0].Quantity != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestNormalizeTakeProfitTargetsRejects(t *testing.T) {
	if _, err := NormalizeTakeProfitTargets([]TakeProfitTarget{
		{Percent: 40, Price: 110},
		{Percent: 40, Price: 120},
	}, 0, 1, 100, 90, false); err == nil {
		t.Fatal("expected shares to sum to 100")
	}
	if _, err := NormalizeTakeProfitTargets([]TakeProfitTarget{
		{Percent: 100, Price: 80},
	}, 0, 1, 100, 90, false); err == nil {
		t.Fatal("expected price above the stop")
	}
	if _, err := NormalizeTakeProfitTargets([]TakeProfitTarget{
		{Percent: 100, OffsetPercent: 5},
	}, 0, 1, 0, 90, false); err == nil {
		t.Fatal("expected a reference for the gain")
	}
	if _, err := NormalizeTakeProfitTargets([]TakeProfitTarget{
		{Percent: 100, Price: 100, OffsetPercent: 5},
	}, 0, 1, 100, 90, false); err == nil {
		t.Fatal("expected one of price or gain")
	}
	if _, err := NormalizeTakeProfitTargets([]TakeProfitTarget{
		{Percent: 100, Price: 95},
	}, 0, 1, 100, 90, true); err == nil {
		t.Fatal("expected the target above the entry")
	}
}

func TestSplitTargetQuantitiesRemainder(t *testing.T) {
	got, err := SplitTargetQuantities(1, []float64{25, 25, 50})
	if err != nil {
		t.Fatal(err)
	}
	var sum float64
	for _, q := range got {
		sum += q
	}
	if math.Abs(sum-1) > 1e-12 {
		t.Fatalf("sum %v parts %v", sum, got)
	}
	partial, err := SplitTargetQuantities(1, []float64{25, 25})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(partial[0]+partial[1]-0.5) > 1e-9 {
		t.Fatalf("canceled share must stay out: %v", partial)
	}
}

func TestScaleRemainings(t *testing.T) {
	got := ScaleRemainings([]float64{0.25, 0.5}, 0.45)
	if math.Abs(got[0]+got[1]-0.45) > 1e-9 {
		t.Fatalf("%v", got)
	}
	same := ScaleRemainings([]float64{0.25, 0.5}, 0.9)
	if same[0] != 0.25 || same[1] != 0.5 {
		t.Fatalf("do not grow slices: %v", same)
	}
}
