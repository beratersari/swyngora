package portfolio

import (
	"context"
	"math"
	"sync"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestCash_ConcurrentDepositAndMarketBuy(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	if _, err := svc.Create(ctx, CreateInput{ClientID: "cash-race", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _, err := svc.Deposit(ctx, CashMoveInput{ClientID: "cash-race", Amount: 2500})
		errCh <- err
	}()
	go func() {
		defer wg.Done()
		_, _, err := svc.PlaceOrder(ctx, OrderInput{
			ClientID: "cash-race", Exchange: "binance", Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
		})
		errCh <- err
	}()
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("op: %v", err)
		}
	}
	view, err := svc.View(ctx, "cash-race")
	if err != nil {
		t.Fatal(err)
	}
	// 10000 + 2500 - 100 buy = 12400 (lost deposit would leave ~9900 or ~12500 without buy)
	if math.Abs(view.CashBalance-12400) > 1e-6 {
		t.Fatalf("cash=%v want 12400", view.CashBalance)
	}
}

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

func TestCash_TransferBetweenOwnBooks(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	main, err := svc.Create(ctx, CreateInput{ClientID: "xfer1", StartingBalance: 10000, Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	risky, err := svc.Create(ctx, CreateInput{ClientID: "xfer1", StartingBalance: 1000, Name: "Risky"})
	if err != nil {
		t.Fatal(err)
	}
	out, in, fromV, toV, err := svc.Transfer(ctx, TransferInput{
		ClientID: "xfer1", FromPortfolioID: main.ID, ToPortfolioID: risky.ID, Amount: 2500, Note: "seed risky",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Kind != domain.CashMovementTransferOut || in.Kind != domain.CashMovementTransferIn {
		t.Fatalf("kinds %s %s", out.Kind, in.Kind)
	}
	if out.CounterpartyPortfolioID != risky.ID || out.CounterpartyPortfolioName != "Risky" {
		t.Fatalf("out counterpart %+v", out)
	}
	if in.CounterpartyPortfolioID != main.ID || in.CounterpartyPortfolioName != "Main" {
		t.Fatalf("in counterpart %+v", in)
	}
	if out.PeerMovementID != in.ID || in.PeerMovementID != out.ID {
		t.Fatalf("peer %s %s", out.PeerMovementID, in.PeerMovementID)
	}
	if math.Abs(fromV.CashBalance-7500) > 1e-9 || math.Abs(toV.CashBalance-3500) > 1e-9 {
		t.Fatalf("cash from=%v to=%v", fromV.CashBalance, toV.CashBalance)
	}
	if math.Abs(fromV.TotalPnL) > 1e-9 || math.Abs(toV.TotalPnL) > 1e-9 {
		t.Fatalf("transfer must not change P&L from=%v to=%v", fromV.TotalPnL, toV.TotalPnL)
	}
	hist, total, err := svc.ListCashMovements(ctx, "xfer1", 10, 0, main.ID)
	if err != nil || total < 2 || hist[0].Kind != domain.CashMovementTransferOut {
		t.Fatalf("main hist %+v total=%d err=%v", hist, total, err)
	}
	hist2, _, err := svc.ListCashMovements(ctx, "xfer1", 10, 0, risky.ID)
	if err != nil || hist2[0].Kind != domain.CashMovementTransferIn {
		t.Fatalf("risky hist %+v err=%v", hist2, err)
	}
	if _, _, _, _, err := svc.Transfer(ctx, TransferInput{
		ClientID: "xfer1", FromPortfolioID: main.ID, ToPortfolioID: main.ID, Amount: 10,
	}); err == nil {
		t.Fatal("same book")
	}
}

func TestCash_TransferAllowsUSDTUSDAlias(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	usdt, err := svc.Create(ctx, CreateInput{ClientID: "xfer-usd", Name: "USDT", StartingBalance: 5000, Currency: "USDT"})
	if err != nil {
		t.Fatal(err)
	}
	usd, err := svc.Create(ctx, CreateInput{ClientID: "xfer-usd", Name: "USD", StartingBalance: 100, Currency: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	_, _, fromV, toV, err := svc.Transfer(ctx, TransferInput{
		ClientID: "xfer-usd", FromPortfolioID: usdt.ID, ToPortfolioID: usd.ID, Amount: 400,
	})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(fromV.CashBalance-4600) > 1e-9 || math.Abs(toV.CashBalance-500) > 1e-9 {
		t.Fatalf("usdt=%v usd=%v", fromV.CashBalance, toV.CashBalance)
	}
}

func TestCash_TransferOwnerOnlyAndNotOtherClient(t *testing.T) {
	svc := newSvc(t, nil)
	ctx := context.Background()
	alice, err := svc.Create(ctx, CreateInput{ClientID: "alice-x", StartingBalance: 5000})
	if err != nil {
		t.Fatal(err)
	}
	risky, err := svc.Create(ctx, CreateInput{ClientID: "alice-x", Name: "Risky", StartingBalance: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "bob-x", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Share(ctx, "alice-x", alice.ID, "bob-x", "trader"); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := svc.Transfer(ctx, TransferInput{
		ClientID: "bob-x", FromPortfolioID: alice.ID, ToPortfolioID: risky.ID, Amount: 10,
	}); err == nil {
		t.Fatal("trader must not transfer")
	}
	if _, _, _, _, err := svc.Transfer(ctx, TransferInput{
		ClientID: "alice-x", FromPortfolioID: alice.ID, ToPortfolioID: "bob-x", Amount: 10,
	}); err == nil {
		t.Fatal("must not transfer to another client's book")
	}
}
