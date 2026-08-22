package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
	"gitlab.com/trace-analysis/swyngora/backend/internal/transport/http/middleware"
)

// DataPort is the data surface MCP tools need (in-process backend or HTTP client).
type DataPort interface {
	GetTicker(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetOrderBook(ctx context.Context, exchange, symbol, group string, limit int, rangePct float64) (json.RawMessage, error)
	AnalyzeOrderBook(ctx context.Context, exchange, symbol string, rangePct float64) (json.RawMessage, error)
	AnalyzeCombinedOrderBook(ctx context.Context, symbol string, rangePct float64) (json.RawMessage, error)
	EstimateOrderBookImpact(ctx context.Context, exchange, symbol, side string, quantity, notional float64) (json.RawMessage, error)
	GetMarketLiquidity(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetOrderBookHeatmap(ctx context.Context, exchange, symbol, group string, windowSec int) (json.RawMessage, error)
	GetLiquidations(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetOpenInterest(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetFundingRate(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error)
	GetLongShortRatio(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error)
	GetFuturesHistory(ctx context.Context, metric, exchange, symbol, from, to string, limit int) (json.RawMessage, error)
	GetLiquidationHunt(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetSqueezeRisk(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetPositioning(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetVenueDivergence(ctx context.Context, symbol string) (json.RawMessage, error)
	GetTakerFlow(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetCVD(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetBasis(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetCorrelation(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetBreadth(ctx context.Context, exchange string, limit int) (json.RawMessage, error)
	GetVolatility(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetSnapshot(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetLevels(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetWhales(ctx context.Context, exchange, symbol string, minNotional float64, limit int) (json.RawMessage, error)
	GetBookHistory(ctx context.Context, exchange, symbol, at, from, to string, limit int) (json.RawMessage, error)
	CompareBookHistory(ctx context.Context, exchange, symbol, from, to string) (json.RawMessage, error)
	GetIcebergs(ctx context.Context, exchange, symbol string, minNotional float64) (json.RawMessage, error)
	GetCandles(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error)
	GetSupply(ctx context.Context, asset string) (json.RawMessage, error)
	GetHolders(ctx context.Context, asset string) (json.RawMessage, error)
	ListSpot(ctx context.Context, exchange, query, quote, sort, order, tag string, limit, offset int) (json.RawMessage, error)
	GetIndicators(ctx context.Context, exchange, symbol, interval string, limit, rsiPeriod int, emaPeriods string) (json.RawMessage, error)
	DetectPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error)
	ScanPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error)
	ListExchanges(ctx context.Context) (json.RawMessage, error)
	GetFxRates(ctx context.Context) (json.RawMessage, error)
	ListDelistSchedule(ctx context.Context, exchange string) (json.RawMessage, error)
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
	CreateOrderBookAlert(ctx context.Context, clientID, exchange, symbol, kind, condition string, threshold, rangePct float64, mode string) (json.RawMessage, error)
	DeletePriceAlert(ctx context.Context, clientID, id string) (json.RawMessage, error)
	GetAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error)
	SetAlertWebhook(ctx context.Context, clientID, url string) (json.RawMessage, error)
	SetAlertWebhookWithMode(ctx context.Context, clientID, url, deliveryMode string) (json.RawMessage, error)
	SetAlertWebhookSettings(ctx context.Context, clientID, url, deliveryMode, timeZone string, quietEnabled bool, quietStart, quietEnd string) (json.RawMessage, error)
	DeleteAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error)
	CreatePortfolio(ctx context.Context, clientID string, startingBalance float64, currency string) (json.RawMessage, error)
	CreateNamedPortfolio(ctx context.Context, clientID string, startingBalance float64, currency, name string) (json.RawMessage, error)
	ListPortfolios(ctx context.Context, clientID string) (json.RawMessage, error)
	RenamePortfolio(ctx context.Context, clientID, id, name string) (json.RawMessage, error)
	DeletePortfolio(ctx context.Context, clientID, id string) (json.RawMessage, error)
	SharePortfolio(ctx context.Context, clientID, portfolioID, granteeClientID, role string) (json.RawMessage, error)
	UpdatePortfolioShare(ctx context.Context, clientID, portfolioID, granteeClientID, role string) (json.RawMessage, error)
	RevokePortfolioShare(ctx context.Context, clientID, portfolioID, granteeClientID string) (json.RawMessage, error)
	ListPortfolioShares(ctx context.Context, clientID, portfolioID string) (json.RawMessage, error)
	ListSharedPortfolios(ctx context.Context, clientID string) (json.RawMessage, error)
	GetPortfolio(ctx context.Context, clientID string) (json.RawMessage, error)
	CreateAPIKey(ctx context.Context, clientID, name, permission string) (json.RawMessage, error)
	ListAPIKeys(ctx context.Context, clientID string) (json.RawMessage, error)
	RevokeAPIKey(ctx context.Context, clientID, id string) (json.RawMessage, error)
	GetPortfolioPerformance(ctx context.Context, clientID, period string) (json.RawMessage, error)
	DepositPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error)
	WithdrawPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error)
	ListPortfolioCashMovements(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	TransferPortfolioCash(ctx context.Context, clientID, fromPortfolioID, toPortfolioID string, amount float64, note string) (json.RawMessage, error)
	GetPortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error)
	PutPortfolioRiskLimits(ctx context.Context, clientID string, maxDailyLossPct, maxAssetWeightPct *float64) (json.RawMessage, error)
	DeletePortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error)
	GetPaperTradingCosts(ctx context.Context, exchange string) (json.RawMessage, error)
	PlacePortfolioOrder(ctx context.Context, clientID, exchange, symbol, side string, quantity float64, lotMethod string) (json.RawMessage, error)
	PlacePortfolioPendingOrder(ctx context.Context, clientID, exchange, symbol, orderType string, quantity, triggerPrice float64, timeInForce, expiresAt, trailType string, trailValue float64, lotMethod string) (json.RawMessage, error)
	ListPortfolioLots(ctx context.Context, clientID, exchange, symbol, status string) (json.RawMessage, error)
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
	AnalyzeSwing(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	ScanSwingSetups(ctx context.Context, clientID, exchange string, limit int) (json.RawMessage, error)
	Health(ctx context.Context) (json.RawMessage, error)
}

// ServerOptions configures the MCP server.
type ServerOptions struct {
	// Data is preferred (in-process). If nil, APIBaseURL is used via HTTP client.
	Data       DataPort
	Accounts   *account.Service // when set, tools with clientId require an active account
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
	registerTools(s, data, opts.Accounts)
	return s
}

// NewInProcessServer wires MCP tools to market/watchlist/alert/portfolio/scanner/export services (same process as HTTP).
func NewInProcessServer(marketSvc *market.Service, watchSvc *watchlist.Service, alertSvc *pricealert.Service, portfolioSvc *portfolio.Service, scannerSvc *scanner.Service, exportSvc *exportsvc.Service, importSvc *dataimport.Service, priceDiffSvc *pricediff.Service, apiKeySvc *apikey.Service, accountSvc *account.Service, swingSvc *swing.Service) *server.MCPServer {
	return NewServer(ServerOptions{
		Data:     &Backend{Market: marketSvc, Watch: watchSvc, Alerts: alertSvc, Portfolio: portfolioSvc, Scanner: scannerSvc, Export: exportSvc, Import: importSvc, PriceDiff: priceDiffSvc, APIKeys: apiKeySvc, Swing: swingSvc},
		Accounts: accountSvc,
		Name:     "swyngora-mcp",
	})
}

// NewHTTPHandler mounts streamable MCP on the shared HTTP process (default path /mcp).
func NewHTTPHandler(mcpServer *server.MCPServer) http.Handler {
	return server.NewStreamableHTTPServer(
		mcpServer,
		server.WithStateLess(true),
	)
}

// toolClientActiveError rejects tenant MCP calls when the clientId account is closed.
// Tools without clientId (market data, health) are unchanged.
func toolClientActiveError(ctx context.Context, accounts *account.Service, req mcp.CallToolRequest) error {
	if accounts == nil {
		return nil
	}
	id := strings.TrimSpace(req.GetString("clientId", ""))
	if id == "" {
		return nil
	}
	return accounts.RequireActive(ctx, id)
}

// bindMCPTenant enforces user-key tenant isolation on MCP tools:
// - force clientId to the key binding (reject mismatches)
// - block key-admin tools for user keys (same as HTTP APIKeyScope)
func bindMCPTenant(ctx context.Context, req *mcp.CallToolRequest) error {
	if req == nil {
		return nil
	}
	id := middleware.IdentityFrom(ctx)
	if id == nil || !id.UserKey {
		return nil
	}
	name := strings.TrimSpace(req.Params.Name)
	switch name {
	case "create_api_key", "list_api_keys", "revoke_api_key":
		return fmt.Errorf("%w: this API key cannot manage account or other keys", domain.ErrForbidden)
	}
	if id.ClientID == "" {
		return fmt.Errorf("%w: API key has no client binding", domain.ErrForbidden)
	}
	requested := strings.TrimSpace(req.GetString("clientId", ""))
	if requested != "" && requested != id.ClientID {
		return fmt.Errorf("%w: clientId does not match API key binding", domain.ErrForbidden)
	}
	args := req.GetArguments()
	if args == nil {
		args = map[string]any{}
	} else {
		// Copy so we do not mutate a shared map unexpectedly.
		cp := make(map[string]any, len(args)+1)
		for k, v := range args {
			cp[k] = v
		}
		args = cp
	}
	args["clientId"] = id.ClientID
	req.Params.Arguments = args
	return nil
}

func registerTools(s *server.MCPServer, api DataPort, accounts *account.Service) {
	// HTTP AccountGate skips /mcp (tools pass clientId in JSON, not headers).
	// Enforce the same active-account rule here when clientId is present.
	// User API keys are bound to their clientId (bindMCPTenant).
	addTool := func(tool mcp.Tool, h func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
		s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if err := bindMCPTenant(ctx, &req); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := toolClientActiveError(ctx, accounts, req); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return h(ctx, req)
		})
	}

	addTool(mcp.NewTool("get_ticker",
		mcp.WithDescription("Get 24h price, volume, and change for a trading pair on an exchange (binance|coinbase|bybit|nasdaq|bist). Use for live quotes."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("Venue id: binance (default), coinbase, bybit, nasdaq, bist")),
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

	addTool(mcp.NewTool("get_orderbook_icebergs",
		mcp.WithDescription("Detect iceberg-style refill at the same price: a visible buy or sell clip is eaten at the touch, then a similar size comes back, repeatedly. Both bid and ask. Returns clip size, refill count, executed notional, and likely vs possible. Not proof of a hidden exchange order. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
		mcp.WithNumber("minNotional", mcp.Description("Minimum visible clip in USD (default 25000)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetIcebergs(ctx, req.GetString("exchange", "all"), symbol, req.GetFloat("minNotional", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_orderbook_history",
		mcp.WithDescription("Stored spot order-book snapshots: bid/ask levels, spread, total liquidity, imbalance, and large walls. Pass at for the book nearest that time (RFC3339 or unix ms). Omit at to list recent samples (from/to optional). 1-minute samples of the live book, not a 24h tape. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance (default) | coinbase | bybit")),
		mcp.WithString("at", mcp.Description("Point in time (RFC3339 or unix ms)")),
		mcp.WithString("from", mcp.Description("List window start")),
		mcp.WithString("to", mcp.Description("List window end")),
		mcp.WithNumber("limit", mcp.Description("Max list rows (default 60, max 500)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetBookHistory(ctx, req.GetString("exchange", "binance"), symbol, req.GetString("at", ""), req.GetString("from", ""), req.GetString("to", ""), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("compare_orderbook_history",
		mcp.WithDescription("Compare two stored spot order books. Shows which price levels gained or lost liquidity, mid/spread/imbalance change, and walls that appeared or were pulled. Use after a strong price move. from and to are RFC3339 or unix ms. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance (default) | coinbase | bybit")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Earlier time (RFC3339 or unix ms)")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Later time (RFC3339 or unix ms)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		from, err := req.RequireString("from")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		to, err := req.RequireString("to")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CompareBookHistory(ctx, req.GetString("exchange", "binance"), symbol, from, to)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_spot_orderbook",
		mcp.WithDescription("Grouped live spot order book (bids/asks) with price steps like 0.1 or 0.01. Includes wall flags, spread, and analysis.pressure / analysis.walls from depth within ±rangePct of mid (default 2%). Each wall has behavior: short, persistent (stayed near the same price), or suspicious (added/removed many times). Binance/Coinbase/Bybit. Spot only."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance (default) | coinbase | bybit | nasdaq | bist")),
		mcp.WithString("group", mcp.Description("Price bucket size e.g. 0.1; omit for a suggested default")),
		mcp.WithNumber("limit", mcp.Description("Grouped rows per side 5–100 (default 20)")),
		mcp.WithNumber("rangePct", mcp.Description("Analyze resting depth within this ±% of mid (0.25–10, default 2)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetOrderBook(ctx, req.GetString("exchange", "binance"), symbol, req.GetString("group", ""), req.GetInt("limit", 20), req.GetFloat("rangePct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("analyze_spot_orderbook",
		mcp.WithDescription("Buy/sell pressure, notional imbalance, and large walls from live spot depth within ±rangePct of mid (default 2%). Walls include behavior short | persistent (resting support/resistance) | suspicious (flicker / pulled often). Prefer this over the ladder when the question is pressure or walls."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance (default) | coinbase | bybit | nasdaq | bist")),
		mcp.WithNumber("rangePct", mcp.Description("±% of mid to include (0.25–10, default 2)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.AnalyzeOrderBook(ctx, req.GetString("exchange", "binance"), symbol, req.GetFloat("rangePct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("analyze_market_orderbook",
		mcp.WithDescription("Market-wide buy/sell pressure for one coin. Sums live Binance + Coinbase + Bybit bid/ask notional only in a symmetric ±% both sides can reach (requested ±rangePct when all cover it both ways). Prefer this for overall market pressure."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithNumber("rangePct", mcp.Description("±% of shared mid (0.25–10, default 2)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.AnalyzeCombinedOrderBook(ctx, symbol, req.GetFloat("rangePct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_orderbook_heatmap",
		mcp.WithDescription("Recent resting bid/ask size over time (order heatmap / bookmap-style tape). Each column is a live book snapshot, not executed volume. Use for 'where has size been sitting' or 'did that wall persist'. window is lookback seconds (60–1800, default 600). Spot only."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance (default) | coinbase | bybit")),
		mcp.WithString("group", mcp.Description("Price bucket size e.g. 0.1; omit for a suggested default")),
		mcp.WithNumber("window", mcp.Description("Lookback seconds 60–1800 (default 600)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetOrderBookHeatmap(ctx, req.GetString("exchange", "binance"), symbol, req.GetString("group", ""), req.GetInt("window", 600))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidations",
		mcp.WithDescription("Futures liquidations for a coin over the last 5 minutes, 1 hour, 4 hours, and 24 hours. Returns long vs short notional, count, and the biggest hit. Fed by Binance USD-M and Bybit linear perpetual streams. exchange=all (default) sums both. complete only counts time the websocket was actually live for that coin and venue; coverage does not grow if the stream never connects or drops. Prefer this for 'how much BTC was liquidated'."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquidations(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_open_interest",
		mcp.WithDescription("Futures open interest for a coin: current outstanding size plus how much it increased or decreased in the last 5 minutes, 1 hour, 4 hours, and 24 hours. Also includes funding (predicted next rate + recent settlements) on the funding field. contracts is base-asset size (e.g. BTC); value is USDT notional. Binance USD-M + Bybit linear perpetual. exchange=all (default) sums OI. Prefer this for 'is OI rising or falling'."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetOpenInterest(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_funding_rate",
		mcp.WithDescription("Perpetual futures funding rate: predicted next payment plus recent settled history. rate is decimal (0.0001 = 0.01%); ratePct is percent. payer=long means longs pay shorts at settlement. Binance USD-M + Bybit linear. exchange=all (default) returns each venue separately (rates are not averaged). Prefer this for 'what is the funding rate' or carry analysis."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithNumber("limit", mcp.Description("Settled history size 1–30 (default 12)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetFundingRate(ctx, req.GetString("exchange", "all"), symbol, int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_long_short_ratio",
		mcp.WithDescription("Futures long/short account ratio: share of accounts that are long vs short (not position size). ratio is long/short (1 = even). bias is long if ratio≥1.05, short if ≤0.95. 5-minute samples. Binance USD-M + Bybit linear. exchange=all returns each venue separately (not averaged). Prefer this for 'are more traders long or short'."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithNumber("limit", mcp.Description("History size 1–100 (default 24, ~2 hours of 5m prints)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLongShortRatio(ctx, req.GetString("exchange", "all"), symbol, int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_futures_history",
		mcp.WithDescription("Durable stored history of futures open interest, funding, long/short ratio, or liquidations (Binance USD-M + Bybit linear). Survives restarts. metric=open_interest|funding|long_short|liquidations. Prefer this for past values, not the live snapshot tools."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("metric", mcp.Required(), mcp.Description("open_interest | funding | long_short | liquidations")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("from", mcp.Description("RFC3339 or unix ms start")),
		mcp.WithString("to", mcp.Description("RFC3339 or unix ms end")),
		mcp.WithNumber("limit", mcp.Description("Max rows, default 200")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		metric, err := req.RequireString("metric")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetFuturesHistory(ctx, metric, req.GetString("exchange", "all"), symbol, req.GetString("from", ""), req.GetString("to", ""), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("estimate_liquidation_hunt",
		mcp.WithDescription("Hypothetical only: if Binance or Bybit walked their own spot book to push price and force futures liquidations, where is the main long/short pressure, how much spot buy/sell the visible book needs, and a rough desk result (book-only unwind vs assuming part of estimated liquidations become exit flow). Venues are never averaged. Not evidence of exchange behavior. Not financial advice. Long/short is account count; leverage mix is assumed; mark price is multi-venue."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both, separately)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquidationHunt(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_squeeze_risk",
		mcp.WithDescription("Long-squeeze and short-squeeze risk scores (0–100) for a coin on Binance USD-M and Bybit linear, plus an OI-weighted combined view. Uses open interest change, funding, account long/short crowding, recent liquidations, and nearby liquidation-pocket estimates. Explains which side is crowded and why risk is high or low. Not a prediction or financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetSqueezeRisk(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_positioning",
		mcp.WithDescription("Price + open interest positioning for a coin: long_buildup, short_buildup, long_unwinding, or short_covering on Binance and Bybit, plus a general market (combined) read. Short summary and reasons (funding / long-short corroboration). Not a prediction or financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPositioning(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_venue_divergence",
		mcp.WithDescription("Compare Binance vs Bybit for one coin: same direction or opposite. Shows which of OI change, funding, account crowding, and price+OI regime differ, plus a short why it matters. Use when the user asks if the two exchanges agree. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetVenueDivergence(ctx, symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_taker_flow",
		mcp.WithDescription("Aggressive futures buy vs sell volume (who is hitting the book) on Binance USD-M and Bybit linear for 5m, 1h, and 4h, plus a combined view and a short read with price/OI/funding. This is taker flow, not the account long/short ratio. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetTakerFlow(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_cvd",
		mcp.WithDescription("Cumulative Volume Delta: running sum of aggressive market-buy minus market-sell notional over time, plotted with price. Futures and spot are separate (spotFutures flags when they disagree). Windows show CVD change over 15m / 1h / 4h / 24h. Points flag price-up/CVD-down or price-down/CVD-up for the current consecutive run only (a later same-kind split is a new episode). Combined uses the overlapping time range and shows when Binance CVD rose while Bybit fell. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetCVD(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_basis",
		mcp.WithDescription("How far the perpetual futures price is from spot/index on Binance and Bybit: premium or discount, dollar and percent gap, whether the gap is expanding or shrinking, plus a short read with funding and open interest. Says if both venues agree. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + agreement)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetBasis(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_price_correlation",
		mcp.WithDescription("How similarly a coin has been moving with BTC and ETH over the last 1 hour, 4 hours, and 24 hours. Returns Pearson correlation, beta, same-direction share, and whether the coin is following with a delay or moving independently. Default venue is Binance spot. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. SOLUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | coinbase (default binance)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetCorrelation(ctx, req.GetString("exchange", ""), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_market_breadth",
		mcp.WithDescription("How much of the market is going up or down among the liquid spot coins we follow. Counts and percents for 1h, 4h, and 24h, plus whether BTC and ETH are moving with the rest of the market or a few large coins are carrying it. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | coinbase (default binance)")),
		mcp.WithNumber("limit", mcp.Description("How many liquid coins to include (default 80, max 150)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetBreadth(ctx, req.GetString("exchange", ""), req.GetInt("limit", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_price_volatility",
		mcp.WithDescription("How volatile a coin has been over the last 1 hour, 4 hours, and 24 hours: net move, high-low range, whether the range is bigger or smaller than normal and than the previous window, and whether the coin is jumpy or calm versus BTC and ETH. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. SOLUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | coinbase (default binance)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetVolatility(ctx, req.GetString("exchange", ""), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_market_snapshot",
		mcp.WithDescription("Price, volume, market cap, open interest, funding, long/short, and taker buy/sell together for one coin. Current values plus 1h / 4h / 24h changes. Useful when volume or OI start to build before price moves. Binance and Bybit separately when exchange=all. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. SOLUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetSnapshot(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_support_resistance",
		mcp.WithDescription("Support and resistance areas from price history, volume, and the live order book. Each area has distance from last price, how many times it was tested, and nearby bid/ask liquidity. When price is close to or through a level, a breakout score uses volume, book thickness, and taker buy/sell. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | coinbase (default binance)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLevels(ctx, req.GetString("exchange", "binance"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_whale_trades",
		mcp.WithDescription("Largest recent aggressive futures buys/sells (taker long/short) and large liquidations, sorted biggest first. Each row has average price, first/last trade time, total size, and whether the print is large versus that coin's market cap (small-cap + huge trade). Omit symbol to scan the top liquid USDT coins. Tape is the newest ~1000 prints per coin, not 24h. Not financial advice."),
		mcp.WithString("symbol", mcp.Description("Pair e.g. BTCUSDT. Omit to scan top liquid USDT coins.")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithNumber("minNotional", mcp.Description("Minimum USD size (default 100000)")),
		mcp.WithNumber("limit", mcp.Description("Max events (default 30, max 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetWhales(ctx, req.GetString("exchange", "all"), req.GetString("symbol", ""), req.GetFloat("minNotional", 0), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_market_liquidity",
		mcp.WithDescription("How liquid a coin is right now. Scores resting buy/sell notional only in ±0.1 / ±0.5 / ±1% bands the book actually reaches on both sides (usedRangePct). Market-wide uses the overlap all venues can reach. 0–100 plus grade; weakerSide is the thinner side. exchange=all (default) returns Binance, Coinbase, Bybit, and a market-wide score."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetMarketLiquidity(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("estimate_market_impact",
		mcp.WithDescription("Simulate a market buy or sell against live order-book depth. Walks asks (buy) or bids (sell) level by level. Use quantity for base size (e.g. 5 BTC) or notional for quote size (e.g. 1000000000 USDT). exchange=all (default) merges Binance+Coinbase+Bybit cheapest-first. Returns average fill, slippage, and price impact as the new best ask/bid after leftover size (0 if the touch still has quantity). If the visible book is wiped, impactAvailable is false — do not invent impact from the last fill. Simulation only."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("side", mcp.Description("buy (default) | sell")),
		mcp.WithNumber("quantity", mcp.Description("Base size to fill (e.g. 5). Do not set together with notional.")),
		mcp.WithNumber("notional", mcp.Description("Quote size to spend/receive (e.g. 1000000000). Do not set together with quantity.")),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.EstimateOrderBookImpact(ctx, req.GetString("exchange", "all"), symbol, req.GetString("side", "buy"), req.GetFloat("quantity", 0), req.GetFloat("notional", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_candles",
		mcp.WithDescription("Fetch OHLCV candlesticks for a symbol. Chronological oldest-first."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
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

	addTool(mcp.NewTool("get_holders",
		mcp.WithDescription("On-chain holder count, concentration (top 10/50/100 %), and top wallets for a crypto base asset (BTC, ETH, or BTCUSDT). CoinMarketCap public snapshot; 404 if unpublished or not a crypto asset. Informational only."),
		mcp.WithString("asset", mcp.Required(), mcp.Description("Base asset ticker e.g. BTC (pairs like BTCUSDT also work)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		asset, err := req.RequireString("asset")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetHolders(ctx, asset)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_supply",
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

	addTool(mcp.NewTool("list_spot_markets",
		mcp.WithDescription("List/search/sort spot markets on an exchange. Supports quote filter, product tags, mcap sorts, pagination."),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
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

	addTool(mcp.NewTool("get_indicators",
		mcp.WithDescription("Compute RSI (Wilder) and EMA series for a symbol. Informational only — not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
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

	addTool(mcp.NewTool("detect_pump_events",
		mcp.WithDescription(
			"Detect pump/dump events on one symbol from OHLCV candles. "+
				"Configurable minReturnPct, windowBars, candle interval, lookbackHours or start/end time, "+
				"mode (close_return|candle_body|high_from_low), direction (up|down|both), minVolumeRatio. "+
				"Informational mechanical filter — not a trade signal.",
		),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or JUVUSDT")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist (default binance)")),
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

	addTool(mcp.NewTool("scan_pump_events",
		mcp.WithDescription(
			"Scan top quote-volume symbols on an exchange for recent pump/dump events. "+
				"Same thresholds as detect_pump_events (minReturnPct, interval, lookbackHours, mode, direction, minVolumeRatio). "+
				"Use for 'what pumped in the last N hours'. Informational only.",
		),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
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

	addTool(mcp.NewTool("get_fx_rates",
		mcp.WithDescription("Spot FX rates (units per 1 USD) for converting BIST TRY, Nasdaq USD, and crypto USDT display values. USDT is treated as USD. Display only."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetFxRates(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_exchanges",
		mcp.WithDescription("List configured market venues and the default exchange."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ListExchanges(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_delist_schedule",
		mcp.WithDescription("List spot delistings for a venue (last 30 days and next ~31 days). Binance official schedule plus CMS titles; Bybit announcements. Empty if none or unsupported."),
		mcp.WithString("exchange", mcp.Description("Venue id; default binance")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ListDelistSchedule(ctx, req.GetString("exchange", "binance"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_watchlist",
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

	addTool(mcp.NewTool("add_watchlist_item",
		mcp.WithDescription("Add or update a symbol on a watchlist. Owner or editor may mutate; optional ownerClientId for shared lists."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Actor opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
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

	addTool(mcp.NewTool("remove_watchlist_item",
		mcp.WithDescription("Remove a symbol from a watchlist. Owner or editor may mutate; optional ownerClientId for shared lists."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Actor opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
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

	addTool(mcp.NewTool("share_watchlist",
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

	addTool(mcp.NewTool("update_watchlist_share",
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

	addTool(mcp.NewTool("revoke_watchlist_share",
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

	addTool(mcp.NewTool("list_watchlist_shares",
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

	addTool(mcp.NewTool("list_shared_watchlists",
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

	addTool(mcp.NewTool("list_watchlist_audit",
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

	addTool(mcp.NewTool("list_price_alerts",
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

	addTool(mcp.NewTool("create_price_alert",
		mcp.WithDescription("Create a price alert (above/below). mode=one_time (default) fires once; mode=repeating re-fires on each re-cross after returning to the safe side."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair symbol e.g. BTCUSDT")),
		mcp.WithString("condition", mcp.Required(), mcp.Description("above | below")),
		mcp.WithNumber("targetPrice", mcp.Required(), mcp.Description("Threshold price")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist (default binance)")),
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

	addTool(mcp.NewTool("create_orderbook_alert",
		mcp.WithDescription("Create a live order-book alert. kind=imbalance (condition above=buy pressure, below=sell) with threshold 0.05–0.95, or kind=wall (condition bid|ask|any). Checked in the background from the local book. Repeating by default: fires when the condition appears, stays quiet while it remains true, re-arms after it clears."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("imbalance | wall")),
		mcp.WithString("condition", mcp.Required(), mcp.Description("imbalance: above|below; wall: bid|ask|any")),
		mcp.WithNumber("threshold", mcp.Description("Imbalance |value| 0.05–0.95, or wall min share 0–1 (0 = any detected wall)")),
		mcp.WithNumber("rangePct", mcp.Description("±% of mid to analyze (default 2)")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist (default binance)")),
		mcp.WithString("mode", mcp.Description("repeating (default) | one_time")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		kind, err := req.RequireString("kind")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		condition, err := req.RequireString("condition")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateOrderBookAlert(ctx, clientID, req.GetString("exchange", "binance"), symbol, kind, condition, req.GetFloat("threshold", 0), req.GetFloat("rangePct", 0), req.GetString("mode", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("delete_price_alert",
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

	addTool(mcp.NewTool("get_alert_webhook",
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

	addTool(mcp.NewTool("set_alert_webhook",
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

	addTool(mcp.NewTool("delete_alert_webhook",
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

	addTool(mcp.NewTool("create_api_key",
		mcp.WithDescription("Create a named API key for a clientId. permission=read (GET only) or trade. Secret is returned once. Use the main account, not another user key."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Label e.g. Trading bot")),
		mcp.WithString("permission", mcp.Description("read (default) or trade")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateAPIKey(ctx, clientID, name, req.GetString("permission", "read"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_api_keys",
		mcp.WithDescription("List named API keys for a clientId (metadata only, no secrets)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListAPIKeys(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("revoke_api_key",
		mcp.WithDescription("Revoke a named API key so it can no longer authenticate."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Key id from list_api_keys")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RevokeAPIKey(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_portfolio",
		mcp.WithDescription("Create a named paper-trading portfolio with starting cash. A client may have multiple books (e.g. Main and Risky). Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("startingBalance", mcp.Required(), mcp.Description("Starting cash e.g. 10000")),
		mcp.WithString("currency", mcp.Description("Accounting currency default USDT")),
		mcp.WithString("name", mcp.Description("Book name; default Main. Must be unique per client.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		bal, err := req.RequireFloat("startingBalance")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateNamedPortfolio(ctx, clientID, bal, req.GetString("currency", "USDT"), req.GetString("name", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_portfolios",
		mcp.WithDescription("List paper portfolios for a client (id, name, cash). Select one with portfolioId on other tools."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPortfolios(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("rename_portfolio",
		mcp.WithDescription("Rename a paper portfolio. Names are unique per client."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Required(), mcp.Description("Book id or current name")),
		mcp.WithString("name", mcp.Required(), mcp.Description("New display name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("portfolioId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RenamePortfolio(ctx, clientID, id, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("delete_portfolio",
		mcp.WithDescription("Delete a paper portfolio and all of its positions, orders, and history. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Required(), mcp.Description("Book id or name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("portfolioId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeletePortfolio(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("share_portfolio",
		mcp.WithDescription("Share one of your paper portfolios with another client as viewer (read) or trader (can place/cancel orders). Owner only. Deposit/withdraw/delete/share stay owner-only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name")),
		mcp.WithString("granteeClientId", mcp.Required(), mcp.Description("Client to share with")),
		mcp.WithString("role", mcp.Required(), mcp.Description("viewer or trader")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
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
		raw, err := api.SharePortfolio(ctx, clientID, req.GetString("portfolioId", ""), grantee, role)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("update_portfolio_share",
		mcp.WithDescription("Change a paper portfolio share role (viewer|trader). Owner only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name")),
		mcp.WithString("granteeClientId", mcp.Required(), mcp.Description("Shared-with client")),
		mcp.WithString("role", mcp.Required(), mcp.Description("viewer or trader")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
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
		raw, err := api.UpdatePortfolioShare(ctx, clientID, req.GetString("portfolioId", ""), grantee, role)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("revoke_portfolio_share",
		mcp.WithDescription("Remove another client's access to a paper portfolio. Owner only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name")),
		mcp.WithString("granteeClientId", mcp.Required(), mcp.Description("Client to revoke")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		grantee, err := req.RequireString("granteeClientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.RevokePortfolioShare(ctx, clientID, req.GetString("portfolioId", ""), grantee)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_portfolio_shares",
		mcp.WithDescription("List who you shared a paper portfolio with (owner)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("portfolioId", mcp.Description("Optional book id; omit for all books")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPortfolioShares(ctx, clientID, req.GetString("portfolioId", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_shared_portfolios",
		mcp.WithDescription("List paper portfolios other clients shared with you, plus your role (viewer|trader)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Your client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListSharedPortfolios(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio",
		mcp.WithDescription("Get paper portfolio cash, positions, realized/unrealized P&L (mark-to-market). Pass portfolioId when the client has more than one book. Works for shared books if you have access."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name; required if multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.GetPortfolio(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("deposit_portfolio_cash",
		mcp.WithDescription("Add virtual cash to a paper portfolio. Does not count as trading profit. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Positive cash to add")),
		mcp.WithString("note", mcp.Description("Optional label (e.g. salary)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amt, err := req.RequireFloat("amount")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.DepositPortfolioCash(ctx, clientID, amt, req.GetString("note", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("withdraw_portfolio_cash",
		mcp.WithDescription("Withdraw available virtual cash from a paper portfolio. Does not count as trading loss. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Positive cash to withdraw")),
		mcp.WithString("note", mcp.Description("Optional label")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amt, err := req.RequireFloat("amount")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.WithdrawPortfolioCash(ctx, clientID, amt, req.GetString("note", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("transfer_portfolio_cash",
		mcp.WithDescription("Move virtual cash from one of your paper portfolios to another. Owner only. Recorded as transfer_out/transfer_in, not deposit/withdrawal. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("fromPortfolioId", mcp.Description("Source book id or name")),
		mcp.WithString("toPortfolioId", mcp.Required(), mcp.Description("Destination book id or name")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Positive cash to move")),
		mcp.WithString("note", mcp.Description("Optional label")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		toID, err := req.RequireString("toPortfolioId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amt, err := req.RequireFloat("amount")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.TransferPortfolioCash(ctx, clientID, req.GetString("fromPortfolioId", ""), toID, amt, req.GetString("note", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_portfolio_cash_movements",
		mcp.WithDescription("List paper portfolio deposits, withdrawals, and internal transfers (newest first), including the opening balance."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("limit", mcp.Description("Max rows (default 50)")),
		mcp.WithNumber("offset", mcp.Description("Pagination offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListPortfolioCashMovements(ctx, clientID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio_performance",
		mcp.WithDescription("Paper portfolio equity over 1d/1w/1m/3m plus P&L amount and percent from the start of that window. Use for history/charts. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("period", mcp.Description("Lookback: 1d, 1w (default), 1m, or 3m")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.GetPortfolioPerformance(ctx, clientID, req.GetString("period", "1w"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio_risk_limits",
		mcp.WithDescription("Get optional paper risk limits and live status (daily loss %, per-coin weights, whether new buys/margin opens are blocked). Limits never close positions."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPortfolioRiskLimits(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("set_portfolio_risk_limits",
		mcp.WithDescription("Set optional paper risk limits. maxDailyLossPct e.g. 5 blocks new buys/margin when today's MTM loss hits 5%. maxAssetWeightPct e.g. 30 blocks new buys that would push one coin over 30%. Omit a value to disable that rule. Does not close positions."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("maxDailyLossPct", mcp.Description("0 or omit to disable; else 0.01-100")),
		mcp.WithNumber("maxAssetWeightPct", mcp.Description("0 or omit to disable; else 0.01-100")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var daily, weight *float64
		if v := req.GetFloat("maxDailyLossPct", 0); v > 0 {
			daily = &v
		}
		if v := req.GetFloat("maxAssetWeightPct", 0); v > 0 {
			weight = &v
		}
		raw, err := api.PutPortfolioRiskLimits(ctx, clientID, daily, weight)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("clear_portfolio_risk_limits",
		mcp.WithDescription("Remove all paper risk limits for a client."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeletePortfolioRiskLimits(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_paper_trading_costs",
		mcp.WithDescription("Paper taker fee and slippage rates per exchange (binance, coinbase, bybit, nasdaq, bist). Fills use slipped last price; buy cash/lot cost include the fee; sell PnL is after the fee. Pending buy reservations cover slip + fee."),
		mcp.WithString("exchange", mcp.Description("Optional venue; omit to list all")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetPaperTradingCosts(ctx, req.GetString("exchange", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("place_portfolio_order",
		mcp.WithDescription("Paper market buy/sell. Fill is last price plus adverse slippage; a taker fee is charged. Buy lot cost includes the fee; sell realized PnL is after the fee. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("side", mcp.Required(), mcp.Description("buy | sell")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base asset quantity")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithString("lotMethod", mcp.Description("fifo (default) or lifo — sell lot matching")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original fill")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
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
		raw, err := api.PlacePortfolioOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol, side, qty, req.GetString("lotMethod", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_portfolio_trades",
		mcp.WithDescription("List paper trade history. Each fill includes slipped price, lastPrice, and taker fee."),
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

	addTool(mcp.NewTool("list_portfolio_lots",
		mcp.WithDescription("List paper tax lots (open remaining buys). Sells use FIFO or LIFO against these lots."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name")),
		mcp.WithString("exchange", mcp.Description("Filter venue")),
		mcp.WithString("symbol", mcp.Description("Filter pair")),
		mcp.WithString("status", mcp.Description("open (default) | closed | all")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListPortfolioLots(ctx, clientID, req.GetString("exchange", ""), req.GetString("symbol", ""), req.GetString("status", "open"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("place_portfolio_pending_order",
		mcp.WithDescription("Place a paper pending order: limit_buy, limit_sell, stop_loss, or trailing_stop. Buy reservations include slippage and the taker fee. Fills use slipped last price. For trailing_stop use trailType+trailValue. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("type", mcp.Required(), mcp.Description("limit_buy | limit_sell | stop_loss | trailing_stop")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base asset quantity")),
		mcp.WithNumber("triggerPrice", mcp.Description("Limit or stop price (not used for trailing_stop)")),
		mcp.WithString("trailType", mcp.Description("trailing_stop: percent | offset")),
		mcp.WithNumber("trailValue", mcp.Description("trailing_stop: fraction e.g. 0.05 or fixed offset")),
		mcp.WithString("lotMethod", mcp.Description("fifo (default) or lifo for sell types")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithString("timeInForce", mcp.Description("gtc (default) | ioc | fok")),
		mcp.WithString("expiresAt", mcp.Description("RFC3339 expiry for GTC only")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original order")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
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
			req.GetString("trailType", ""), req.GetFloat("trailValue", 0), req.GetString("lotMethod", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("place_portfolio_oco_order",
		mcp.WithDescription("Place a paper OCO: take-profit limit sell + stop-loss for the same quantity. Full fill of one cancels the other; partial fill shrinks both remainings. Same tick fills at most one leg. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base size for both legs")),
		mcp.WithNumber("takeProfitPrice", mcp.Required(), mcp.Description("Limit sell price (must be above stop)")),
		mcp.WithNumber("stopLossPrice", mcp.Required(), mcp.Description("Stop-loss trigger (must be below take-profit)")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithString("expiresAt", mcp.Description("RFC3339 expiry for GTC legs")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original pair")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
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

	addTool(mcp.NewTool("place_portfolio_bracket_order",
		mcp.WithDescription("Place a paper bracket: limit-buy entry with take-profit + stop-loss. Exits stay pending until entry fills; exit size tracks filled qty; exits are OCO. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Entry size")),
		mcp.WithNumber("entryPrice", mcp.Required(), mcp.Description("Limit buy price")),
		mcp.WithNumber("takeProfitPrice", mcp.Required(), mcp.Description("Limit sell above entry")),
		mcp.WithNumber("stopLossPrice", mcp.Required(), mcp.Description("Stop below entry")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithString("expiresAt", mcp.Description("RFC3339 expiry")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original bracket")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
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

	addTool(mcp.NewTool("list_portfolio_orders",
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

	addTool(mcp.NewTool("get_portfolio_order",
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

	addTool(mcp.NewTool("amend_portfolio_order",
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

	addTool(mcp.NewTool("cancel_all_portfolio_orders",
		mcp.WithDescription("Cancel all open paper pending orders for a client, or only one market when symbol is set. Releases reservations. Empty result is success."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Description("When set, cancel only this pair (e.g. BTCUSDT); omit for all markets")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist; default binance when symbol is set")),
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

	addTool(mcp.NewTool("cancel_portfolio_order",
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

	addTool(mcp.NewTool("create_recurring_buy",
		mcp.WithDescription("Create a named paper recurring buy: cash amount at market on daily|weekly|monthly|interval. Use weekday (monday) for weekly, dayOfMonth (1-31) for salary day, intervalHours (e.g. 12) for interval. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Cash notional per run e.g. 500")),
		mcp.WithString("frequency", mcp.Required(), mcp.Description("daily | weekly | monthly | interval")),
		mcp.WithString("name", mcp.Description("Label e.g. Salary Day Buy")),
		mcp.WithString("weekday", mcp.Description("Weekly: monday..sunday")),
		mcp.WithNumber("dayOfMonth", mcp.Description("Monthly salary day 1-31")),
		mcp.WithNumber("intervalHours", mcp.Description("Interval frequency: 1-168 hours")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
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

	addTool(mcp.NewTool("update_recurring_buy",
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

	addTool(mcp.NewTool("list_recurring_buys",
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

	addTool(mcp.NewTool("get_recurring_buy",
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

	addTool(mcp.NewTool("pause_recurring_buy",
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

	addTool(mcp.NewTool("resume_recurring_buy",
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

	addTool(mcp.NewTool("delete_recurring_buy",
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

	addTool(mcp.NewTool("list_recurring_buy_runs",
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

	addTool(mcp.NewTool("create_portfolio_basket",
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

	addTool(mcp.NewTool("list_portfolio_baskets",
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

	addTool(mcp.NewTool("get_portfolio_basket",
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

	addTool(mcp.NewTool("update_portfolio_basket",
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

	addTool(mcp.NewTool("delete_portfolio_basket",
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

	addTool(mcp.NewTool("preview_portfolio_rebalance",
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

	addTool(mcp.NewTool("rebalance_portfolio_basket",
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

	addTool(mcp.NewTool("set_margin_mode",
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

	addTool(mcp.NewTool("adjust_margin",
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

	addTool(mcp.NewTool("repay_margin_debt",
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

	addTool(mcp.NewTool("place_margin_order",
		mcp.WithDescription("Paper margin open: long|short, leverage 1-10, market or limit (uses account isolated|cross mode). Optional stopLoss/takeProfit. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("side", mcp.Required(), mcp.Description("long | short")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Base quantity")),
		mcp.WithNumber("leverage", mcp.Required(), mcp.Description("1–10")),
		mcp.WithString("type", mcp.Description("market (default) | limit")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithNumber("limitPrice", mcp.Description("Required for limit")),
		mcp.WithNumber("stopLoss", mcp.Description("Optional stop-loss price")),
		mcp.WithNumber("takeProfit", mcp.Description("Optional take-profit price")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original open")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
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

	addTool(mcp.NewTool("list_margin_positions",
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

	addTool(mcp.NewTool("close_margin_position",
		mcp.WithDescription("Close all or part of a paper margin position at market. quantity 0 = full close."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("quantity", mcp.Description("Partial size; omit/0 for full")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original close")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
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

	addTool(mcp.NewTool("set_margin_brackets",
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

	addTool(mcp.NewTool("list_margin_orders",
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

	addTool(mcp.NewTool("cancel_margin_order",
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

	addTool(mcp.NewTool("list_margin_trades",
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

	addTool(mcp.NewTool("create_scanner_rule",
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
			"maDirection":    req.GetString("maDirection", "golden_cross"),
			"volumeLookback": req.GetFloat("volumeLookback", 20), "volumeMinRatio": req.GetFloat("volumeMinRatio", 2),
		}
		raw, err := api.CreateScannerRule(ctx, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_scanner_rules",
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

	addTool(mcp.NewTool("delete_scanner_rule",
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

	addTool(mcp.NewTool("analyze_swing",
		mcp.WithDescription("Analyze one symbol with the swing engine (4h+1d, Wilder RSI/ADX/ATR, quality gates, ATR structure stops). Informational only."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. ETHUSDT")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist, default binance")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ex := req.GetString("exchange", "binance")
		raw, err := api.AnalyzeSwing(ctx, ex, symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_swing_setups",
		mcp.WithDescription("Scan the client's watchlist for quality-gated swing setups (entry/stop/TP). Informational only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("exchange", mcp.Description("Optional venue filter")),
		mcp.WithNumber("limit", mcp.Description("Max symbols, default 25")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ScanSwingSetups(ctx, clientID, req.GetString("exchange", ""), int(req.GetFloat("limit", 25)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_scanner_results",
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

	addTool(mcp.NewTool("start_export",
		mcp.WithDescription("Start a background export of the user's watchlist, shares, alerts, backtests, and/or paper portfolios as json or csv. Only one active export per client. Poll get_export for progress; download via HTTP when completed."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("format", mcp.Description("json (default) or csv")),
		mcp.WithString("sections", mcp.Description("Comma-separated: watchlist,shares,alerts,backtests,portfolios (default all)")),
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

	addTool(mcp.NewTool("get_export",
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

	addTool(mcp.NewTool("list_exports",
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

	addTool(mcp.NewTool("cancel_export",
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

	addTool(mcp.NewTool("preview_import",
		mcp.WithDescription("Preview restoring a prior JSON export of watchlist/shares/alerts/backtests/portfolios. Returns valid/invalid/willAdd counts without applying. Pass export JSON as content. Then call confirm_import with mode merge|replace."),
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

	addTool(mcp.NewTool("confirm_import",
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

	addTool(mcp.NewTool("get_import",
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

	addTool(mcp.NewTool("list_imports",
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

	addTool(mcp.NewTool("cancel_import",
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

	addTool(mcp.NewTool("create_price_diff_watch",
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

	addTool(mcp.NewTool("list_price_diff_watches",
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

	addTool(mcp.NewTool("get_price_diff_watch",
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

	addTool(mcp.NewTool("delete_price_diff_watch",
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

	addTool(mcp.NewTool("list_price_diff_opportunities",
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

	addTool(mcp.NewTool("get_price_diff_opportunity",
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

	addTool(mcp.NewTool("health",
		mcp.WithDescription("Check Swyngora MCP connectivity (in-process when embedded in API server)."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.Health(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("unhealthy: %v", err)), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("realtime_stream_info",
		mcp.WithDescription("Describe the WebSocket realtime API: subscribe/unsubscribe prices and paper portfolio order/position updates. Use when a user asks how live prices or portfolio updates work."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, _ := json.Marshal(map[string]any{
			"path":        "/api/v1/ws",
			"httpInfo":    "GET /api/v1/realtime",
			"protocol":    1,
			"maxSymbols":  100,
			"auth":        "Same as REST (Bearer / X-API-Key). Browsers may pass ?token= and ?clientId=.",
			"reconnect":   "On reconnect, resend subscribe_prices and subscribe_portfolio. Server snapshots current state.",
			"clientTypes": []string{"subscribe_prices", "unsubscribe_prices", "subscribe_portfolio", "unsubscribe_portfolio", "ping"},
			"serverTypes": []string{"hello", "ack", "price", "portfolio", "error", "pong"},
			"access":      "Portfolio events only if the client can view that book (owner, trader, or viewer).",
			"exampleSubscribePrices": map[string]any{
				"type":    "subscribe_prices",
				"symbols": []map[string]string{{"exchange": "binance", "symbol": "BTCUSDT"}},
			},
			"exampleSubscribePortfolio": map[string]any{
				"type": "subscribe_portfolio", "portfolioId": "<book-id>",
			},
		})
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})
}
