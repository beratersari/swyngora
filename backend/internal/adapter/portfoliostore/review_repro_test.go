package portfoliostore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Review claim: fill computed from remaining=1, then remaining is amended up,
// then ExecutePendingFill applies caller reserved=0 against the new remaining.
// Correct: either conflict, or leftover remaining keeps a matching reserve.
func TestRepro_FillAfterAmendDoesNotDropReserve(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "fill-amend.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePendingOrder(ctx, domain.PendingOrder{
		ID: "po-race", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, RemainingQuantity: 1,
		TriggerPrice: 100, ReservedCash: 100, Status: domain.PendingStatusOpen, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Amend-up after the filler sized a full fill of the original remaining.
	if _, err := s.AmendPendingOrder(ctx, "c", "po-race", domain.PendingOrderAmend{
		RemainingQuantity: 2, TriggerPrice: 100, Quantity: 2,
		ReservedCash: 200, ExpectedRemaining: 1, ExpectedTrigger: 100, At: now,
	}); err != nil {
		t.Fatal(err)
	}

	// Fill payload as computed from the pre-amend snapshot (qty=1, reserved leftover=0).
	fillOrder := domain.PendingOrder{
		ID: "po-race", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Type: domain.PendingLimitBuy, Side: domain.TradeSideBuy, Quantity: 1, RemainingQuantity: 0,
		ReservedCash: 0, ReservedQuantity: 0, Status: domain.PendingStatusFilled,
	}
	err = s.ExecutePendingFill(ctx, &fillOrder, &domain.Portfolio{
		ClientID: "c", CashBalance: 900,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100,
	}, domain.Trade{
		ID: "t-race", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, PendingOrderID: "po-race", CreatedAt: now,
	}, now, nil)
	if err != nil && err != domain.ErrNotFound && err != domain.ErrConflict {
		t.Fatal(err)
	}

	got, gerr := s.GetPendingOrder(ctx, "c", "po-race")
	if gerr != nil {
		t.Fatal(gerr)
	}
	if got.Status == domain.PendingStatusOpen && got.RemainingQuantity > domain.PositionEpsilon && got.ReservedCash <= domain.PositionEpsilon {
		t.Fatalf("fill after amend left open remaining=%v with reserved_cash=%v", got.RemainingQuantity, got.ReservedCash)
	}
}
