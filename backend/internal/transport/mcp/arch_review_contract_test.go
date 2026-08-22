package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/portfoliostore"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type archPx struct{}

func (archPx) GetTicker24h(_ context.Context, _, symbol string) (*domain.Ticker24h, error) {
	return &domain.Ticker24h{Symbol: symbol, LastPrice: "100"}, nil
}

func newArchPortfolio(t *testing.T) *portfolio.Service {
	t.Helper()
	st, err := portfoliostore.Open(filepath.Join(t.TempDir(), "pf.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return portfolio.New(st, archPx{}).WithPaperCosts(domain.ZeroTradingCosts)
}

func mcpToolBlock(src, name string) string {
	marker := `addTool(mcp.NewTool("` + name + `"`
	i := strings.Index(src, marker)
	if i < 0 {
		return ""
	}
	rest := src[i:]
	// Cut at the next addTool after this tool's opening, or at 6k chars.
	next := strings.Index(rest[len(marker):], `addTool(mcp.NewTool("`)
	if next < 0 || next > 8000 {
		if len(rest) > 8000 {
			return rest[:8000]
		}
		return rest
	}
	return rest[:len(marker)+next]
}

func TestArchReview_MCPToolSchemasOmitPortfolioID(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	// These tools call Backend methods that pass PortfolioIDFrom(ctx) into
	// requireAccess, which 400s when the tenant has more than one book.
	missing := []string{
		"list_portfolio_trades",
		"list_portfolio_orders",
		"place_portfolio_pending_order",
		"place_margin_order",
		"list_margin_positions",
		"list_margin_orders",
		"list_margin_trades",
		"create_recurring_buy",
		"list_recurring_buys",
		"create_portfolio_basket",
		"list_portfolio_baskets",
		"get_portfolio_risk_limits",
		"list_portfolio_cash_movements",
	}
	var failed []string
	for _, name := range missing {
		block := mcpToolBlock(text, name)
		if block == "" {
			t.Fatalf("tool %s not found in server.go", name)
		}
		if !strings.Contains(block, `WithString("portfolioId"`) {
			failed = append(failed, name)
		}
	}
	if len(failed) > 0 {
		t.Fatalf("MCP tools omit portfolioId: %s", strings.Join(failed, ", "))
	}
}

func TestArchReview_MCPListTradesFailsWithTwoBooks(t *testing.T) {
	ctx := context.Background()
	svc := newArchPortfolio(t)
	if _, err := svc.Create(ctx, portfolio.CreateInput{ClientID: "arch-mcp", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, portfolio.CreateInput{ClientID: "arch-mcp", Name: "Risky", StartingBalance: 2000}); err != nil {
		t.Fatal(err)
	}
	b := &Backend{Portfolio: svc}
	if _, err := b.ListPortfolioTrades(ctx, "arch-mcp", 50, 0); err == nil {
		t.Fatal("expected list trades without portfolioId to fail once two books exist")
	}
	main, err := svc.Get(ctx, "arch-mcp", "arch-mcp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.ListPortfolioTrades(WithPortfolioID(ctx, main.ID), "arch-mcp", 50, 0); err != nil {
		t.Fatalf("list trades with portfolioId: %v", err)
	}
}

func TestArchReview_MCPPlaceMarginFailsWithTwoBooks(t *testing.T) {
	ctx := context.Background()
	svc := newArchPortfolio(t)
	if _, err := svc.Create(ctx, portfolio.CreateInput{ClientID: "arch-mcp-m", StartingBalance: 10000}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, portfolio.CreateInput{ClientID: "arch-mcp-m", Name: "Risky", StartingBalance: 5000}); err != nil {
		t.Fatal(err)
	}
	b := &Backend{Portfolio: svc}
	if _, err := b.PlaceMarginOrder(ctx, "arch-mcp-m", "binance", "BTCUSDT", "long", "market", 1, 2, 0, nil, nil); err == nil {
		t.Fatal("expected margin open without portfolioId to fail once two books exist")
	}
	main, err := svc.Get(ctx, "arch-mcp-m", "arch-mcp-m")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.PlaceMarginOrder(WithPortfolioID(ctx, main.ID), "arch-mcp-m", "binance", "BTCUSDT", "long", "market", 1, 2, 0, nil, nil); err != nil {
		t.Fatalf("margin open with portfolioId: %v", err)
	}
}

func TestArchReview_StdioAPIClientSendsNoAuth(t *testing.T) {
	var sawAuth, sawAPIKey string
	var sawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawPath = r.URL.Path
		sawAuth = r.Header.Get("Authorization")
		sawAPIKey = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"valid API token required"}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewAPIClientWithAuth(srv.URL, 0, "secret-token")
	_, err := c.GetPortfolio(context.Background(), "tenant-a")
	if err == nil {
		t.Fatal("expected unauthorized from stub")
	}
	if sawPath != "/api/v1/portfolio" {
		t.Fatalf("path=%s", sawPath)
	}
	if sawAuth != "Bearer secret-token" {
		t.Fatalf("Authorization=%q", sawAuth)
	}
	if sawAPIKey != "secret-token" {
		t.Fatalf("X-API-Key=%q", sawAPIKey)
	}
}
