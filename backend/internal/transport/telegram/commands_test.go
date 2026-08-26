package telegram

import (
	"context"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
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
func (f *fakeMarket) GetOrderBook(_ context.Context, q domain.OrderBookQuery) (*domain.RawOrderBook, error) {
	return &domain.RawOrderBook{
		Symbol: q.Symbol,
		Bids:   []domain.PriceLevel{{Price: 100, Quantity: 1}},
		Asks:   []domain.PriceLevel{{Price: 100.1, Quantity: 1}},
	}, nil
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

func fptr(f float64) *float64 { return &f }

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	fm := &fakeMarket{}
	ms := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  fm,
		domain.ExchangeCoinbase: fm,
		domain.ExchangeBybit:    fm,
	}, fakeSupply{})
	ws := watchlist.New(watchliststore.NewMemory())
	return NewRouter(ms, ws, Options{
		DefaultExchange: "binance", LowMcapLimit: 10, AllowAll: true,
		Identities: accountstore.NewMemory(),
	})
}

func TestLongShortCommand(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 14, 14, "/ls BTCUSDT")
	if strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "BTCUSDT") {
		t.Fatalf("%s", out)
	}
}

func TestFundingCommand(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 13, 13, "/funding BTCUSDT")
	if strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "BTCUSDT") {
		t.Fatalf("%s", out)
	}
}

func TestFundingArbCommand(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 13, 13, "/fundingarb BTCUSDT 10000")
	if strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "BTCUSDT") {
		t.Fatalf("%s", out)
	}
}

func TestFundingArbScanCommand(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 13, 13, "/fundingarb scan 5000")
	if strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(strings.ToLower(out), "scan") && !strings.Contains(out, "funding") {
		t.Fatalf("%s", out)
	}
}

func TestOpenInterestCommand(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 12, 12, "/oi BTCUSDT")
	if strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "BTCUSDT") {
		t.Fatalf("%s", out)
	}
}

func TestRSIArgOrder_EitherWay(t *testing.T) {
	r := newTestRouter(t)
	// Allow rate limit by using different chats.
	out1 := r.Handle(context.Background(), 10, 10, "/rsi BTCUSDT 1h binance")
	out2 := r.Handle(context.Background(), 11, 11, "/rsi BTCUSDT binance 1h")
	if !strings.Contains(out1, "RSI") && !strings.Contains(out1, "50") {
		t.Fatalf("interval,exchange failed: %s", out1)
	}
	if strings.Contains(strings.ToLower(out2), "error") {
		t.Fatalf("exchange,interval should work: %s", out2)
	}
}

func TestWatchUsesStableUnguessableClientID(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 99, 42, "/watch add BTCUSDT")
	if strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("add: %s", out)
	}
	id1, err := r.clientIDForUser(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	id2, err := r.clientIDForUser(context.Background(), 42)
	if err != nil || id1 != id2 {
		t.Fatalf("stable id %q vs %q err=%v", id1, id2, err)
	}
	if strings.HasPrefix(id1, "tg-") {
		t.Fatalf("enumerable telegram id leaked: %s", id1)
	}
	wl, err := r.watch.Get(context.Background(), id1, "")
	if err != nil || len(wl.Items) != 1 || wl.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("watchlist via mapped id: %+v %v", wl, err)
	}
}

func TestWatchAddRejectedWhenAccountClosed(t *testing.T) {
	ids := accountstore.NewMemory()
	acc := account.New(ids, account.DataPurgeDeps{})
	fm := &fakeMarket{}
	ms := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{
		domain.ExchangeBinance:  fm,
		domain.ExchangeCoinbase: fm,
		domain.ExchangeBybit:    fm,
	}, fakeSupply{})
	ws := watchlist.New(watchliststore.NewMemory())
	r := NewRouter(ms, ws, Options{
		DefaultExchange: "binance", LowMcapLimit: 10, AllowAll: true,
		Identities: ids, Accounts: acc,
	})
	ctx := context.Background()
	out := r.Handle(ctx, 7, 99, "/watch add ETHUSDT")
	if strings.Contains(strings.ToLower(out), "error") {
		t.Fatalf("pre-close add: %s", out)
	}
	id, err := ids.ClientIDForTelegramUser(ctx, 99)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acc.Close(ctx, id); err != nil {
		t.Fatal(err)
	}
	out = r.Handle(ctx, 8, 99, "/watch add BTCUSDT")
	if !strings.Contains(strings.ToLower(out), "closed") {
		t.Fatalf("want closed, got %s", out)
	}
	wl, err := ws.Get(ctx, id, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 1 || wl.Items[0].Symbol != "ETHUSDT" {
		t.Fatalf("watchlist mutated after close: %+v", wl.Items)
	}
}

func TestFreeTextNotPrice(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 12, 12, "hello there")
	// Must not render a ticker card (FormatTicker uses 📈).
	if strings.Contains(out, "📈") || strings.Contains(out, "Last") {
		t.Fatalf("free text should not be /price: %s", out)
	}
	if !strings.Contains(strings.ToLower(out), "command") && !strings.Contains(out, "/help") {
		t.Fatalf("want help hint: %s", out)
	}
}

func TestAllowAllRequiredWhenEmpty(t *testing.T) {
	fm := &fakeMarket{}
	ms := market.NewMulti(map[domain.Exchange]domain.MarketDataPort{domain.ExchangeBinance: fm}, fakeSupply{})
	r := NewRouter(ms, nil, Options{DefaultExchange: "binance"}) // no allowlist, AllowAll false
	out := r.Handle(context.Background(), 1, 1, "/help")
	if !strings.Contains(out, "private") {
		t.Fatalf("fail closed expected: %s", out)
	}
}

func TestAsk_RequiresAI(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 20, 20, "/ask what is btc")
	if !strings.Contains(strings.ToLower(out), "ai") {
		t.Fatalf("want AI not configured message: %s", out)
	}
}

func TestAsk_UsageWhenEmpty(t *testing.T) {
	r := newTestRouter(t)
	out := r.Handle(context.Background(), 21, 21, "/ask")
	if !strings.Contains(out, "Usage") {
		t.Fatalf("%s", out)
	}
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
