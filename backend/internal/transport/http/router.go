package httpx

import (
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/handler"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

// RouterOptions configures transport middleware.
type RouterOptions struct {
	// RateLimitRPS is tokens/sec per IP; 0 disables.
	RateLimitRPS   float64
	RateLimitBurst int
}

// NewRouter wires HTTP routes for the API with default rate limits.
func NewRouter(marketSvc *market.Service) http.Handler {
	return NewRouterWithOptions(marketSvc, RouterOptions{
		RateLimitRPS:   20,
		RateLimitBurst: 40,
	})
}

// NewRouterWithOptions wires HTTP routes with explicit middleware options.
func NewRouterWithOptions(marketSvc *market.Service, opts RouterOptions) http.Handler {
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

	var h http.Handler = mux
	h = middleware.RateLimit(opts.RateLimitRPS, opts.RateLimitBurst)(h)
	h = middleware.CORS(h)
	return h
}
