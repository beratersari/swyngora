package httpx

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/fundingarb"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/realtime"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
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
	// APIAuthToken when set requires Bearer / X-API-Key on non-public routes (see middleware.APIAuth).
	APIAuthToken string
	// APIKeys enables per-user named keys (read|trade) and authenticates them.
	APIKeys *apikey.Service
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
	// FundingArb enables funding-arb follow watches when non-nil.
	FundingArb *fundingarb.Service
	// Scanner enables technical indicator scanner routes when non-nil.
	Scanner *scanner.Service
	// Swing enables swing-setup scan routes when non-nil.
	Swing *swing.Service
	// Export enables user data export routes when non-nil.
	Export *exportsvc.Service
	// Import enables user data import routes when non-nil.
	Import *dataimport.Service
	// Accounts enables account close/reopen and closed-client gate when non-nil.
	Accounts *account.Service
	// Realtime is the WebSocket hub for live prices + paper portfolio events.
	Realtime *realtime.Hub
	// StrictMasterTenant rejects a remote process-master token from acting as
	// an arbitrary clientId (loopback and user keys are unchanged). cmd/server
	// sets this unless ALLOW_MASTER_IMPERSONATE=true. Tests stay loose by default.
	StrictMasterTenant bool
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
	tickets := middleware.NewWSTicketIssuer()

	mux.Handle("GET /health", health)
	if opts.Realtime != nil {
		rh := handler.NewRealtimeHandler(opts.Realtime, opts.CORSAllowOrigins).WithTickets(tickets)
		mux.HandleFunc("GET /api/v1/realtime", rh.Info)
		mux.HandleFunc("POST /api/v1/realtime/ticket", rh.IssueTicket)
		mux.HandleFunc("GET /api/v1/ws", rh.ServeWS)
	}
	mux.HandleFunc("GET /api/v1/market/candles", mh.GetCandles)
	mux.HandleFunc("GET /api/v1/market/ticker/24h", mh.GetTicker24h)
	mux.HandleFunc("GET /api/v1/market/orderbook/history/compare", mh.CompareBookHistory)
	mux.HandleFunc("GET /api/v1/market/orderbook/history", mh.GetBookHistory)
	mux.HandleFunc("GET /api/v1/market/orderbook/icebergs", mh.GetIcebergs)
	mux.HandleFunc("GET /api/v1/market/orderbook", mh.GetOrderBook)
	mux.HandleFunc("GET /api/v1/market/orderbook/combined", mh.GetCombinedOrderBook)
	mux.HandleFunc("GET /api/v1/market/orderbook/impact", mh.GetOrderBookImpact)
	mux.HandleFunc("GET /api/v1/market/orderbook/liquidity", mh.GetMarketLiquidity)
	mux.HandleFunc("GET /api/v1/market/orderbook/heatmap", mh.GetOrderBookHeatmap)
	mux.HandleFunc("GET /api/v1/market/liquidations", mh.GetLiquidations)
	mux.HandleFunc("GET /api/v1/market/open-interest", mh.GetOpenInterest)
	mux.HandleFunc("GET /api/v1/market/funding-rate", mh.GetFundingRate)
	mux.HandleFunc("GET /api/v1/market/funding-arb/scan", mh.ScanFundingArb)
	mux.HandleFunc("GET /api/v1/market/funding-arb/history", mh.GetFundingArbHistory)
	mux.HandleFunc("GET /api/v1/market/funding-arb", mh.GetFundingArb)
	mux.HandleFunc("GET /api/v1/market/long-short-ratio", mh.GetLongShortRatio)
	mux.HandleFunc("GET /api/v1/market/futures-history", mh.GetFuturesHistory)
	mux.HandleFunc("GET /api/v1/market/liquidation-hunt/heatmap", mh.GetLiquidationHuntHeatmap)
	mux.HandleFunc("GET /api/v1/market/liquidation-hunt", mh.GetLiquidationHunt)
	mux.HandleFunc("GET /api/v1/market/squeeze-risk", mh.GetSqueezeRisk)
	mux.HandleFunc("GET /api/v1/market/positioning", mh.GetPositioning)
	mux.HandleFunc("GET /api/v1/market/venue-divergence", mh.GetVenueDivergence)
	mux.HandleFunc("GET /api/v1/market/taker-flow", mh.GetTakerFlow)
	mux.HandleFunc("GET /api/v1/market/cvd", mh.GetCVD)
	mux.HandleFunc("GET /api/v1/market/volume-profile", mh.GetVolumeProfile)
	mux.HandleFunc("GET /api/v1/market/vwap", mh.GetVWAP)
	mux.HandleFunc("GET /api/v1/market/around/compare", mh.GetAroundCompare)
	mux.HandleFunc("GET /api/v1/market/around/precursors", mh.GetAroundPrecursors)
	mux.HandleFunc("GET /api/v1/market/around/similar", mh.GetAroundSimilar)
	mux.HandleFunc("GET /api/v1/market/around/moves", mh.GetAroundMoves)
	mux.HandleFunc("GET /api/v1/market/around", mh.GetAround)
	mux.HandleFunc("GET /api/v1/market/absorption", mh.GetAbsorption)
	mux.HandleFunc("GET /api/v1/market/liquidity-sweeps", mh.GetLiquiditySweeps)
	mux.HandleFunc("GET /api/v1/market/basis", mh.GetBasis)
	mux.HandleFunc("GET /api/v1/market/correlation", mh.GetCorrelation)
	mux.HandleFunc("GET /api/v1/market/breadth", mh.GetBreadth)
	mux.HandleFunc("GET /api/v1/market/volatility", mh.GetVolatility)
	mux.HandleFunc("GET /api/v1/market/snapshot", mh.GetSnapshot)
	mux.HandleFunc("GET /api/v1/market/levels", mh.GetLevels)
	mux.HandleFunc("GET /api/v1/market/whales", mh.GetWhales)
	mux.HandleFunc("GET /api/v1/market/supply", mh.GetSupply)
	mux.HandleFunc("GET /api/v1/market/holders", mh.GetHolders)
	mux.HandleFunc("GET /api/v1/market/asset-profile", mh.GetAssetProfile)
	mux.HandleFunc("GET /api/v1/market/exchanges", mh.ListExchanges)
	mux.HandleFunc("GET /api/v1/market/fx", mh.GetFxRates)
	mux.HandleFunc("GET /api/v1/market/intervals", mh.GetIntervals)
	mux.HandleFunc("GET /api/v1/market/tags", mh.ListProductTags)
	mux.HandleFunc("GET /api/v1/market/spot", mh.ListSpotMarkets)
	mux.HandleFunc("GET /api/v1/market/delist-schedule", mh.ListDelistSchedule)
	mux.HandleFunc("GET /api/v1/market/post-delist", mh.GetPostDelist)
	mux.HandleFunc("GET /api/v1/market/indicators", mh.GetIndicators)
	mux.HandleFunc("POST /api/v1/market/indicators/batch", mh.PostIndicatorsBatch)
	mux.HandleFunc("GET /api/v1/market/rsi-heatmap", mh.GetRSIHeatmap)
	mux.HandleFunc("GET /api/v1/market/pumps", mh.GetPumpEvents)
	mux.HandleFunc("GET /api/v1/market/pumps/scan", mh.ScanPumpEvents)
	mux.HandleFunc("GET /api/v1/market/volume-surge/scan", mh.ScanVolumeSurges)
	mux.HandleFunc("GET /api/v1/market/volume-surge", mh.GetVolumeSurge)
	if opts.Swing != nil {
		sh := handler.NewSwingHandler(opts.Swing)
		mux.HandleFunc("GET /api/v1/market/swing", sh.Analyze)
	}

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
		mux.HandleFunc("GET /api/v1/portfolios", ph.List)
		mux.HandleFunc("GET /api/v1/portfolios/shared", ph.ListSharedPortfolios)
		mux.HandleFunc("PATCH /api/v1/portfolios/{id}", ph.Rename)
		mux.HandleFunc("DELETE /api/v1/portfolios/{id}", ph.DeleteBook)
		mux.HandleFunc("POST /api/v1/portfolio/shares", ph.SharePortfolio)
		mux.HandleFunc("GET /api/v1/portfolio/shares", ph.ListPortfolioShares)
		mux.HandleFunc("PATCH /api/v1/portfolio/shares", ph.UpdatePortfolioShare)
		mux.HandleFunc("DELETE /api/v1/portfolio/shares", ph.RevokePortfolioShare)
		mux.HandleFunc("GET /api/v1/portfolio/performance", ph.GetPerformance)
		mux.HandleFunc("POST /api/v1/portfolio/deposits", ph.Deposit)
		mux.HandleFunc("POST /api/v1/portfolio/withdrawals", ph.Withdraw)
		mux.HandleFunc("POST /api/v1/portfolio/transfers", ph.Transfer)
		mux.HandleFunc("GET /api/v1/portfolio/cash-movements", ph.ListCashMovements)
		mux.HandleFunc("GET /api/v1/portfolio/trading-costs", ph.GetTradingCosts)
		mux.HandleFunc("GET /api/v1/portfolio/risk-limits", ph.GetRiskLimits)
		mux.HandleFunc("PUT /api/v1/portfolio/risk-limits", ph.PutRiskLimits)
		mux.HandleFunc("DELETE /api/v1/portfolio/risk-limits", ph.DeleteRiskLimits)
		mux.HandleFunc("POST /api/v1/portfolio/orders", ph.PlaceOrder)
		mux.HandleFunc("GET /api/v1/portfolio/orders", ph.ListOrders)
		mux.HandleFunc("POST /api/v1/portfolio/orders/cancel-all", ph.CancelAllOrders)
		mux.HandleFunc("GET /api/v1/portfolio/orders/{id}", ph.GetOrder)
		mux.HandleFunc("PATCH /api/v1/portfolio/orders/{id}", ph.AmendOrder)
		mux.HandleFunc("DELETE /api/v1/portfolio/orders/{id}", ph.CancelOrder)
		mux.HandleFunc("GET /api/v1/portfolio/trades", ph.ListTrades)
		mux.HandleFunc("GET /api/v1/portfolio/lots", ph.ListLots)
		// Recurring buys: static subpaths before {id}
		mux.HandleFunc("POST /api/v1/portfolio/recurring-buys", ph.CreateRecurringBuy)
		mux.HandleFunc("GET /api/v1/portfolio/recurring-buys", ph.ListRecurringBuys)
		mux.HandleFunc("POST /api/v1/portfolio/recurring-buys/{id}/pause", ph.PauseRecurringBuy)
		mux.HandleFunc("POST /api/v1/portfolio/recurring-buys/{id}/resume", ph.ResumeRecurringBuy)
		mux.HandleFunc("GET /api/v1/portfolio/recurring-buys/{id}/runs", ph.ListRecurringBuyRuns)
		mux.HandleFunc("GET /api/v1/portfolio/recurring-buys/{id}", ph.GetRecurringBuy)
		mux.HandleFunc("PATCH /api/v1/portfolio/recurring-buys/{id}", ph.UpdateRecurringBuy)
		mux.HandleFunc("DELETE /api/v1/portfolio/recurring-buys/{id}", ph.DeleteRecurringBuy)
		// Allocation baskets (manual rebalance only)
		mux.HandleFunc("POST /api/v1/portfolio/baskets", ph.CreateBasket)
		mux.HandleFunc("GET /api/v1/portfolio/baskets", ph.ListBaskets)
		mux.HandleFunc("GET /api/v1/portfolio/baskets/{id}/preview", ph.PreviewBasketRebalance)
		mux.HandleFunc("POST /api/v1/portfolio/baskets/{id}/rebalance", ph.RebalanceBasket)
		mux.HandleFunc("GET /api/v1/portfolio/baskets/{id}", ph.GetBasket)
		mux.HandleFunc("PATCH /api/v1/portfolio/baskets/{id}", ph.UpdateBasket)
		mux.HandleFunc("DELETE /api/v1/portfolio/baskets/{id}", ph.DeleteBasket)
		// Margin (isolated / cross)
		mux.HandleFunc("PUT /api/v1/portfolio/margin/mode", ph.SetMarginMode)
		mux.HandleFunc("POST /api/v1/portfolio/margin/orders", ph.PlaceMarginOrder)
		mux.HandleFunc("GET /api/v1/portfolio/margin/orders", ph.ListMarginOrders)
		mux.HandleFunc("DELETE /api/v1/portfolio/margin/orders/{id}", ph.CancelMarginOrder)
		mux.HandleFunc("GET /api/v1/portfolio/margin/positions", ph.ListMarginPositions)
		mux.HandleFunc("POST /api/v1/portfolio/margin/positions/{id}/close", ph.CloseMarginPosition)
		mux.HandleFunc("POST /api/v1/portfolio/margin/positions/{id}/margin", ph.AdjustMargin)
		mux.HandleFunc("POST /api/v1/portfolio/margin/positions/{id}/repay", ph.RepayMarginDebt)
		mux.HandleFunc("PUT /api/v1/portfolio/margin/positions/{id}/brackets", ph.SetMarginBrackets)
		mux.HandleFunc("GET /api/v1/portfolio/margin/positions/{id}", ph.GetMarginPosition)
		mux.HandleFunc("GET /api/v1/portfolio/margin/trades", ph.ListMarginTrades)
	}

	if opts.PriceDiff != nil {
		pdh := handler.NewPriceDiffHandler(opts.PriceDiff)
		mux.HandleFunc("POST /api/v1/price-diff/watches", pdh.CreateWatch)
		mux.HandleFunc("GET /api/v1/price-diff/watches", pdh.ListWatches)
		mux.HandleFunc("GET /api/v1/price-diff/watches/{id}/quote", pdh.QuoteWatch)
		mux.HandleFunc("POST /api/v1/price-diff/watches/{id}/pause", pdh.PauseWatch)
		mux.HandleFunc("POST /api/v1/price-diff/watches/{id}/resume", pdh.ResumeWatch)
		mux.HandleFunc("GET /api/v1/price-diff/watches/{id}", pdh.GetWatch)
		mux.HandleFunc("PATCH /api/v1/price-diff/watches/{id}", pdh.UpdateWatch)
		mux.HandleFunc("DELETE /api/v1/price-diff/watches/{id}", pdh.DeleteWatch)
		mux.HandleFunc("GET /api/v1/price-diff/quote/scan", pdh.QuoteScan)
		mux.HandleFunc("GET /api/v1/price-diff/quote", pdh.QuoteRoute)
		mux.HandleFunc("GET /api/v1/price-diff/opportunities", pdh.ListOpportunities)
		mux.HandleFunc("GET /api/v1/price-diff/opportunities/{id}/quote", pdh.QuoteOpportunity)
		mux.HandleFunc("GET /api/v1/price-diff/opportunities/{id}", pdh.GetOpportunity)
	}

	if opts.FundingArb != nil {
		fah := handler.NewFundingArbWatchHandler(opts.FundingArb)
		mux.HandleFunc("POST /api/v1/funding-arb/watches", fah.CreateWatch)
		mux.HandleFunc("GET /api/v1/funding-arb/watches", fah.ListWatches)
		mux.HandleFunc("PATCH /api/v1/funding-arb/watches/{id}", fah.UpdateWatch)
		mux.HandleFunc("POST /api/v1/funding-arb/watches/{id}/pause", fah.PauseWatch)
		mux.HandleFunc("POST /api/v1/funding-arb/watches/{id}/resume", fah.ResumeWatch)
		mux.HandleFunc("GET /api/v1/funding-arb/watches/{id}", fah.GetWatch)
		mux.HandleFunc("DELETE /api/v1/funding-arb/watches/{id}", fah.DeleteWatch)
		mux.HandleFunc("GET /api/v1/funding-arb/signals", fah.ListSignals)
	}

	if opts.Swing != nil {
		swh := handler.NewSwingHandler(opts.Swing)
		mux.HandleFunc("GET /api/v1/swing/setups", swh.ListSetups)
	}

	if opts.Scanner != nil {
		sh := handler.NewScannerHandler(opts.Scanner)
		mux.HandleFunc("POST /api/v1/scanner/rules", sh.Create)
		mux.HandleFunc("GET /api/v1/scanner/rules", sh.ListRules)
		mux.HandleFunc("GET /api/v1/scanner/rules/{id}", sh.GetRule)
		mux.HandleFunc("PATCH /api/v1/scanner/rules/{id}", sh.Update)
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

	if opts.APIKeys != nil {
		kh := handler.NewAPIKeyHandler(opts.APIKeys)
		mux.HandleFunc("POST /api/v1/account/api-keys", kh.Create)
		mux.HandleFunc("GET /api/v1/account/api-keys", kh.List)
		mux.HandleFunc("DELETE /api/v1/account/api-keys/{id}", kh.Revoke)
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
		mux.HandleFunc("POST /api/v1/ai/chat/stream", ah.ChatStream)
	}

	var h http.Handler = mux
	h = middleware.AccountGate(opts.Accounts)(h)
	h = middleware.APIKeyScope(h)
	h = middleware.MasterTenant(opts.StrictMasterTenant, opts.APIKeys)(h)
	// Auth wraps the mux so /mcp and tenant APIs are protected when a token is configured.
	h = middleware.APIAuthWithOptions(middleware.APIAuthOptions{
		Master:                opts.APIAuthToken,
		Keys:                  opts.APIKeys,
		Tickets:               tickets,
		DenyMasterImpersonate: opts.StrictMasterTenant,
	})(h)
	h = middleware.RateLimit(opts.RateLimitRPS, opts.RateLimitBurst)(h)
	h = middleware.CORSWithOrigins(opts.CORSAllowOrigins)(h)
	return h
}
