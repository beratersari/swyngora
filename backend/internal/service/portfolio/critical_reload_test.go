package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
)

// AdjustMargin used to commit cash then reload via GetMarginPosition(bookID),
// which treats the first arg as the actor. A second book made that lookup fail
// so a client retry double-debited.
func TestAdjustMarginOnSecondBookDoesNotErrorAfterDebit(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "adj-2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	desk, err := svc.Create(ctx, CreateInput{ClientID: "adj-2", Name: "Desk", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "adj-2", PortfolioID: desk.ID, Symbol: "BTCUSDT",
		Side: "long", Type: "market", Quantity: 1, Leverage: 5,
	})
	if err != nil || pos == nil {
		t.Fatalf("open: pos=%+v err=%v", pos, err)
	}
	before, err := svc.View(ctx, "adj-2", desk.ID)
	if err != nil {
		t.Fatal(err)
	}
	const add = 10.0
	got, err := svc.AdjustMargin(ctx, MarginAdjustInput{
		ClientID: "adj-2", PortfolioID: desk.ID, PositionID: pos.ID, Delta: add,
	})
	if err != nil {
		t.Fatalf("AdjustMargin: %v", err)
	}
	if got == nil {
		t.Fatal("nil position")
	}
	after, err := svc.View(ctx, "adj-2", desk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(after.CashBalance-(before.CashBalance-add)) > 1e-9 {
		t.Fatalf("cash after adjust=%g want %g", after.CashBalance, before.CashBalance-add)
	}
}

func TestAdjustMarginOnPrimaryBookAfterSecondExists(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	main, err := svc.Create(ctx, CreateInput{ClientID: "adj-main", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "adj-main", Name: "Desk", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "adj-main", PortfolioID: main.ID, Symbol: "BTCUSDT",
		Side: "long", Type: "market", Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AdjustMargin(ctx, MarginAdjustInput{
		ClientID: "adj-main", PortfolioID: main.ID, PositionID: pos.ID, Delta: 10,
	}); err != nil {
		t.Fatalf("AdjustMargin on primary after second book: %v", err)
	}
}

func TestRepayOnSecondBookDoesNotErrorAfterDebit(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "repay-2", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	desk, err := svc.Create(ctx, CreateInput{ClientID: "repay-2", Name: "Desk", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	pos, _, err := svc.PlaceMarginOrder(ctx, MarginOrderInput{
		ClientID: "repay-2", PortfolioID: desk.ID, Symbol: "BTCUSDT",
		Side: "long", Type: "market", Quantity: 1, Leverage: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := svc.View(ctx, "repay-2", desk.ID)
	if err != nil {
		t.Fatal(err)
	}
	const pay = 5.0
	_, tr, err := svc.RepayMarginDebt(ctx, MarginRepayInput{
		ClientID: "repay-2", PortfolioID: desk.ID, PositionID: pos.ID, Amount: pay,
	})
	if err != nil {
		t.Fatalf("RepayMarginDebt: %v trade=%+v", err, tr)
	}
	after, err := svc.View(ctx, "repay-2", desk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(after.CashBalance-(before.CashBalance-pay)) > 1e-9 {
		t.Fatalf("cash after repay=%g want %g", after.CashBalance, before.CashBalance-pay)
	}
}

func TestPlaceOrderRejectsClosedOwner(t *testing.T) {
	svc := newSvc(t, &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}})
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "closed-ord", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	acct := account.New(accountstore.NewMemory(), account.DataPurgeDeps{Paper: svc})
	svc.SetAccountChecker(acct)
	if _, err := acct.Close(ctx, "closed-ord"); err != nil {
		t.Fatal(err)
	}
	_, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "closed-ord", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err == nil {
		t.Fatal("expected closed account to reject PlaceOrder")
	}
	var closed *domain.ErrAccountClosed
	if !errors.As(err, &closed) {
		t.Fatalf("want ErrAccountClosed, got %v", err)
	}
}
