package portfolio

import (
	"context"
	"math"
	"sync"
	"testing"
)

// Concurrent partial fills must not desync counters or exceed order quantity.
func TestPending_ConcurrentPartialFillsDoNotExceedOrderQty(t *testing.T) {
	ctx := context.Background()
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	if _, err := svc.Create(ctx, CreateInput{ClientID: "race-fill", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	o, err := svc.PlacePendingOrder(ctx, PendingOrderInput{
		ClientID: "race-fill", Symbol: "BTCUSDT", Type: "limit_buy",
		Quantity: 2, TriggerPrice: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	snap := *o
	const workers = 2
	const maxFill = 0.5
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			_, _, _ = svc.TryFillPendingOrder(ctx, snap, 100, maxFill)
		}()
	}
	wg.Wait()

	final, err := svc.store.GetPendingOrder(ctx, "race-fill", o.ID)
	if err != nil {
		t.Fatal(err)
	}
	trades, _, err := svc.ListTrades(ctx, "race-fill", 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	var filledSum float64
	for _, tr := range trades {
		filledSum += tr.Quantity
	}
	orderQty := o.Quantity
	if filledSum > orderQty+1e-9 {
		t.Fatalf("trade qty sum %v exceeds order qty %v", filledSum, orderQty)
	}
	if math.Abs(final.FilledQuantity+final.RemainingQuantity-orderQty) > 1e-6 {
		t.Fatalf("filled(%v)+remaining(%v) != qty(%v)", final.FilledQuantity, final.RemainingQuantity, orderQty)
	}
	if math.Abs(final.FilledQuantity-filledSum) > 1e-6 {
		t.Fatalf("order.filled=%v != sum(trades)=%v", final.FilledQuantity, filledSum)
	}
	if math.Abs(filledSum-1.0) > 1e-6 {
		t.Fatalf("expected both partials (1.0 total), got sum=%v filled=%v remaining=%v",
			filledSum, final.FilledQuantity, final.RemainingQuantity)
	}
}
