package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestHeatmapWarmCandidate(t *testing.T) {
	if heatmapWarmCandidate(domain.SpotMarket{}) {
		t.Fatal("empty symbol")
	}
	if !heatmapWarmCandidate(domain.SpotMarket{Symbol: "BTCUSDT", Status: "TRADING"}) {
		t.Fatal("trading")
	}
	if heatmapWarmCandidate(domain.SpotMarket{Symbol: "XRPUSDT", Status: "BREAK"}) {
		t.Fatal("break")
	}
	if !heatmapWarmCandidate(domain.SpotMarket{Symbol: "ETH-USD", Status: ""}) {
		t.Fatal("empty status is ok")
	}
}

func TestRefreshHeatmapUniverseAndSample(t *testing.T) {
	fx := &fakeMarket{}
	svc := New(fx, &fakeSupply{})
	svc.refreshHeatmapUniverse(context.Background())
	got := svc.heatmapUniverse(domain.ExchangeBinance)
	if len(got) < 3 {
		t.Fatalf("universe %v", got)
	}
	if got[0] != "BTCUSDT" {
		t.Fatalf("highest quote volume first, got %v", got)
	}
	for _, sym := range got {
		if sym == "XRPUSDT" {
			t.Fatalf("BREAK pair should be excluded: %v", got)
		}
	}
	svc.sampleHeatSnapshot(context.Background(), domain.ExchangeBinance, "ETHUSDT")
	if !fx.lastBookQ.SnapshotOnly || fx.lastBookQ.Limit != heatmapWarmLimit {
		t.Fatalf("warm sample must be snapshot-only limit=%d q=%+v", heatmapWarmLimit, fx.lastBookQ)
	}
	view := svc.heat.View("binance", "ETHUSDT", time.Minute)
	if len(view.Columns) != 1 {
		t.Fatalf("expected a recorded column, got %d", len(view.Columns))
	}
}
