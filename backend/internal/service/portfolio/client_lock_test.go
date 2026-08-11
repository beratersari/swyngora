package portfolio

import (
	"context"
	"sync"
	"testing"

)

// Concurrent market buys on two symbols must both debit cash (no last-write-wins).
func TestPortfolio_ConcurrentMultiSymbolBuysDebitCashOnceEach(t *testing.T) {
	px := &fakePx{prices: map[string]string{
		"binance|BTCUSDT": "100",
		"binance|ETHUSDT": "100",
	}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "race1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for _, sym := range []string{"BTCUSDT", "ETHUSDT"} {
		sym := sym
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.PlaceOrder(ctx, OrderInput{
				ClientID: "race1", Exchange: "binance", Symbol: sym, Side: "buy", Quantity: 1,
			})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("place: %v", err)
		}
	}

	view, err := svc.View(ctx, "race1")
	if err != nil {
		t.Fatal(err)
	}
	// 10000 - 100 - 100
	if view.CashBalance < 9799.999 || view.CashBalance > 9800.001 {
		t.Fatalf("cash=%v want 9800 (lost-update would leave ~9900)", view.CashBalance)
	}
	if len(view.Positions) != 2 {
		t.Fatalf("positions=%d want 2: %+v", len(view.Positions), view.Positions)
	}
}
