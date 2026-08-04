package watchlist

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const uncond = domain.WatchlistUnconditionalVersion

func TestWatchlist_AddRemove(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	wl, err := svc.Add(context.Background(), "me", "", "binance", "btcusdt", "", uncond)
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 1 || wl.Items[0].Symbol != "BTCUSDT" || wl.Role != domain.WatchlistRoleOwner || wl.Version != 1 {
		t.Fatalf("%+v", wl)
	}
	wl, err = svc.Remove(context.Background(), "me", "", "binance", "BTCUSDT", uncond)
	if err != nil || len(wl.Items) != 0 || wl.Version != 2 {
		t.Fatalf("%+v %v", wl, err)
	}
}

func TestWatchlist_ShareViewerEditorAudit(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	_, err := svc.Add(ctx, "owner1", "", "binance", "BTCUSDT", "note", uncond)
	if err != nil {
		t.Fatal(err)
	}

	sh, err := svc.Share(ctx, "owner1", "viewer1", "viewer")
	if err != nil || sh.Role != domain.WatchlistRoleViewer {
		t.Fatalf("%+v %v", sh, err)
	}
	_, err = svc.Share(ctx, "owner1", "viewer1", "editor")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("dup share: %v", err)
	}

	acc, err := svc.Get(ctx, "viewer1", "owner1")
	if err != nil || acc.Role != domain.WatchlistRoleViewer || len(acc.Items) != 1 {
		t.Fatalf("%+v %v", acc, err)
	}
	_, err = svc.Add(ctx, "viewer1", "owner1", "binance", "ETHUSDT", "", uncond)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer add: %v", err)
	}
	_, err = svc.Replace(ctx, "viewer1", "owner1", nil, uncond, nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer replace: %v", err)
	}

	sh2, err := svc.Share(ctx, "owner1", "editor1", "editor")
	if err != nil || sh2.Role != domain.WatchlistRoleEditor {
		t.Fatalf("%+v %v", sh2, err)
	}
	acc, err = svc.Add(ctx, "editor1", "owner1", "binance", "ETHUSDT", "", uncond)
	if err != nil || len(acc.Items) != 2 {
		t.Fatalf("editor add %+v %v", acc, err)
	}
	_, err = svc.Replace(ctx, "editor1", "owner1", nil, uncond, nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("editor replace: %v", err)
	}

	if err := svc.RevokeShare(ctx, "owner1", "viewer1"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Get(ctx, "viewer1", "owner1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("after revoke: %v", err)
	}

	list, err := svc.ListSharedWithMe(ctx, "editor1")
	if err != nil || len(list) != 1 || list[0].OwnerClientID != "owner1" {
		t.Fatalf("%+v %v", list, err)
	}

	audit, err := svc.ListAudit(ctx, "owner1", 50, 0)
	if err != nil || len(audit) < 4 {
		t.Fatalf("audit len=%d err=%v", len(audit), err)
	}
	found := false
	for _, ev := range audit {
		if ev.Action == domain.WatchlistAuditItemAdded && ev.ActorClientID == "editor1" && ev.Symbol == "ETHUSDT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing editor add audit: %+v", audit)
	}
}

func TestWatchlist_StrangerForbidden(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	_, _ = svc.Add(ctx, "a", "", "binance", "BTCUSDT", "", uncond)
	_, err := svc.Get(ctx, "stranger", "a")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("%v", err)
	}
}

func TestWatchlist_CannotShareSelf(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	_, err := svc.Share(context.Background(), "me", "me", "viewer")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("%v", err)
	}
}

func TestWatchlist_MultiDevice_AutoMergeDifferentSymbols(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	// Device A loads v0, adds BTC
	a, err := svc.Add(ctx, "user", "", "binance", "BTCUSDT", "", 0)
	if err != nil || a.Version != 1 {
		t.Fatalf("%+v %v", a, err)
	}
	// Device B still has baseVersion 0, adds ETH → auto-merge
	b, err := svc.Add(ctx, "user", "", "binance", "ETHUSDT", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.Version != 2 || len(b.Items) != 2 {
		t.Fatalf("auto-merge want 2 items v2: %+v", b)
	}
}

func TestWatchlist_MultiDevice_DeleteVsUpdateConflict(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	a, err := svc.Add(ctx, "user", "", "binance", "BTCUSDT", "old", 0)
	if err != nil {
		t.Fatal(err)
	}
	// Device B updates note with correct version
	_, err = svc.Add(ctx, "user", "", "binance", "BTCUSDT", "new-note", a.Version)
	if err != nil {
		t.Fatal(err)
	}
	// Device A still on v1 tries to delete → conflict
	_, err = svc.Remove(ctx, "user", "", "binance", "BTCUSDT", a.Version)
	var sc *domain.WatchlistSyncConflict
	if !errors.As(err, &sc) || len(sc.Conflicts) != 1 {
		t.Fatalf("want delete_vs_update conflict, got %v", err)
	}
	if sc.Conflicts[0].Type != domain.ConflictDeleteVsUpdate {
		t.Fatalf("%+v", sc.Conflicts[0])
	}
	if sc.Conflicts[0].ServerItem == nil || sc.Conflicts[0].ServerItem.Note != "new-note" {
		t.Fatalf("%+v", sc.Conflicts[0])
	}
}

func TestWatchlist_MultiDevice_ReplaceMergeAndConflict(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	base := []domain.WatchlistItem{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "n"},
	}
	// Server has BTC
	cur, err := svc.Replace(ctx, "user", "", base, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Other device adds SOL (version advances)
	cur, err = svc.Add(ctx, "user", "", "binance", "SOLUSDT", "", cur.Version)
	if err != nil {
		t.Fatal(err)
	}
	// Client still on v1, replaces with BTC+ETH (adds ETH, doesn't know SOL)
	// baseItems = original BTC only → 3-way: keep SOL (server add), add ETH (client add)
	client := []domain.WatchlistItem{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "n"},
		{Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT", Note: ""},
	}
	out, err := svc.Replace(ctx, "user", "", client, 1, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Items) != 3 {
		t.Fatalf("want 3 merged items: %+v", out.Items)
	}

	// Now conflict: client deletes BTC while server changed note
	got, _ := svc.Get(ctx, "user", "")
	// Update BTC note on server
	got, err = svc.Add(ctx, "user", "", "binance", "BTCUSDT", "changed", got.Version)
	if err != nil {
		t.Fatal(err)
	}
	// Client with old base deletes BTC (list without BTC)
	client2 := []domain.WatchlistItem{
		{Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT"},
		{Exchange: domain.ExchangeBinance, Symbol: "SOLUSDT"},
	}
	base2 := []domain.WatchlistItem{
		{Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT", Note: "n"},
		{Exchange: domain.ExchangeBinance, Symbol: "ETHUSDT"},
		{Exchange: domain.ExchangeBinance, Symbol: "SOLUSDT"},
	}
	_, err = svc.Replace(ctx, "user", "", client2, got.Version-1, base2)
	var sc *domain.WatchlistSyncConflict
	if !errors.As(err, &sc) {
		t.Fatalf("want conflict, got %v", err)
	}
	found := false
	for _, c := range sc.Conflicts {
		if c.Symbol == "BTCUSDT" && c.Type == domain.ConflictDeleteVsUpdate {
			found = true
		}
	}
	if !found {
		t.Fatalf("conflicts: %+v", sc.Conflicts)
	}
}

// keep compile of fmt in older helpers if needed
var _ = fmt.Sprintf
var _ = time.Now
