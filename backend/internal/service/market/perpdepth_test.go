package market

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/perpdepthstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestGetPerpDepth_BidHeavyOnePercent(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetPerpDepth(context.Background(), "binance", "BTCUSDT", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Combined != nil || len(got.Venues) != 1 {
		t.Fatalf("%+v", got)
	}
	h := got.Venues[0].Headline
	if h.Below.Notional < 39_000_000 || h.Above.Notional < 14_000_000 || h.Lean != domain.PerpDepthLeanBid {
		t.Fatalf("headline %+v", h)
	}
}

func TestGetPerpDepth_Combined(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetPerpDepth(context.Background(), "all", "BTCUSDT", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Combined == nil || len(got.Venues) != 2 {
		t.Fatalf("%+v", got)
	}
	if got.Combined.Headline.Below.Notional < 70_000_000 {
		t.Fatalf("combined %+v", got.Combined.Headline)
	}
}

func TestGetPerpDepth_MissingSymbol(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetPerpDepth(context.Background(), "all", "", 1); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestGetPerpDepthOverview_TotalsAndLeaders(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{})
	got, err := svc.GetPerpDepthOverview(context.Background(), "all", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if got.Known < 2 || got.Bid <= 0 || got.Ask <= 0 {
		t.Fatalf("%+v", got)
	}
	if len(got.Coins) == 0 || got.Coins[0].SharePct <= 0 {
		t.Fatalf("coins %+v", got.Coins)
	}
	if got.Summary == "" || !strings.Contains(strings.ToLower(got.Summary), "bid") {
		t.Fatalf("summary %s", got.Summary)
	}
	if len(got.Bands) < 2 {
		t.Fatalf("bands %+v", got.Bands)
	}
	var near domain.PerpDepthOverviewBand
	for _, b := range got.Bands {
		if b.RangePct == 0.1 {
			near = b
		}
	}
	if near.Total <= 0 || near.OfHeadlinePct <= 0 || near.OfHeadlinePct >= 100 {
		t.Fatalf("near band %+v", near)
	}
}

func TestGetPerpDepthOverview_WindowsFromHistory(t *testing.T) {
	store := &perpdepthstore.Memory{}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, &fakeSupply{}).WithPerpDepthHistory(store)
	past := time.Now().UTC().Add(-time.Hour).Truncate(time.Minute)
	_, _ = store.InsertSamples(context.Background(), []domain.PerpDepthSample{
		{Time: past, Exchange: "binance", RangePct: 1, Symbol: "BTCUSDT", Base: "BTC", Bid: 10, Ask: 90},
		{Time: past, Exchange: "binance", RangePct: 1, Symbol: "ETHUSDT", Base: "ETH", Bid: 10, Ask: 90},
	})
	got, err := svc.GetPerpDepthOverview(context.Background(), "binance", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	var hour domain.PerpDepthShareWindow
	for _, w := range got.Windows {
		if w.Window == domain.ParticipationWindow1h {
			hour = w
		}
	}
	if !hour.Complete || hour.Kind == "" {
		t.Fatalf("windows %+v", got.Windows)
	}
}

func TestGetPerpDepthOverview_BadExchange(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	if _, err := svc.GetPerpDepthOverview(context.Background(), "okx", 1, 10); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}
