package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// DataPort is the data surface MCP tools need (in-process backend or HTTP client).
type DataPort interface {
	GetTicker(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetCandles(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error)
	GetSupply(ctx context.Context, asset string) (json.RawMessage, error)
	ListSpot(ctx context.Context, exchange, query, quote, sort, order, tag string, limit, offset int) (json.RawMessage, error)
	GetIndicators(ctx context.Context, exchange, symbol, interval string, limit, rsiPeriod int, emaPeriods string) (json.RawMessage, error)
	DetectPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error)
	ScanPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error)
	ListExchanges(ctx context.Context) (json.RawMessage, error)
	GetWatchlist(ctx context.Context, clientID string) (json.RawMessage, error)
	AddWatchlistItem(ctx context.Context, clientID, exchange, symbol, note string) (json.RawMessage, error)
	RemoveWatchlistItem(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error)
	ListPriceAlerts(ctx context.Context, clientID string) (json.RawMessage, error)
	CreatePriceAlert(ctx context.Context, clientID, exchange, symbol, condition string, targetPrice float64, mode string) (json.RawMessage, error)
	DeletePriceAlert(ctx context.Context, clientID, id string) (json.RawMessage, error)
	GetAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error)
	SetAlertWebhook(ctx context.Context, clientID, url string) (json.RawMessage, error)
	DeleteAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error)
	Health(ctx context.Context) (json.RawMessage, error)
}

// ServerOptions configures the MCP server.
type ServerOptions struct {
	// Data is preferred (in-process). If nil, APIBaseURL is used via HTTP client.
	Data       DataPort
	APIBaseURL string
	Name       string
	Version    string
}

// NewServer builds an MCP tool server.
func NewServer(opts ServerOptions) *server.MCPServer {
	if opts.Name == "" {
		opts.Name = "swyngora-mcp"
	}
	if opts.Version == "" {
		opts.Version = "0.1.0"
	}
	var data DataPort = opts.Data
	if data == nil {
		data = NewAPIClient(opts.APIBaseURL, 0)
	}

	s := server.NewMCPServer(
		opts.Name,
		opts.Version,
		server.WithToolCapabilities(true),
	)
	registerTools(s, data)
	return s
}

// NewInProcessServer wires MCP tools to market/watchlist/alert services (same process as HTTP).
func NewInProcessServer(marketSvc *market.Service, watchSvc *watchlist.Service, alertSvc *pricealert.Service) *server.MCPServer {
	return NewServer(ServerOptions{
		Data: &Backend{Market: marketSvc, Watch: watchSvc, Alerts: alertSvc},
		Name: "swyngora-mcp",
	})
}

// NewHTTPHandler mounts streamable MCP on the shared HTTP process (default path /mcp).
func NewHTTPHandler(mcpServer *server.MCPServer) http.Handler {
	return server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
	)
}

func registerTools(s *server.MCPServer, api DataPort) {
	s.AddTool(mcp.NewTool("get_ticker",
		mcp.WithDescription("Get 24h price, volume, and change for a trading pair on an exchange (binance|coinbase|bybit). Use for live quotes."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("Venue id: binance (default), coinbase, bybit")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetTicker(ctx, req.GetString("exchange", "binance"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_candles",
		mcp.WithDescription("Fetch OHLCV candlesticks for a symbol. Chronological oldest-first."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("interval", mcp.Description("Candle interval e.g. 1h, 15m, 1d (default 1h)")),
		mcp.WithNumber("limit", mcp.Description("Number of bars 1–1000 (default 50)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetCandles(ctx, req.GetString("exchange", "binance"), symbol, req.GetString("interval", "1h"), req.GetInt("limit", 50))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_supply",
		mcp.WithDescription("Get circulating/total/max supply and snapshot USD price for a base asset (e.g. BTC, ETH). Cache-only daily Binance snapshot."),
		mcp.WithString("asset", mcp.Required(), mcp.Description("Base asset ticker e.g. BTC")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		asset, err := req.RequireString("asset")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetSupply(ctx, asset)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_spot_markets",
		mcp.WithDescription("List/search/sort spot markets on an exchange. Supports quote filter, product tags, mcap sorts, pagination."),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("q", mcp.Description("Search substring on symbol/base/quote")),
		mcp.WithString("quote", mcp.Description("Quote asset filter e.g. USDT or USD")),
		mcp.WithString("sort", mcp.Description("Sort field e.g. quoteVolume, marketCapCirculating, lastPrice")),
		mcp.WithString("order", mcp.Description("asc or desc")),
		mcp.WithString("tag", mcp.Description("Product tag filter e.g. Meme, defi")),
		mcp.WithNumber("limit", mcp.Description("Page size default 20 max 500")),
		mcp.WithNumber("offset", mcp.Description("Pagination offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ListSpot(ctx,
			req.GetString("exchange", "binance"),
			req.GetString("q", ""),
			req.GetString("quote", ""),
			req.GetString("sort", "quoteVolume"),
			req.GetString("order", "desc"),
			req.GetString("tag", ""),
			req.GetInt("limit", 20),
			req.GetInt("offset", 0),
		)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_indicators",
		mcp.WithDescription("Compute RSI (Wilder) and EMA series for a symbol. Informational only — not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("interval", mcp.Description("Candle interval default 1h")),
		mcp.WithNumber("limit", mcp.Description("Output bars default 30")),
		mcp.WithNumber("rsiPeriod", mcp.Description("RSI period default 14")),
		mcp.WithString("emaPeriods", mcp.Description("Comma-separated EMA periods e.g. 12,26")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetIndicators(ctx,
			req.GetString("exchange", "binance"),
			symbol,
			req.GetString("interval", "1h"),
			req.GetInt("limit", 30),
			req.GetInt("rsiPeriod", 14),
			req.GetString("emaPeriods", "12,26"),
		)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("detect_pump_events",
		mcp.WithDescription(
			"Detect pump/dump events on one symbol from OHLCV candles. "+
				"Configurable minReturnPct, windowBars, candle interval, lookbackHours or start/end time, "+
				"mode (close_return|candle_body|high_from_low), direction (up|down|both), minVolumeRatio. "+
				"Informational mechanical filter — not a trade signal.",
		),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or JUVUSDT")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit (default binance)")),
		mcp.WithString("interval", mcp.Description("Candle interval e.g. 1m,5m,15m,1h,4h (default 1h)")),
		mcp.WithNumber("lookbackHours", mcp.Description("Hours of history to analyze (derives bar count; e.g. 24 or 48)")),
		mcp.WithNumber("limit", mcp.Description("Explicit candle bar count if lookbackHours omitted (default 100, max 1000)")),
		mcp.WithNumber("minReturnPct", mcp.Description("Threshold percent e.g. 5 = +5% (default 5)")),
		mcp.WithNumber("windowBars", mcp.Description("Bars for close_return window (default 1)")),
		mcp.WithString("mode", mcp.Description("close_return|candle_body|high_from_low (default close_return)")),
		mcp.WithString("direction", mcp.Description("up|down|both (default up)")),
		mcp.WithNumber("minVolumeRatio", mcp.Description("Require volume ≥ N× series median (0=off)")),
		mcp.WithNumber("maxEvents", mcp.Description("Max events returned (default 20)")),
		mcp.WithString("startTime", mcp.Description("Optional RFC3339 range start")),
		mcp.WithString("endTime", mcp.Description("Optional RFC3339 range end")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args := map[string]any{
			"symbol":         symbol,
			"exchange":       req.GetString("exchange", "binance"),
			"interval":       req.GetString("interval", "1h"),
			"lookbackHours":  req.GetFloat("lookbackHours", 0),
			"limit":          req.GetInt("limit", 0),
			"minReturnPct":   req.GetFloat("minReturnPct", 5),
			"windowBars":     req.GetInt("windowBars", 1),
			"mode":           req.GetString("mode", "close_return"),
			"direction":      req.GetString("direction", "up"),
			"minVolumeRatio": req.GetFloat("minVolumeRatio", 0),
			"maxEvents":      req.GetInt("maxEvents", 20),
			"startTime":      req.GetString("startTime", ""),
			"endTime":        req.GetString("endTime", ""),
		}
		raw, err := api.DetectPumpEvents(ctx, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("scan_pump_events",
		mcp.WithDescription(
			"Scan top quote-volume symbols on an exchange for recent pump/dump events. "+
				"Same thresholds as detect_pump_events (minReturnPct, interval, lookbackHours, mode, direction, minVolumeRatio). "+
				"Use for 'what pumped in the last N hours'. Informational only.",
		),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("quote", mcp.Description("Quote filter e.g. USDT or USD")),
		mcp.WithString("interval", mcp.Description("Candle interval default 15m")),
		mcp.WithNumber("lookbackHours", mcp.Description("Hours to scan (default 24)")),
		mcp.WithNumber("minReturnPct", mcp.Description("Threshold percent (default 8)")),
		mcp.WithNumber("windowBars", mcp.Description("close_return window bars (default 1)")),
		mcp.WithString("mode", mcp.Description("close_return|candle_body|high_from_low")),
		mcp.WithString("direction", mcp.Description("up|down|both")),
		mcp.WithNumber("minVolumeRatio", mcp.Description("Volume vs median filter (0=off)")),
		mcp.WithNumber("symbolLimit", mcp.Description("How many top-volume symbols to scan (default 15, max 40)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := map[string]any{
			"exchange":       req.GetString("exchange", "binance"),
			"quote":          req.GetString("quote", "USDT"),
			"interval":       req.GetString("interval", "15m"),
			"lookbackHours":  req.GetFloat("lookbackHours", 24),
			"minReturnPct":   req.GetFloat("minReturnPct", 8),
			"windowBars":     req.GetInt("windowBars", 1),
			"mode":           req.GetString("mode", "close_return"),
			"direction":      req.GetString("direction", "up"),
			"minVolumeRatio": req.GetFloat("minVolumeRatio", 0),
			"symbolLimit":    req.GetInt("symbolLimit", 15),
		}
		raw, err := api.ScanPumpEvents(ctx, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_exchanges",
		mcp.WithDescription("List configured market venues and the default exchange."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ListExchanges(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_watchlist",
		mcp.WithDescription("Get a user's watchlist by clientId (required non-empty opaque id; not 'default')."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id e.g. web-abc123 or tg-42")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetWatchlist(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("add_watchlist_item",
		mcp.WithDescription("Add or update a symbol on a watchlist."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("note", mcp.Description("Optional note")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.AddWatchlistItem(ctx, clientID, req.GetString("exchange", "binance"), symbol, req.GetString("note", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("remove_watchlist_item",
		mcp.WithDescription("Remove a symbol from a watchlist."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RemoveWatchlistItem(ctx, clientID, req.GetString("exchange", "binance"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_price_alerts",
		mcp.WithDescription("List price alerts for a clientId (active and triggered)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPriceAlerts(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("create_price_alert",
		mcp.WithDescription("Create a price alert (above/below). mode=one_time (default) fires once; mode=repeating re-fires on each re-cross after returning to the safe side."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol e.g. BTCUSDT")),
		mcp.WithString("condition", mcp.Required(), mcp.Description("above | below")),
		mcp.WithNumber("targetPrice", mcp.Required(), mcp.Description("Threshold price")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit (default binance)")),
		mcp.WithString("mode", mcp.Description("one_time | repeating (default one_time)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		condition, err := req.RequireString("condition")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		target, err := req.RequireFloat("targetPrice")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreatePriceAlert(ctx, clientID, req.GetString("exchange", "binance"), symbol, condition, target, req.GetString("mode", "one_time"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("delete_price_alert",
		mcp.WithDescription("Delete a price alert by id for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Alert id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeletePriceAlert(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_alert_webhook",
		mcp.WithDescription("Get the client's price-alert webhook URL (empty if not set)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetAlertWebhook(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("set_alert_webhook",
		mcp.WithDescription("Set absolute http(s) webhook URL for price-alert notifications (durable outbox, at most once per alert)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("url", mcp.Required(), mcp.Description("https://hooks.example.com/...")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		u, err := req.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.SetAlertWebhook(ctx, clientID, u)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("delete_alert_webhook",
		mcp.WithDescription("Clear the client's price-alert webhook URL."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeleteAlertWebhook(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("health",
		mcp.WithDescription("Check Swyngora MCP connectivity (in-process when embedded in API server)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.Health(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("unhealthy: %v", err)), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})
}
