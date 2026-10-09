package market

import (
	"context"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/perpdepthstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSamplePerpBookBands_StoresOnlyWhenEveryVenueIsFresh(t *testing.T) {
	store := &perpdepthstore.Memory{}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{}).WithFuturesDepth(map[domain.Exchange]domain.FuturesDepthPort{
		domain.ExchangeOKX: &fakeMarket{},
	}).WithPerpDepthHistory(store)
	stored, gaps, err := svc.SamplePerpBookBands(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stored < 1 || gaps != 0 {
		t.Fatalf("stored %d gaps %d", stored, gaps)
	}
	got, err := svc.GetPerpBookHistory(context.Background(), "all", "BTCUSDT", 3600, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Columns) != 1 || got.Columns[0].Mid <= 0 {
		t.Fatalf("%+v", got.Columns)
	}
	var near float64
	for _, b := range got.Columns[0].Bands {
		if b.RangePct == 0.1 {
			near = b.Bid
		}
	}
	if near <= 0 {
		t.Fatalf("bands %+v", got.Columns[0].Bands)
	}
}

func TestSamplePerpBookBands_NamesTheMissingVenue(t *testing.T) {
	store := &perpdepthstore.Memory{}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
		domain.ExchangeBybit:   &fakeMarket{},
	}, &fakeSupply{}).WithPerpDepthHistory(store)
	_, gaps, err := svc.SamplePerpBookBands(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gaps < 1 {
		t.Fatal("expected gaps")
	}
	got, err := svc.GetPerpBookHistory(context.Background(), "all", "BTCUSDT", 3600, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Columns) != 0 || len(got.Gaps) == 0 || got.Gaps[0].Missing[0] != "okx" {
		t.Fatalf("cols %d gaps %+v", len(got.Columns), got.Gaps)
	}
}

func TestGetPerpBookHistory_NoStore(t *testing.T) {
	svc := New(&fakeMarket{}, &fakeSupply{})
	got, err := svc.GetPerpBookHistory(context.Background(), "all", "BTCUSDT", 0, nil, nil)
	if err != nil || got.Symbol != "BTCUSDT" || len(got.Columns) != 0 {
		t.Fatalf("%v %+v", err, got)
	}
}

func TestListBucketTotals_KeepsTheNewestMinute(t *testing.T) {
	store := &perpdepthstore.Memory{}
	base := time.Now().UTC().Truncate(2 * time.Hour)
	_, _ = store.InsertSamples(context.Background(), []domain.PerpDepthSample{
		{Time: base.Add(10 * time.Minute), Exchange: "all", RangePct: 1, Symbol: "BTCUSDT", Bid: 1, Ask: 1},
		{Time: base.Add(50 * time.Minute), Exchange: "all", RangePct: 1, Symbol: "BTCUSDT", Bid: 4, Ask: 2},
	})
	got, err := store.ListBucketTotals(context.Background(), "all", 1, base, base.Add(2*time.Hour), 2*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Total != 6 {
		t.Fatalf("%+v", got)
	}
}
