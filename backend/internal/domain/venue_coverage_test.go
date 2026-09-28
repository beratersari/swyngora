package domain

import (
	"strings"
	"testing"
	"time"
)

func TestCoverVenues_MissingAndStaleBlockTheRead(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	got := CoverVenues(FuturesVenues(), []VenueSample{
		{Exchange: ExchangeBinance, OK: true, At: now.Add(-20 * time.Second)},
		{Exchange: ExchangeBybit, OK: true, At: now.Add(-6 * time.Minute)},
	}, now, VenueTapeMaxAge)
	if got.Complete {
		t.Fatal("expected incomplete coverage")
	}
	if len(got.Missing) != 1 || got.Missing[0] != "okx" {
		t.Fatalf("missing = %v", got.Missing)
	}
	if len(got.Stale) != 1 || got.Stale[0] != "bybit" {
		t.Fatalf("stale = %v", got.Stale)
	}
	explain := got.Explain()
	if !strings.Contains(explain, "No market read") || !strings.Contains(explain, "okx") || !strings.Contains(explain, "bybit") {
		t.Fatalf("explain = %q", explain)
	}
}

func TestCoverVenues_QuietAnswerIsFresh(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	got := CoverVenues([]Exchange{ExchangeBinance}, []VenueSample{
		{Exchange: ExchangeBinance, OK: true},
	}, now, VenueTapeMaxAge)
	if !got.Complete || got.Venues[0].Status != VenueReadFresh {
		t.Fatalf("quiet answer should be fresh: %+v", got)
	}
	if got.Explain() != "" {
		t.Fatalf("explain = %q", got.Explain())
	}
}

func TestCoverageFromLiquidationFeed_DownWithOldClockIsStale(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	feed := LiquidationFeed{
		Venues: []LiquidationVenueHealth{
			{Exchange: "binance", Live: true, LastSeenAt: now.Add(-5 * time.Second)},
			{Exchange: "bybit", Live: false, LastSeenAt: now.Add(-8 * time.Minute), LastEventAt: now.Add(-9 * time.Minute)},
			{Exchange: "okx", Live: false},
		},
	}
	got := CoverageFromLiquidationFeed(feed, "all", now)
	if got.Complete {
		t.Fatal("expected incomplete")
	}
	if len(got.Stale) != 1 || got.Stale[0] != "bybit" {
		t.Fatalf("stale = %v", got.Stale)
	}
	if len(got.Missing) != 1 || got.Missing[0] != "okx" {
		t.Fatalf("missing = %v", got.Missing)
	}
}
