package pricediffstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestOpportunityPathPersists(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "pd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	ctx := context.Background()
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	w := domain.PriceDiffWatch{
		ID: "w1", ClientID: "c1", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 10,
		Status: domain.PriceDiffWatchActive, CreatedAt: now, UpdatedAt: now,
		Exchanges: []domain.Exchange{domain.ExchangeBinance, domain.ExchangeBybit},
	}
	if _, err := s.CreateWatch(ctx, w); err != nil {
		t.Fatal(err)
	}
	o := domain.PriceDiffOpportunity{
		ID: "o1", WatchID: "w1", ClientID: "c1", Symbol: "BTCUSDT",
		BuyExchange: domain.ExchangeBinance, SellExchange: domain.ExchangeBybit,
		MinNetDiffPct: 0, Status: domain.PriceDiffOppOpen,
	}
	domain.InitPriceDiffOpportunity(&o, domain.PriceDiffOppSample{
		At: now, BuyPrice: 100, SellPrice: 101.5, GrossDiffPct: 1.5, NetDiffPct: 1.3, Profit: 130, Executable: true,
	})
	if _, err := s.CreateOpportunity(ctx, o); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TouchOpportunity(ctx, "o1", domain.PriceDiffOppSample{
		At: now.Add(30 * time.Second), BuyPrice: 100, SellPrice: 102, GrossDiffPct: 2, NetDiffPct: 1.8, Profit: 180, Executable: true,
	}); err != nil {
		t.Fatal(err)
	}
	last := domain.PriceDiffOppSample{
		At: now.Add(time.Minute), BuyPrice: 100, SellPrice: 100.2, GrossDiffPct: 0.2, NetDiffPct: 0.0, Profit: 1, Executable: true,
	}
	got, err := s.CloseOpportunity(ctx, "o1", now.Add(time.Minute), domain.PriceDiffCloseBelowMinProfit, &last)
	if err != nil {
		t.Fatal(err)
	}
	if got.CloseReason != domain.PriceDiffCloseBelowMinProfit || got.PeakNetDiffPct < 1.7 || got.FadeStartedAt == nil {
		t.Fatalf("%+v", got)
	}
	ticks, err := s.ListOpportunityTicks(ctx, "o1", 50)
	if err != nil || len(ticks) < 3 {
		t.Fatalf("ticks=%d err=%v", len(ticks), err)
	}
}
