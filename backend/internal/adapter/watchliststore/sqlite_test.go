package watchliststore

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func openTempSQLite(t *testing.T) *SQLite {
	t.Helper()
	path := filepath.Join(t.TempDir(), "watchlist.db")
	s, err := OpenSQLite(path)
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSQLite_CRUD(t *testing.T) {
	s := openTempSQLite(t)
	ctx := context.Background()
	_, err := s.Add(ctx, "c1", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "top"}, -1)
	if err != nil {
		t.Fatal(err)
	}
	wl, err := s.Get(ctx, "c1")
	if err != nil || len(wl.Items) != 1 || wl.Items[0].Symbol != "BTCUSDT" || wl.Items[0].Note != "top" {
		t.Fatalf("%+v %v", wl, err)
	}
	wl, err = s.Remove(ctx, "c1", domain.ExchangeBinance, "BTCUSDT", -1)
	if err != nil || len(wl.Items) != 0 {
		t.Fatalf("%+v %v", wl, err)
	}
}

func TestSQLite_PersistsAcrossReopen(t *testing.T) {
	// Simulates backend restart: write, close handle, open same file again.
	dir := t.TempDir()
	path := filepath.Join(dir, "watchlist.db")
	ctx := context.Background()

	s1, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	addedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	if _, err := s1.Add(ctx, "web-user-1", domain.WatchlistItem{
		Exchange: domain.ExchangeBinance,
		Symbol:   "ETHUSDT",
		Note:     "hold",
		AddedAt:  addedAt,
	}, -1); err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Add(ctx, "web-user-1", domain.WatchlistItem{
		Exchange: domain.ExchangeBybit,
		Symbol:   "SOLUSDT",
	}, -1); err != nil {
		t.Fatal(err)
	}
	if _, err := s1.Add(ctx, "tg-42", domain.WatchlistItem{
		Exchange: domain.ExchangeCoinbase,
		Symbol:   "BTC-USD",
	}, -1); err != nil {
		t.Fatal(err)
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	// "Restart": new process opens the same DB path.
	s2, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	wl, err := s2.Get(ctx, "web-user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 2 {
		t.Fatalf("after reopen want 2 items, got %d: %+v", len(wl.Items), wl.Items)
	}
	bySym := map[string]domain.WatchlistItem{}
	for _, it := range wl.Items {
		bySym[string(it.Exchange)+"|"+it.Symbol] = it
	}
	eth, ok := bySym["binance|ETHUSDT"]
	if !ok {
		t.Fatalf("missing ETHUSDT: %+v", wl.Items)
	}
	if eth.Note != "hold" {
		t.Fatalf("note lost: %q", eth.Note)
	}
	if !eth.AddedAt.Equal(addedAt) {
		t.Fatalf("addedAt want %v got %v", addedAt, eth.AddedAt)
	}
	if _, ok := bySym["bybit|SOLUSDT"]; !ok {
		t.Fatalf("missing SOLUSDT: %+v", wl.Items)
	}

	tg, err := s2.Get(ctx, "tg-42")
	if err != nil || len(tg.Items) != 1 || tg.Items[0].Symbol != "BTC-USD" {
		t.Fatalf("tg list after reopen: %+v err=%v", tg, err)
	}

	// Mutations still work after reopen.
	if _, err := s2.Remove(ctx, "web-user-1", domain.ExchangeBinance, "ETHUSDT", -1); err != nil {
		t.Fatal(err)
	}
	if err := s2.Close(); err != nil {
		t.Fatal(err)
	}

	s3, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s3.Close()
	wl, err = s3.Get(ctx, "web-user-1")
	if err != nil || len(wl.Items) != 1 || wl.Items[0].Symbol != "SOLUSDT" {
		t.Fatalf("after second restart: %+v err=%v", wl, err)
	}
}

func TestSQLite_ServiceSurvivesRestart(t *testing.T) {
	// Service layer on top of SQLite must retain data after store close/reopen.
	dir := t.TempDir()
	path := filepath.Join(dir, "watchlist.db")
	ctx := context.Background()

	store1, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise via port methods used by the service.
	if _, err := store1.Add(ctx, "client-a", domain.WatchlistItem{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "core",
	}, -1); err != nil {
		t.Fatal(err)
	}
	if _, err := store1.Set(ctx, "client-a", []domain.WatchlistItem{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "core"},
		{Exchange: domain.ExchangeBinance, Symbol: "ADAUSDT"},
	}, -1); err != nil {
		t.Fatal(err)
	}
	_ = store1.Close()

	store2, err := OpenSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()
	wl, err := store2.Get(ctx, "client-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 2 {
		t.Fatalf("want 2 after restart, got %d", len(wl.Items))
	}
}

func TestSQLite_GetUnknownEmpty(t *testing.T) {
	s := openTempSQLite(t)
	wl, err := s.Get(context.Background(), "nobody")
	if err != nil || len(wl.Items) != 0 {
		t.Fatalf("%+v %v", wl, err)
	}
}

func TestSQLite_MaxItemsEnforced(t *testing.T) {
	s := openTempSQLite(t)
	ctx := context.Background()
	for i := 0; i < domain.MaxWatchlistItems; i++ {
		_, err := s.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "S" + strconv.Itoa(i)}, -1)
		if err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	_, err := s.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "OVERFLOW"}, -1)
	if err == nil {
		t.Fatal("expected max items error")
	}
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestSQLite_MaxClients(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wl.db")
	s, err := OpenSQLiteWithMaxClients(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if _, err := s.Add(ctx, "c1", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"}, -1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(ctx, "c2", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT"}, -1); err != nil {
		t.Fatal(err)
	}
	_, err = s.Add(ctx, "c3", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "SOLUSDT"}, -1)
	if err == nil {
		t.Fatal("expected capacity error")
	}
	// Existing client still writable.
	if _, err := s.Add(ctx, "c1", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "XRPUSDT"}, -1); err != nil {
		t.Fatalf("existing client: %v", err)
	}
}

func TestSQLite_GetReturnsCopy(t *testing.T) {
	s := openTempSQLite(t)
	ctx := context.Background()
	_, _ = s.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT"}, -1)
	a, _ := s.Get(ctx, "c")
	a.Items[0].Symbol = "HACKED"
	b, _ := s.Get(ctx, "c")
	if b.Items[0].Symbol != "BTCUSDT" {
		t.Fatal("Get must return independent slice")
	}
}

func TestSQLite_ConcurrentAdds(t *testing.T) {
	s := openTempSQLite(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sym := "SYM" + strconv.Itoa(i)
			_, _ = s.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: sym}, -1)
		}(i)
	}
	wg.Wait()
	wl, err := s.Get(ctx, "c")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 40 {
		t.Fatalf("want 40 concurrent unique adds, got %d", len(wl.Items))
	}
}

func TestSQLite_SetReplaceAndUpsertNote(t *testing.T) {
	s := openTempSQLite(t)
	ctx := context.Background()
	_, err := s.Set(ctx, "c", []domain.WatchlistItem{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "a"},
		{Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT"},
	}, -1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Add(ctx, "c", domain.WatchlistItem{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "b"}, -1)
	if err != nil {
		t.Fatal(err)
	}
	wl, err := s.Get(ctx, "c")
	if err != nil || len(wl.Items) != 2 {
		t.Fatalf("%+v %v", wl, err)
	}
	for _, it := range wl.Items {
		if it.Symbol == "BTCUSDT" && it.Note != "b" {
			t.Fatalf("upsert note: %+v", it)
		}
	}
}