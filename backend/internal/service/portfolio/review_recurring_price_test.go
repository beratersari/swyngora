package portfolio

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// flipPx returns a different last price on each GetTicker24h call.
type flipPx struct {
	n      atomic.Int64
	first  string
	second string
}

func (f *flipPx) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	p := f.first
	if f.n.Add(1) > 1 {
		p = f.second
	}
	return &domain.Ticker24h{Symbol: symbol, LastPrice: p}, nil
}

// executeRecurringCashBuy sizes qty from one lastPrice, then PlaceOrder fetches
// lastPrice again and spends qty * newPrice — so a move overspends the plan.
func TestReview_RecurringBuyOverspendsWhenPriceMovesBetweenSizingAndFill(t *testing.T) {
	px := &flipPx{first: "100", second: "200"}
	svc := newSvc(t, nil)
	svc.market = px
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "rb-gap", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Minute)
	const planAmount = 200.0
	plan, err := svc.CreateRecurringBuyPlan(ctx, RecurringBuyCreateInput{
		ClientID: "rb-gap", Symbol: "BTCUSDT", Amount: planAmount, Frequency: "daily", StartAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := svc.ProcessDueRecurringBuys(ctx, time.Now().UTC())
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	view, err := svc.View(ctx, "rb-gap")
	if err != nil {
		t.Fatal(err)
	}
	spent := 10000 - view.CashBalance
	if spent > planAmount+1 {
		t.Fatalf("overspent: spent=%v plan=%v cash=%v", spent, planAmount, view.CashBalance)
	}
	if spent <= 0 {
		t.Fatalf("expected a cash debit, spent=%v cash=%v", spent, view.CashBalance)
	}
	runs, _ := svc.ListRecurringBuyRuns(ctx, "rb-gap", plan.ID, 10, 0)
	if len(runs) == 0 || runs[0].Status != domain.RecurringBuyRunSucceeded {
		t.Fatalf("run=%+v", runs)
	}
}
