package pricediff

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/pricediffstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type fakeTicker struct {
	// key: exchange|symbol -> ticker
	data map[string]*domain.Ticker24h
	err  map[string]error
}

func (f *fakeTicker) GetTicker24h(_ context.Context, exchange, symbol string) (*domain.Ticker24h, error) {
	k := exchange + "|" + symbol
	if f.err != nil {
		if e, ok := f.err[k]; ok {
			return nil, e
		}
	}
	if f.data == nil {
		return nil, domain.ErrNotFound
	}
	t, ok := f.data[k]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func newSvc(t *testing.T, m TickerFetcher) *Service {
	t.Helper()
	store, err := pricediffstore.Open(filepath.Join(t.TempDir(), "pd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := New(store, m)
	return svc
}

func venueBook(bid, ask, qty float64, now time.Time) *domain.RawOrderBook {
	return &domain.RawOrderBook{
		Live: true, FetchedAt: now,
		Bids: []domain.PriceLevel{{Price: bid, Quantity: qty}},
		Asks: []domain.PriceLevel{{Price: ask, Quantity: qty}},
	}
}

func booksAt(now time.Time, binance, coinbase, bybit float64) *fakeBooks {
	return &fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT":  venueBook(binance, binance, 1000, now),
		"coinbase|BTC-USD": venueBook(coinbase, coinbase, 1000, now),
		"bybit|BTCUSDT":    venueBook(bybit, bybit, 1000, now),
	}}
}

func freshTicker(price string, now time.Time) *domain.Ticker24h {
	return &domain.Ticker24h{LastPrice: price, CloseTime: now.Add(-10 * time.Second)}
}

type closedAccounts map[string]bool

func (m closedAccounts) IsClosed(_ context.Context, clientID string) (bool, *domain.Account, error) {
	return m[clientID], nil, nil
}

func TestProcessActiveWatches_SkipsClosedAccount(t *testing.T) {
	now := time.Now().UTC()
	m := &fakeTicker{data: map[string]*domain.Ticker24h{
		"binance|BTCUSDT":  freshTicker("100", now),
		"coinbase|BTC-USD": freshTicker("102", now),
		"bybit|BTCUSDT":    freshTicker("100.2", now),
	}}
	svc := newSvc(t, m)
	svc.MaxPriceAge = 2 * time.Minute
	svc.SetAccountChecker(closedAccounts{"arb-closed": true})
	ctx := context.Background()
	if _, err := svc.CreateWatch(ctx, CreateInput{
		ClientID: "arb-closed", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 1,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	}); err != nil {
		t.Fatal(err)
	}
	c, cl, touched, err := svc.ProcessActiveWatches(ctx, now)
	if err != nil || c != 0 || cl != 0 || touched != 0 {
		t.Fatalf("closed tenant must not process created=%d closed=%d touched=%d err=%v", c, cl, touched, err)
	}
}

func TestCreateWatch_RequiresNotional(t *testing.T) {
	svc := newSvc(t, &fakeTicker{})
	_, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "u-floor", Symbol: "BTCUSDT", MinProfit: 1,
	})
	if err == nil {
		t.Fatal("expected missing notional to be rejected")
	}
}

func TestCreateListDeleteWatch(t *testing.T) {
	svc := newSvc(t, &fakeTicker{})
	ctx := context.Background()
	w, err := svc.CreateWatch(ctx, CreateInput{
		ClientID: "u1", Symbol: "btcusdt", Notional: 10000, MinProfit: 5,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.6, FeeBybitPct: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if w.Symbol != "BTCUSDT" || w.Status != domain.PriceDiffWatchActive {
		t.Fatalf("%+v", w)
	}
	list, err := svc.ListWatches(ctx, "u1")
	if err != nil || len(list) != 1 {
		t.Fatalf("%+v %v", list, err)
	}
	if err := svc.DeleteWatch(ctx, "u1", w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetWatch(ctx, "u1", w.ID); err != domain.ErrNotFound {
		t.Fatalf("%v", err)
	}
}

func TestOpportunity_OpenTouchCloseReopen(t *testing.T) {
	now := time.Now().UTC()
	m := &fakeTicker{data: map[string]*domain.Ticker24h{
		"binance|BTCUSDT":  freshTicker("100", now),
		"coinbase|BTC-USD": freshTicker("102", now),
		"bybit|BTCUSDT":    freshTicker("100.2", now),
	}}
	books := booksAt(now, 100, 102, 100.2)
	svc := newSvc(t, m).WithBooks(books)
	svc.MaxPriceAge = 2 * time.Minute
	ctx := context.Background()
	if _, err := svc.CreateWatch(ctx, CreateInput{
		ClientID: "arb", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 10,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	}); err != nil {
		t.Fatal(err)
	}

	// First tick: open opportunity buy binance sell coinbase
	c, cl, _, err := svc.ProcessActiveWatches(ctx, now)
	if err != nil || c < 1 {
		t.Fatalf("created=%d closed=%d err=%v", c, cl, err)
	}
	opps, err := svc.ListOpportunities(ctx, "arb", "open", 20, 0)
	if err != nil || len(opps) < 1 {
		t.Fatalf("%+v %v", opps, err)
	}
	firstID := opps[0].ID
	if opps[0].BuyExchange != domain.ExchangeBinance || opps[0].SellExchange != domain.ExchangeCoinbase {
		// may have multiple; find binance->coinbase
		found := false
		for _, o := range opps {
			if o.BuyExchange == domain.ExchangeBinance && o.SellExchange == domain.ExchangeCoinbase {
				firstID = o.ID
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no binance->coinbase in %+v", opps)
		}
	}

	// Second tick with still-wide spread: do not create duplicate open
	c2, _, tch, err := svc.ProcessActiveWatches(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if c2 != 0 {
		t.Fatalf("want 0 new creates while open, got %d", c2)
	}
	if tch < 1 {
		t.Fatalf("expected touch, got %d", tch)
	}
	opps, _ = svc.ListOpportunities(ctx, "arb", "open", 20, 0)
	openCount := 0
	for _, o := range opps {
		if o.BuyExchange == domain.ExchangeBinance && o.SellExchange == domain.ExchangeCoinbase {
			openCount++
			if o.ID != firstID {
				t.Fatalf("duplicate open opp %s vs %s", o.ID, firstID)
			}
		}
	}
	if openCount != 1 {
		t.Fatalf("open routes count=%d", openCount)
	}

	// Collapse spread → close
	m.data["coinbase|BTC-USD"] = freshTicker("100.1", now)
	books.data["coinbase|BTC-USD"] = venueBook(100.1, 100.1, 1000, now)
	_, cl3, _, err := svc.ProcessActiveWatches(ctx, now.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if cl3 < 1 {
		t.Fatalf("expected close, closed=%d", cl3)
	}
	got, err := svc.GetOpportunity(ctx, "arb", firstID)
	if err != nil || got.Status != domain.PriceDiffOppClosed {
		t.Fatalf("%+v %v", got, err)
	}

	// Spread returns → new opportunity
	m.data["coinbase|BTC-USD"] = freshTicker("103", now)
	books.data["coinbase|BTC-USD"] = venueBook(103, 103, 1000, now)
	c4, _, _, err := svc.ProcessActiveWatches(ctx, now.Add(3*time.Second))
	if err != nil || c4 < 1 {
		t.Fatalf("created=%d err=%v", c4, err)
	}
	opps, _ = svc.ListOpportunities(ctx, "arb", "open", 20, 0)
	newID := ""
	for _, o := range opps {
		if o.BuyExchange == domain.ExchangeBinance && o.SellExchange == domain.ExchangeCoinbase {
			newID = o.ID
		}
	}
	if newID == "" || newID == firstID {
		t.Fatalf("want new opportunity id, got %q (old %q)", newID, firstID)
	}
}

func TestSkipMissingBook(t *testing.T) {
	now := time.Now().UTC()
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"coinbase|BTC-USD": venueBook(110, 110, 1000, now),
		"bybit|BTCUSDT":    venueBook(100, 100, 1000, now),
	}})
	ctx := context.Background()
	_, err := svc.CreateWatch(ctx, CreateInput{
		ClientID: "stale", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 1,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	c, _, _, err := svc.ProcessActiveWatches(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if c < 1 {
		t.Fatalf("expected opp from bybit/coinbase books, created=%d", c)
	}
	svc2 := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|ETHUSDT": venueBook(100, 100, 1000, now),
	}})
	_, err = svc2.CreateWatch(ctx, CreateInput{ClientID: "one", Symbol: "ETHUSDT", Notional: 10000, MinProfit: 1})
	if err != nil {
		t.Fatal(err)
	}
	c2, _, _, err := svc2.ProcessActiveWatches(ctx, now)
	if err != nil || c2 != 0 {
		t.Fatalf("want 0 created with a single book, got %d err=%v", c2, err)
	}
}

func TestTickerSpreadWithoutBookDoesNotOpen(t *testing.T) {
	now := time.Now().UTC()
	m := &fakeTicker{data: map[string]*domain.Ticker24h{
		"binance|BTCUSDT":  freshTicker("100", now),
		"coinbase|BTC-USD": freshTicker("110", now),
		"bybit|BTCUSDT":    freshTicker("100", now),
	}}
	svc := newSvc(t, m).WithBooks(booksAt(now, 100, 100, 100))
	if _, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "tick", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 1,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	}); err != nil {
		t.Fatal(err)
	}
	c, _, _, err := svc.ProcessActiveWatches(context.Background(), now)
	if err != nil || c != 0 {
		t.Fatalf("ticker gap without book edge must not open, created=%d err=%v", c, err)
	}
}

func TestThinBookDoesNotOpen(t *testing.T) {
	now := time.Now().UTC()
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT":  venueBook(100, 100, 0.01, now),
		"coinbase|BTC-USD": venueBook(110, 110, 0.01, now),
		"bybit|BTCUSDT":    venueBook(100, 100, 0.01, now),
	}})
	if _, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "thin", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 1,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	}); err != nil {
		t.Fatal(err)
	}
	c, _, _, err := svc.ProcessActiveWatches(context.Background(), now)
	if err != nil || c != 0 {
		t.Fatalf("thin book must not open, created=%d err=%v", c, err)
	}
}

func TestStaleBookDoesNotOpen(t *testing.T) {
	now := time.Now().UTC()
	books := booksAt(now.Add(-10*time.Minute), 100, 110, 100)
	svc := newSvc(t, &fakeTicker{}).WithBooks(books)
	svc.MaxPriceAge = 2 * time.Minute
	if _, err := svc.CreateWatch(context.Background(), CreateInput{
		ClientID: "stale-book", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 1,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	}); err != nil {
		t.Fatal(err)
	}
	c, _, _, err := svc.ProcessActiveWatches(context.Background(), now)
	if err != nil || c != 0 {
		t.Fatalf("stale books must not open, created=%d err=%v", c, err)
	}
}

func TestMinDurationHoldsThenOpensAndResets(t *testing.T) {
	t0 := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	books := booksAt(t0, 100, 110, 100)
	svc := newSvc(t, &fakeTicker{}).WithBooks(books)
	svc.MaxPriceAge = 10 * time.Minute
	ctx := context.Background()
	if _, err := svc.CreateWatch(ctx, CreateInput{
		ClientID: "hold", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 1,
		MinDurationSec: 60,
		FeeBinancePct:  0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	}); err != nil {
		t.Fatal(err)
	}
	stampBooks(books, t0)
	c, _, _, err := svc.ProcessActiveWatches(ctx, t0)
	if err != nil || c != 0 {
		t.Fatalf("first tick must wait, created=%d err=%v", c, err)
	}
	stampBooks(books, t0.Add(30*time.Second))
	c, _, _, err = svc.ProcessActiveWatches(ctx, t0.Add(30*time.Second))
	if err != nil || c != 0 {
		t.Fatalf("30s < 60s, created=%d err=%v", c, err)
	}
	stampBooks(books, t0.Add(60*time.Second))
	c, _, _, err = svc.ProcessActiveWatches(ctx, t0.Add(60*time.Second))
	if err != nil || c < 1 {
		t.Fatalf("held 60s should open, created=%d err=%v", c, err)
	}

	// Close then one-tick spike must not open until duration holds again.
	stampBooks(books, t0.Add(70*time.Second))
	books.data["coinbase|BTC-USD"] = venueBook(100, 100, 1000, t0.Add(70*time.Second))
	_, cl, _, err := svc.ProcessActiveWatches(ctx, t0.Add(70*time.Second))
	if err != nil || cl < 1 {
		t.Fatalf("expected close after break, closed=%d err=%v", cl, err)
	}
	books.data["coinbase|BTC-USD"] = venueBook(110, 110, 1000, t0.Add(80*time.Second))
	c, _, _, err = svc.ProcessActiveWatches(ctx, t0.Add(80*time.Second))
	if err != nil || c != 0 {
		t.Fatalf("re-arm first tick must wait, created=%d err=%v", c, err)
	}
	books.data["coinbase|BTC-USD"] = venueBook(100, 100, 1000, t0.Add(90*time.Second))
	c, _, _, err = svc.ProcessActiveWatches(ctx, t0.Add(90*time.Second))
	if err != nil || c != 0 {
		t.Fatalf("break should reset arm, created=%d err=%v", c, err)
	}
	books.data["coinbase|BTC-USD"] = venueBook(110, 110, 1000, t0.Add(100*time.Second))
	c, _, _, err = svc.ProcessActiveWatches(ctx, t0.Add(100*time.Second))
	if err != nil || c != 0 {
		t.Fatalf("new arm first tick, created=%d err=%v", c, err)
	}
	books.data["coinbase|BTC-USD"] = venueBook(110, 110, 1000, t0.Add(160*time.Second))
	c, _, _, err = svc.ProcessActiveWatches(ctx, t0.Add(160*time.Second))
	if err != nil || c < 1 {
		t.Fatalf("held again should open, created=%d err=%v", c, err)
	}
}

func stampBooks(f *fakeBooks, now time.Time) {
	for _, b := range f.data {
		b.FetchedAt = now
	}
}

type fakeBooks struct {
	data map[string]*domain.RawOrderBook
	err  map[string]error
}

func (f *fakeBooks) GetRawOrderBook(_ context.Context, exchange, symbol string) (*domain.RawOrderBook, error) {
	k := exchange + "|" + symbol
	if f.err != nil {
		if e, ok := f.err[k]; ok {
			return nil, e
		}
	}
	if f.data == nil {
		return nil, domain.ErrNotFound
	}
	b, ok := f.data[k]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return b, nil
}

func quoteBooks() (buy, sell *domain.RawOrderBook) {
	now := time.Now().UTC()
	buy = &domain.RawOrderBook{
		Symbol: "BTCUSDT", Live: true, FetchedAt: now,
		Asks: []domain.PriceLevel{{Price: 100, Quantity: 1}, {Price: 101, Quantity: 1}},
	}
	sell = &domain.RawOrderBook{
		Symbol: "BTC-USD", Live: true, FetchedAt: now,
		Bids: []domain.PriceLevel{{Price: 103, Quantity: 1}, {Price: 102, Quantity: 1}},
	}
	return buy, sell
}

func TestQuote_WalksBothBooks(t *testing.T) {
	buy, sell := quoteBooks()
	svc := newSvc(t, &fakeTicker{})
	svc.WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT":  buy,
		"coinbase|BTC-USD": sell,
	}})
	got, err := svc.Quote(context.Background(), QuoteInput{
		Symbol: "btcusdt", BuyExchange: "binance", SellExchange: "coinbase",
		BuyFeePct: 0.1, SellFeePct: 0.1, Notional: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AverageBuyPrice != "100" || got.AverageSellPrice != "103" || !got.Executable {
		t.Fatalf("%+v", got)
	}
}

func TestQuote_MissingBooksStillQuotes(t *testing.T) {
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{}})
	got, err := svc.Quote(context.Background(), QuoteInput{
		Symbol: "BTCUSDT", BuyExchange: "binance", SellExchange: "bybit", Notional: 10000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Executable || got.Profitable {
		t.Fatalf("empty books should not be executable: %+v", got)
	}
	if got.MaxLimitedBy != domain.PriceDiffMaxLimitedByEmpty && got.MaxLimitedBy != domain.PriceDiffMaxLimitedByBuyBook {
		t.Fatalf("limitedBy=%s", got.MaxLimitedBy)
	}
}

func TestQuote_RejectsSameVenue(t *testing.T) {
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{}})
	_, err := svc.Quote(context.Background(), QuoteInput{
		Symbol: "BTCUSDT", BuyExchange: "binance", SellExchange: "binance", Notional: 100,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQuoteScan_RanksAllPairs(t *testing.T) {
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT": {
			Asks: []domain.PriceLevel{{Price: 100, Quantity: 2}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 2}},
			Live: true,
		},
		"bybit|BTCUSDT": {
			Asks: []domain.PriceLevel{{Price: 104, Quantity: 2}},
			Bids: []domain.PriceLevel{{Price: 103, Quantity: 2}},
			Live: true,
		},
		"coinbase|BTC-USD": {
			Asks: []domain.PriceLevel{{Price: 110, Quantity: 2}},
			Bids: []domain.PriceLevel{{Price: 109, Quantity: 2}},
			Live: true,
		},
	}})
	got, err := svc.QuoteScan(context.Background(), ScanInput{
		Symbol: "BTCUSDT", Notional: 200,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.BestRoute == nil || got.BestRoute.BuyExchange != domain.ExchangeBinance || got.BestRoute.SellExchange != domain.ExchangeCoinbase {
		t.Fatalf("best=%+v", got.BestRoute)
	}
	if len(got.Routes) != 6 || got.ProfitableCount < 1 {
		t.Fatalf("routes=%d profitable=%d", len(got.Routes), got.ProfitableCount)
	}
}

func TestQuoteScan_MissingBookIsUnavailable(t *testing.T) {
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT": {
			Asks: []domain.PriceLevel{{Price: 100, Quantity: 2}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 2}},
		},
		"bybit|BTCUSDT": {
			Asks: []domain.PriceLevel{{Price: 101, Quantity: 2}},
			Bids: []domain.PriceLevel{{Price: 100.5, Quantity: 2}},
		},
	}})
	got, err := svc.QuoteScan(context.Background(), ScanInput{Symbol: "BTCUSDT", Notional: 100})
	if err != nil {
		t.Fatal(err)
	}
	if got.BestRoute != nil && (got.BestRoute.BuyExchange == domain.ExchangeCoinbase || got.BestRoute.SellExchange == domain.ExchangeCoinbase) {
		t.Fatalf("best used missing venue: %+v", got.BestRoute)
	}
	found := false
	for _, u := range got.Unavailable {
		if u.Exchange == "coinbase" {
			found = true
			if u.Message == "" {
				t.Fatal("expected load-failure message")
			}
		}
	}
	if !found {
		t.Fatalf("unavailable=%+v", got.Unavailable)
	}
	for _, r := range got.Routes {
		if r.BuyExchange == domain.ExchangeCoinbase || r.SellExchange == domain.ExchangeCoinbase {
			t.Fatalf("missing venue listed as a route: %+v", r)
		}
	}
}

func TestQuoteWatch_UsesWatchFees(t *testing.T) {
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT": {
			Asks: []domain.PriceLevel{{Price: 100, Quantity: 1}},
			Bids: []domain.PriceLevel{{Price: 99, Quantity: 1}},
		},
		"bybit|BTCUSDT": {
			Asks: []domain.PriceLevel{{Price: 102, Quantity: 1}},
			Bids: []domain.PriceLevel{{Price: 101, Quantity: 1}},
		},
		"coinbase|BTC-USD": {
			Asks: []domain.PriceLevel{{Price: 103, Quantity: 1}},
			Bids: []domain.PriceLevel{{Price: 102, Quantity: 1}},
		},
	}})
	ctx := context.Background()
	w, err := svc.CreateWatch(ctx, CreateInput{
		ClientID: "scan1", Symbol: "BTCUSDT", Notional: 100, MinProfit: 0, MinNetDiffPct: 0.5,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.6, FeeBybitPct: 0.1,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.QuoteWatch(ctx, "scan1", w.ID, ScanInput{Notional: 100})
	if err != nil {
		t.Fatal(err)
	}
	if got.Symbol != "BTCUSDT" || len(got.Routes) != 6 {
		t.Fatalf("%+v", got)
	}
	for i := range got.Routes {
		if got.Routes[i].BuyExchange == domain.ExchangeBinance && got.Routes[i].BuyFeePct != 0.1 {
			t.Fatalf("binance buy fee %+v", got.Routes[i])
		}
		if got.Routes[i].SellExchange == domain.ExchangeCoinbase && got.Routes[i].SellFeePct != 0.6 {
			t.Fatalf("coinbase sell fee %+v", got.Routes[i])
		}
	}
}

func TestQuoteOpportunity_UsesWatchFees(t *testing.T) {
	now := time.Now().UTC()
	m := &fakeTicker{data: map[string]*domain.Ticker24h{
		"binance|BTCUSDT":  freshTicker("100", now),
		"coinbase|BTC-USD": freshTicker("103", now),
		"bybit|BTCUSDT":    freshTicker("100.2", now),
	}}
	svc := newSvc(t, m)
	buy, sell := quoteBooks()
	svc.WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{
		"binance|BTCUSDT":  buy,
		"coinbase|BTC-USD": sell,
	}})
	ctx := context.Background()
	if _, err := svc.CreateWatch(ctx, CreateInput{
		ClientID: "q1", Symbol: "BTCUSDT", Notional: 100, MinProfit: 0, MinNetDiffPct: 0.5,
		FeeBinancePct: 0.1, FeeCoinbasePct: 0.1, FeeBybitPct: 0.1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := svc.ProcessActiveWatches(ctx, now); err != nil {
		t.Fatal(err)
	}
	opps, err := svc.ListOpportunities(ctx, "q1", "open", 20, 0)
	if err != nil || len(opps) == 0 {
		t.Fatalf("%+v %v", opps, err)
	}
	var id string
	for _, o := range opps {
		if o.BuyExchange == domain.ExchangeBinance && o.SellExchange == domain.ExchangeCoinbase {
			id = o.ID
			break
		}
	}
	if id == "" {
		t.Fatalf("no binance->coinbase in %+v", opps)
	}
	got, err := svc.QuoteOpportunity(ctx, "q1", id, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.BuyFeePct != 0.1 || got.SellFeePct != 0.1 || !got.Profitable {
		t.Fatalf("%+v", got)
	}
	if got.MinNetDiffPct != 0.5 || !got.MeetsMinNet {
		t.Fatalf("min net %+v", got)
	}
}

func TestQuoteOpportunity_NotFound(t *testing.T) {
	svc := newSvc(t, &fakeTicker{}).WithBooks(&fakeBooks{data: map[string]*domain.RawOrderBook{}})
	_, err := svc.QuoteOpportunity(context.Background(), "q1", "missing", 100, 0)
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestOpenPersistsAcrossNewService(t *testing.T) {
	// Simulates worker restart: same store, new service instance still sees open opp.
	now := time.Now().UTC()
	m := &fakeTicker{data: map[string]*domain.Ticker24h{
		"binance|BTCUSDT":  freshTicker("100", now),
		"coinbase|BTC-USD": freshTicker("105", now),
		"bybit|BTCUSDT":    freshTicker("100", now),
	}}
	store, err := pricediffstore.Open(filepath.Join(t.TempDir(), "persist.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	svc1 := New(store, m).WithBooks(booksAt(now, 100, 105, 100))
	w, err := svc1.CreateWatch(ctx, CreateInput{ClientID: "p", Symbol: "BTCUSDT", Notional: 10000, MinProfit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := svc1.ProcessActiveWatches(ctx, now); err != nil {
		t.Fatal(err)
	}
	// "Restart"
	svc2 := New(store, m)
	opps, err := svc2.ListOpportunities(ctx, "p", "open", 10, 0)
	if err != nil || len(opps) == 0 {
		t.Fatalf("open should survive restart: %+v %v", opps, err)
	}
	// Still no duplicate on next process
	c, _, _, err := svc2.ProcessActiveWatches(ctx, now.Add(time.Second))
	if err != nil || c != 0 {
		t.Fatalf("created=%d err=%v watch=%s", c, err, w.ID)
	}
}
