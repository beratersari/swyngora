package swing

import (
	"context"
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

type fakePort struct {
	candles []domain.Candle
}

func (f fakePort) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	return append([]domain.Candle(nil), f.candles...), nil
}
func (f fakePort) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "100", QuoteVolume: "5000000", Volume: "1000",
		OpenTime: time.Now().Add(-24 * time.Hour), CloseTime: time.Now().Add(-time.Minute)}, nil
}
func (f fakePort) GetOrderBook(context.Context, domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{}, nil
}
func (f fakePort) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) { return nil, nil }
func (f fakePort) ListProductTags(context.Context) ([]string, error)            { return nil, nil }
func (f fakePort) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

type fakeSupply struct{}

func (fakeSupply) GetSupply(_ context.Context, asset string) (*domain.AssetSupply, error) {
	c := 1e6
	return &domain.AssetSupply{Asset: asset, CirculatingSupply: &c}, nil
}
func (fakeSupply) Refresh(context.Context) (int, error) { return 0, nil }

func trendCandles(n int) []domain.Candle {
	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	px := 50.0
	out := make([]domain.Candle, n)
	for i := 0; i < n; i++ {
		o := px
		px += 0.7
		out[i] = domain.Candle{
			OpenTime: t0.Add(time.Duration(i) * 4 * time.Hour),
			CloseTime: t0.Add(time.Duration(i+1) * 4 * time.Hour),
			Open: strconv.FormatFloat(o, 'f', -1, 64),
			High: strconv.FormatFloat(px+0.3, 'f', -1, 64),
			Low:  strconv.FormatFloat(o-0.2, 'f', -1, 64),
			Close: strconv.FormatFloat(px, 'f', -1, 64),
			Volume: "1000",
		}
	}
	// last bar closed
	out[n-1].CloseTime = time.Now().UTC().Add(-time.Minute)
	return out
}

func TestAnalyze_ReturnsDecision(t *testing.T) {
	bars := trendCandles(120)
	m := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fakePort{candles: bars},
	}, fakeSupply{})
	svc := New(m, watchlist.New(watchliststore.NewMemory()))
	dec, err := svc.Analyze(context.Background(), "binance", "ETHUSDT")
	if err != nil {
		t.Fatal(err)
	}
	if dec == nil || dec.Price <= 0 || dec.Interval != "4h" {
		t.Fatalf("%+v", dec)
	}
	if dec.Levels != nil && dec.Levels.StopLoss >= dec.Levels.Entry {
		t.Fatalf("levels %+v", dec.Levels)
	}
}

func TestScanWatchlist_UsesClientList(t *testing.T) {
	bars := trendCandles(80)
	m := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance: fakePort{candles: bars},
	}, fakeSupply{})
	wl := watchlist.New(watchliststore.NewMemory())
	_, err := wl.Add(context.Background(), "user-sw", "", "binance", "ETHUSDT", "", domain.WatchlistUnconditionalVersion)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(m, wl)
	list, err := svc.ScanWatchlist(context.Background(), "user-sw", "binance", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Symbol != "ETHUSDT" {
		t.Fatalf("%+v", list)
	}
}
