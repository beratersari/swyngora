package market

import (
	"context"
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/deliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeOffVenue struct {
	quote *domain.OffVenueQuote
	bars  []domain.Candle
	err   error
}

func (f *fakeOffVenue) QuoteByBase(context.Context, string) (*domain.OffVenueQuote, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.quote, nil
}

func (f *fakeOffVenue) OHLCByBase(context.Context, string, int) ([]domain.Candle, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]domain.Candle(nil), f.bars...), nil
}

func pastDelistStore(symbol string) *deliststore.Memory {
	store := deliststore.NewMemory()
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: symbol, DelistTime: time.Now().UTC().Add(-48 * time.Hour)},
	})
	return store
}

func TestGetPostDelist_ListedPair(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, nil).
		WithDelistStore(deliststore.NewMemory())
	view, err := svc.GetPostDelist(context.Background(), "binance", "BTCUSDT", "1d", 30)
	if err != nil {
		t.Fatal(err)
	}
	if view.Available {
		t.Fatalf("listed pair should not be available: %+v", view)
	}
	if !strings.Contains(view.Note, "still listed") {
		t.Fatalf("note=%q", view.Note)
	}
}

func TestGetPostDelist_UpcomingDelist(t *testing.T) {
	store := deliststore.NewMemory()
	store.ReplaceAll(domain.ExchangeBinance, []domain.SpotDelistEntry{
		{Symbol: "SOONUSDT", DelistTime: time.Now().UTC().Add(48 * time.Hour)},
	})
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, nil).WithDelistStore(store)
	view, err := svc.GetPostDelist(context.Background(), "binance", "SOONUSDT", "1d", 10)
	if err != nil {
		t.Fatal(err)
	}
	if view.Available {
		t.Fatalf("upcoming delist should not be available: %+v", view)
	}
}

func TestGetPostDelist_CoinGeckoAfterHalt(t *testing.T) {
	pct := 4.25
	gecko := &fakeOffVenue{
		quote: &domain.OffVenueQuote{LastUSD: 0.1234, ChangePct: &pct, AsOf: time.Now().UTC()},
		bars: []domain.Candle{
			{OpenTime: time.Now().UTC().Add(-24 * time.Hour), Open: "0.11", High: "0.13", Low: "0.10", Close: "0.1234"},
		},
	}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeBybit:    &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeCoinbase: &fakeMarket{tickerErr: domain.ErrNotFound},
	}, nil).WithDelistStore(pastDelistStore("VICUSDT")).WithOffVenuePrice(gecko)

	view, err := svc.GetPostDelist(context.Background(), "binance", "VICUSDT", "1d", 30)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Available || view.Source != "coingecko" || view.LastPrice != "0.1234" {
		t.Fatalf("view=%+v", view)
	}
	if view.PriceChangePercent != "4.250" {
		t.Fatalf("change=%q", view.PriceChangePercent)
	}
	if len(view.Candles) != 1 {
		t.Fatalf("candles=%d", len(view.Candles))
	}
	if !strings.Contains(view.Note, "not this exchange") {
		t.Fatalf("note=%q", view.Note)
	}
}

func TestGetPostDelist_PrefersLiveOtherVenue(t *testing.T) {
	bybit := &fakeMarket{
		ticker: &domain.Ticker24h{LastPrice: "0.21", PriceChangePercent: "-1.5"},
		candles: []domain.Candle{
			{Open: "0.20", High: "0.22", Low: "0.19", Close: "0.21"},
		},
	}
	gecko := &fakeOffVenue{quote: &domain.OffVenueQuote{LastUSD: 0.11}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeBybit:   bybit,
	}, nil).WithDelistStore(pastDelistStore("VICUSDT")).WithOffVenuePrice(gecko)

	view, err := svc.GetPostDelist(context.Background(), "binance", "VICUSDT", "1d", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Available || view.Source != "bybit" || view.LastPrice != "0.21" {
		t.Fatalf("view=%+v", view)
	}
	if len(view.Candles) != 1 {
		t.Fatalf("candles=%d", len(view.Candles))
	}
	if bybit.lastQ.StartTime.IsZero() {
		t.Fatal("expected other-venue candles to start at halt")
	}
}

func TestFillHaltedOffVenueChangeUsesCoinGecko(t *testing.T) {
	pct := 4.25
	abs := 0.00489
	past := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeBybit:    &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeCoinbase: &fakeMarket{tickerErr: domain.ErrNotFound},
	}, nil).WithOffVenuePrice(&fakeOffVenue{
		quote: &domain.OffVenueQuote{LastUSD: 0.12, ChangePct: &pct, ChangeAbs: &abs},
	})
	items := []domain.SpotMarket{
		{Symbol: "VICUSDT", BaseAsset: "VIC", QuoteAsset: "USDT", PriceChangePercent: "", DelistTime: &past},
		{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", PriceChangePercent: "1.2", PriceChange: "100"},
	}
	svc.fillHaltedOffVenueChange(context.Background(), domain.ExchangeBinance, items)
	if items[0].PriceChangePercent != "4.250" || items[0].PriceChange != "0.00489" {
		t.Fatalf("halted row=%+v", items[0])
	}
	if items[1].PriceChangePercent != "1.2" || items[1].PriceChange != "100" {
		t.Fatalf("live row overwritten: %+v", items[1])
	}
}

func TestBarsAfter_DropsPreHalt(t *testing.T) {
	halt := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	got := barsAfter([]domain.Candle{
		{OpenTime: halt.Add(-time.Hour), Close: "1"},
		{OpenTime: halt, Close: "2"},
		{OpenTime: halt.Add(time.Hour), Close: "3"},
	}, halt)
	if len(got) != 2 || got[0].Close != "2" || got[1].Close != "3" {
		t.Fatalf("got=%+v", got)
	}
}

func TestGetPostDelist_SkipsOtherVenueWhenAlsoDelisted(t *testing.T) {
	store := pastDelistStore("VICUSDT")
	store.ReplaceAll(domain.ExchangeBybit, []domain.SpotDelistEntry{
		{Symbol: "VICUSDT", DelistTime: time.Now().UTC().Add(-24 * time.Hour)},
	})
	gecko := &fakeOffVenue{quote: &domain.OffVenueQuote{LastUSD: 0.09, AsOf: time.Now().UTC()}}
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeBybit: &fakeMarket{
			ticker: &domain.Ticker24h{LastPrice: "0.50", Halted: false},
		},
	}, nil).WithDelistStore(store).WithOffVenuePrice(gecko)

	view, err := svc.GetPostDelist(context.Background(), "binance", "VICUSDT", "1d", 10)
	if err != nil {
		t.Fatal(err)
	}
	if view.Source != "coingecko" || view.LastPrice != "0.09" {
		t.Fatalf("expected gecko after bybit delist, got %+v", view)
	}
}

func TestGetPostDelist_RequiresSymbol(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{},
	}, nil)
	_, err := svc.GetPostDelist(context.Background(), "binance", "", "1d", 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetPostDelist_NoSource(t *testing.T) {
	svc := NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: &fakeMarket{tickerErr: domain.ErrNotFound},
		domain.ExchangeBybit:   &fakeMarket{tickerErr: domain.ErrNotFound},
	}, nil).WithDelistStore(pastDelistStore("VICUSDT"))
	view, err := svc.GetPostDelist(context.Background(), "binance", "VICUSDT", "1d", 10)
	if err != nil {
		t.Fatal(err)
	}
	if view.Available {
		t.Fatalf("available=%+v", view)
	}
	if !strings.Contains(view.Note, "No public") {
		t.Fatalf("note=%q", view.Note)
	}
}
