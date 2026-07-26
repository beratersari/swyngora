package httpx

import (
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
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

	if watchSvc != nil {
		wh := handler.NewWatchlistHandler(watchSvc)
		mux.HandleFunc("GET /api/v1/watchlist", wh.Get)
		mux.HandleFunc("PUT /api/v1/watchlist", wh.Replace)
		mux.HandleFunc("POST /api/v1/watchlist/items", wh.Add)
		mux.HandleFunc("DELETE /api/v1/watchlist/items", wh.Remove)
	}

	var h http.Handler = mux
	h = middleware.RateLimit(opts.RateLimitRPS, opts.RateLimitBurst)(h)
	h = middleware.CORSWithOrigins(opts.CORSAllowOrigins)(h)
	return h
}
