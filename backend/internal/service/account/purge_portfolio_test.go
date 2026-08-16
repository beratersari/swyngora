package account

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type closedPx struct{}

func (closedPx) GetTicker24h(context.Context, string, string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "100"}, nil
}

// Finding 2c: DataPurgeDeps has no portfolio port, so purge leaves paper books.
// Status() treats a missing account row as active, so the same clientId inherits them.
func TestVerify_PurgeLeavesPaperBookForReusedClientID(t *testing.T) {
	ctx := context.Background()
	pfStore, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pfStore.Close() })
	pf := portfolio.New(pfStore, closedPx{}).WithPaperCosts(domain.ZeroTradingCosts)
	if _, err := pf.Create(ctx, portfolio.CreateInput{ClientID: "purge-user", StartingBalance: 7777}); err != nil {
		t.Fatal(err)
	}

	acctStore := accountstore.NewMemory()
	svc := New(acctStore, DataPurgeDeps{Paper: pf}).WithGrace(time.Millisecond)
	base := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }
	if _, err := svc.Close(ctx, "purge-user"); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return base.Add(2 * time.Millisecond) }
	n, err := svc.PurgeDue(ctx)
	if err != nil || n != 1 {
		t.Fatalf("purge n=%d err=%v", n, err)
	}

	st, err := svc.Status(ctx, "purge-user")
	if err != nil {
		t.Fatal(err)
	}
	view, verr := pf.View(ctx, "purge-user")
	bookStillThere := verr == nil && view != nil && view.CashBalance > 0
	statusActive := st != nil && st.Status == domain.AccountActive
	if bookStillThere && statusActive {
		t.Errorf("CONFIRMED finding 2/3: purge left paper book cash=%v and Status treats client as %s",
			view.CashBalance, st.Status)
		return
	}
	t.Logf("NOT REPRODUCED: bookStill=%v status=%v viewErr=%v", bookStillThere, st.Status, verr)
}
