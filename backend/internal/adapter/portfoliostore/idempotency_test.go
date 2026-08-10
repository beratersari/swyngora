package portfoliostore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_IdempotencyClaimAndExpire(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "idemp.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	p, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := &domain.IdempotencyRecord{
		ClientID: p.BookID(), Key: "k1", RequestHash: "abc", Kind: domain.IdempotencyKindPending,
		ResultJSON: `{"orderId":"po-idemp"}`, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	ctx = domain.ContextWithIdempotency(ctx, rec)
	if _, err := s.CreatePendingOrder(ctx, domain.PendingOrder{
		ID: "po-idemp", ClientID: p.BookID(), Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, RemainingQuantity: 1,
		TriggerPrice: 90, ReservedCash: 90, Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetIdempotency(ctx, p.BookID(), "k1")
	if err != nil || got.RequestHash != "abc" || got.Kind != domain.IdempotencyKindPending {
		t.Fatalf("%+v %v", got, err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	hitErr := s.txInsertIdempotency(domain.ContextWithIdempotency(context.Background(), rec), tx)
	_ = tx.Rollback()
	if hitErr == nil || !errors.Is(hitErr, domain.ErrIdempotencyHit) {
		t.Fatalf("want hit, got %v", hitErr)
	}

	expired := &domain.IdempotencyRecord{
		ClientID: p.BookID(), Key: "old", RequestHash: "xyz", Kind: domain.IdempotencyKindTrade,
		ResultJSON: `{"tradeId":"t1"}`, CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	}
	tx, err = s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.txInsertIdempotency(domain.ContextWithIdempotency(ctx, expired), tx); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetIdempotency(ctx, p.BookID(), "old"); err != domain.ErrNotFound {
		t.Fatalf("expired should be not found, got %v", err)
	}
}
