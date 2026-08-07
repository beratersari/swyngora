package portfoliostore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_CashMovements(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "cash.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	p, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	p.CashBalance = 1500
	p.NetDeposits = 500
	p.UpdatedAt = now
	m, err := s.ApplyCashMovement(ctx, p, domain.CashMovement{
		ID: "m1", Kind: domain.CashMovementDeposit, Amount: 500,
		CashAfter: 1500, NetDepositsAfter: 500, Note: "top up", CreatedAt: now,
	})
	if err != nil || m.Amount != 500 {
		t.Fatalf("%+v %v", m, err)
	}
	got, err := s.GetPortfolio(ctx, "c")
	if err != nil || got.CashBalance != 1500 || got.NetDeposits != 500 {
		t.Fatalf("%+v %v", got, err)
	}
	list, err := s.ListCashMovements(ctx, "c", 10, 0)
	if err != nil || len(list) != 1 || list[0].Note != "top up" {
		t.Fatalf("%+v %v", list, err)
	}
	n, err := s.CountCashMovements(ctx, "c")
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
}
