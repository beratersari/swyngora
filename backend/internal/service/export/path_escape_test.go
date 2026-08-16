package export

import (
	"context"
	"path/filepath"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/exportstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestVerify_DotDotClientIDWritesOutsideFileDir(t *testing.T) {
	ctx := context.Background()
	fileDir := filepath.Join(t.TempDir(), "exports")
	wl := watchliststore.NewMemory()
	svc, err := New(exportstore.NewMemory(), DataSources{
		Watchlist: wl, Alerts: &fakeAlerts{}, Scanner: &fakeScanner{},
	}, Options{FileDir: fileDir, FileTTL: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.NormalizeClientID(".."); err == nil {
		t.Fatal("NormalizeClientID must reject ..")
	}
	if _, err := svc.Start(ctx, StartInput{ClientID: "..", Format: "json"}); err == nil {
		t.Fatal("export Start must reject clientId ..")
	}
}
