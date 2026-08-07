package portfolio

import (
	"context"
	"math"
	"testing"
)

func TestCash_DepositWithdrawHistoryAndPnL(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cash1", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	m, view, err := svc.Deposit(ctx, CashMoveInput{ClientID: "cash1", Amount: 2500, Note: "salary"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Kind != "deposit" || m.Amount != 2500 || math.Abs(view.CashBalance-12500) > 1e-9 {
		t.Fatalf("dep %+v view cash=%v", m, view.CashBalance)
	}
	if math.Abs(view.NetDeposits-2500) > 1e-9 || math.Abs(view.TotalPnL) > 1e-9 {
		t.Fatalf("deposit must not create P&L: net=%v pnl=%v", view.NetDeposits, view.TotalPnL)
	}
	if math.Abs(view.ContributedCapital-12500) > 1e-9 {
		t.Fatalf("contributed=%v", view.ContributedCapital)
	}

	m, view, err = svc.Withdraw(ctx, CashMoveInput{ClientID: "cash1", Amount: 500, Note: "atm"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Kind != "withdrawal" || math.Abs(view.CashBalance-12000) > 1e-9 {
		t.Fatalf("wd %+v cash=%v", m, view.CashBalance)
	}
	if math.Abs(view.NetDeposits-2000) > 1e-9 || math.Abs(view.TotalPnL) > 1e-9 {
		t.Fatalf("withdraw pnl=%v net=%v", view.TotalPnL, view.NetDeposits)
	}

	list, total, err := svc.ListCashMovements(ctx, "cash1", 10, 0)
	if err != nil || total != 3 || len(list) != 3 {
		t.Fatalf("hist total=%d n=%d err=%v", total, len(list), err)
	}
	if list[0].Kind != "withdrawal" || list[2].Note != "Opening balance" {
		t.Fatalf("order %+v", list)
	}

	if _, _, err := svc.Withdraw(ctx, CashMoveInput{ClientID: "cash1", Amount: 999999}); err == nil {
		t.Fatal("want insufficient cash")
	}
}
