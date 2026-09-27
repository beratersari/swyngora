package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestScanSuspiciousWalls_ReadsTracker(t *testing.T) {
	s := &Service{walls: domain.NewWallMemory()}
	left := time.Now().UTC().Add(-12 * time.Second)
	s.walls.Observe(left.Add(-5*time.Second), "bybit", "ETHUSDT", []domain.OrderBookWall{{
		Side: "ask", Price: "2000", Quantity: "1", Notional: "800000", Share: 0.5,
	}})
	s.walls.Observe(left, "bybit", "ETHUSDT", nil)
	s.watchMu.Lock()
	s.wallScanPairs = map[domain.Exchange][]string{domain.ExchangeBybit: {"ETHUSDT"}}
	s.watchMu.Unlock()

	got, err := s.ScanSuspiciousWalls(context.Background(), "bybit", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Quote != "USDT" || got.Universe != 1 || len(got.Hits) != 1 || got.Hits[0].Kind != domain.WallScanKindFlicker {
		t.Fatalf("%+v", got)
	}
	if got.Hits[0].Symbol != "ETHUSDT" || got.Hits[0].Present {
		t.Fatalf("hit %+v", got.Hits[0])
	}
	if _, err := s.ScanSuspiciousWalls(context.Background(), "okx", "USDT", 10); err == nil {
		t.Fatal("expected exchange error")
	}
}
