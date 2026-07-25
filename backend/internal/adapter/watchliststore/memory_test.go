package watchliststore

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestMemory_CRUD(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_, err := m.Add(ctx, "c1", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"})
	if err != nil {
		t.Fatal(err)
	}
	wl, err := m.Get(ctx, "c1")
	if err != nil || len(wl.Items) != 1 {
		t.Fatalf("%+v %v", wl, err)
	}
	wl, err = m.Remove(ctx, "c1", domain.ExchangeBinance, "BTCUSDT")
	if err != nil || len(wl.Items) != 0 {
		t.Fatalf("%+v %v", wl, err)
	}
}

func TestMemory_SixItems_Stable(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	syms := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT", "DOGEUSDT", "ADAUSDT"}
	for _, s := range syms {
		if _, err := m.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: s}); err != nil {
			t.Fatal(err)
		}
	}
	wl, err := m.Get(ctx, "c")
	if err != nil || len(wl.Items) != 6 {
		t.Fatalf("len=%d err=%v", len(wl.Items), err)
	}
	// Set replace with same 6 — still 6
	wl, err = m.Set(ctx, "c", wl.Items)
	if err != nil || len(wl.Items) != 6 {
		t.Fatalf("after set len=%d", len(wl.Items))
	}
}

func TestMemory_GetReturnsCopy(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_, _ = m.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"})
	a, _ := m.Get(ctx, "c")
	a.Items[0].Symbol = "HACKED"
	b, _ := m.Get(ctx, "c")
	if b.Items[0].Symbol != "BTCUSDT" {
		t.Fatal("Get must return a copy")
	}
}

func TestMemory_ConcurrentAdds(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sym := fmt.Sprintf("SYM%d", i)
			_, _ = m.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: sym})
		}(i)
	}
	wg.Wait()
	wl, err := m.Get(ctx, "c")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 50 {
		t.Fatalf("want 50 concurrent unique adds, got %d", len(wl.Items))
	}
}
