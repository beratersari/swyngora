package portfoliostore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_PortfolioRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	if _, err := s.CreatePortfolio(ctx, domain.Portfolio{
		ClientID: "c", Currency: "USDT", StartingBalance: 1000, CashBalance: 1000, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.ExecuteTrade(ctx, &domain.Portfolio{
		ClientID: "c", CashBalance: 900, RealizedPnLTotal: 0, UpdatedAt: now,
	}, &domain.Position{
		ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Quantity: 1, AvgCost: 100, UpdatedAt: now,
	}, domain.Trade{
		ID: "t1", ClientID: "c", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		Side: domain.TradeSideBuy, Quantity: 1, Price: 100, Notional: 100, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	got, err := s2.GetPortfolio(ctx, "c")
	if err != nil || got.CashBalance != 900 {
		t.Fatalf("%+v %v", got, err)
	}
	pos, err := s2.ListPositions(ctx, "c")
	if err != nil || len(pos) != 1 || pos[0].Quantity != 1 {
		t.Fatalf("%+v %v", pos, err)
	}
	tr, err := s2.ListTrades(ctx, "c", 10, 0)
	if err != nil || len(tr) != 1 {
		t.Fatalf("%+v %v", tr, err)
	}
}