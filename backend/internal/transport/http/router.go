package httpx

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/handler"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

// RouterOptions configures transport middleware.
type RouterOptions struct {
	// RateLimitRPS is tokens/sec per IP; 0 disables.
	RateLimitRPS   float64
	RateLimitBurst int
	// CORSAllowOrigins: empty or ["*"] = any origin; otherwise exact match list.
	CORSAllowOrigins []string
	// MCPHandler mounts streamable MCP (typically at /mcp) in the same process.
	MCPHandler http.Handler
	// AI client for POST /api/v1/ai/chat (optional).
	AI        *aiagent.Client
	AITimeout time.Duration
	// Alerts enables price-alert routes when non-nil.
	Alerts *pricealert.Service
	// Portfolio enables paper-trading routes when non-nil.
	Portfolio *portfolio.Service
	// PriceDiff enables cross-exchange price difference tracking when non-nil.
	PriceDiff *pricediff.Service
	// Scanner enables technical indicator scanner routes when non-nil.
	Scanner *scanner.Service
	// Export enables user data export routes when non-nil.
	Export *exportsvc.Service
	// Import enables user data import routes when non-nil.
	Import *dataimport.Service
	// Accounts enables account close/reopen and closed-client gate when non-nil.
	Accounts *account.Service
}

// NewRouter wires HTTP routes for the API with default rate limits.
func NewRouter(marketSvc *market.Service) http.Handler {
	return NewRouterWithOptions(marketSvc, nil, RouterOptions{
		RateLimitRPS:   20,
		RateLimitBurst: 40,
	})
}

// NewRouterWithOptions wires HTTP routes with explicit middleware options.
func NewRouterWithOptions(marketSvc *market.Service, watchSvc *watchlist.Service, opts RouterOptions) http.Handler {
	mux := http.NewServeMux()

	health := handler.NewHealthHandler()
	mh := handler.NewMarketHandler(marketSvc)

	mux.Handle("GET /health", health)
	mux.HandleFunc("GET /api/v1/market/candles", mh.GetCandles)
	mux.HandleFunc("GET /api/v1/market/ticker/24h", mh.GetTicker24h)
	mux.HandleFunc("GET /api/v1/market/supply", mh.GetSupply)
	mux.HandleFunc("GET /api/v1/market/exchanges", mh.ListExchanges)
	mux.HandleFunc("GET /api/v1/market/intervals", mh.GetIntervals)
	mux.HandleFunc("GET /api/v1/market/tags", mh.ListProductTags)
	mux.HandleFunc("GET /api/v1/market/spot", mh.ListSpotMarkets)
	mux.HandleFunc("GET /api/v1/market/indicators", mh.GetIndicators)
	mux.HandleFunc("POST /api/v1/market/indicators/batch", mh.PostIndicatorsBatch)
	mux.HandleFunc("GET /api/v1/market/pumps", mh.GetPumpEvents)
	mux.HandleFunc("GET /api/v1/market/pumps/scan", mh.ScanPumpEvents)

	if watchSvc != nil {
		wh := handler.NewWatchlistHandler(watchSvc)
		mux.HandleFunc("GET /api/v1/watchlist", wh.Get)
		mux.HandleFunc("PUT /api/v1/watchlist", wh.Replace)
		mux.HandleFunc("POST /api/v1/watchlist/items", wh.Add)
		mux.HandleFunc("DELETE /api/v1/watchlist/items", wh.Remove)
		// Sharing + audit (static paths before any future {id} routes)
		mux.HandleFunc("GET /api/v1/watchlist/shares", wh.ListShares)
		mux.HandleFunc("POST /api/v1/watchlist/shares", wh.Share)
		mux.HandleFunc("PATCH /api/v1/watchlist/shares", wh.UpdateShare)
		mux.HandleFunc("DELETE /api/v1/watchlist/shares", wh.RevokeShare)
		mux.HandleFunc("GET /api/v1/watchlist/shared", wh.ListSharedWithMe)
		mux.HandleFunc("GET /api/v1/watchlist/audit", wh.ListAudit)
	}

	if opts.Alerts != nil {
		ah := handler.NewAlertHandler(opts.Alerts)
		// Static paths before /{id} so "webhook" is not captured as an id.
		mux.HandleFunc("GET /api/v1/alerts/webhook", ah.GetWebhook)
		mux.HandleFunc("PUT /api/v1/alerts/webhook", ah.PutWebhook)
		mux.HandleFunc("DELETE /api/v1/alerts/webhook", ah.DeleteWebhook)
		mux.HandleFunc("GET /api/v1/alerts", ah.List)
		mux.HandleFunc("POST /api/v1/alerts", ah.Create)
		mux.HandleFunc("GET /api/v1/alerts/{id}", ah.Get)
		mux.HandleFunc("DELETE /api/v1/alerts/{id}", ah.Delete)
	}

	if opts.Portfolio != nil {
		ph := handler.NewPortfolioHandler(opts.Portfolio)
		mux.HandleFunc("POST /api/v1/portfolio", ph.Create)
		mux.HandleFunc("GET /api/v1/portfolio", ph.Get)
		mux.HandleFunc("POST /api/v1/portfolio/orders", ph.PlaceOrder)
		mux.HandleFunc("GET /api/v1/portfolio/orders", ph.ListOrders)
		mux.HandleFunc("DELETE /api/v1/portfolio/orders/{id}", ph.CancelOrder)
		mux.HandleFunc("GET /api/v1/portfolio/trades", ph.ListTrades)
		// Recurring buys: static subpaths before {id}
		mux.HandleFunc("POST /api/v1/portfolio/recurring-buys", ph.CreateRecurringBuy)
		mux.HandleFunc("GET /api/v1/portfolio/recurring-buys", ph.ListRecurringBuys)
		mux.HandleFunc("POST /api/v1/portfolio/recurring-buys/{id}/pause", ph.PauseRecurringBuy)
		mux.HandleFunc("POST /api/v1/portfolio/recurring-buys/{id}/resume", ph.ResumeRecurringBuy)
		mux.HandleFunc("GET /api/v1/portfolio/recurring-buys/{id}/runs", ph.ListRecurringBuyRuns)
		mux.HandleFunc("GET /api/v1/portfolio/recurring-buys/{id}", ph.GetRecurringBuy)
		mux.HandleFunc("DELETE /api/v1/portfolio/recurring-buys/{id}", ph.DeleteRecurringBuy)
	}

	if opts.PriceDiff != nil {
		pdh := handler.NewPriceDiffHandler(opts.PriceDiff)
		mux.HandleFunc("POST /api/v1/price-diff/watches", pdh.CreateWatch)
		mux.HandleFunc("GET /api/v1/price-diff/watches", pdh.ListWatches)
		mux.HandleFunc("GET /api/v1/price-diff/watches/{id}", pdh.GetWatch)
		mux.HandleFunc("DELETE /api/v1/price-diff/watches/{id}", pdh.DeleteWatch)
		mux.HandleFunc("GET /api/v1/price-diff/opportunities", pdh.ListOpportunities)
		mux.HandleFunc("GET /api/v1/price-diff/opportunities/{id}", pdh.GetOpportunity)
	}

	if opts.Scanner != nil {
		sh := handler.NewScannerHandler(opts.Scanner)
		mux.HandleFunc("POST /api/v1/scanner/rules", sh.Create)
		mux.HandleFunc("GET /api/v1/scanner/rules", sh.ListRules)
		mux.HandleFunc("GET /api/v1/scanner/rules/{id}", sh.GetRule)
		mux.HandleFunc("DELETE /api/v1/scanner/rules/{id}", sh.DeleteRule)
		mux.HandleFunc("GET /api/v1/scanner/results", sh.ListResults)
		// Backtests: static subpaths before {id} where needed
		mux.HandleFunc("POST /api/v1/scanner/backtests", sh.StartBacktest)
		mux.HandleFunc("GET /api/v1/scanner/backtests", sh.ListBacktests)
		mux.HandleFunc("POST /api/v1/scanner/backtests/{id}/cancel", sh.CancelBacktest)
		mux.HandleFunc("GET /api/v1/scanner/backtests/{id}/signals", sh.ListBacktestSignals)
		mux.HandleFunc("GET /api/v1/scanner/backtests/{id}", sh.GetBacktest)
	}

	if opts.Export != nil {
		eh := handler.NewExportHandler(opts.Export)
		mux.HandleFunc("POST /api/v1/export", eh.Start)
		mux.HandleFunc("GET /api/v1/export", eh.List)
		// Static subpaths before bare {id}
		mux.HandleFunc("POST /api/v1/export/{id}/cancel", eh.Cancel)
		mux.HandleFunc("GET /api/v1/export/{id}/download", eh.Download)
		mux.HandleFunc("GET /api/v1/export/{id}", eh.Get)
	}

	if opts.Import != nil {
		ih := handler.NewImportHandler(opts.Import)
		mux.HandleFunc("POST /api/v1/import/preview", ih.Preview)
		mux.HandleFunc("GET /api/v1/import", ih.List)
		mux.HandleFunc("POST /api/v1/import/{id}/confirm", ih.Confirm)
		mux.HandleFunc("POST /api/v1/import/{id}/cancel", ih.Cancel)
		mux.HandleFunc("GET /api/v1/import/{id}", ih.Get)
	}

	if opts.Accounts != nil {
		ah := handler.NewAccountHandler(opts.Accounts)
		mux.HandleFunc("GET /api/v1/account", ah.Status)
		mux.HandleFunc("POST /api/v1/account/close", ah.Close)
		mux.HandleFunc("POST /api/v1/account/reopen", ah.Reopen)
	}

	// MCP streamable HTTP — same process as REST API (no second server).
	if opts.MCPHandler != nil {
		mux.Handle("/mcp", opts.MCPHandler)
		mux.Handle("/mcp/", opts.MCPHandler)
	}

	// AI chat proxy → Python multi-agent service (may be auto-started by cmd/server).
	if opts.AI != nil {
		ah := handler.NewAIHandler(opts.AI, opts.AITimeout)
		mux.HandleFunc("POST /api/v1/ai/chat", ah.Chat)
	}

	var h http.Handler = mux
	h = middleware.AccountGate(opts.Accounts)(h)
	h = middleware.RateLimit(opts.RateLimitRPS, opts.RateLimitBurst)(h)
	h = middleware.CORSWithOrigins(opts.CORSAllowOrigins)(h)
	return h
}
