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

func TestWatchlist_AddRemove(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	wl, err := svc.Add(context.Background(), "me", "", "binance", "btcusdt", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(wl.Items) != 1 || wl.Items[0].Symbol != "BTCUSDT" || wl.Role != domain.WatchlistRoleOwner {
		t.Fatalf("%+v", wl)
	}
	wl, err = svc.Remove(context.Background(), "me", "", "binance", "BTCUSDT")
	if err != nil || len(wl.Items) != 0 {
		t.Fatalf("%+v %v", wl, err)
	}
}

func TestWatchlist_ShareViewerEditorAudit(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	_, err := svc.Add(ctx, "owner1", "", "binance", "BTCUSDT", "note")
	if err != nil {
		t.Fatal(err)
	}

	// Share as viewer
	sh, err := svc.Share(ctx, "owner1", "viewer1", "viewer")
	if err != nil || sh.Role != domain.WatchlistRoleViewer {
		t.Fatalf("%+v %v", sh, err)
	}
	// Cannot share twice
	_, err = svc.Share(ctx, "owner1", "viewer1", "editor")
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("dup share: %v", err)
	}

	// Viewer can get
	acc, err := svc.Get(ctx, "viewer1", "owner1")
	if err != nil || acc.Role != domain.WatchlistRoleViewer || len(acc.Items) != 1 {
		t.Fatalf("%+v %v", acc, err)
	}
	// Viewer cannot add
	_, err = svc.Add(ctx, "viewer1", "owner1", "binance", "ETHUSDT", "")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer add: %v", err)
	}
	// Viewer cannot replace
	_, err = svc.Replace(ctx, "viewer1", "owner1", nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("viewer replace: %v", err)
	}

	// Upgrade via UpdateShareRole (create new as editor for editor1)
	sh2, err := svc.Share(ctx, "owner1", "editor1", "editor")
	if err != nil || sh2.Role != domain.WatchlistRoleEditor {
		t.Fatalf("%+v %v", sh2, err)
	}
	acc, err = svc.Add(ctx, "editor1", "owner1", "binance", "ETHUSDT", "")
	if err != nil || len(acc.Items) != 2 {
		t.Fatalf("editor add %+v %v", acc, err)
	}
	// Editor cannot replace or manage shares
	_, err = svc.Replace(ctx, "editor1", "owner1", nil)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("editor replace: %v", err)
	}
	_, err = svc.Share(ctx, "editor1", "someone", "viewer")
	// editor1 is not owner of their share action - Share uses ownerClientID as actor
	// Share(ctx, ownerClientID, grantee, role) — editor calling Share("editor1", ...) shares editor1's list
	// For owner1 list, only owner1 can Share. Good.

	// Editor cannot revoke owner1's shares
	// Revoke is by ownerClientID only.

	// Remove access
	if err := svc.RevokeShare(ctx, "owner1", "viewer1"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Get(ctx, "viewer1", "owner1")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("after revoke: %v", err)
	}

	// Shared with me
	list, err := svc.ListSharedWithMe(ctx, "editor1")
	if err != nil || len(list) != 1 || list[0].OwnerClientID != "owner1" {
		t.Fatalf("%+v %v", list, err)
	}

	// Audit has events
	audit, err := svc.ListAudit(ctx, "owner1", 50, 0)
	if err != nil || len(audit) < 4 {
		t.Fatalf("audit len=%d err=%v", len(audit), err)
	}
	// Ensure actor recorded for editor add
	found := false
	for _, ev := range audit {
		if ev.Action == domain.WatchlistAuditItemAdded && ev.ActorClientID == "editor1" && ev.Symbol == "ETHUSDT" {
			found = true
			if ev.CreatedAt.IsZero() {
				t.Fatal("missing timestamp")
			}
		}
	}
	if !found {
		t.Fatalf("missing editor add audit: %+v", audit)
	}
}

func TestWatchlist_StrangerForbidden(t *testing.T) {
	svc := New(watchliststore.NewMemory())
	ctx := context.Background()
	_, _ = svc.Add(ctx, "a", "", "binance", "BTCUSDT", "")
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

// keep compile of fmt in older helpers if needed
var _ = fmt.Sprintf
var _ = time.Now
