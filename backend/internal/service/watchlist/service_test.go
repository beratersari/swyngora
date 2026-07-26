package watchlist

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type mem struct {
	mu   sync.Mutex
	data map[string]*domain.Watchlist
}

func newMem() *mem {
	return &mem{data: map[string]*domain.Watchlist{}}
}

func (m *mem) Get(_ context.Context, clientID string) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if wl, ok := m.data[clientID]; ok {
		cp := *wl
		cp.Items = append([]domain.WatchlistItem(nil), wl.Items...)
		return &cp, nil
	}
	return &domain.Watchlist{ClientID: clientID, Items: nil, Updated: time.Now().UTC()}, nil
}

func (m *mem) Set(_ context.Context, clientID string, items []domain.WatchlistItem) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(items) > maxItems {
		return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, maxItems)
	}
	wl := &domain.Watchlist{ClientID: clientID, Items: append([]domain.WatchlistItem(nil), items...), Updated: time.Now().UTC()}
	m.data[clientID] = wl
	cp := *wl
	cp.Items = append([]domain.WatchlistItem(nil), items...)
	return &cp, nil
}

func (m *mem) Add(ctx context.Context, clientID string, item domain.WatchlistItem) (*domain.Watchlist, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wl := m.data[clientID]
	if wl == nil {
		wl = &domain.Watchlist{ClientID: clientID}
	}
	items := append([]domain.WatchlistItem(nil), wl.Items...)
	found := false
	for i, it := range items {
		if it.Exchange == item.Exchange && it.Symbol == item.Symbol {
			items[i] = item
			found = true
			break
		}
	}
	if !found {
		if len(items) >= maxItems {
			return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, maxItems)
		}
		items = append(items, item)
	}
	out := &domain.Watchlist{ClientID: clientID, Items: items, Updated: time.Now().UTC()}
	m.data[clientID] = out
	cp := *out
	cp.Items = append([]domain.WatchlistItem(nil), items...)
	return &cp, nil
}

func (m *mem) Remove(ctx context.Context, clientID string, exchange domain.Exchange, symbol string) (*domain.Watchlist, error) {
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

func TestWatchlist_AddRemove(t *testing.T) {
	svc := New(newMem())
	wl, err := svc.Add(context.Background(), "me", "binance", "btcusdt", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 1 || wl.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("%+v", wl.Items)
	}
	wl, err = svc.Remove(context.Background(), "me", "binance", "BTCUSDT")
	if err != nil || len(wl.Items) != 0 {
		t.Fatalf("%+v %v", wl, err)
	}
}

func TestWatchlist_SixItems_StableMembership(t *testing.T) {
	svc := New(newMem())
	ctx := context.Background()
	syms := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT", "DOGEUSDT", "ADAUSDT"}
	for _, s := range syms {
		if _, err := svc.Add(ctx, "user1", "binance", s, ""); err != nil {
			t.Fatal(err)
		}
	}
	wl, err := svc.Get(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 6 {
		t.Fatalf("want 6 items, got %d", len(wl.Items))
	}
	// Re-get after "sort" simulation: membership must not shrink
	wl2, err := svc.Get(ctx, "user1")
	if err != nil || len(wl2.Items) != 6 {
		t.Fatalf("membership unstable: %d", len(wl2.Items))
	}
}

func TestWatchlist_AddEnforcesMaxItems(t *testing.T) {
	svc := New(newMem())
	ctx := context.Background()
	for i := 0; i < maxItems; i++ {
		sym := "SYM" + strconv.Itoa(i) + "USDT"
		if _, err := svc.Add(ctx, "cap", "binance", sym, ""); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	_, err := svc.Add(ctx, "cap", "binance", "OVERFLOWUSDT", "")
	if err == nil {
		t.Fatal("expected max items error")
	}
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want invalid argument, got %v", err)
	}
	// Upsert existing must still work.
	if _, err := svc.Add(ctx, "cap", "binance", "SYM0USDT", "note"); err != nil {
		t.Fatalf("upsert existing: %v", err)
	}
}

func TestWatchlist_MultiExchange(t *testing.T) {
	svc := New(newMem())
	ctx := context.Background()
	_, _ = svc.Add(ctx, "u", "binance", "BTCUSDT", "")
	_, _ = svc.Add(ctx, "u", "bybit", "BTCUSDT", "")
	_, _ = svc.Add(ctx, "u", "coinbase", "BTC-USD", "")
	wl, err := svc.Get(ctx, "u")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 3 {
		t.Fatalf("want 3 (same base, different venues), got %d %+v", len(wl.Items), wl.Items)
	}
}

func TestWatchlist_Replace_Dedupes(t *testing.T) {
	svc := New(newMem())
	wl, err := svc.Replace(context.Background(), "u", []domain.WatchlistItem{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"},
		{Exchange: domain.ExchangeBinance, Symbol: "btcusdt"},
		{Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 2 {
		t.Fatalf("dedupe want 2 got %d", len(wl.Items))
	}
}

func TestWatchlist_ClientIsolation(t *testing.T) {
	svc := New(newMem())
	ctx := context.Background()
	_, _ = svc.Add(ctx, "alice", "binance", "BTCUSDT", "")
	_, _ = svc.Add(ctx, "bob", "binance", "ETHUSDT", "")
	a, _ := svc.Get(ctx, "alice")
	b, _ := svc.Get(ctx, "bob")
	if len(a.Items) != 1 || a.Items[0].Symbol != "BTCUSDT" {
		t.Fatalf("alice=%+v", a.Items)
	}
	if len(b.Items) != 1 || b.Items[0].Symbol != "ETHUSDT" {
		t.Fatalf("bob=%+v", b.Items)
	}
}

func TestWatchlist_InvalidClientID(t *testing.T) {
	svc := New(newMem())
	_, err := svc.Get(context.Background(), "bad id!")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestWatchlist_EmptyClientIDRejected(t *testing.T) {
	svc := New(newMem())
	_, err := svc.Get(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("empty clientId must be rejected, got %v", err)
	}
	_, err = svc.Add(context.Background(), "default", "binance", "BTCUSDT", "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("shared default name must be rejected, got %v", err)
	}
}

func TestWatchlist_CoinbaseSymbolNormalized(t *testing.T) {
	svc := New(newMem())
	wl, err := svc.Add(context.Background(), "c1", "coinbase", "BTCUSD", "")
	if err != nil {
		t.Fatal(err)
	}
	if wl.Items[0].Symbol != "BTC-USD" {
		t.Fatalf("want BTC-USD, got %q", wl.Items[0].Symbol)
	}
}

func TestWatchlist_ConcurrentAddRespectsMax(t *testing.T) {
	// Real memory store enforces max under one lock (no TOCTOU).
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	items := make([]domain.WatchlistItem, 0, maxItems-1)
	for i := 0; i < maxItems-1; i++ {
		items = append(items, domain.WatchlistItem{
			Exchange: domain.ExchangeBinance,
			Symbol:   "S" + strconv.Itoa(i) + "USDT",
		})
	}
	if _, err := svc.Replace(ctx, "raceclient", items); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = svc.Add(ctx, "raceclient", "binance", "X"+strconv.Itoa(i)+"USDT", "")
		}(i)
	}
	wg.Wait()
	wl, err := svc.Get(ctx, "raceclient")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) > maxItems {
		t.Fatalf("exceeded maxItems: %d > %d", len(wl.Items), maxItems)
	}
}

func TestWatchlist_InvalidExchange(t *testing.T) {
	svc := New(newMem())
	_, err := svc.Add(context.Background(), "u", "kraken", "BTCUSDT", "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestWatchlist_EmptySymbol(t *testing.T) {
	svc := New(newMem())
	_, err := svc.Add(context.Background(), "u", "binance", "  ", "")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestWatchlist_MaxItems(t *testing.T) {
	svc := New(newMem())
	items := make([]domain.WatchlistItem, maxItems+1)
	for i := range items {
		items[i] = domain.WatchlistItem{
			Exchange: domain.ExchangeBinance,
			Symbol:   "S" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
		}
	}
	// use unique symbols
	for i := range items {
		items[i].Symbol = "SYM" + strconv.Itoa(i)
	}
	_, err := svc.Replace(context.Background(), "u", items)
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("err=%v", err)
	}
}

func TestWatchlist_AddIdempotent(t *testing.T) {
	svc := New(newMem())
	ctx := context.Background()
	_, _ = svc.Add(ctx, "u", "binance", "BTCUSDT", "note1")
	wl, err := svc.Add(ctx, "u", "binance", "BTCUSDT", "note2")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 1 {
		t.Fatalf("want 1 after re-add, got %d", len(wl.Items))
	}
	if wl.Items[0].Note != "note2" {
		t.Fatalf("note=%q", wl.Items[0].Note)
	}
}
