package account

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/exportstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/importstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestAccount_CloseReopenAndBlock(t *testing.T) {
	store := accountstore.NewMemory()
	svc := New(store, DataPurgeDeps{})
	// short grace for nothing yet
	ctx := context.Background()
	a, err := svc.Close(ctx, "user1")
	if err != nil || a.Status != domain.AccountClosed || a.PurgeAt == nil {
		t.Fatalf("%+v %v", a, err)
	}
	err = svc.RequireActive(ctx, "user1")
	if err == nil {
		t.Fatal("expected closed")
	}
	var ac *domain.ErrAccountClosed
	if !errors.As(err, &ac) {
		t.Fatalf("%v", err)
	}
	// reopen
	a, err = svc.Reopen(ctx, "user1")
	if err != nil || a.Status != domain.AccountActive {
		t.Fatalf("%+v %v", a, err)
	}
	if err := svc.RequireActive(ctx, "user1"); err != nil {
		t.Fatal(err)
	}
}

func TestAccount_PurgeAfterGrace(t *testing.T) {
	store := accountstore.NewMemory()
	wl := watchliststore.NewMemory()
	ex := exportstore.NewMemory()
	im := importstore.NewMemory()
	svc := New(store, DataPurgeDeps{Watchlist: wl, Exports: ex, Imports: im}).WithGrace(time.Millisecond)
	// control clock
	base := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }

	ctx := context.Background()
	_, _ = wl.Add(ctx, "user1", domain.WatchlistItem{
		Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", AddedAt: base,
	}, domain.WatchlistUnconditionalVersion)
	// seed export job with file
	dir := t.TempDir()
	path := filepath.Join(dir, "f.json")
	_ = writeFile(path, []byte(`{}`))
	_, _ = ex.Create(ctx, domain.ExportJob{
		ID: "e1", ClientID: "user1", Format: domain.ExportFormatJSON,
		Status: domain.ExportCompleted, FilePath: path, CreatedAt: base,
	})

	if _, err := svc.Close(ctx, "user1"); err != nil {
		t.Fatal(err)
	}
	// still within grace (purgeAt = base + 1ms)
	svc.now = func() time.Time { return base.Add(500 * time.Microsecond) }
	n, err := svc.PurgeDue(ctx)
	if err != nil || n != 0 {
		t.Fatalf("within grace n=%d err=%v", n, err)
	}
	// past grace
	svc.now = func() time.Time { return base.Add(2 * time.Millisecond) }
	n, err = svc.PurgeDue(ctx)
	if err != nil || n != 1 {
		t.Fatalf("after grace n=%d err=%v", n, err)
	}
	got, _ := wl.Get(ctx, "user1")
	if len(got.Items) != 0 {
		t.Fatalf("watchlist not purged: %+v", got.Items)
	}
	// reopen after purge fails
	svc.now = func() time.Time { return base.Add(time.Hour) }
	_, err = svc.Reopen(ctx, "user1")
	if err == nil {
		t.Fatal("expected reopen fail after purge")
	}
}

func writeFile(path string, b []byte) error {
	return os.WriteFile(path, b, 0o600)
}
