package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
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
	GetWatchlistOwned(ctx context.Context, actorClientID, ownerClientID string) (json.RawMessage, error)
	AddWatchlistItem(ctx context.Context, clientID, exchange, symbol, note string) (json.RawMessage, error)
	AddWatchlistItemOwned(ctx context.Context, actorClientID, ownerClientID, exchange, symbol, note string) (json.RawMessage, error)
	RemoveWatchlistItem(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error)
	RemoveWatchlistItemOwned(ctx context.Context, actorClientID, ownerClientID, exchange, symbol string) (json.RawMessage, error)
	ShareWatchlist(ctx context.Context, ownerClientID, granteeClientID, role string) (json.RawMessage, error)
	UpdateWatchlistShare(ctx context.Context, ownerClientID, granteeClientID, role string) (json.RawMessage, error)
	RevokeWatchlistShare(ctx context.Context, ownerClientID, granteeClientID string) (json.RawMessage, error)
	ListWatchlistShares(ctx context.Context, ownerClientID string) (json.RawMessage, error)
	ListSharedWatchlists(ctx context.Context, granteeClientID string) (json.RawMessage, error)
	ListWatchlistAudit(ctx context.Context, ownerClientID string, limit, offset int) (json.RawMessage, error)
	ListPriceAlerts(ctx context.Context, clientID string) (json.RawMessage, error)
	CreatePriceAlert(ctx context.Context, clientID, exchange, symbol, condition string, targetPrice float64, mode string) (json.RawMessage, error)
	DeletePriceAlert(ctx context.Context, clientID, id string) (json.RawMessage, error)
	GetAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error)
	SetAlertWebhook(ctx context.Context, clientID, url string) (json.RawMessage, error)
	SetAlertWebhookWithMode(ctx context.Context, clientID, url, deliveryMode string) (json.RawMessage, error)
	SetAlertWebhookSettings(ctx context.Context, clientID, url, deliveryMode, timeZone string, quietEnabled bool, quietStart, quietEnd string) (json.RawMessage, error)
	DeleteAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error)
	CreatePortfolio(ctx context.Context, clientID string, startingBalance float64, currency string) (json.RawMessage, error)
	GetPortfolio(ctx context.Context, clientID string) (json.RawMessage, error)
	PlacePortfolioOrder(ctx context.Context, clientID, exchange, symbol, side string, quantity float64) (json.RawMessage, error)
	PlacePortfolioPendingOrder(ctx context.Context, clientID, exchange, symbol, orderType string, quantity, triggerPrice float64, timeInForce, expiresAt, trailType string, trailValue float64) (json.RawMessage, error)
	PlacePortfolioOCOOrder(ctx context.Context, clientID, exchange, symbol string, quantity, takeProfitPrice, stopLossPrice float64, expiresAt string) (json.RawMessage, error)
	PlacePortfolioBracketOrder(ctx context.Context, clientID, exchange, symbol string, quantity, entryPrice, takeProfitPrice, stopLossPrice float64, expiresAt string) (json.RawMessage, error)
	ListPortfolioOrders(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error)
	GetPortfolioOrder(ctx context.Context, clientID, id string) (json.RawMessage, error)
	AmendPortfolioOrder(ctx context.Context, clientID, id string, triggerPrice, remainingQuantity *float64) (json.RawMessage, error)
	CancelPortfolioOrder(ctx context.Context, clientID, id string) (json.RawMessage, error)
	CancelAllPortfolioOrders(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error)
	ListPortfolioTrades(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	CreateRecurringBuyPlan(ctx context.Context, clientID, exchange, symbol string, amount float64, frequency, startAt, name, weekday string, dayOfMonth, intervalHours int) (json.RawMessage, error)
	UpdateRecurringBuyPlan(ctx context.Context, clientID, id, name, frequency, weekday, startAt string, amount float64, dayOfMonth, intervalHours int) (json.RawMessage, error)
	ListRecurringBuyPlans(ctx context.Context, clientID string) (json.RawMessage, error)
	GetRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	PauseRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ResumeRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	DeleteRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int) (json.RawMessage, error)
	CreatePortfolioBasket(ctx context.Context, clientID, name, targetsJSON string) (json.RawMessage, error)
	ListPortfolioBaskets(ctx context.Context, clientID string) (json.RawMessage, error)
	GetPortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error)
	UpdatePortfolioBasket(ctx context.Context, clientID, id, name, targetsJSON string) (json.RawMessage, error)
	DeletePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error)
	PreviewPortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error)
	RebalancePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error)
	PlaceMarginOrder(ctx context.Context, clientID, exchange, symbol, side, orderType string, quantity float64, leverage int, limitPrice float64, stopLoss, takeProfit *float64) (json.RawMessage, error)
	ListMarginPositions(ctx context.Context, clientID string) (json.RawMessage, error)
	GetMarginPosition(ctx context.Context, clientID, id string) (json.RawMessage, error)
	CloseMarginPosition(ctx context.Context, clientID, id string, quantity float64) (json.RawMessage, error)
	SetMarginBrackets(ctx context.Context, clientID, id string, stopLoss, takeProfit *float64, clearSL, clearTP bool) (json.RawMessage, error)
	ListMarginOrders(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error)
	CancelMarginOrder(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListMarginTrades(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	SetMarginMode(ctx context.Context, clientID, mode string) (json.RawMessage, error)
	AdjustMargin(ctx context.Context, clientID, positionID string, delta float64) (json.RawMessage, error)
	RepayMarginDebt(ctx context.Context, clientID, positionID string, amount float64) (json.RawMessage, error)
	CreateScannerRule(ctx context.Context, args map[string]any) (json.RawMessage, error)
	ListScannerRules(ctx context.Context, clientID string) (json.RawMessage, error)
	DeleteScannerRule(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListScannerResults(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	StartExport(ctx context.Context, clientID, format string, sections []string) (json.RawMessage, error)
	GetExport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListExports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	CancelExport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	PreviewImport(ctx context.Context, clientID, fileName, format string, fileBytes []byte) (json.RawMessage, error)
	ConfirmImport(ctx context.Context, clientID, id, mode string) (json.RawMessage, error)
	GetImport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListImports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	CancelImport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	CreatePriceDiffWatch(ctx context.Context, clientID, symbol string, minNetDiffPct, feeBinance, feeCoinbase, feeBybit float64) (json.RawMessage, error)
	ListPriceDiffWatches(ctx context.Context, clientID string) (json.RawMessage, error)
	GetPriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	DeletePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListPriceDiffOpportunities(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error)
	GetPriceDiffOpportunity(ctx context.Context, clientID, id string) (json.RawMessage, error)
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

// NewInProcessServer wires MCP tools to market/watchlist/alert/portfolio/scanner/export services (same process as HTTP).
func NewInProcessServer(marketSvc *market.Service, watchSvc *watchlist.Service, alertSvc *pricealert.Service, portfolioSvc *portfolio.Service, scannerSvc *scanner.Service, exportSvc *exportsvc.Service, importSvc *dataimport.Service, priceDiffSvc *pricediff.Service) *server.MCPServer {
	return NewServer(ServerOptions{
		Data: &Backend{Market: marketSvc, Watch: watchSvc, Alerts: alertSvc, Portfolio: portfolioSvc, Scanner: scannerSvc, Export: exportSvc, Import: importSvc, PriceDiff: priceDiffSvc},
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
		mcp.WithNumber("maxTotalEvents", mcp.Description("Cap on total events across all symbols (default 30)")),
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
			"maxTotalEvents": req.GetInt("maxTotalEvents", 30),
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
		mcp.WithDescription("Get a watchlist by clientId. Optional ownerClientId reads a list shared with the actor (viewer/editor/owner)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Actor opaque client id (not 'default')")),
		mcp.WithString("ownerClientId", mcp.Description("List owner when viewing a shared list; omit for own list")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetWatchlistOwned(ctx, clientID, req.GetString("ownerClientId", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("add_watchlist_item",
		mcp.WithDescription("Add or update a symbol on a watchlist. Owner or editor may mutate; optional ownerClientId for shared lists."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Actor opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("note", mcp.Description("Optional note")),
		mcp.WithString("ownerClientId", mcp.Description("List owner when editing a shared list")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.AddWatchlistItemOwned(ctx, clientID, req.GetString("ownerClientId", ""),
			req.GetString("exchange", "binance"), symbol, req.GetString("note", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("remove_watchlist_item",
		mcp.WithDescription("Remove a symbol from a watchlist. Owner or editor may mutate; optional ownerClientId for shared lists."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Actor opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("ownerClientId", mcp.Description("List owner when editing a shared list")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RemoveWatchlistItemOwned(ctx, clientID, req.GetString("ownerClientId", ""),
			req.GetString("exchange", "binance"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("share_watchlist",
		mcp.WithDescription("Share your watchlist with another clientId as viewer (read-only) or editor (add/remove symbols). Owner only; cannot share twice with same user."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("granteeClientId", mcp.Required(), mcp.Description("Client to share with")),
		mcp.WithString("role", mcp.Required(), mcp.Description("viewer or editor")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		grantee, err := req.RequireString("granteeClientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		role, err := req.RequireString("role")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ShareWatchlist(ctx, owner, grantee, role)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("update_watchlist_share",
		mcp.WithDescription("Change role (viewer|editor) for an existing watchlist share. Owner only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("granteeClientId", mcp.Required(), mcp.Description("Shared-with client id")),
		mcp.WithString("role", mcp.Required(), mcp.Description("viewer or editor")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		grantee, err := req.RequireString("granteeClientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		role, err := req.RequireString("role")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.UpdateWatchlistShare(ctx, owner, grantee, role)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("revoke_watchlist_share",
		mcp.WithDescription("Remove a user's access to your watchlist. Owner only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("granteeClientId", mcp.Required(), mcp.Description("Client to revoke")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		grantee, err := req.RequireString("granteeClientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RevokeWatchlistShare(ctx, owner, grantee)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_watchlist_shares",
		mcp.WithDescription("List who has access to your watchlist and their roles. Owner only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListWatchlistShares(ctx, owner)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_shared_watchlists",
		mcp.WithDescription("List watchlists shared with this clientId (incoming shares)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Grantee client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListSharedWatchlists(ctx, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_watchlist_audit",
		mcp.WithDescription("List who changed a watchlist and when (share grants, item add/remove, replace). Owner only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithNumber("limit", mcp.Description("Max events (default 50, max 200)")),
		mcp.WithNumber("offset", mcp.Description("Pagination offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		owner, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := int(req.GetFloat("limit", 50))
		offset := int(req.GetFloat("offset", 0))
		raw, err := api.ListWatchlistAudit(ctx, owner, limit, offset)
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
		mcp.WithDescription("Set webhook URL, deliveryMode (immediate|hourly_digest), timeZone, and optional quietHours (start/end HH:MM, may cross midnight)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("url", mcp.Required(), mcp.Description("https://hooks.example.com/...")),
		mcp.WithString("deliveryMode", mcp.Description("immediate | hourly_digest (default immediate)")),
		mcp.WithString("timeZone", mcp.Description("IANA timezone, default UTC")),
		mcp.WithBoolean("quietEnabled", mcp.Description("Defer delivery during quiet hours")),
		mcp.WithString("quietStart", mcp.Description("Local quiet start HH:MM")),
		mcp.WithString("quietEnd", mcp.Description("Local quiet end HH:MM")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		u, err := req.RequireString("url")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.SetAlertWebhookSettings(ctx, clientID, u,
			req.GetString("deliveryMode", "immediate"),
			req.GetString("timeZone", "UTC"),
			req.GetBool("quietEnabled", false),
			req.GetString("quietStart", ""),
			req.GetString("quietEnd", ""),
		)
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

	s.AddTool(mcp.NewTool("create_portfolio",
		mcp.WithDescription("Create a paper-trading portfolio with a starting cash balance (one per clientId). Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("startingBalance", mcp.Required(), mcp.Description("Starting cash e.g. 10000")),
		mcp.WithString("currency", mcp.Description("Accounting currency default USDT")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		bal, err := req.RequireFloat("startingBalance")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreatePortfolio(ctx, clientID, bal, req.GetString("currency", "USDT"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_portfolio",
		mcp.WithDescription("Get paper portfolio cash, positions, realized/unrealized P&L (mark-to-market)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPortfolio(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("place_portfolio_order",
		mcp.WithDescription("Paper market buy/sell at last price. Simulated only — not real trading."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("side", mcp.Required(), mcp.Description("buy | sell")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base asset quantity")),
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
		side, err := req.RequireString("side")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		qty, err := req.RequireFloat("quantity")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.PlacePortfolioOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol, side, qty)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_portfolio_trades",
		mcp.WithDescription("List paper trade history for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPortfolioTrades(ctx, clientID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("place_portfolio_pending_order",
		mcp.WithDescription("Place a paper pending order: limit_buy, limit_sell, stop_loss, or trailing_stop. For trailing_stop use trailType+trailValue (triggerPrice optional/ignored). Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("type", mcp.Required(), mcp.Description("limit_buy | limit_sell | stop_loss | trailing_stop")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base asset quantity")),
		mcp.WithNumber("triggerPrice", mcp.Description("Limit or stop price (not used for trailing_stop)")),
		mcp.WithString("trailType", mcp.Description("trailing_stop: percent | offset")),
		mcp.WithNumber("trailValue", mcp.Description("trailing_stop: fraction e.g. 0.05 or fixed offset")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("timeInForce", mcp.Description("gtc (default) | ioc | fok")),
		mcp.WithString("expiresAt", mcp.Description("RFC3339 expiry for GTC only")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		typ, err := req.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		qty, err := req.RequireFloat("quantity")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		trig := req.GetFloat("triggerPrice", 0)
		if typ != "trailing_stop" {
			var terr error
			trig, terr = req.RequireFloat("triggerPrice")
			if terr != nil {
				return mcp.NewToolResultError(terr.Error()), nil
			}
		}
		raw, err := api.PlacePortfolioPendingOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol, typ, qty, trig,
			req.GetString("timeInForce", "gtc"), req.GetString("expiresAt", ""),
			req.GetString("trailType", ""), req.GetFloat("trailValue", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("place_portfolio_oco_order",
		mcp.WithDescription("Place a paper OCO: take-profit limit sell + stop-loss for the same quantity. Full fill of one cancels the other; partial fill shrinks both remainings. Same tick fills at most one leg. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base size for both legs")),
		mcp.WithNumber("takeProfitPrice", mcp.Required(), mcp.Description("Limit sell price (must be above stop)")),
		mcp.WithNumber("stopLossPrice", mcp.Required(), mcp.Description("Stop-loss trigger (must be below take-profit)")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("expiresAt", mcp.Description("RFC3339 expiry for GTC legs")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		qty, err := req.RequireFloat("quantity")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tp, err := req.RequireFloat("takeProfitPrice")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sl, err := req.RequireFloat("stopLossPrice")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.PlacePortfolioOCOOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol, qty, tp, sl, req.GetString("expiresAt", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("place_portfolio_bracket_order",
		mcp.WithDescription("Place a paper bracket: limit-buy entry with take-profit + stop-loss. Exits stay pending until entry fills; exit size tracks filled qty; exits are OCO. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Entry size")),
		mcp.WithNumber("entryPrice", mcp.Required(), mcp.Description("Limit buy price")),
		mcp.WithNumber("takeProfitPrice", mcp.Required(), mcp.Description("Limit sell above entry")),
		mcp.WithNumber("stopLossPrice", mcp.Required(), mcp.Description("Stop below entry")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("expiresAt", mcp.Description("RFC3339 expiry")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		qty, err := req.RequireFloat("quantity")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		entry, err := req.RequireFloat("entryPrice")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tp, err := req.RequireFloat("takeProfitPrice")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sl, err := req.RequireFloat("stopLossPrice")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.PlacePortfolioBracketOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol, qty, entry, tp, sl, req.GetString("expiresAt", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_portfolio_orders",
		mcp.WithDescription("List paper pending orders (default status=open)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("status", mcp.Description("open|filled|canceled|rejected|all")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPortfolioOrders(ctx, clientID, req.GetString("status", "open"), req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_portfolio_order",
		mcp.WithDescription("Get one paper pending order plus last price and amend hints (editable, max remaining, available cash/qty including this order's reservation)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Pending order id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		orderID, err := req.RequireString("orderId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPortfolioOrder(ctx, clientID, orderID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("amend_portfolio_order",
		mcp.WithDescription("Amend an open paper GTC limit_buy, limit_sell, or stop_loss in place (same id): triggerPrice and/or remainingQuantity. Recalculates reservations; fills immediately if newly marketable. OCO, bracket, trailing, IOC/FOK cannot be amended."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Pending order id")),
		mcp.WithNumber("triggerPrice", mcp.Description("New limit/stop price")),
		mcp.WithNumber("remainingQuantity", mcp.Description("New remaining size (must stay > 0)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		orderID, err := req.RequireString("orderId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var trigger, remaining *float64
		if v := req.GetFloat("triggerPrice", -1); v >= 0 {
			trigger = &v
		}
		if v := req.GetFloat("remainingQuantity", -1); v >= 0 {
			remaining = &v
		}
		raw, err := api.AmendPortfolioOrder(ctx, clientID, orderID, trigger, remaining)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("cancel_all_portfolio_orders",
		mcp.WithDescription("Cancel all open paper pending orders for a client, or only one market when symbol is set. Releases reservations. Empty result is success."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Description("When set, cancel only this pair (e.g. BTCUSDT); omit for all markets")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit; default binance when symbol is set")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CancelAllPortfolioOrders(ctx, clientID, req.GetString("exchange", ""), req.GetString("symbol", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("cancel_portfolio_order",
		mcp.WithDescription("Cancel an open paper pending order. Canceled orders never fill."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Pending order id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		orderID, err := req.RequireString("orderId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CancelPortfolioOrder(ctx, clientID, orderID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("create_recurring_buy",
		mcp.WithDescription("Create a named paper recurring buy: cash amount at market on daily|weekly|monthly|interval. Use weekday (monday) for weekly, dayOfMonth (1-31) for salary day, intervalHours (e.g. 12) for interval. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Cash notional per run e.g. 500")),
		mcp.WithString("frequency", mcp.Required(), mcp.Description("daily | weekly | monthly | interval")),
		mcp.WithString("name", mcp.Description("Label e.g. Salary Day Buy")),
		mcp.WithString("weekday", mcp.Description("Weekly: monday..sunday")),
		mcp.WithNumber("dayOfMonth", mcp.Description("Monthly salary day 1-31")),
		mcp.WithNumber("intervalHours", mcp.Description("Interval frequency: 1-168 hours")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithString("startAt", mcp.Description("RFC3339 first run; default now")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amount, err := req.RequireFloat("amount")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		freq, err := req.RequireString("frequency")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateRecurringBuyPlan(ctx, clientID, req.GetString("exchange", "binance"), symbol, amount, freq, req.GetString("startAt", ""),
			req.GetString("name", ""), req.GetString("weekday", ""), int(req.GetFloat("dayOfMonth", 0)), int(req.GetFloat("intervalHours", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("update_recurring_buy",
		mcp.WithDescription("Update a paper recurring buy name, amount, or schedule (frequency/weekday/dayOfMonth/intervalHours). Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
		mcp.WithString("name", mcp.Description("New label")),
		mcp.WithNumber("amount", mcp.Description("New cash notional per run")),
		mcp.WithString("frequency", mcp.Description("daily | weekly | monthly | interval")),
		mcp.WithString("weekday", mcp.Description("monday..sunday")),
		mcp.WithNumber("dayOfMonth", mcp.Description("1-31")),
		mcp.WithNumber("intervalHours", mcp.Description("1-168")),
		mcp.WithString("startAt", mcp.Description("RFC3339 to reschedule first/next run")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.UpdateRecurringBuyPlan(ctx, clientID, planID, req.GetString("name", ""), req.GetString("frequency", ""),
			req.GetString("weekday", ""), req.GetString("startAt", ""), req.GetFloat("amount", 0), int(req.GetFloat("dayOfMonth", 0)), int(req.GetFloat("intervalHours", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_recurring_buys",
		mcp.WithDescription("List paper recurring buy plans for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListRecurringBuyPlans(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_recurring_buy",
		mcp.WithDescription("Get one paper recurring buy plan by id."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetRecurringBuyPlan(ctx, clientID, planID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("pause_recurring_buy",
		mcp.WithDescription("Pause a paper recurring buy plan (no further executions until resume)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.PauseRecurringBuyPlan(ctx, clientID, planID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("resume_recurring_buy",
		mcp.WithDescription("Resume a paused paper recurring buy plan."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ResumeRecurringBuyPlan(ctx, clientID, planID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("delete_recurring_buy",
		mcp.WithDescription("Delete a paper recurring buy plan and its run history."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeleteRecurringBuyPlan(ctx, clientID, planID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_recurring_buy_runs",
		mcp.WithDescription("List execution history for a paper recurring buy plan (succeeded/failed runs)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListRecurringBuyRuns(ctx, clientID, planID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("create_portfolio_basket",
		mcp.WithDescription("Create a named paper allocation basket (target percent mix). targetsJSON example: [{\"asset\":\"BTC\",\"weightPct\":50},{\"asset\":\"ETH\",\"weightPct\":30},{\"asset\":\"USDT\",\"weightPct\":20}]. Does not trade. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Basket label e.g. Core 50/30/20")),
		mcp.WithString("targetsJSON", mcp.Required(), mcp.Description("JSON array of {asset,weightPct,exchange?}")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tg, err := req.RequireString("targetsJSON")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreatePortfolioBasket(ctx, clientID, name, tg)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_portfolio_baskets",
		mcp.WithDescription("List saved paper allocation baskets for a client."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPortfolioBaskets(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_portfolio_basket",
		mcp.WithDescription("Get a paper allocation basket with live actual vs target weights and proposed rebalance legs (preview, no trades)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("basketId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPortfolioBasket(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("update_portfolio_basket",
		mcp.WithDescription("Update a paper allocation basket name and/or targetsJSON. Does not trade."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
		mcp.WithString("name", mcp.Description("New label")),
		mcp.WithString("targetsJSON", mcp.Description("JSON array of {asset,weightPct,exchange?}")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("basketId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.UpdatePortfolioBasket(ctx, clientID, id, req.GetString("name", ""), req.GetString("targetsJSON", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("delete_portfolio_basket",
		mcp.WithDescription("Delete a paper allocation basket. Does not trade."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("basketId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeletePortfolioBasket(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("preview_portfolio_rebalance",
		mcp.WithDescription("Preview market sells/buys to move a paper portfolio toward a basket. Does not trade. Rebalance is never automatic."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("basketId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.PreviewPortfolioRebalance(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("rebalance_portfolio_basket",
		mcp.WithDescription("USER-TRIGGERED paper rebalance toward a basket (sell overweight, buy underweight at last price). Drift is allowed until this is called. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("basketId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RebalancePortfolioBasket(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("set_margin_mode",
		mcp.WithDescription("Set paper portfolio margin mode isolated|cross. Blocked while open margin positions or pending margin orders exist."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("mode", mcp.Required(), mcp.Description("isolated | cross")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		mode, err := req.RequireString("mode")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.SetMarginMode(ctx, clientID, mode)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("adjust_margin",
		mcp.WithDescription("Add (delta>0) or remove (delta<0) margin from an isolated paper position; recalculates liquidation. Isolated mode only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("delta", mcp.Required(), mcp.Description("Positive add, negative remove")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("positionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		delta, err := req.RequireFloat("delta")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.AdjustMargin(ctx, clientID, id, delta)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("repay_margin_debt",
		mcp.WithDescription("Repay margin debt without closing: interest first, then principal. Amount in debt units (quote for long, base coins for short)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Debt units to repay")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("positionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amount, err := req.RequireFloat("amount")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RepayMarginDebt(ctx, clientID, id, amount)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("place_margin_order",
		mcp.WithDescription("Paper margin open: long|short, leverage 1-10, market or limit (uses account isolated|cross mode). Optional stopLoss/takeProfit. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("side", mcp.Required(), mcp.Description("long | short")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base quantity")),
		mcp.WithNumber("leverage", mcp.Required(), mcp.Description("1–10")),
		mcp.WithString("type", mcp.Description("market (default) | limit")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit")),
		mcp.WithNumber("limitPrice", mcp.Description("Required for limit")),
		mcp.WithNumber("stopLoss", mcp.Description("Optional stop-loss price")),
		mcp.WithNumber("takeProfit", mcp.Description("Optional take-profit price")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		side, err := req.RequireString("side")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		qty, err := req.RequireFloat("quantity")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		levF, err := req.RequireFloat("leverage")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var sl, tp *float64
		if v := req.GetFloat("stopLoss", 0); v > 0 {
			sl = &v
		}
		if v := req.GetFloat("takeProfit", 0); v > 0 {
			tp = &v
		}
		raw, err := api.PlaceMarginOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol, side,
			req.GetString("type", "market"), qty, int(levF), req.GetFloat("limitPrice", 0), sl, tp)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_margin_positions",
		mcp.WithDescription("List open paper margin positions with mark, unrealized PnL, liquidation price."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListMarginPositions(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("close_margin_position",
		mcp.WithDescription("Close all or part of a paper margin position at market. quantity 0 = full close."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("quantity", mcp.Description("Partial size; omit/0 for full")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("positionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CloseMarginPosition(ctx, clientID, id, req.GetFloat("quantity", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("set_margin_brackets",
		mcp.WithDescription("Set or clear stop-loss / take-profit on an open paper margin position."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("stopLoss", mcp.Description("Stop-loss price")),
		mcp.WithNumber("takeProfit", mcp.Description("Take-profit price")),
		mcp.WithString("clearStopLoss", mcp.Description("true to clear SL")),
		mcp.WithString("clearTakeProfit", mcp.Description("true to clear TP")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("positionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var sl, tp *float64
		if v := req.GetFloat("stopLoss", 0); v > 0 {
			sl = &v
		}
		if v := req.GetFloat("takeProfit", 0); v > 0 {
			tp = &v
		}
		clearSL := strings.EqualFold(req.GetString("clearStopLoss", ""), "true")
		clearTP := strings.EqualFold(req.GetString("clearTakeProfit", ""), "true")
		raw, err := api.SetMarginBrackets(ctx, clientID, id, sl, tp, clearSL, clearTP)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_margin_orders",
		mcp.WithDescription("List paper margin orders (default status=open)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("status", mcp.Description("open|filled|canceled|rejected|all")),
		mcp.WithNumber("limit", mcp.Description("Max rows")),
		mcp.WithNumber("offset", mcp.Description("Offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListMarginOrders(ctx, clientID, req.GetString("status", "open"), req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("cancel_margin_order",
		mcp.WithDescription("Cancel an open paper margin limit order."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Margin order id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("orderId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CancelMarginOrder(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_margin_trades",
		mcp.WithDescription("List paper margin trade history (open/close/liquidation/sl/tp)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("limit", mcp.Description("Max rows")),
		mcp.WithNumber("offset", mcp.Description("Offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListMarginTrades(ctx, clientID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("create_scanner_rule",
		mcp.WithDescription("Create a technical scanner rule for the client's watchlist: rsi, ma_crossover, or volume_increase."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("type", mcp.Required(), mcp.Description("rsi | ma_crossover | volume_increase")),
		mcp.WithString("interval", mcp.Description("Candle interval default 1h")),
		mcp.WithNumber("rsiPeriod", mcp.Description("RSI period default 14")),
		mcp.WithString("rsiCondition", mcp.Description("above | below")),
		mcp.WithNumber("rsiThreshold", mcp.Description("RSI threshold 0-100")),
		mcp.WithNumber("maFastPeriod", mcp.Description("EMA fast period")),
		mcp.WithNumber("maSlowPeriod", mcp.Description("EMA slow period")),
		mcp.WithString("maDirection", mcp.Description("golden_cross | death_cross")),
		mcp.WithNumber("volumeLookback", mcp.Description("Bars for volume average")),
		mcp.WithNumber("volumeMinRatio", mcp.Description("Min last/avg volume ratio")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		typ, err := req.RequireString("type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args := map[string]any{
			"clientId": clientID, "type": typ, "interval": req.GetString("interval", "1h"),
			"rsiPeriod": req.GetFloat("rsiPeriod", 14), "rsiCondition": req.GetString("rsiCondition", "below"),
			"rsiThreshold": req.GetFloat("rsiThreshold", 30),
			"maFastPeriod": req.GetFloat("maFastPeriod", 12), "maSlowPeriod": req.GetFloat("maSlowPeriod", 26),
			"maDirection": req.GetString("maDirection", "golden_cross"),
			"volumeLookback": req.GetFloat("volumeLookback", 20), "volumeMinRatio": req.GetFloat("volumeMinRatio", 2),
		}
		raw, err := api.CreateScannerRule(ctx, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_scanner_rules",
		mcp.WithDescription("List technical scanner rules for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListScannerRules(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("delete_scanner_rule",
		mcp.WithDescription("Delete a scanner rule by id."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("ruleId", mcp.Required(), mcp.Description("Rule id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ruleID, err := req.RequireString("ruleId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeleteScannerRule(ctx, clientID, ruleID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_scanner_results",
		mcp.WithDescription("List saved scanner match history for a clientId (deduped by rule/symbol/bar)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListScannerResults(ctx, clientID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("start_export",
		mcp.WithDescription("Start a background export of the user's watchlist, shares, alerts, and/or backtests as json or csv. Only one active export per client. Poll get_export for progress; download via HTTP when completed."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("format", mcp.Description("json (default) or csv")),
		mcp.WithString("sections", mcp.Description("Comma-separated: watchlist,shares,alerts,backtests (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var sections []string
		if raw := strings.TrimSpace(req.GetString("sections", "")); raw != "" {
			for _, p := range strings.Split(raw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					sections = append(sections, p)
				}
			}
		}
		raw, err := api.StartExport(ctx, clientID, req.GetString("format", "json"), sections)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_export",
		mcp.WithDescription("Get export job status and progressPct (0-100). When completed, downloadUrl is set for HTTP download."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("exportId", mcp.Required(), mcp.Description("Export job id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("exportId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetExport(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_exports",
		mcp.WithDescription("List recent data export jobs for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 20")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListExports(ctx, clientID, req.GetInt("limit", 20), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("cancel_export",
		mcp.WithDescription("Cancel a pending or running data export job."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("exportId", mcp.Required(), mcp.Description("Export job id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("exportId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CancelExport(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("preview_import",
		mcp.WithDescription("Preview restoring a prior JSON export of watchlist/shares/alerts/backtests. Returns valid/invalid/willAdd counts without applying. Pass export JSON as content. Then call confirm_import with mode merge|replace."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Full export file text (JSON preferred)")),
		mcp.WithString("format", mcp.Description("json (default) or csv")),
		mcp.WithString("fileName", mcp.Description("Optional original file name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		content, err := req.RequireString("content")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.PreviewImport(ctx, clientID, req.GetString("fileName", "export.json"),
			req.GetString("format", "json"), []byte(content))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("confirm_import",
		mcp.WithDescription("Start applying a previewed import. mode=merge (skip duplicates) or replace (clear then import). One active apply per client."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("importId", mcp.Required(), mcp.Description("Import job id from preview")),
		mcp.WithString("mode", mcp.Required(), mcp.Description("merge or replace")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("importId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		mode, err := req.RequireString("mode")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ConfirmImport(ctx, clientID, id, mode)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_import",
		mcp.WithDescription("Get import job status, progress, and section counts."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("importId", mcp.Required(), mcp.Description("Import job id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("importId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetImport(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_imports",
		mcp.WithDescription("List recent data import jobs for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 20")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListImports(ctx, clientID, req.GetInt("limit", 20), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("cancel_import",
		mcp.WithDescription("Cancel a previewed, pending, or running import."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("importId", mcp.Required(), mcp.Description("Import job id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("importId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CancelImport(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("create_price_diff_watch",
		mcp.WithDescription("Track cross-exchange price differences for a coin (Binance/Coinbase/Bybit). Opens opportunities when net edge after fees exceeds minNetDiffPct."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("minNetDiffPct", mcp.Required(), mcp.Description("Minimum net difference % after fees e.g. 0.5")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance fee % e.g. 0.1")),
		mcp.WithNumber("feeCoinbasePct", mcp.Description("Coinbase fee % e.g. 0.6")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit fee % e.g. 0.1")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		minNet, err := req.RequireFloat("minNetDiffPct")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreatePriceDiffWatch(ctx, clientID, symbol, minNet,
			req.GetFloat("feeBinancePct", 0), req.GetFloat("feeCoinbasePct", 0), req.GetFloat("feeBybitPct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_price_diff_watches",
		mcp.WithDescription("List cross-exchange price difference watches for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPriceDiffWatches(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_price_diff_watch",
		mcp.WithDescription("Get one price-diff watch by id."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("watchId", mcp.Required(), mcp.Description("Watch id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("watchId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPriceDiffWatch(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("delete_price_diff_watch",
		mcp.WithDescription("Delete a price-diff watch and its opportunities."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("watchId", mcp.Required(), mcp.Description("Watch id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("watchId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeletePriceDiffWatch(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("list_price_diff_opportunities",
		mcp.WithDescription("List cross-exchange price opportunities (status open|closed|all). Open opportunities persist across restarts."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("status", mcp.Description("open | closed | all")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPriceDiffOpportunities(ctx, clientID, req.GetString("status", "open"), req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	s.AddTool(mcp.NewTool("get_price_diff_opportunity",
		mcp.WithDescription("Get one price-diff opportunity by id."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("opportunityId", mcp.Required(), mcp.Description("Opportunity id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("opportunityId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPriceDiffOpportunity(ctx, clientID, id)
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
