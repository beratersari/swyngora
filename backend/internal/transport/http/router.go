package httpx

import (
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
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
	h = middleware.RateLimit(opts.RateLimitRPS, opts.RateLimitBurst)(h)
	h = middleware.CORSWithOrigins(opts.CORSAllowOrigins)(h)
	return h
}
