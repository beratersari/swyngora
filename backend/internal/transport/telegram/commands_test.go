package telegram

import (
	"context"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

type fakeMarket struct {
	spot []domain.SpotMarket
}

func (f *fakeMarket) GetCandles(context.Context, domain.CandleQuery) ([]domain.Candle, error) {
	base := make([]domain.Candle, 40)
	for i := range base {
		base[i] = domain.Candle{Close: "100"}
	}
	return base, nil
}
func (f *fakeMarket) GetTicker24h(_ context.Context, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "100", PriceChangePercent: "1.5", HighPrice: "110", LowPrice: "90"}, nil
}
func (f *fakeMarket) ListSpotMarkets(context.Context) ([]domain.SpotMarket, error) {
	if f.spot != nil {
		return append([]domain.SpotMarket(nil), f.spot...), nil
	}
	return []domain.SpotMarket{
		{Symbol: "AAAUSDT", BaseAsset: "AAA", QuoteAsset: "USDT", Status: "TRADING", LastPrice: "1", QuoteVolume: "10", MarketCapCirculating: fptr(1e6)},
		{Symbol: "BBBUSDT", BaseAsset: "BBB", QuoteAsset: "USDT", Status: "TRADING", LastPrice: "2", QuoteVolume: "20", MarketCapCirculating: fptr(2e6)},
	}, nil
}
func (f *fakeMarket) ListProductTags(context.Context) ([]string, error) { return nil, nil }
func (f *fakeMarket) TagsByBase(context.Context) (map[string][]string, error) {
	return map[string][]string{}, nil
}

type fakeSupply struct{}

func (fakeSupply) GetSupply(_ context.Context, asset string) (*domain.AssetSupply, error) {
	return &domain.AssetSupply{Asset: strings.ToUpper(asset), Name: "Test"}, nil
}
func (fakeSupply) Refresh(context.Context) (int, error) { return 0, nil }

type memWatch struct {
	items map[string][]domain.WatchlistItem
}

func newMemWatch() *memWatch { return &memWatch{items: map[string][]domain.WatchlistItem{}} }

func (m *memWatch) Get(_ context.Context, clientID string) (*domain.Watchlist, error) {
	return &domain.Watchlist{ClientID: clientID, Items: append([]domain.WatchlistItem(nil), m.items[clientID]...)}, nil
}
func (m *memWatch) Set(_ context.Context, clientID string, items []domain.WatchlistItem) (*domain.Watchlist, error) {
	m.items[clientID] = append([]domain.WatchlistItem(nil), items...)
	return m.Get(context.Background(), clientID)
}
func (m *memWatch) Add(ctx context.Context, clientID string, item domain.WatchlistItem) (*domain.Watchlist, error) {
	wl, _ := m.Get(ctx, clientID)
	for i, it := range wl.Items {
		if it.Exchange == item.Exchange && it.Symbol == item.Symbol {
			wl.Items[i] = item
			return m.Set(ctx, clientID, wl.Items)
		}
	}
	return m.Set(ctx, clientID, append(wl.Items, item))
}
func (m *memWatch) Remove(ctx context.Context, clientID string, exchange domain.Exchange, symbol string) (*domain.Watchlist, error) {
	wl, _ := m.Get(ctx, clientID)
	next := make([]domain.WatchlistItem, 0, len(wl.Items))
	for _, it := range wl.Items {
		if it.Exchange == exchange && it.Symbol == symbol {
			continue
		}
		next = append(next, it)
	}
	return m.Set(ctx, clientID, next)
}

func fptr(f float64) *float64 { return &f }

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	fm := &fakeMarket{}
	ms := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  fm,
		domain.ExchangeCoinbase: fm,
		domain.ExchangeBybit:    fm,
	}, fakeSupply{})
	ws := watchlist.New(newMemWatch())
	return NewRouter(ms, ws, Options{DefaultExchange: "binance", LowMcapLimit: 10})
}

func TestHelp(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 1, 1, "/help")
	if !strings.Contains(out, "/lowmcap") {
		t.Fatalf("%s", out)
	}
}

func TestPrice(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 1, 1, "/price BTCUSDT")
	if !strings.Contains(out, "BTCUSDT") || !strings.Contains(out, "100") {
		t.Fatalf("%s", out)
	}
}

func TestLowMcapAll(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 1, 1, "/lowmcap all 2")
	if !strings.Contains(out, "BINANCE") || !strings.Contains(out, "COINBASE") || !strings.Contains(out, "<b>") {
		t.Fatalf("%s", out)
	}
}

func TestWatchAdd(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 1, 99, "/watch add ETHUSDT")
	if !strings.Contains(out, "Added") {
		t.Fatalf("%s", out)
	}
}

func TestAllowlist(t *testing.T) {
	fm := &fakeMarket{}
	ms := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{domain.ExchangeBinance: fm}, fakeSupply{})
	r := NewRouter(ms, nil, Options{
		DefaultExchange: "binance",
		AllowedChatIDs:  map[int64]struct{}{42: {}},
	})
	out := r.Handle(context.Background(), 99, 1, "/help")
	if !strings.Contains(out, "private") {
		t.Fatalf("%s", out)
	}
	out = r.Handle(context.Background(), 42, 1, "/help")
	if !strings.Contains(out, "Swyngora") {
		t.Fatalf("%s", out)
	}
}
