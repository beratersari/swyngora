package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestPortfolioShare_ViewerTraderOwner(t *testing.T) {
	px := &fakePx{prices: map[string]string{"binance|BTCUSDT": "100"}}
	svc := newSvc(t, px)
	ctx := context.Background()
	ownerBook, err := svc.Create(ctx, CreateInput{ClientID: "alice", StartingBalance: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, CreateInput{ClientID: "bob", StartingBalance: 1000}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Share(ctx, "alice", ownerBook.ID, "alice", "viewer"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("self share: %v", err)
	}
	sh, err := svc.Share(ctx, "alice", ownerBook.ID, "bob", "viewer")
	if err != nil || sh.Role != domain.PortfolioRoleViewer {
		t.Fatalf("%+v %v", sh, err)
	}
	if _, err := svc.Share(ctx, "alice", ownerBook.ID, "bob", "trader"); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("dup share: %v", err)
	}

	v, err := svc.View(ctx, "bob", ownerBook.ID)
	if err != nil || v.Role != domain.PortfolioRoleViewer || math.Abs(v.CashBalance-10000) > 1e-9 {
		t.Fatalf("viewer view %+v %v", v, err)
	}
	if _, _, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "bob", PortfolioID: ownerBook.ID, Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer trade: %v", err)
	}
	if _, _, err := svc.Deposit(ctx, CashMoveInput{ClientID: "bob", PortfolioID: ownerBook.ID, Amount: 10}); err == nil {
		t.Fatal("viewer must not deposit")
	}

	sh, err = svc.UpdateShareRole(ctx, "alice", ownerBook.ID, "bob", "trader")
	if err != nil || sh.Role != domain.PortfolioRoleTrader {
		t.Fatalf("promote %+v %v", sh, err)
	}
	tr, view, err := svc.PlaceOrder(ctx, OrderInput{
		ClientID: "bob", PortfolioID: ownerBook.ID, Symbol: "BTCUSDT", Side: "buy", Quantity: 1,
	})
	if err != nil || tr == nil || math.Abs(view.CashBalance-9900) > 1e-6 {
		t.Fatalf("trader buy %+v %+v %v", tr, view, err)
	}
	if view.Role != domain.PortfolioRoleTrader {
		t.Fatalf("role after trade %s", view.Role)
	}
	if _, _, err := svc.Withdraw(ctx, CashMoveInput{ClientID: "bob", PortfolioID: ownerBook.ID, Amount: 10}); err == nil {
		t.Fatal("trader must not withdraw")
	}
	if err := svc.Delete(ctx, "bob", ownerBook.ID); !errors.Is(err, domain.ErrForbidden) && !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("trader delete: %v", err)
	}

	shared, err := svc.ListSharedWithMe(ctx, "bob")
	if err != nil || len(shared) != 1 || shared[0].Role != domain.PortfolioRoleTrader {
		t.Fatalf("shared with me %+v %v", shared, err)
	}
	if err := svc.RevokeShare(ctx, "alice", ownerBook.ID, "bob"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.View(ctx, "bob", ownerBook.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("after revoke: %v", err)
	}
	vAlice, err := svc.View(ctx, "alice", ownerBook.ID)
	if err != nil || math.Abs(vAlice.CashBalance-9900) > 1e-6 {
		t.Fatalf("owner still has trade %+v %v", vAlice, err)
	}
}
