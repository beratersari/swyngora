package httpx

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/accountstore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/watchliststore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/mcp"
)

type pfTicker struct{}

func (pfTicker) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "100"}, nil
}

func newTenantPaperRouter(t *testing.T) (http.Handler, *portfolio.Service, *account.Service) {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	paper := portfolio.New(st, pfTicker{}).WithPaperCosts(domain.ZeroTradingCosts)
	acctStore := accountstore.NewMemory()
	acct := account.New(acctStore, account.DataPurgeDeps{Paper: paper})
	paper.SetAccountChecker(acct)
	keys := apikey.New(acctStore)
	watch := watchlist.New(watchliststore.NewMemory())
	mcpSrv := mcp.NewInProcessServer(evidenceMarket(), watch, nil, paper, nil, nil, nil, nil, keys, acct, nil, nil)
	h := NewRouterWithOptions(evidenceMarket(), watch, RouterOptions{
		RateLimitRPS: 0, APIAuthToken: evidenceMaster,
		Portfolio: paper, Accounts: acct, APIKeys: keys,
		MCPHandler: mcp.NewHTTPHandler(mcpSrv),
	})
	return h, paper, acct
}

func TestAccountGateHeaderBodyMismatchCannotTradeClosedTenant(t *testing.T) {
	h, paper, acct := newTenantPaperRouter(t)
	ctx := context.Background()
	const closedID = "closed-desk-1"
	const decoyID = "active-decoy-1"

	if _, err := paper.Create(ctx, portfolio.CreateInput{ClientID: closedID, StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, err := acct.Close(ctx, closedID); err != nil {
		t.Fatal(err)
	}

	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio/orders", evidenceMaster, closedID, map[string]any{
		"clientId": closedID, "symbol": "BTCUSDT", "side": "buy", "quantity": 1,
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("control same-id: want 403, got %d %s", rr.Code, rr.Body.String())
	}

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/portfolio/orders", evidenceMaster, decoyID, map[string]any{
		"clientId": closedID, "symbol": "BTCUSDT", "side": "buy", "quantity": 1,
	})
	view, err := paper.View(ctx, closedID)
	if err != nil {
		t.Fatal(err)
	}
	if rr.Code == http.StatusOK || (view != nil && view.CashBalance < 10000-1e-6) {
		t.Fatalf("closed tenant traded via decoy X-Client-Id (status=%d cash=%v body=%s)",
			rr.Code, view.CashBalance, rr.Body.String())
	}
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Fatalf("want 400 mismatch or 403 closed, got %d %s", rr.Code, rr.Body.String())
	}
}

func TestAccountGateHeaderBodyMismatchCannotMutateClosedWatchlist(t *testing.T) {
	h, _, acct := newTenantPaperRouter(t)
	ctx := context.Background()
	const closedID = "closed-wl-1"
	const decoyID = "active-decoy-wl"

	rr := evidenceDo(t, h, http.MethodPost, "/api/v1/watchlist/items", evidenceMaster, closedID, map[string]any{
		"clientId": closedID, "exchange": "binance", "symbol": "ETHUSDT",
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("seed: %d %s", rr.Code, rr.Body.String())
	}
	if _, err := acct.Close(ctx, closedID); err != nil {
		t.Fatal(err)
	}

	rr = evidenceDo(t, h, http.MethodPost, "/api/v1/watchlist/items", evidenceMaster, decoyID, map[string]any{
		"clientId": closedID, "exchange": "binance", "symbol": "BTCUSDT",
	})
	if rr.Code == http.StatusOK {
		t.Fatalf("closed watchlist mutated via decoy header: %s", rr.Body.String())
	}
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusForbidden {
		t.Fatalf("want 400 or 403, got %d %s", rr.Code, rr.Body.String())
	}
}
