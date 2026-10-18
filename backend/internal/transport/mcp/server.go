package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/account"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/apikey"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/fundingarb"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/radar"
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
	ScanOrderBookImpact(ctx context.Context, exchange, quote, side string, notional float64, limit int) (json.RawMessage, error)
	ScanSuspiciousWalls(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error)
	GetBookGaps(ctx context.Context, exchange, symbol string, rangePct float64) (json.RawMessage, error)
	ScanBookGaps(ctx context.Context, exchange, quote string, rangePct float64, limit int) (json.RawMessage, error)
	GetWideSpread(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	ScanWideSpreads(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error)
	GetMarketLiquidity(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetOrderBookHeatmap(ctx context.Context, exchange, symbol, group string, windowSec int) (json.RawMessage, error)
	GetLiquidations(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetLiquidationOverview(ctx context.Context, exchange, window string, limit int) (json.RawMessage, error)
	GetLiquidationLevels(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error)
	GetLiquidationCascade(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	ScanLiquidationCascades(ctx context.Context, exchange string) (json.RawMessage, error)
	GetOpenInterest(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetPerpDepth(ctx context.Context, exchange, symbol string, rangePct float64) (json.RawMessage, error)
	CompareSpotPerpLiquidity(ctx context.Context, symbol string, rangePct float64) (json.RawMessage, error)
	GetPerpDepthOverview(ctx context.Context, exchange string, rangePct float64, limit int) (json.RawMessage, error)
	GetPerpBookHistory(ctx context.Context, exchange, symbol string, windowSec int) (json.RawMessage, error)
	GetDerivativesBoard(ctx context.Context, exchange string, limit int) (json.RawMessage, error)
	GetDerivativesGrid(ctx context.Context, exchange, metric, rangeID string, limit int) (json.RawMessage, error)
	GetOptions(ctx context.Context, exchange, asset string) (json.RawMessage, error)
	GetFundingRate(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error)
	GetFundingArb(ctx context.Context, symbol string, notional, holdHours float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error)
	ScanFundingArb(ctx context.Context, quote string, notional, holdHours float64, feeBinancePct, feeBybitPct *float64, limit int) (json.RawMessage, error)
	GetFundingArbHistory(ctx context.Context, symbol, from, to string, notional float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error)
	CreateFundingArbWatch(ctx context.Context, clientID, symbol string, notional, holdHours, minProfit float64, quote string, limit int, durationHours float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error)
	ListFundingArbWatches(ctx context.Context, clientID string) (json.RawMessage, error)
	GetFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	UpdateFundingArbWatch(ctx context.Context, clientID, id string, notional, holdHours, minProfit *float64, quote *string, limit *int, durationHours *float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error)
	PauseFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ResumeFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	DeleteFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListFundingArbSignals(ctx context.Context, clientID, status string, limit int) (json.RawMessage, error)
	GetFundingArbSignal(ctx context.Context, clientID, id string) (json.RawMessage, error)
	GetLongShortRatio(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error)
	GetFuturesHistory(ctx context.Context, metric, exchange, symbol, from, to, bucket string, limit int) (json.RawMessage, error)
	GetPositioningAnchor(ctx context.Context, exchange, symbol, anchor, bucket, from, to string) (json.RawMessage, error)
	GetHistorySpan(ctx context.Context, symbol string) (json.RawMessage, error)
	GetUnusual(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	ScanUnusual(ctx context.Context, exchange, metric string, limit int) (json.RawMessage, error)
	GetLiquidationHunt(ctx context.Context, exchange, symbol string, weights domain.HuntScoreWeights) (json.RawMessage, error)
	GetLiquidationHuntHeatmap(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error)
	GetLiquidationPrintHeatmap(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error)
	GetLiquidationOpenLevels(ctx context.Context, exchange, symbol, rawRange, anchor string) (json.RawMessage, error)
	GetLiquidationMaxPain(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetSqueezeRisk(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetPositioning(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetVenueDivergence(ctx context.Context, symbol string) (json.RawMessage, error)
	GetTakerFlow(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetNetPosition(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetParticipation(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetVolumeShare(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetPrintSize(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetPrintSizeOverview(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error)
	GetBotTracker(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetOrderCount(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetOrderCountOverview(ctx context.Context, exchange, quote string, limit int) (json.RawMessage, error)
	GetCVD(ctx context.Context, exchange, symbol, rangeID, anchor string) (json.RawMessage, error)
	GetTapeBacktest(ctx context.Context, exchange, symbol, condition, side string, threshold float64, from, to string) (json.RawMessage, error)
	GetMarketPressure(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetVolumeProfile(ctx context.Context, exchange, symbol, window, startTime, endTime string, tickSize float64) (json.RawMessage, error)
	GetFlowAtPrices(ctx context.Context, exchange, symbol, window, startTime, endTime string, tickSize, minPrice, maxPrice float64) (json.RawMessage, error)
	GetCandleFootprint(ctx context.Context, exchange, symbol, interval, startTime, endTime string) (json.RawMessage, error)
	GetVWAP(ctx context.Context, exchange, symbol, window, startTime, endTime string) (json.RawMessage, error)
	GetAround(ctx context.Context, exchange, symbol, at, window, during string) (json.RawMessage, error)
	CompareAround(ctx context.Context, exchange, symbol, from, to, window, during string) (json.RawMessage, error)
	FindAroundMoves(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (json.RawMessage, error)
	GetAroundPrecursors(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (json.RawMessage, error)
	GetAroundSimilar(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during, fields, weights, minCoverage, horizons, at string) (json.RawMessage, error)
	GetForensics(ctx context.Context, exchange, symbol, at, window, during, lookback, interval, direction string, minReturnPct float64, limit int, fields, weights, minCoverage, horizons, asOf string) (json.RawMessage, error)
	ScanForensics(ctx context.Context, exchange, quote, lookback, interval, direction string, minReturnPct float64, limit int, window, during, fields, weights, minCoverage, horizons string) (json.RawMessage, error)
	ReviewSetupRadar(ctx context.Context, exchange, symbol, since string, limit int) (json.RawMessage, error)
	GetAbsorption(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetLiquiditySweeps(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetSweepRecovery(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetVolumeSurge(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	ScanVolumeSurges(ctx context.Context, exchange, quote string, minRatio float64, limit int) (json.RawMessage, error)
	GetRSIHeatmap(ctx context.Context, exchange, quote, intervals, sort string, limit, period int) (json.RawMessage, error)
	GetBasis(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetCorrelation(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetBreadth(ctx context.Context, exchange string, limit int) (json.RawMessage, error)
	GetFearGreed(ctx context.Context) (json.RawMessage, error)
	GetCoinSituation(ctx context.Context, symbol, at string) (json.RawMessage, error)
	GetFillPocket(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error)
	GetVolatility(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetSnapshot(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetLevels(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	GetWhales(ctx context.Context, exchange, symbol string, minNotional float64, limit int) (json.RawMessage, error)
	GetBookHistory(ctx context.Context, exchange, symbol, at, from, to string, limit int) (json.RawMessage, error)
	CompareBookHistory(ctx context.Context, exchange, symbol, from, to string) (json.RawMessage, error)
	GetBookLiquidity(ctx context.Context, exchange, symbol, window, startTime, endTime string, minPrice, maxPrice float64) (json.RawMessage, error)
	GetIcebergs(ctx context.Context, exchange, symbol string, minNotional float64) (json.RawMessage, error)
	GetCandles(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error)
	GetSupply(ctx context.Context, asset string) (json.RawMessage, error)
	GetHolders(ctx context.Context, asset string) (json.RawMessage, error)
	GetAssetProfile(ctx context.Context, asset string) (json.RawMessage, error)
	ListSpot(ctx context.Context, exchange, query, quote, sort, order, tag string, limit, offset int) (json.RawMessage, error)
	GetIndicators(ctx context.Context, exchange, symbol, interval string, limit, rsiPeriod int, emaPeriods string) (json.RawMessage, error)
	DetectPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error)
	ScanPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error)
	ListExchanges(ctx context.Context) (json.RawMessage, error)
	GetFxRates(ctx context.Context) (json.RawMessage, error)
	ListDelistSchedule(ctx context.Context, exchange string) (json.RawMessage, error)
	GetPostDelist(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error)
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
	CreateSituationAlert(ctx context.Context, clientID, exchange, symbol, mode string) (json.RawMessage, error)
	CreateTapeAlert(ctx context.Context, clientID, exchange, symbol, kind, condition string, target float64, mode string) (json.RawMessage, error)
	CreateOrderBookAlert(ctx context.Context, clientID, exchange, symbol, kind, condition string, threshold, rangePct float64, mode string) (json.RawMessage, error)
	CreateLiquidationFeedAlert(ctx context.Context, clientID, exchange string, minDownSeconds float64, mode string) (json.RawMessage, error)
	CreateLiquidationCascadeAlert(ctx context.Context, clientID, exchange, symbol, minGrade, mode string) (json.RawMessage, error)
	CreateLiquidationNotionalAlert(ctx context.Context, clientID, exchange, symbol, side string, notional float64, window, mode string) (json.RawMessage, error)
	CreateLiquidationLevelAlert(ctx context.Context, clientID, exchange, symbol, minReliability string, minConfidence float64, mode string) (json.RawMessage, error)
	CreateSetupRadarAlert(ctx context.Context, clientID, exchange, symbol, lean string, minSimilarity float64, mode string) (json.RawMessage, error)
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
	GetSituationChange(ctx context.Context, clientID, exchange, symbol, tradeID string) (json.RawMessage, error)
	GetSituationRecord(ctx context.Context, clientID string) (json.RawMessage, error)
	GetSituationLeave(ctx context.Context, clientID string) (json.RawMessage, error)
	CreateAPIKey(ctx context.Context, clientID, name, permission string) (json.RawMessage, error)
	ListAPIKeys(ctx context.Context, clientID string) (json.RawMessage, error)
	RevokeAPIKey(ctx context.Context, clientID, id string) (json.RawMessage, error)
	GetPortfolioPerformance(ctx context.Context, clientID, period string) (json.RawMessage, error)
	DepositPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error)
	WithdrawPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error)
	ListPortfolioCashMovements(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	TransferPortfolioCash(ctx context.Context, clientID, fromPortfolioID, toPortfolioID string, amount float64, note string) (json.RawMessage, error)
	ListTransferAttempts(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	StressPortfolio(ctx context.Context, clientID, moves, from, to string) (json.RawMessage, error)
	GetPortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error)
	PutPortfolioRiskLimits(ctx context.Context, clientID string, maxDailyLossPct, maxAssetWeightPct, maxExposurePct *float64) (json.RawMessage, error)
	DeletePortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error)
	GetPaperTradingCosts(ctx context.Context, exchange string) (json.RawMessage, error)
	PreviewPortfolioOrder(ctx context.Context, clientID, exchange, symbol, side, orderType, product, trailType, lotMethod string, quantity, triggerPrice, takeProfitPrice, stopLossPrice, trailValue, limitPrice float64, leverage int, riskPercent float64) (json.RawMessage, error)
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
	CreateRecurringBuyPlan(ctx context.Context, clientID, exchange, symbol string, amount float64, frequency, startAt, name, weekday string, dayOfMonth, intervalHours int, timeZone string, hour, minute int, maxPrice, budget float64, endDate, endsAt string) (json.RawMessage, error)
	UpdateRecurringBuyPlan(ctx context.Context, clientID, id, name, frequency, weekday, startAt string, amount float64, dayOfMonth, intervalHours int, timeZone string, hour, minute int, maxPrice, budget float64, endDate, endsAt string, clearEnds bool) (json.RawMessage, error)
	ListRecurringBuyPlans(ctx context.Context, clientID string) (json.RawMessage, error)
	GetRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	PauseRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ResumeRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	DeleteRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int) (json.RawMessage, error)
	CompleteRecurringBuyRun(ctx context.Context, clientID, planID, runID string) (json.RawMessage, error)
	CreatePortfolioBasket(ctx context.Context, clientID, name, targetsJSON string) (json.RawMessage, error)
	ListPortfolioBaskets(ctx context.Context, clientID string) (json.RawMessage, error)
	GetPortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error)
	UpdatePortfolioBasket(ctx context.Context, clientID, id, name, targetsJSON string) (json.RawMessage, error)
	DeletePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error)
	PreviewPortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error)
	RebalancePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListPortfolioRebalances(ctx context.Context, clientID, basketID string, limit, offset int) (json.RawMessage, error)
	GetPortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ContinuePortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error)
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
	UpdateScannerRule(ctx context.Context, args map[string]any) (json.RawMessage, error)
	DeleteScannerRule(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListScannerResults(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	StartBacktest(ctx context.Context, clientID, ruleID, profileID, exchange, symbol, rangeStart, rangeEnd string, startingBalance, riskPct, maxOpenRiskPct float64, useTP2 bool, tp1ClosePct float64, stopAfterTP1 string, feePct, slipPct float64) (json.RawMessage, error)
	ListBacktests(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	GetBacktest(ctx context.Context, clientID, id string) (json.RawMessage, error)
	CancelBacktest(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListBacktestSignals(ctx context.Context, clientID, id string, limit, offset int) (json.RawMessage, error)
	StartExport(ctx context.Context, clientID, format string, sections []string) (json.RawMessage, error)
	GetExport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListExports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	CancelExport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	PreviewImport(ctx context.Context, clientID, fileName, format string, fileBytes []byte) (json.RawMessage, error)
	ConfirmImport(ctx context.Context, clientID, id, mode string, resolutions []domain.CustomSignalResolution) (json.RawMessage, error)
	GetImport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListImports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error)
	CancelImport(ctx context.Context, clientID, id string) (json.RawMessage, error)
	CreatePriceDiffWatch(ctx context.Context, clientID, symbol string, notional, minProfit, minNetDiffPct, minDurationSec, feeBinance, feeCoinbase, feeBybit float64, exchanges []string) (json.RawMessage, error)
	ListPriceDiffWatches(ctx context.Context, clientID string) (json.RawMessage, error)
	GetPriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	UpdatePriceDiffWatch(ctx context.Context, clientID, id string, notional, minProfit, minNetDiffPct, minDurationSec, feeBinance, feeCoinbase, feeBybit *float64, exchanges []string) (json.RawMessage, error)
	PausePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ResumePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	DeletePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ListPriceDiffOpportunities(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error)
	GetPriceDiffOpportunity(ctx context.Context, clientID, id string) (json.RawMessage, error)
	QuotePriceDiff(ctx context.Context, symbol, buyExchange, sellExchange string, notional, quantity, feeBuyPct, feeSellPct, minNetDiffPct float64) (json.RawMessage, error)
	QuotePriceDiffOpportunity(ctx context.Context, clientID, id string, notional, quantity float64) (json.RawMessage, error)
	ScanPriceDiffQuotes(ctx context.Context, symbol string, notional, quantity, feeBinance, feeCoinbase, feeBybit, minNetDiffPct, minProfitPct, minProfitAmount float64) (json.RawMessage, error)
	QuotePriceDiffWatch(ctx context.Context, clientID, watchID string, notional, quantity, minProfitPct, minProfitAmount float64) (json.RawMessage, error)
	AnalyzeSwing(ctx context.Context, exchange, symbol string) (json.RawMessage, error)
	ScanSwingSetups(ctx context.Context, clientID, exchange string, limit int) (json.RawMessage, error)
	ListCustomSignalFactors(ctx context.Context) (json.RawMessage, error)
	ListCustomSignalProfiles(ctx context.Context, clientID string) (json.RawMessage, error)
	CreateCustomSignalProfile(ctx context.Context, clientID, name string, weights map[string]float64, isDefault bool) (json.RawMessage, error)
	UpdateCustomSignalProfile(ctx context.Context, clientID, id, name string, weights map[string]float64, isDefault *bool) (json.RawMessage, error)
	DuplicateCustomSignalProfile(ctx context.Context, clientID, id, name string) (json.RawMessage, error)
	SetDefaultCustomSignalProfile(ctx context.Context, clientID, id string) (json.RawMessage, error)
	DeleteCustomSignalProfile(ctx context.Context, clientID, id string) (json.RawMessage, error)
	ScanCustomSignals(ctx context.Context, clientID, profileID, exchange string, limit int, weights domain.CustomSignalWeights, hasWeights bool) (json.RawMessage, error)
	Health(ctx context.Context) (json.RawMessage, error)
}

// ServerOptions configures the MCP server.
type ServerOptions struct {
	// Data is preferred (in-process). If nil, APIBaseURL is used via HTTP client.
	Data       DataPort
	Accounts   *account.Service // when set, tools with clientId require an active account
	APIBaseURL string
	APIToken   string
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
		data = NewAPIClientWithAuth(opts.APIBaseURL, 0, opts.APIToken)
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
func NewInProcessServer(marketSvc *market.Service, watchSvc *watchlist.Service, alertSvc *pricealert.Service, portfolioSvc *portfolio.Service, scannerSvc *scanner.Service, exportSvc *exportsvc.Service, importSvc *dataimport.Service, priceDiffSvc *pricediff.Service, apiKeySvc *apikey.Service, accountSvc *account.Service, swingSvc *swing.Service, fundingArbSvc *fundingarb.Service, radarSvc *radar.Service) *server.MCPServer {
	return NewServer(ServerOptions{
		Data:     &Backend{Market: marketSvc, Watch: watchSvc, Alerts: alertSvc, Portfolio: portfolioSvc, Scanner: scannerSvc, Export: exportSvc, Import: importSvc, PriceDiff: priceDiffSvc, APIKeys: apiKeySvc, Swing: swingSvc, FundingArb: fundingArbSvc, Radar: radarSvc},
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
	if id == nil {
		return nil
	}
	if id.Master && id.DenyImpersonate && !id.Loopback {
		if strings.TrimSpace(req.GetString("clientId", "")) != "" {
			return fmt.Errorf("%w: master token cannot select a tenant from this client", domain.ErrForbidden)
		}
		return nil
	}
	if !id.UserKey {
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
		mcp.WithDescription("Stored spot order-book snapshots: bid/ask levels, spread, total liquidity, imbalance, and large walls. Pass at for the book nearest that time (RFC3339 or unix ms); flashes on that snapshot are walls that rested only a few seconds inside the minute and were gone before the ladder was stored. moves are other resting-size changes in that minute that the ladder does not show. Omit at to list recent samples (from/to optional). 1-minute ladders, not a 24h tape. Not financial advice."),
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

	addTool(mcp.NewTool("get_orderbook_liquidity_flow",
		mcp.WithDescription("When resting size at a price shrinks (e.g. 8M bid at 65500 down to 2M), split how much was actually traded vs pulled from the book. Same for asks. series shows liquidity added/pulled/traded over time. Uses stored spot books plus spot prints. window 5m/15m/1h (default 5m) or startTime/endTime (max 1h). Optional minPrice/maxPrice. Not L3 order-by-order. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
		mcp.WithString("window", mcp.Description("5m | 15m | 1h (default 5m). Ignored when startTime/endTime are set.")),
		mcp.WithString("startTime", mcp.Description("Range start (RFC3339 or unix ms). Max 1h.")),
		mcp.WithString("endTime", mcp.Description("Range end (RFC3339 or unix ms).")),
		mcp.WithNumber("minPrice", mcp.Description("Only include this price and above (e.g. 65400).")),
		mcp.WithNumber("maxPrice", mcp.Description("Only include this price and below (e.g. 65600).")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetBookLiquidity(ctx, req.GetString("exchange", "all"), symbol, req.GetString("window", ""), req.GetString("startTime", ""), req.GetString("endTime", ""), req.GetFloat("minPrice", 0), req.GetFloat("maxPrice", 0))
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

	addTool(mcp.NewTool("scan_market_impact",
		mcp.WithDescription("Rank liquid coins by slippage for the same quote-sized market order. Walks live spot books. Buy consumes asks, sell consumes bids. exchange=all merges Binance, Coinbase, and Bybit. Default notional is 100000 quote. Wider slippage ranks first. exhausted means the visible book could not fill the whole order. Simulation only. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
		mcp.WithString("quote", mcp.Description("Quote asset, default USDT")),
		mcp.WithString("side", mcp.Description("buy or sell (default buy)")),
		mcp.WithNumber("notional", mcp.Description("Shared quote size (default 100000)")),
		mcp.WithNumber("limit", mcp.Description("How many liquid pairs (default 20, max 30)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ScanOrderBookImpact(ctx, req.GetString("exchange", "all"), req.GetString("quote", "USDT"), req.GetString("side", "buy"), req.GetFloat("notional", 0), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_suspicious_walls",
		mcp.WithDescription("Rank suspicious and flickering spot walls across the liquid USDT pairs. Suspicious means the same wall was added and pulled at least 4 times inside 2 minutes. A flicker rested under about 20 seconds and then left. exchange is binance, coinbase, bybit, or all. Empty until the desk has been sampling. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
		mcp.WithString("quote", mcp.Description("Quote asset, default USDT")),
		mcp.WithNumber("limit", mcp.Description("How many walls to return (default 40, max 80)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ScanSuspiciousWalls(ctx, req.GetString("exchange", "all"), req.GetString("quote", "USDT"), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_book_gaps",
		mcp.WithDescription("Find empty areas in the live spot book inside ±rangePct of mid (default 2%). A gap is a run of 0.1% steps with very little resting size. easier=up means asks are much thinner than bids, so a move up meets less size. easier=down is the mirror. exchange=all sums Binance, Coinbase, and Bybit. One snapshot, not a forecast. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
		mcp.WithNumber("rangePct", mcp.Description("±% of mid (0.5–5, default 2)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetBookGaps(ctx, req.GetString("exchange", "all"), symbol, req.GetFloat("rangePct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_book_gaps",
		mcp.WithDescription("Rank liquid USDT pairs by which side of the spot book is thinner. easier=up means a move up meets less resting size. The thinnest one-sided books come first. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
		mcp.WithString("quote", mcp.Description("Quote asset, default USDT")),
		mcp.WithNumber("rangePct", mcp.Description("±% of mid (0.5–5, default 2)")),
		mcp.WithNumber("limit", mcp.Description("How many liquid pairs (default 20, max 30)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ScanBookGaps(ctx, req.GetString("exchange", "all"), req.GetString("quote", "USDT"), req.GetFloat("rangePct", 0), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_wide_spread",
		mcp.WithDescription("Compare a coin's current top-of-book bid-ask spread with the median of earlier samples of that same book. opened is true at 1.5× its own typical. Samples collect only while this process watches the book. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetWideSpread(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_wide_spreads",
		mcp.WithDescription("Rank spot books whose latest bid-ask spread is at least 1.5× that book's own typical spread. Widest ratio first. Empty until this process has sampled the books. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | all (default all)")),
		mcp.WithString("quote", mcp.Description("Quote label, default USDT")),
		mcp.WithNumber("limit", mcp.Description("How many coins (default 20, max 40)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ScanWideSpreads(ctx, req.GetString("exchange", "all"), req.GetString("quote", "USDT"), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_orderbook_heatmap",
		mcp.WithDescription("Resting bid/ask size over time (order heatmap / bookmap-style tape). Not executed volume. Windows up to 30 minutes are the live 1s tape. Longer windows (up to 30 days) use stored 1-minute ladders, including brief walls and other size moves between those minutes. window is lookback seconds (60–2592000, default 600). Spot only."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance (default) | coinbase | bybit")),
		mcp.WithString("group", mcp.Description("Price bucket size e.g. 0.1; omit for a suggested default")),
		mcp.WithNumber("window", mcp.Description("Lookback seconds 60–2592000 (default 600). Above 1800 uses stored book history, about 30 days.")),
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
		mcp.WithDescription("Futures liquidations for a coin over the last 5 minutes, 1 hour, 4 hours, and 24 hours. Returns long vs short notional, count, the biggest hit, and feed health (lastEventAt, lastSeenAt, gaps in the last 6h, missingSeconds still unfilled after history backfill, missing venues). After a reconnect each venue fills its own hole from history when available; overlapping prints are not counted twice. exchange=all (default) sums Binance USD-M, Bybit linear, and OKX USDT-swap but never substitutes one venue for another; combined live/complete require every stream. Prefer this for 'how much BTC was liquidated' or 'is the Bybit feed down'."),
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

	addTool(mcp.NewTool("get_liquidation_overview",
		mcp.WithDescription("Market-wide futures liquidations: long vs short notional for the last 1 hour, 4 hours, 12 hours, and 24 hours, plus coins ranked by total notional for a treemap. Totals are stored force-order prints, plus prints newer than the newest saved row. feed lists last print, disconnects, and missingSeconds still unfilled after a same-venue history fill. Combined live/complete require Binance, Bybit, and OKX; a missing venue is listed, not replaced. window (1h|4h|12h|24h, default 24h) selects which window ranks coins."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("window", mcp.Description("1h | 4h | 12h | 24h — ranks coins (default 24h)")),
		mcp.WithNumber("limit", mcp.Description("Max coins (default 50, max 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := int(req.GetFloat("limit", 0))
		raw, err := api.GetLiquidationOverview(ctx, req.GetString("exchange", "all"), req.GetString("window", "24h"), limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidation_levels",
		mcp.WithDescription("CoinGlass-style liquidation bar chart. symbol=BTCUSDT returns estimated notional at each price, split by 10x/25x/50x/100x, plus the walk from last to that bar (cumLong/cumShort/cumByLeverage). Live bars shrink bands that rarely printed later (longConfidence/shortConfidence). Bands that did print are also shortened by sizeScale, the share of the predicted amount that later showed up, once that band has enough hits; prints larger than the model do not grow the next map. review rebuilds past maps and scores later same-venue liquidations (1h/4h/12h). Combined lastPrice is the OI-weighted last of venues that have one. symbol=all returns stored force-order time bars for the requested range (12h|24h|3d|7d|14d|30d). A zero bar was quiet or was not recorded."),
		mcp.WithString("symbol", mcp.Description("Pair e.g. BTCUSDT, or all (default all)")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = combined)")),
		mcp.WithString("range", mcp.Description("12h | 24h | 3d | 7d | 14d | 30d (default 24h)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetLiquidationLevels(ctx, req.GetString("exchange", "all"), req.GetString("symbol", "all"), req.GetString("range", "24h"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidation_cascade",
		mcp.WithDescription("Detect a liquidation cascade: a short burst of long or short liquidations far above that stream's own typical rate (1m / 5m / 15m vs the prior 6 hours). episodes lists each wave in the last 24h (when it started, how long, long/short notional, price move). Combined waves include startedBy, venueStarts.lagSec (seconds after the first venue print), venueStarts.totalNotional and sharePct (how much of the wave that venue produced), and largestBy; orderUnknown when two venues share the same first-print second. Binance, Bybit, and OKX are scored separately; a combined episode is the same-side wave on more than one venue at once. both.agree is true only when the same side is cascade or hotter on Binance, Bybit, and OKX together. A quiet or opposite venue keeps it off. symbol=all is market-wide (pooled coins). Prefer this for 'when did the BTC liquidation cascade start', 'which exchange started first', 'where did most of the wave come from', or 'how much was liquidated during the wave'."),
		mcp.WithString("symbol", mcp.Description("Pair e.g. BTCUSDT, or all for the market (default all)")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetLiquidationCascade(ctx, req.GetString("exchange", "all"), req.GetString("symbol", "all"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_liquidation_cascades",
		mcp.WithDescription("Market-wide liquidation cascade scan: pooled market risk plus coins currently bursting (elevated / cascade / extreme). Binance, Bybit, and OKX stay separate; a hit's both flag means the same side is cascade or hotter on all three. Prefer this for 'which coins are in a liquidation cascade'."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ScanLiquidationCascades(ctx, req.GetString("exchange", "all"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_options_overview",
		mcp.WithDescription("BTC/ETH options desk from Deribit: listed open interest, 24h volume, put/call ratios, DVOL (30-day implied vol), 30-day realized vol from Binance daily closes, volCompare (dvol / realized / gap in vol points — e.g. DVOL 70 vs realized 45 is gap +25), ATM IV near 30 days, IV term structure, call/put skew, expiry OI, per-expiry 1-SD expected move plus max pain, dealer gamma (gamma.netUsd / callUsd / putUsd: USD delta per 1% spot move; short calls / long puts; regime long=stabler, short=faster), gamma flip (price where net GEX crosses zero; flipSide above/below), history (1h/4h/24h/7d change + series for OI, put/call, DVOL, 30-day realized, volGap, net GEX, flip vs spot; windows also say if regime or flip-side changed), strikeOi (call and put open-interest change at each strike over those windows; opened/closed when a strike appears or leaves), moves (unusual short-window OI / put-call / IV vs that series' typical, plus gamma_to_short / gamma_to_long / flip_crossed_up / flip_crossed_down), and vsMarket findings when implied vol or put/call lean disagrees with live spot range or futures funding / account long-short. asset is BTC or ETH (BTCUSDT is accepted). Not Binance/Bybit futures OI. Informational only."),
		mcp.WithString("asset", mcp.Required(), mcp.Description("BTC or ETH (or BTCUSDT / ETHUSDT)")),
		mcp.WithString("exchange", mcp.Description("deribit (default)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		asset, err := req.RequireString("asset")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetOptions(ctx, req.GetString("exchange", "deribit"), asset)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_open_interest",
		mcp.WithDescription("Futures open interest for a coin: current outstanding size plus how much OI opened or closed in the last 5 minutes, 1 hour, 4 hours, and 24 hours — USDT amount and percent, per venue and combined. flow=opened|closed|flat; openedValue/closedValue is the USDT notional added or closed. Also includes funding on the funding field. contracts is base-asset size (e.g. BTC); value is USDT notional. Binance USD-M + Bybit linear + OKX USDT-swap. exchange=all (default) sums those three and keeps per-venue rows. A failed or missing asked venue blanks the combined current and windows and is named in missing, and is not counted as zero. A window that lacks a sample on any asked venue is blank even if the others have it. exchange=hyperliquid is that venue alone: value is mark times coin size, USDC margin lined up with the USDT name, and it is not inside exchange=all. Prefer this for 'how much OI was added or closed'."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | hyperliquid | all (default all)")),
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

	addTool(mcp.NewTool("compare_spot_perp_liquidity",
		mcp.WithDescription("Compare resting liquidity in the spot book and the perpetual book at the same distance from mid (0.1% / 0.5% / 1%). Says which book is thinner and which side (bid or ask) is thinner on each. Spot sums Binance, Coinbase, and Bybit. Perp sums Binance USD-M, Bybit linear, and OKX. Binance and Bybit rows compare that venue's own two books. A venue counts in a band only when its book reaches that % on both sides. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("rangePct", mcp.Description("Headline ±% of mid (default 1, max 1)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CompareSpotPerpLiquidity(ctx, symbol, req.GetFloat("rangePct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_perp_depth",
		mcp.WithDescription("How much perpetual liquidity is waiting below vs above the price. For example 40M bid within 1% below mid vs 15M ask above. Binance USD-M, Bybit linear, and OKX USDT-swap separately plus combined. rangePct default 1; bands also include 0.1/0.25/0.5/2. Combined adds each venue's own ±% band. This is the futures book, not spot. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithNumber("rangePct", mcp.Description("Headline ±% of mid (default 1, max 5)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPerpDepth(ctx, req.GetString("exchange", "all"), symbol, req.GetFloat("rangePct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_perp_depth_overview",
		mcp.WithDescription("How much total perpetual bid vs ask liquidity is waiting across the liquid coins we track, plus which coins make up the biggest part of that book. Example: 2.4B bid vs 1.8B ask within 1% of mid; BTC 48%, ETH 17%. bands show how close that book is to mid (0.1% may be 80M / 8% of the 1% book while 1% is 1.0B) and which coins fill each distance. windows (15m/1h/4h) compare bid vs ask share of that book (bids 55% → 65%) and drivers name the coins that moved the mix. history is total bid+ask each minute for the last 6 hours with BTC's close. moves are the largest jumps and drops of at least 5% that finish within 15 minutes, with BTC during the move and 15 minutes later. drivers name the coins behind each move and span says whether one coin, a few coins, or the whole book caused it. follow says whether depth came back, kept going, or held in the next 15 minutes. venues and venueSpan say whether Binance, Bybit, and OKX each moved at least 5% the same way (all), at least two did (some), or only one venue did. Sampled while the desk runs. exchange=all (default) adds Binance USD-M, Bybit linear, and OKX USDT-swap per coin. rangePct default 1. limit default 20 (max 40). Futures book, not spot. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithNumber("rangePct", mcp.Description("Headline ±% of mid (default 1, max 5)")),
		mcp.WithNumber("limit", mcp.Description("How many liquid perps (default 20, max 40)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetPerpDepthOverview(ctx, req.GetString("exchange", "all"), req.GetFloat("rangePct", 0), req.GetInt("limit", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_perp_book_history",
		mcp.WithDescription("Stored perpetual bid and ask within 0.1%, 0.5%, and 1% of mid for one coin. A minute is kept only when Binance, Bybit, and OKX are all fresh. Bands the snapshot does not reach are left out. windowSec is the lookback (default 24h, max 30d). The liquid USDT list is sampled while the server runs. exchange=all is the combined book. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithNumber("windowSec", mcp.Description("Lookback in seconds (default 86400, max 30 days)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPerpBookHistory(ctx, req.GetString("exchange", "all"), symbol, req.GetInt("windowSec", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_derivatives_board",
		mcp.WithDescription("One sortable board for the liquid USDT perpetuals: mark price, predicted funding, open interest, 1h and 24h open-interest change, crowd long percent, top-minus-crowd gap, day CVD, 15m print-size change, 15m order-count ratio, the farthest unusual reading, 5m liquidation heat, 1% bid share, 1h taker buy share, and 24h long and short liquidations. exchange=all reads Binance USD-M, Bybit linear, and OKX. A combined number is present only when every expected venue is fresh for that column. Crowd long is the account long share, weighted by open interest when that open interest is also fresh. gapPct is top-trader long percent minus crowd long percent. Bybit does not publish a top-trader ratio, so gapScope names who is in the average and Bybit is unsupported. cvdDay is aggressive buy minus sell since 00:00 UTC. cvdSplit is the current run where price and CVD disagree (price_up_cvd_down or price_down_cvd_up) and cvdSplitDuration says how long it has run, or agrees when they are in line. The cell stays blank until the tape reaches 00:00 UTC, the newest minute is fresh, and the newest bar has a price. Combined uses the summed series and stays blank unless every expected venue's 24h tape is complete. A later bar where price and CVD move the same way ends the run. Flat bars after it do not. A 4-hour window that disagrees without a current run is agrees, not a run. printChangePct compares the last 15 minutes of average fill size with the 15 minutes before. orderRatio is (buy − sell) / (buy + sell) over those same 15 minutes, blank unless every expected venue's tape covers the window. unusualMetric names the farthest high or low reading versus that coin's own stored samples and unusualVenue says which venue it is; venues are not averaged, and a coin with no standout stays blank. liqHeat is the hottest 5-minute liquidation grade versus that coin's own typical rate, blank until the venue has been watched for 6 hours; liqHeatVenue is set when the venues disagree. bidPct is resting bid share within 1% of mid. Taker buy share sums the last hour of aggressive volume; an incomplete hour stays blank. definite is false when any coin is incomplete on the original columns; missing and stale venues are named and are not counted as zero. The 24h open-interest change uses the stored sample within 90 minutes of 24 hours ago when the live series does not reach that hour. change24hMissing names each venue with no sample, and that venue is not treated as zero. recorded says how far back open interest, the spot book, flow-at-prices, and stored liquidation prints go for that coin. exchange=hyperliquid fills mark, open interest, and the hourly funding rate. Liquidation, tape, and book columns stay blank and are named unsupported. Hyperliquid is not part of exchange=all. limit default 36, max 100 (the recorder keeps 100). Opening an unusual standout shows the same clauses as get_coin_situation for that symbol. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | hyperliquid | all (default all)")),
		mcp.WithNumber("limit", mcp.Description("How many liquid perps (default 36, max 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetDerivativesBoard(ctx, req.GetString("exchange", "all"), req.GetInt("limit", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_derivatives_grid",
		mcp.WithDescription("Time grid for the liquid USDT perpetuals. Coins run down the side. 24h is one column per clock hour. 7d is one column per UTC day. metric is funding (latest settled rate in the cell, or the latest predicted rate when that cell has no settlement), oi (percent change from the previous cell), cvd (aggressive buy minus sell), or liquidations (stored force-order notional; a watched hour with no print is a real zero). exchange=all reads Binance USD-M, Bybit linear, and OKX. A cell stays blank until every expected venue has that column. missing names the venue that is absent, and that venue is not counted as zero. venues lists each expected exchange and its own number when that exchange has the column. A missing exchange on that list has no value and is not a zero. The open column also stays blank when its newest sample is older than five minutes, or when the tape does not reach both edges. heat on a liquidation cell is that hour or day versus the coin's own median. exchange=hyperliquid is mark, open interest, and hourly funding only. It is not folded into exchange=all. CVD and liquidation cells stay blank and name Hyperliquid. They are not zeros. limit default 36, max 100. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | hyperliquid | all (default all)")),
		mcp.WithString("metric", mcp.Description("funding | oi | cvd | liquidations (default funding)")),
		mcp.WithString("range", mcp.Description("24h | 7d (default 24h)")),
		mcp.WithNumber("limit", mcp.Description("How many liquid perps (default 36, max 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetDerivativesGrid(ctx, req.GetString("exchange", "all"), req.GetString("metric", "funding"), req.GetString("range", "24h"), req.GetInt("limit", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_funding_rate",
		mcp.WithDescription("Perpetual futures funding rate: predicted next payment plus recent settled history. rate is decimal (0.0001 = 0.01%); ratePct is percent. payer=long means longs pay shorts at settlement. Binance USD-M + Bybit linear + OKX USDT-swap. Per-venue rows stay separate. pressure is OI-weighted combined predicted funding (USDT open interest share) so an 8B book outweighs a 1B book; pressureWindows (1h/4h/24h) and pressureMoves show how that blend changed. Combined pressure stays blank until every asked venue is in the blend and is named in pressure.missing. A venue with no open interest is still skipped inside a complete blend. exchange=hyperliquid is the hourly rate and is not inside pressure or exchange=all. Prefer this for 'what is the funding rate' or carry analysis."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | hyperliquid | all (default all)")),
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

	addTool(mcp.NewTool("get_funding_arb",
		mcp.WithDescription("Cross-exchange perpetual funding: long the cheaper-funding venue and short the richer one on Binance USD-M vs Bybit linear. Counts only published settlement clocks in the hold window (8h at 00:00/08:00/16:00 UTC, 4h at 00/04/08/12/16/20 UTC). No clock in the window means funding is 0. trade is omitted when after-fee net is not positive. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("notional", mcp.Description("Quote size on each leg (default 10000)")),
		mcp.WithNumber("holdHours", mcp.Description("Hold window for the horizon payout (default 24)")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance taker fee percent (default paper 0.10)")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit taker fee percent (default paper 0.10)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fb, fy := optionalMCPFee(req, "feeBinancePct"), optionalMCPFee(req, "feeBybitPct")
		raw, err := api.GetFundingArb(ctx, symbol, req.GetFloat("notional", 0), req.GetFloat("holdHours", 0), fb, fy)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_funding_arb",
		mcp.WithDescription("Rank liquid USDT coins that are after-fee winners on the Binance vs Bybit funding spread. Only rows whose published settlements in holdHours beat round-trip taker fees. Not financial advice."),
		mcp.WithNumber("notional", mcp.Description("Quote size on each leg (default 10000)")),
		mcp.WithNumber("holdHours", mcp.Description("Hold window for the horizon payout (default 24)")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance taker fee percent (default paper 0.10)")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit taker fee percent (default paper 0.10)")),
		mcp.WithString("quote", mcp.Description("Quote asset (default USDT)")),
		mcp.WithNumber("limit", mcp.Description("How many top-volume coins to scan (default 15, max 40)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fb, fy := optionalMCPFee(req, "feeBinancePct"), optionalMCPFee(req, "feeBybitPct")
		raw, err := api.ScanFundingArb(ctx, req.GetString("quote", "USDT"), req.GetFloat("notional", 0), req.GetFloat("holdHours", 0), fb, fy, int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_funding_arb_history",
		mcp.WithDescription("Past Binance, Bybit, and OKX funding opportunities for one coin in a date range. Uses settled funding prints only. Long is the lowest known rate and short is the highest. Each run is a stretch of the same long/short pair; listed only when settled funding minus round-trip fees is positive. The first clock of a run is the entry signal — that payment is not collected because the position was not open before it. A direction flip at settlement still pays the old sides, then starts a new run. from/to are RFC3339 or YYYY-MM-DD (UTC), max 30 days. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("from", mcp.Required(), mcp.Description("Range start RFC3339 or YYYY-MM-DD")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Range end RFC3339 or YYYY-MM-DD")),
		mcp.WithNumber("notional", mcp.Description("Quote size on each leg (default 10000)")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance taker fee percent (default paper 0.10)")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit taker fee percent. Also the OKX leg (default paper 0.10)")),
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
		fb, fy := optionalMCPFee(req, "feeBinancePct"), optionalMCPFee(req, "feeBybitPct")
		raw, err := api.GetFundingArbHistory(ctx, symbol, from, to, req.GetFloat("notional", 0), fb, fy)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_funding_arb_watch",
		mcp.WithDescription("Follow funding-arb opportunities. Omit symbol (or use scan/*) to continuously scan liquid coins and notify when a new coin's after-fee net is at least minProfit. Same coin + same long/short while still above the floor does not re-notify; it must drop below and come back to notify again. Optional symbol follows one pair only. Not financial advice."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Description("Pair e.g. BTCUSDT, or omit / scan / * for the full scan follow")),
		mcp.WithNumber("minProfit", mcp.Required(), mcp.Description("Minimum after-fee profit in quote currency (e.g. 10)")),
		mcp.WithNumber("notional", mcp.Description("Quote size on each leg (default 10000)")),
		mcp.WithNumber("holdHours", mcp.Description("Hold window for the horizon payout (default 24)")),
		mcp.WithString("quote", mcp.Description("Scan quote asset (default USDT)")),
		mcp.WithNumber("limit", mcp.Description("How many top-volume coins to scan (default 15, max 40)")),
		mcp.WithNumber("durationHours", mcp.Description("How long the follow should run; omit for no end")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance taker fee percent (default paper 0.10)")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit taker fee percent (default paper 0.10)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		minP, err := req.RequireFloat("minProfit")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fb, fy := optionalMCPFee(req, "feeBinancePct"), optionalMCPFee(req, "feeBybitPct")
		raw, err := api.CreateFundingArbWatch(ctx, clientID, req.GetString("symbol", ""), req.GetFloat("notional", 0), req.GetFloat("holdHours", 0), minP, req.GetString("quote", ""), int(req.GetFloat("limit", 0)), req.GetFloat("durationHours", 0), fb, fy)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_funding_arb_watches",
		mcp.WithDescription("List funding-arb follow watches for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListFundingArbWatches(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_funding_arb_watch",
		mcp.WithDescription("Get one funding-arb follow watch by id."),
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
		raw, err := api.GetFundingArbWatch(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("update_funding_arb_watch",
		mcp.WithDescription("Change minProfit, notional, holdHours, fees, or scan size on an existing funding-arb follow. Does not delete signals."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("watchId", mcp.Required(), mcp.Description("Watch id")),
		mcp.WithNumber("minProfit", mcp.Description("New after-fee profit floor")),
		mcp.WithNumber("notional", mcp.Description("New quote size on each leg")),
		mcp.WithNumber("holdHours", mcp.Description("New hold window hours")),
		mcp.WithString("quote", mcp.Description("Scan quote asset")),
		mcp.WithNumber("limit", mcp.Description("How many top-volume coins to scan")),
		mcp.WithNumber("durationHours", mcp.Description("How long the follow should keep running from now; 0 = no end")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance taker fee percent")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit taker fee percent")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("watchId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var quote *string
		if q := strings.TrimSpace(req.GetString("quote", "")); q != "" {
			quote = &q
		}
		var limit *int
		if _, ok := req.GetArguments()["limit"]; ok {
			n := int(req.GetFloat("limit", 0))
			limit = &n
		}
		raw, err := api.UpdateFundingArbWatch(ctx, clientID, id,
			optionalMCPFee(req, "notional"), optionalMCPFee(req, "holdHours"), optionalMCPFee(req, "minProfit"),
			quote, limit, optionalMCPFee(req, "durationHours"), optionalMCPFee(req, "feeBinancePct"), optionalMCPFee(req, "feeBybitPct"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("pause_funding_arb_watch",
		mcp.WithDescription("Pause a funding-arb follow without deleting it or its signals."),
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
		raw, err := api.PauseFundingArbWatch(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("resume_funding_arb_watch",
		mcp.WithDescription("Resume a paused funding-arb follow."),
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
		raw, err := api.ResumeFundingArbWatch(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("delete_funding_arb_watch",
		mcp.WithDescription("Delete a funding-arb follow watch and its signals."),
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
		raw, err := api.DeleteFundingArbWatch(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_funding_arb_signal",
		mcp.WithDescription("One funding-arb crossing plus predicted-vs-settled review after funding clocks pass."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Signal id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetFundingArbSignal(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_funding_arb_signals",
		mcp.WithDescription("List funding-arb crossings above minProfit (status open|closed|all). Includes predicted-vs-settled review when clocks have passed."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("status", mcp.Description("open | closed | all")),
		mcp.WithNumber("limit", mcp.Description("Max rows (default 50)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListFundingArbSignals(ctx, clientID, req.GetString("status", ""), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_long_short_ratio",
		mcp.WithDescription("Futures long/short: current is all-account count (not size). On Binance, topAccounts is the top 20% of traders by margin (account count) and topPositions is their position size. gap is one signed value: top long% minus crowd long% in percentage points (top 65% vs crowd 55% → +10). gapWindows (1h/4h/24h) and gapMoves say if that spread widened, narrowed, or flipped. divergence is when top traders get more long while price falls (or more short while price rises), with since/duration (e.g. +3pp long vs −1.5% over 30m). split.opposite means the crowd and bigger books lean different ways — hunt/squeeze/positioning then treat size as who is positioned. ratio is long/short (1 = even). bias is long if ratio≥1.05, short if ≤0.95. 5-minute samples. Binance USD-M, Bybit linear, and OKX USDT-swap. OKX top traders are the top 5% by position value. exchange=all returns each venue separately (not averaged). Prefer this for 'are more traders long or short', 'are top traders fading the crowd', 'is the crowd vs top-trader gap widening', or 'are top traders diverging from price'."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
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
		mcp.WithDescription("Durable stored history of futures open interest, funding, long/short ratio, or liquidations for Binance USD-M, Bybit linear, and OKX USDT-swap. Survives restarts. metric=open_interest|funding|long_short|liquidations. bucket=5m|30m|2h|4h returns one sample per step across from..to, capped at 90 days: last open interest, last funding (settled preferred), last crowd and top-trader long share, or summed liquidation notional. A bucket with no sample is omitted and is not zero. Bybit long/short has no top-trader share. exchange=hyperliquid is sampled open interest and settled hourly funding. It has no long/short ratio and no liquidation history. Omit bucket for the newest raw rows. The coin chart uses these steps for the candles on screen and draws open interest and the top-minus-crowd gap since 00:00 UTC and since the current 4-hour open. Prefer this for past values, not the live snapshot tools."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("metric", mcp.Required(), mcp.Description("open_interest | funding | long_short | liquidations")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | hyperliquid | all (default all)")),
		mcp.WithString("from", mcp.Description("RFC3339 or unix ms start")),
		mcp.WithString("to", mcp.Description("RFC3339 or unix ms end")),
		mcp.WithString("bucket", mcp.Description("5m | 30m | 2h | 4h. Omit for the newest raw rows.")),
		mcp.WithNumber("limit", mcp.Description("Max raw rows, default 200. Ignored when bucket is set.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		metric, err := req.RequireString("metric")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetFuturesHistory(ctx, metric, req.GetString("exchange", "all"), symbol, req.GetString("from", ""), req.GetString("to", ""), req.GetString("bucket", ""), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_positioning_anchor",
		mcp.WithDescription("Restart stored open interest, funding, and liquidation notional at one clicked candle for Binance USD-M, Bybit linear, and OKX USDT-swap. Open interest and funding are the change from the last sample at or before that candle. A venue with no sample by then is named in missingOpenInterest or missingFunding and is left off. Liquidation notional is the running sum from that candle, short positive and long negative. A print at or before the candle starts the sum at zero. A venue whose first print is later is listed in lateLiquidations and the line starts at that print. A venue with no prints is named in missingLiquidations. A missing venue is not zero. exchange=all includes Hyperliquid on open interest and hourly funding when a sample exists. Hyperliquid publishes no liquidation prints, so it is not named in missingLiquidations. anchor is the candle time (RFC3339). bucket is 5m, 30m, 2h, or 4h; omit it and the step follows from..to the same way the coin chart does. from and to bound the stored window, capped at 90 days. Informational only — not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("anchor", mcp.Required(), mcp.Description("RFC3339 candle time to restart from")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | hyperliquid | all (default all)")),
		mcp.WithString("bucket", mcp.Description("5m | 30m | 2h | 4h. Omit to follow the from..to span.")),
		mcp.WithString("from", mcp.Description("RFC3339 or unix ms start of the stored window")),
		mcp.WithString("to", mcp.Description("RFC3339 or unix ms end of the stored window")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		anchor, err := req.RequireString("anchor")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPositioningAnchor(ctx, req.GetString("exchange", "all"), symbol, anchor, req.GetString("bucket", ""), req.GetString("from", ""), req.GetString("to", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_history_span",
		mcp.WithDescription("How long this coin has actually been recorded: oldest stored futures open interest, spot book, flow-at-prices, and liquidation print per venue. A venue with no rows is omitted. The startup archive backfills the liquid USDT perpetuals, up to 100 names with the seed pairs first, and only as far as each public endpoint documents. Binance USD-M aggTrades are walked into taker minutes and flow-at-prices for that same list, about 7 days, one hour at a time. Bybit and OKX flow stay on the minutes this process recorded. Spot books and flow-at-prices are kept about 30 days. Recorded liquidation prints are kept with the futures archive (about 90 days); public refill is still about 7 days. spanSec is from that oldest sample to now. Informational only — not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetHistorySpan(ctx, symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_unusual_context",
		mcp.WithDescription("Says whether funding, the latest 24h open-interest change, the last completed hour of liquidations, the last completed UTC day of CVD, and the latest hour's average spread are high or low versus this coin's own stored history on that venue. Print size stays unknown because only the previous 15 minutes are kept. exchange=all returns Binance, Bybit, and OKX separately. The span is the stored sample span. Too few samples stay unknown. Informational only — not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetUnusual(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_unusual",
		mcp.WithDescription("Ranks the liquid USDT perpetuals, up to 100 with the seed pairs first, whose funding, latest 24h open-interest change, last completed liquidation hour, last completed CVD day, or latest hour of average spread is high or low versus that coin's own stored samples. Typical readings and readings with too few samples are left off. Venues stay separate and are not merged. Print size is not scanned. metric filters one series. Opening a row shows the same clauses as get_coin_situation for that symbol. Informational only — not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithString("metric", mcp.Description("all | funding | open_interest | liquidations | cvd | spread (default all)")),
		mcp.WithNumber("limit", mcp.Description("How many liquid names to score, 5 to 36. Default 36.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ScanUnusual(ctx, req.GetString("exchange", "all"), req.GetString("metric", "all"), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("estimate_liquidation_hunt",
		mcp.WithDescription("Hypothetical only: Binance and Bybit walk their own spot book. OKX is priced from its USDT-swap mark and has no spot book, so its walk cost stays empty. Where is the main long/short pressure, how much spot buy/sell the visible book needs, and a rough desk result (book-only unwind vs assuming part of estimated liquidations become exit flow). Also returns upScore/downScore (0–100) and bias (up|down|even) with reasons: zone distance, visible book cost, price+OI trend, crowding/funding, recent taker and liquidation flow. Optional weight* mix must sum to 100. Requested percents stay as given — a missing factor keeps its percent and adds 0 (neutral 50); remaining factors are not increased. scoreCompare is default vs applied (up/down scores, lean, separate upFactors/downFactors, and upLargestChange/downLargestChange for the factor that moved that direction the most) for analysis. Direction score = 50 + sum(effect). upCascade/downCascade list zones from last price outward. Each step has role (start|self|helped|stall|unreachable|missing|observed), remaining hop after prior assumed exit flow, assistancePct/strength only when the hop can be scored, zoneEst (model|observed|missing), and the path marks feedsUntil / stallsAt where self-fueling stops. Self-fueling means prior flow covers the visible WalkBookToPrice hop cost — remaining and assistance always match that final state. Scores and paths do not change the zone or P&L numbers. coverage says how complete the inputs are (have/need/coverPct/age on time windows; a few minutes of taker is not a full hour; a 2h-old OI print is not 1h OI). Each factor has sharePct (requested percent) and effect (points versus 50); a venue with an error is shown but excluded from the combined lean. Venues are never averaged. Not evidence of exchange behavior. Not financial advice. Long/short is account count; leverage mix is assumed; mark price is multi-venue."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both, separately)")),
		mcp.WithNumber("weightProximity", mcp.Description("Custom mix % for distance to zone. If any weight* is set, omitted factors are 0 and the six must sum to 100. Percents are not rescaled when a factor is missing.")),
		mcp.WithNumber("weightBook", mcp.Description("Custom mix % for spot walk cost")),
		mcp.WithNumber("weightEfficiency", mcp.Description("Custom mix % for liq per spot")),
		mcp.WithNumber("weightTrend", mcp.Description("Custom mix % for price + OI trend")),
		mcp.WithNumber("weightCrowding", mcp.Description("Custom mix % for crowding + funding")),
		mcp.WithNumber("weightFlow", mcp.Description("Custom mix % for 1h taker + 1h liquidations")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		weights, err := domain.ParseHuntScoreWeights(func(k string) string {
			args := req.GetArguments()
			if args == nil {
				return ""
			}
			v, ok := args[k]
			if !ok || v == nil {
				return ""
			}
			return fmt.Sprint(v)
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquidationHunt(ctx, req.GetString("exchange", "all"), symbol, weights)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidation_max_pain",
		mcp.WithDescription("Largest estimated liquidation pockets above and below last price for one coin (CoinGlass-style max pain). Above = shorts that liquidate if price rises; below = longs that liquidate if price falls. Returns price, estimated size, distance from last, and long/short. This is NOT the hunt target (hunt also weighs spot-walk cost). Venues stay separate; combined above/below is the single largest pocket on that side. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both, separately)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquidationMaxPain(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidation_heatmap",
		mcp.WithDescription("CoinGlass-style price × time liquidation intensity map from historical open interest, each venue's own price (never borrowed from another venue), and the hunt leverage mix. Force-order prints are not added here; use get_liquidation_print_heatmap for measured prints. range=12h|24h|3d|7d|14d|30d|90d. Windows past 24 hours read one stored open-interest, funding, and long/short sample per bucket from the archive. A column with no sample stays empty. The review's liquidation check still uses the newest 2,500 prints. Public backfill follows each endpoint: Binance open interest and long/short are 30 days, OKX hourly open interest is the latest 1,440 points, and Bybit open interest goes back to listing (this archive keeps 90 days). 90d is one column per day. Binance columns older than the public 30 days are only hours this process already recorded. OKX prices are that venue's USDT-swap candles. Each venue uses its stored account ratio. A column with no stored ratio, including OKX before the first sample, is an even split. A column with no stored open interest uses that venue's public hourly open-interest history, clamped to the documented lookback (OKX: the latest 1,440 hours), and does not extend the archive. Combined sums a column only when Binance, Bybit, and OKX all have open interest. Public liquidation history refills about 7 days; older prints are only what this process already stored. Returns Binance, Bybit, OKX, and combined grids, the leverage mix, and a review: totals and a per-signal list (price area, 1h/4h/12h hit or miss, time-to-hit, how much later price and liquidation history was available, labeled gaps). Rates use only validated signals. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("range", mcp.Description("12h | 24h | 3d | 7d | 14d | 30d | 90d (default 24h)")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all = three venues + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquidationHuntHeatmap(ctx, req.GetString("exchange", "all"), symbol, req.GetString("range", "24h"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidation_print_heatmap",
		mcp.WithDescription("Measured liquidation prints by price and time. Each cell is stored force-order notional, not an open-interest leverage estimate. Longs are positions closed by a sell. Shorts are positions closed by a buy. range=12h|24h|3d|7d|14d|30d|90d. Public REST refill is about 7 days; older cells are only prints this process already stored. A column is recorded when a print landed there, or when that venue was watched and the column is outside a known feed gap. A quiet recorded column is a real zero with hasData true. A column before the watch, or inside a gap with no print, has hasData false. That flag is unrecorded time. Binance, Bybit, and OKX never borrow each other's prints. Combined sums a column only when all three were recorded. columnsWithOi and coverage count recorded columns. leverageMix is empty and review has no horizons. kind is prints. Each requested book includes printAnchors for long and short notional since 00:00 UTC and since the current 4-hour open. complete is false and missing names the venue when that clock was not fully recorded. A quiet recorded clock is side quiet with zero notionals. Combined anchors are present only for exchange=all. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("range", mcp.Description("12h | 24h | 3d | 7d | 14d | 30d | 90d (default 24h)")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all = three venues + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquidationPrintHeatmap(ctx, req.GetString("exchange", "all"), symbol, req.GetString("range", "24h"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidation_open_levels",
		mcp.WithDescription("Estimated liquidation levels that stay at the price where that slice of open interest was placed, until a later candle trades through them. range=12h|24h|3d|7d|14d|30d|90d. Windows past 24 hours read one stored open-interest sample per bucket from the archive. A column with no sample stays empty. The first column places the whole open interest then on the books. Later columns only add an increase. A traded-through level is not put back. When estimated open interest on a side falls, the oldest open slices close; that is not a liquidation. Longs are hit by a later candle low; shorts by a later candle high. The birth candle cannot hit its own level. Binance, Bybit, and OKX each use only their own price and open interest. Combined sums a column only when all three have both, and combined is withheld (definite=false, empty segments and anchors) when any venue has none. Each definite book reports still-open long, short, and long-minus-short change since 00:00 UTC and since the current 4-hour open. The baseline is the last cumulative column at or before that clock. complete is false when the stored book starts later, and that is not a zero change. Optional anchor (RFC3339) adds the same read from a chosen candle and does not move the window. Segments are capped for drawing; cumulative, openLong, openShort, and delta stay uncapped. The leverage mix is the labeled hunt model with 0.40% maintenance. Binance public open interest is 30 days. Not exchange liquidation-engine output. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("range", mcp.Description("12h | 24h | 3d | 7d | 14d | 30d | 90d (default 24h)")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all = three venues + combined)")),
		mcp.WithString("anchor", mcp.Description("Optional RFC3339 candle time. Adds the still-open size change since that candle. Empty omits it.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquidationOpenLevels(ctx, req.GetString("exchange", "all"), symbol, req.GetString("range", "24h"), req.GetString("anchor", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_squeeze_risk",
		mcp.WithDescription("Long-squeeze and short-squeeze risk scores (0–100) for a coin on Binance USD-M and Bybit linear, plus an OI-weighted combined view. Uses open interest change, funding, account long/short crowding, recent liquidations, and nearby liquidation-pocket estimates. Explains which side is crowded and why risk is high or low. An errored or zero-OI venue withholds the combined blend (agreement=withheld). A zero score is not that blend. Not a prediction or financial advice."),
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
		mcp.WithDescription("Price + open interest positioning for a coin: long_buildup, short_buildup, long_unwinding, or short_covering on Binance and Bybit, plus a general market (combined) read. Short summary and reasons (funding / long-short corroboration). An errored or zero-OI venue withholds the combined blend (agreement=withheld) and missing names that venue, before a combined regime is published. Not a prediction or financial advice."),
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
		mcp.WithDescription("Compare Binance, Bybit, and OKX for one coin. groups lists venues moving the same way. outliers names the venue moving differently when the other two agree. Each metric (OI change, funding, account crowding, price+OI regime) includes a reading per venue. OKX has no account long/short. Not financial advice."),
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

	addTool(mcp.NewTool("get_volume_share",
		mcp.WithDescription("How BTC (or another coin) futures volume splits across Binance USD-M, Bybit linear, and OKX USDT-swap. Example: Binance 55%, Bybit 30%, OKX 15%. Each window (15m / 1h / 4h) also compares to the same length just before so you can see who gained or lost share in the last hour. Prefer this for 'where is the volume' or 'did Binance lose share'. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetVolumeShare(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_participation",
		mcp.WithDescription("Share of futures volume that stuck as new (or closed) open interest: OI USDT delta divided by taker volume. Example: 2B volume and OI +300M is ratio 0.15 (15% new positions); the rest is transfer between existing books. Windows 15m, 1h, and 4h. Binance USD-M + Bybit linear + combined. A venue without a complete window blanks the combined ratio and is named in missing, and is not counted as zero. Prefer this for 'how much of this volume is new positions'. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetParticipation(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_print_size",
		mcp.WithDescription("Average aggressive futures print size (USDT notional per taker fill) on Binance USD-M, Bybit linear, and OKX USDT-swap. Windows are 5m and 15m vs the same length just before. Use this when volume is high and you need to know whether it was a few large market buys/sells or many small ones (e.g. buy avg jumped from 20K to 180K). kind=jumped/faded/stable; read=few_large/many_small. OKX is the newest 500 public trades. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPrintSize(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_print_size_overview",
		mcp.WithDescription("Whole-market average print size. Sums aggressive futures notional and fill counts across the recorder's liquid USDT perpetuals (up to 100, seed pairs first), then divides. Compares the last 5m and 15m with the same length just before, so you can see if prints got larger or smaller. A busier coin weighs more. largerCoins and smallerCoins are one vote per pair. Use this for the whole market, not one coin. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithString("quote", mcp.Description("Quote asset, default USDT")),
		mcp.WithNumber("limit", mcp.Description("How many names from the recorder list (default 100, max 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetPrintSizeOverview(ctx, req.GetString("exchange", "all"), req.GetString("quote", "USDT"), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_bot_tracker",
		mcp.WithDescription("Repeated same-size aggressive futures prints. A size is flagged when it hits at least 4 times inside any 60 seconds during the last 5 minutes. Buy and sell are separate. Sizes within about 0.2% count as the same clip. Prints under 5,000 USDT are ignored. delta is flagged buy prints minus flagged sell prints. Use this when you want to know if the same market-order size keeps coming back, not the average print size. Not proof of a named bot. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetBotTracker(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_order_count_ratio",
		mcp.WithDescription("Market order count ratio: (aggressive buy print count − sell print count) / (buy + sell), from −1 (all sells) to +1 (all buys). This counts prints, not USDT size. Windows 1m, 5m, and 15m. Prefer this for 'are more market orders buying or selling', not for repeated clip size. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetOrderCount(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_order_count_overview",
		mcp.WithDescription("Whole-market order count ratio. Sums aggressive buy and sell print counts across the recorder's liquid USDT perpetuals (up to 100, seed pairs first), then divides. Ratio is −1 (all sells) to +1 (all buys). A busier coin counts for more of the ratio. buyCoins, sellCoins, balancedCoins, and quietCoins are one vote per pair. Use this for whether the market tape is buy-heavy or sell-heavy, not for one coin. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithString("quote", mcp.Description("Quote asset, default USDT")),
		mcp.WithNumber("limit", mcp.Description("How many names from the recorder list (default 100, max 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetOrderCountOverview(ctx, req.GetString("exchange", "all"), req.GetString("quote", "USDT"), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_market_pressure",
		mcp.WithDescription("One −1…+1 score for which side has more pressure when the resting order book, CVD, and absorption disagree. +1 is buy, −1 is sell. When the book looks bid-heavy but CVD is selling, absorption is weighted more so the net is not a wash. change1h is this score minus ~1h ago. Prefer this for 'who has the pressure' or 'book is bullish but CVD is negative'. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetMarketPressure(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_net_position",
		mcp.WithDescription("Estimate net long and net short USDT over 15m, 1h, and 4h by combining open-interest change with taker buy and sell. Positive means that side grew, negative means it shrank. The two nets add up to the open-interest change. Binance and Bybit. Not the account long/short ratio. An incomplete window has empty numeric strings and complete false. A missing venue is named and is not a flat zero. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetNetPosition(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_taker_flow",
		mcp.WithDescription("Aggressive futures buy vs sell volume (who is hitting the book) on Binance USD-M, Bybit linear, and OKX USDT-swap for 5m, 1h, and 4h, plus a combined view and a short read with price/OI/funding. Each venue sums measured prints (price × size). Longer windows stay incomplete until this process has recorded them. This is taker flow, not the account long/short ratio. Not financial advice."),
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
		mcp.WithDescription("Cumulative Volume Delta: running sum of aggressive market-buy minus market-sell notional over time, plotted with price. Futures and spot are separate (spotFutures flags when they disagree). Windows show CVD change over 15m / 1h / 4h / 24h on the returned bars. Anchors reset the sum at 00:00 UTC (day) and at the current 4-hour open, so you can see the move since that clock rather than a rolling lookback. anchor is an optional RFC3339 candle time: each series gains a click anchor that sums bars starting at or after that time and is incomplete when the tape starts later. Choose a range that reaches the candle. range is 24h (5-minute bars, default), 7d (30-minute bars), 30d (2-hour bars), or 90d (4-hour bars). Longer ranges use stored minutes and are complete only when that tape reaches the start. Binance minutes for the liquid USDT list are filled in the background from public aggTrades, about 7 days. Bybit and OKX stay on the minutes this process recorded. Points flag price-up/CVD-down or price-down/CVD-up for the current consecutive run only (a later same-kind split is a new episode). Combined uses the overlapping time range and stays incomplete until every requested venue reaches the window open. Futures combined expects Binance, Bybit, and OKX. Spot combined is spotVenues / spotCombined and expects Binance and Bybit. OKX has no spot CVD. A missing venue is not zero. A minute inside the overlap that one venue missed is left out and is not filled from the other venue. A quiet minute nobody printed is not a gap. whaleDelta and retailDelta split a futures bar when print size was stored: a whale is one aggressive print of at least 100,000 USD, and retail is every smaller print. A minute stored before size was known omits both and is not zero. Combined size is set only when every venue's bar at that step was sized. Spot points do not carry the split. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all = venues + combined)")),
		mcp.WithString("range", mcp.Description("24h | 7d | 30d | 90d (default 24h). Use a range that reaches anchor.")),
		mcp.WithString("anchor", mcp.Description("Optional RFC3339 candle time. Adds a click anchor from that bar. Empty keeps day and 4h only.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetCVD(ctx, req.GetString("exchange", "all"), symbol, req.GetString("range", "24h"), req.GetString("anchor", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_tape_backtest",
		mcp.WithDescription("Replay stored tape for one coin and score the spot close 1 hour, 4 hours, and 24 hours later. condition=cvd is a finished UTC hour (at least 50 minute buckets) whose buy or sell share crosses the threshold after the previous hour was below. condition=liquidation is a 5-minute force-order burst at least N times that side's own median printed bucket (side is the position liquidated: long or short). It does not replay estimated leverage levels. condition=book is the last spot-book sample in a 15-minute block whose bid or ask share crosses the threshold. condition=funding is a settled funding print, in percent, after the previous settled hour was on the other side of the line. Predicted funding is left out. condition=oi is the open-interest percent change from a sample about 24 hours earlier. A missing sample is left out and is not zero. condition=gap is top-trader long share minus the crowd, in percentage points. Bybit has no top-trader ratio, so gap uses Binance and OKX. condition=size is a finished UTC hour whose aggressive notional from prints of at least $100,000 reaches the share. Minutes stored before print size was known are left out. Identical clip repeats are not stored. exchange=all keeps per-venue results and withholds combined unless every expected venue has that bar. Each horizon includes the average, median, win rate, and worst drop. Worst drop is the lowest close versus the entry on the way to that horizon. Zero means price did not close below the entry. Liquidations use the newest 2,500 prints. Books use the newest 500 snapshots. Prices are spot 5-minute closes. A horizon is omitted until that candle has closed. Informational only — not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("cvd, liquidation, funding, oi, and size: binance | bybit | okx | all. book: binance | coinbase | bybit | all. gap: binance | okx | all. Default all.")),
		mcp.WithString("condition", mcp.Description("cvd | liquidation | book | funding | oi | gap | size (default cvd)")),
		mcp.WithString("side", mcp.Description("buy|sell, long|short, bid|ask, up|down, top|crowd, or whale|retail. Empty uses the condition default.")),
		mcp.WithNumber("threshold", mcp.Description("Share 0.55–0.90, liquidation multiple 2–12, funding percent 0.005–0.20, open-interest percent 1–50, crowd gap 1–20 points, or large-print share 0.15–0.80. 0 uses the condition default.")),
		mcp.WithString("from", mcp.Description("RFC3339 start. Empty is 7 days before to.")),
		mcp.WithString("to", mcp.Description("RFC3339 end. Empty is now.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetTapeBacktest(ctx, req.GetString("exchange", "all"), symbol, req.GetString("condition", "cvd"), req.GetString("side", ""), req.GetFloat("threshold", 0), req.GetString("from", ""), req.GetString("to", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_volume_surge",
		mcp.WithDescription("How much higher this coin's volume is than its own typical volume over 5 minutes, 15 minutes, and 1 hour. Typical is the median of the prior ~24 hours. Buy and sell are separate when the venue publishes taker-buy (Binance), so you can see if the spike is one-sided. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetVolumeSurge(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_rsi_heatmap",
		mcp.WithDescription("Ranked Wilder RSI scatter for top listed pairs on one venue. Each item is one coin (stables omitted) with market-cap rank and RSI on a single interval (default 1h). Cached about 60s. Informational only — not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit | nasdaq | bist (default binance)")),
		mcp.WithString("quote", mcp.Description("Quote asset (default USDT)")),
		mcp.WithString("interval", mcp.Description("Candle interval (default 1h)")),
		mcp.WithNumber("limit", mcp.Description("How many top pairs (default 24, max 40)")),
		mcp.WithNumber("period", mcp.Description("RSI period (default 14)")),
		mcp.WithString("sort", mcp.Description("quoteVolume or marketCapCirculating (default quoteVolume)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetRSIHeatmap(ctx, req.GetString("exchange", "binance"), req.GetString("quote", "USDT"), req.GetString("interval", req.GetString("intervals", "")), req.GetString("sort", ""), int(req.GetFloat("limit", 0)), int(req.GetFloat("period", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_volume_surges",
		mcp.WithDescription("Which coins have much more volume than their own typical right now. Scans top 24h-volume USDT pairs and ranks by 5m/15m/1h current-vs-median. Buy/sell split when available. minRatio default 2 (2x typical). Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default binance)")),
		mcp.WithString("quote", mcp.Description("Quote asset (default USDT)")),
		mcp.WithNumber("minRatio", mcp.Description("Minimum current/typical to include (default 2)")),
		mcp.WithNumber("limit", mcp.Description("How many top-volume coins to scan (default 30, max 50)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ex := req.GetString("exchange", "binance")
		raw, err := api.ScanVolumeSurges(ctx, ex, req.GetString("quote", "USDT"), req.GetFloat("minRatio", 0), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_liquidity_sweeps",
		mcp.WithDescription("Liquidity sweep: price pokes a little through a prior high or low that had already turned it back, then comes back. Returns the swept level, how far price went through (excursion), how long it took to reclaim, and quote volume (buy/sell when known) during that poke. 15-minute spot bars, last ~7 days. Binance and Bybit separately. A poke that stays through for more than 2 hours is a breakout, not a sweep. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetLiquiditySweeps(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_sweep_recovery",
		mcp.WithDescription("How long spot depth took to refill after a liquidity sweep. Before is resting size near the swept level just before the poke. The clock starts when price comes back and stops when that side is at least 80% restored, or after two hours. Uses stored 1-minute ladders, so it is empty until the desk has been sampling. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetSweepRecovery(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("review_setup_radar",
		mcp.WithDescription("Review saved Setup Radar hits and what price did after each one (15m / 1h / 4h). Usefulness is how often an up/down lean matched the later move. since=24h|7d|30d (default 7d). Informational only — not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("symbol", mcp.Description("Optional pair to filter, e.g. BTCUSDT")),
		mcp.WithString("since", mcp.Description("24h | 7d | 30d (default 7d)")),
		mcp.WithNumber("limit", mcp.Description("How many saved hits to return (default 50, max 200)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ReviewSetupRadar(ctx, req.GetString("exchange", "all"), req.GetString("symbol", ""), req.GetString("since", "7d"), int(req.GetFloat("limit", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_absorption",
		mcp.WithDescription("When big market buys or sells hit but price barely moves: how much buy/sell volume came in, how far price moved, which side is absorbing (bids absorbing sells, or asks absorbing buys), and how strong that is (score 0–100, weak–extreme). Windows 15m / 1h / 4h / 24h plus the current consecutive run. Binance and Bybit separately plus combined; futures and spot. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetAbsorption(ctx, req.GetString("exchange", "all"), symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_around",
		mcp.WithDescription("What happened around a chosen time for a coin: before, during, and after that move on one tape. Price, volume, buy/sell, VWAP, volume vs typical, volume-profile POC, liquidity sweeps, stored book/futures history, listed BTC/ETH options (OI, put/call, IV, unusual prints) when those archives have samples, and the listed futures curve (contango / backwardation and near-vs-far gap) from mark klines. tapeRead is flow at prices: market buy vs sell at the volume node, where dominance started, and if delta flipped while price kept going. Options lead is the real start inside the unusual lookback, not the 4h/24h window open. venueTiming says which of Binance/Bybit printed first, whether the other followed or the move stayed on one venue, and whether OI or taker flow had already changed before the laggard's real start (not the during-window close). The first signal across price, OI, and takers is the leader only if that OI or taker print kept going until price. If the last OI sample or taker bar is much earlier than the price start, the path is omitted (not continued / faded / reversed). The taker bar at the price start is not pre-price tape. Continued / faded / reversed also need a continuous pre-price taker path at the tape's bar size (no hole longer than one missing 1m or 5m bar, and tape after the start print). window is the lookback/lookforward (default 1h). during is the move starting at `at` (default 15m). Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("at", mcp.Required(), mcp.Description("Event time (RFC3339 or unix ms). Start of the during window.")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
		mcp.WithString("window", mcp.Description("15m | 30m | 1h | 2h | 4h (default 1h)")),
		mcp.WithString("during", mcp.Description("5m | 15m | 30m | 1h (default 15m)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		at, err := req.RequireString("at")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetAround(ctx, req.GetString("exchange", "all"), symbol, at, req.GetString("window", ""), req.GetString("during", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("find_around_moves",
		mcp.WithDescription("Find the important up and down moves in a coin's recent history, then show what happened in the market during each one (price, volume, VWAP, vs typical, POC, sweeps, stored book/futures). Ranked by |return|. lookback 4h/12h/24h/3d/7d (default 24h). minReturnPct default 1.5 (15m) or 2.5 (1h). Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("lookback", mcp.Description("4h | 12h | 24h | 3d | 7d (default 24h)")),
		mcp.WithString("interval", mcp.Description("15m | 1h (default 15m)")),
		mcp.WithString("direction", mcp.Description("up | down | both (default both)")),
		mcp.WithNumber("minReturnPct", mcp.Description("Minimum |return| %% (default 1.5 on 15m, 2.5 on 1h)")),
		mcp.WithNumber("limit", mcp.Description("How many moves to keep (default 8, max 15)")),
		mcp.WithString("window", mcp.Description("Around lookback 15m|30m|1h|2h|4h (default 1h)")),
		mcp.WithString("during", mcp.Description("Override move window 5m|15m|30m|1h (default from the leg)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.FindAroundMoves(ctx, req.GetString("exchange", "all"), symbol, req.GetString("lookback", ""), req.GetString("interval", ""), req.GetString("direction", ""), req.GetFloat("minReturnPct", 0), int(req.GetFloat("limit", 0)), req.GetString("window", ""), req.GetString("during", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("find_around_precursors",
		mcp.WithDescription("Scan a coin's candles for important up and down moves, compare the tape before those moves, and list conditions that show up often — singles and combos that fire together (volume + book + OI, etc.). Combos say whether the group is more common before increases or drops. Common means 60%+ of before-windows with 3+ samples. Default lookback 7d. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("lookback", mcp.Description("4h | 12h | 24h | 3d | 7d (default 7d)")),
		mcp.WithString("interval", mcp.Description("15m | 1h (default 15m)")),
		mcp.WithString("direction", mcp.Description("up | down | both (default both)")),
		mcp.WithNumber("minReturnPct", mcp.Description("Minimum |return| %% (default 1.5 on 15m)")),
		mcp.WithNumber("limit", mcp.Description("How many moves to scan (default 15)")),
		mcp.WithString("window", mcp.Description("Before-window length 15m|30m|1h|2h|4h (default 1h)")),
		mcp.WithString("during", mcp.Description("Override move window 5m|15m|30m|1h")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetAroundPrecursors(ctx, req.GetString("exchange", "all"), symbol, req.GetString("lookback", ""), req.GetString("interval", ""), req.GetString("direction", ""), req.GetFloat("minReturnPct", 0), int(req.GetFloat("limit", 0)), req.GetString("window", ""), req.GetString("during", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("find_around_similar",
		mcp.WithDescription("Compare the current market tape to the setup before past important moves. Each past match uses only tape available before that move (dataFrom–dataTo) and lists whySimilar plus per-field compared scores (price, volume, takers, oi, book, options). fields selects price,volume,takers,oi,book,options. Only overlapping price-move ranges count as one event. horizons is a CSV of after-windows (default 15m,1h,4h; e.g. 5m,30m,2h) measured on matching candle size. Each horizon reports avg, median, up, down, sample, and the same stats for similarity 40-60, 60-80, 80-100. Default lookback 7d. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("lookback", mcp.Description("4h | 12h | 24h | 3d | 7d (default 7d)")),
		mcp.WithString("interval", mcp.Description("15m | 1h (default 15m)")),
		mcp.WithString("direction", mcp.Description("up | down | both (default both)")),
		mcp.WithNumber("minReturnPct", mcp.Description("Minimum |return| %% for past moves")),
		mcp.WithNumber("limit", mcp.Description("How many nearest cases (default 5, max 10)")),
		mcp.WithString("window", mcp.Description("Current/before window 15m|30m|1h|2h|4h (default 1h)")),
		mcp.WithString("during", mcp.Description("Override past move window")),
		mcp.WithString("fields", mcp.Description("Which tape to compare: price,volume,takers,oi,book,options (default all). Example: volume,book,oi")),
		mcp.WithString("weights", mcp.Description("Importance per field, e.g. book:3,oi:3,volume:1 (defaults: book and oi 3, options 2, takers 1.5, volume and price 1)")),
		mcp.WithNumber("minCoverage", mcp.Description("Minimum % of selected data that must be present (default 60). Below that goes to skipped, not matches.")),
		mcp.WithString("horizons", mcp.Description("After-windows CSV, e.g. 5m,30m,2h (default 15m,1h,4h). Matching candle size. Each reports avg, median, up, down, sample, and similarity bands.")),
		mcp.WithString("at", mcp.Description("Optional event time (RFC3339 or unix ms). When set, compares the tape before this time instead of the current last window.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		minCov := ""
		if v := req.GetFloat("minCoverage", -1); v >= 0 {
			minCov = strconv.FormatFloat(v, 'f', -1, 64)
		}
		raw, err := api.GetAroundSimilar(ctx, req.GetString("exchange", "all"), symbol, req.GetString("lookback", ""), req.GetString("interval", ""), req.GetString("direction", ""), req.GetFloat("minReturnPct", 0), int(req.GetFloat("limit", 0)), req.GetString("window", ""), req.GetString("during", ""), req.GetString("fields", ""), req.GetString("weights", ""), minCov, req.GetString("horizons", ""), req.GetString("at", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_market_forensics",
		mcp.WithDescription("Market forensics for a coin and a chosen time: what happened before, during, and after that move (price, volume, takers, VWAP, vs typical, POC, sweeps, stored book/futures, and listed BTC/ETH options when sampled), plus past important-move setups that looked like the before-window and how price moved after those. why is a plain-language read of why the move may have happened (not a metric dump). timeline is one clock of those prints in time order (OI, takers, price per venue, follow-through, liquidations, options, dealer gamma / flip vs spot, futures curve contango/backwardation and near-vs-far gap, tapeRead flow-at-prices, sweeps) with offsets from the event. The timeline and the similar-setup after-windows only include prints known by asOf (or by the end of the during window when reviewing a completed event). Unusual options prints in the before-window are called out; lead is the real start inside the unusual lookback, not the 4h/24h window open. On BTC/ETH the timeline also shows whether dealers were long or short gamma before the move and whether spot sat above or below the flip. The timeline also shows whether the listed futures curve was in contango or backwardation before the move and the near-vs-far gap. venueTiming says which of Binance/Bybit printed first, whether the other followed or the move stayed on one venue, and whether OI or taker flow had already changed before the laggard's real start (not the during-window close). The first signal across price, OI, and takers is the leader only if that OI or taker print kept going until price. If the last OI sample or taker bar is much earlier than the price start, the path is omitted (not continued / faded / reversed). The taker bar at the price start is not pre-price tape. Continued / faded / reversed also need a continuous pre-price taker path at the tape's bar size (no hole longer than one missing 1m or 5m bar, and tape after the start print). at is required. Same window/during as get_around. Same similar knobs as find_around_similar. If similar-move search fails, tape is still returned. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("at", mcp.Required(), mcp.Description("Event time (RFC3339 or unix ms). Start of the during window.")),
		mcp.WithString("asOf", mcp.Description("Observer clock (RFC3339 or unix ms). Timeline and tape only include prints at or before this time. Omit to use now — old events keep later prints that have already happened.")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
		mcp.WithString("window", mcp.Description("15m | 30m | 1h | 2h | 4h (default 1h)")),
		mcp.WithString("during", mcp.Description("5m | 15m | 30m | 1h (default 15m)")),
		mcp.WithString("lookback", mcp.Description("4h | 12h | 24h | 3d | 7d (default 7d)")),
		mcp.WithString("interval", mcp.Description("15m | 1h (default 15m)")),
		mcp.WithString("direction", mcp.Description("up | down | both (default both)")),
		mcp.WithNumber("minReturnPct", mcp.Description("Minimum |return| %% for past moves")),
		mcp.WithNumber("limit", mcp.Description("How many nearest cases (default 5, max 10)")),
		mcp.WithString("fields", mcp.Description("Which tape to compare: price,volume,takers,oi,book,options (default all)")),
		mcp.WithString("weights", mcp.Description("Importance per field, e.g. book:3,oi:3,volume:1")),
		mcp.WithNumber("minCoverage", mcp.Description("Minimum % of selected data that must be present (default 60)")),
		mcp.WithString("horizons", mcp.Description("After-windows CSV, e.g. 15m,1h,4h")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		at, err := req.RequireString("at")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		minCov := ""
		if v := req.GetFloat("minCoverage", -1); v >= 0 {
			minCov = strconv.FormatFloat(v, 'f', -1, 64)
		}
		raw, err := api.GetForensics(ctx, req.GetString("exchange", "all"), symbol, at, req.GetString("window", ""), req.GetString("during", ""), req.GetString("lookback", ""), req.GetString("interval", ""), req.GetString("direction", ""), req.GetFloat("minReturnPct", 0), int(req.GetFloat("limit", 0)), req.GetString("fields", ""), req.GetString("weights", ""), minCov, req.GetString("horizons", ""), req.GetString("asOf", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_market_forensics",
		mcp.WithDescription("Scan top-volume coins and find which ones currently look like the setup before their own important past moves. Returns ranked coins, how close the tape is, which pieces matched (whySimilar + compared scores for price, volume, takers, oi, book, options), and what usually happened after those similar cases (afterStory, lean, afterHorizons). Default 12 coins, 3d lookback. Not financial advice."),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("quote", mcp.Description("Quote asset, default USDT")),
		mcp.WithString("lookback", mcp.Description("4h | 12h | 24h | 3d | 7d (default 3d)")),
		mcp.WithString("interval", mcp.Description("15m | 1h (default 15m)")),
		mcp.WithString("direction", mcp.Description("up | down | both (default both)")),
		mcp.WithNumber("minReturnPct", mcp.Description("Minimum |return| %% for past moves")),
		mcp.WithNumber("limit", mcp.Description("How many top-volume coins to scan (default 12, max 20)")),
		mcp.WithString("window", mcp.Description("Current/before window 15m|30m|1h|2h|4h (default 1h)")),
		mcp.WithString("during", mcp.Description("Override past move window")),
		mcp.WithString("fields", mcp.Description("Tape to compare: price,volume,takers,oi,book")),
		mcp.WithString("weights", mcp.Description("Importance per field, e.g. book:3,oi:3,volume:1")),
		mcp.WithNumber("minCoverage", mcp.Description("Minimum % of selected data that must be present (default 60)")),
		mcp.WithString("horizons", mcp.Description("After-windows CSV, e.g. 15m,1h,4h")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		minCov := ""
		if v := req.GetFloat("minCoverage", -1); v >= 0 {
			minCov = strconv.FormatFloat(v, 'f', -1, 64)
		}
		raw, err := api.ScanForensics(ctx, req.GetString("exchange", "all"), req.GetString("quote", ""), req.GetString("lookback", ""), req.GetString("interval", ""), req.GetString("direction", ""), req.GetFloat("minReturnPct", 0), int(req.GetFloat("limit", 0)), req.GetString("window", ""), req.GetString("during", ""), req.GetString("fields", ""), req.GetString("weights", ""), minCov, req.GetString("horizons", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("compare_around",
		mcp.WithDescription("Compare two different times for the same coin: how the two moves differed in price, volume, vs typical, order book, open interest, funding, liquidations, and sweeps. from and to are the two event times. Same window/during as get_around. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("from", mcp.Required(), mcp.Description("First event time (RFC3339 or unix ms)")),
		mcp.WithString("to", mcp.Required(), mcp.Description("Second event time (RFC3339 or unix ms)")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
		mcp.WithString("window", mcp.Description("15m | 30m | 1h | 2h | 4h (default 1h)")),
		mcp.WithString("during", mcp.Description("5m | 15m | 30m | 1h (default 15m)")),
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
		raw, err := api.CompareAround(ctx, req.GetString("exchange", "all"), symbol, from, to, req.GetString("window", ""), req.GetString("during", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_vwap",
		mcp.WithDescription("Volume-weighted average price from a start time (or window 1h/4h/24h/7d/30d) until now. Prices with more quote volume pull the average more. Returns VWAP, last vs VWAP (distance and percent), and total volume. Binance and Bybit separately plus combined. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
		mcp.WithString("window", mcp.Description("1h | 4h | 24h | 7d | 30d (default 24h). Ignored when startTime is set.")),
		mcp.WithString("startTime", mcp.Description("Range start (RFC3339 or unix ms). Use with optional endTime.")),
		mcp.WithString("endTime", mcp.Description("Range end (RFC3339 or unix ms). Default now.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetVWAP(ctx, req.GetString("exchange", "all"), symbol, req.GetString("window", ""), req.GetString("startTime", ""), req.GetString("endTime", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_volume_profile",
		mcp.WithDescription("Where aggressive futures prints happened by price: buy and sell USDT on the price that printed, plus open interest opened or closed on those prices and long/short liquidation notional on the same rows. The point of control is the busiest price. The value area holds about 70% of that size. Closed minutes come from the stored ladder (about 30 days). Binance minutes for the liquid list are also filled from public aggTrades, about 7 days. Bybit and OKX stay on minutes this process recorded. The live tape is used when it already reaches the window open. exchange=all returns each venue. combined is present only when every venue reaches that open. missing names a venue that starts later, and that venue is not treated as zero. window 1h/4h/24h/7d/30d or startTime/endTime. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
		mcp.WithString("window", mcp.Description("1h | 4h | 24h | 7d | 30d (default 24h). Ignored when startTime/endTime are set.")),
		mcp.WithString("startTime", mcp.Description("Range start (RFC3339 or unix ms). Use with endTime.")),
		mcp.WithString("endTime", mcp.Description("Range end (RFC3339 or unix ms).")),
		mcp.WithNumber("tickSize", mcp.Description("Price row width (e.g. 50). Omit to pick one automatically.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetVolumeProfile(ctx, req.GetString("exchange", "all"), symbol, req.GetString("window", ""), req.GetString("startTime", ""), req.GetString("endTime", ""), req.GetFloat("tickSize", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_flow_at_prices",
		mcp.WithDescription("What printed at each price: market buy vs sell USDT, print count, and delta (buy − sell) on that row. Use when the user asks how much bought/sold at a level (e.g. 65520 while BTC traded 65400–65600), not candle totals. Built from aggressive futures prints, not candle smear. window 5m/15m/1h/4h/1d/3d/7d/30d (default 15m) or startTime/endTime (max 30d). The live tape is about 1h on Binance and the newest ~1000 Bybit trades. Closed minutes stored while this process runs (about 30 days) fill longer lookbacks and Bybit further back. exchange=all leaves combined off until every requested venue reaches the window open. missing names a late venue, and that venue is not counted as zero. Optional minPrice/maxPrice clips the ladder; tickSize sets row width. truncated when stored rows do not cover the whole window. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all = both + combined)")),
		mcp.WithString("window", mcp.Description("5m | 15m | 1h (default 15m). Ignored when startTime/endTime are set.")),
		mcp.WithString("startTime", mcp.Description("Range start (RFC3339 or unix ms). Use with endTime. Max 1h.")),
		mcp.WithString("endTime", mcp.Description("Range end (RFC3339 or unix ms).")),
		mcp.WithNumber("tickSize", mcp.Description("Price row width (e.g. 10). Omit to pick one automatically.")),
		mcp.WithNumber("minPrice", mcp.Description("Only include prints at or above this price.")),
		mcp.WithNumber("maxPrice", mcp.Description("Only include prints at or below this price.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetFlowAtPrices(ctx, req.GetString("exchange", "all"), symbol, req.GetString("window", ""), req.GetString("startTime", ""), req.GetString("endTime", ""), req.GetFloat("tickSize", 0), req.GetFloat("minPrice", 0), req.GetFloat("maxPrice", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_candle_footprint",
		mcp.WithDescription("Buy versus sell inside each candle, at the prices that actually printed. Each candle has price rows with buy and sell USDT, print count, delta, the busiest row (isPoc), and imbalance buy or sell when one side is at least 3× the other. interval matches the chart (1m, 5m, 15m, 1h, 4h, 1d, 1w, 1M). Optional startTime and endTime together; otherwise the newest 120 candles. At most 400 candles and 30 days; truncated when the range is shortened. Closed minutes are the stored ladder (about 30 days). The current minute uses the live tape when that tape already covers it. This request does not walk public trade history. Binance minutes for the liquid USDT list are filled in the background from public aggTrades, about 7 days. Bybit and OKX stay on the minutes this process recorded. exchange=all returns each venue plus combined. combined is present only when every venue was recorded for the range. missing names a venue with no stored minutes and that venue is not treated as zero. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithString("interval", mcp.Required(), mcp.Description("Candle size: 1m, 5m, 15m, 1h, 4h, 1d, 1w, or 1M.")),
		mcp.WithString("startTime", mcp.Description("Range start (RFC3339 or unix ms). Set with endTime.")),
		mcp.WithString("endTime", mcp.Description("Range end (RFC3339 or unix ms). Set with startTime.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		interval, err := req.RequireString("interval")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetCandleFootprint(ctx, req.GetString("exchange", "all"), symbol, interval, req.GetString("startTime", ""), req.GetString("endTime", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_basis",
		mcp.WithDescription("How far the perpetual futures price is from spot/index on Binance, Bybit, and OKX: premium or discount, dollar and percent gap, whether the gap is expanding or shrinking, plus a short read with funding and open interest. curve compares listed dated futures: near vs far expiry, contango (far richer / up) or backwardation (near richer / down). curve.history / windows / moves show when that near-far spread flipped or got steeper over 1h/4h/24h/7d. Agreement needs a usable quote from all three. A missing or unusable quote leaves the comparison unknown. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all = three venues + agreement)")),
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

	addTool(mcp.NewTool("get_coin_situation",
		mcp.WithDescription("Short read for one coin from data Swyngora already has. priceCvd.kind is price_up_cvd_down, price_down_cvd_up, or agrees. openInterest.flow is opened, closed, or flat over 1h. liquidation.side is the leverage model: long (longs below last are closer), short (shorts above last are closer), or none. prints.side is measured force-orders over the last 24h: long, short, even, or quiet, with longNotional and shortNotional in USDT. Prints are definite only when Binance, Bybit, and OKX recorded that whole window. printAnchors repeat that read since 00:00 UTC (day) and since the current 4-hour open (4h). An anchor stays blank and missing names the venue when that clock was not fully recorded. A quiet recorded clock is a real zero. A situation alert still compares prints.side only. funding.payer is long, short, or none for the OI-weighted predicted rate. crowd.lean is top_more_long, top_more_short, or even (Binance and OKX only; Bybit has no top-trader ratio). size.kind is whale versus retail on the latest sized bar. A clause is omitted and its *Missing list names the venues that do not cover it. The same read is stored when a paper order fills. A resting order stores a new read at the fill, not the read from placement. An unusual standout on the derivatives board and each unusual-scan row open these same clauses. A live clause may include since, the time that reading began in the stored minutes. since is absent when the archive does not reach a different reading. The liquidation model has no minute series, so its since stays absent. changes lists each minute a stored kind changed (clause, at, and the new kind). A gap or an unreadable minute is not a change, and the liquidation model is not listed. A stored fill omits since and changes. Omit at for that live read. Pass at as RFC3339 to read the stored five-minute minute at a candle close. That past read leaves liquidation, liquidationMissing, and changes absent. A named venue is a minute the archive does not have and is not treated as zero. Not financial advice."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("at", mcp.Description("Candle close, RFC3339. Omit for the live read. A past read leaves out the estimated liquidation pocket.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetCoinSituation(ctx, symbol, req.GetString("at", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_fear_greed",
		mcp.WithDescription("Swyngora fear and greed index, 0 to 100, calculated from our own tape (24h breadth, BTC move, volatility versus usual, OI-weighted funding, account long/short, 24h liquidations, perp bid share, taker flow). Not the Binance or CoinGlass index. factors and top say which input moved the score the most. history and windows (1h/4h/24h/7d) are past scores stored while this process runs. Levels: extreme fear, fear, neutral, greed, extreme greed. Not financial advice."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.GetFearGreed(ctx)
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
		mcp.WithDescription("Largest recent aggressive futures buys/sells and large liquidations, sorted biggest first. intent uses open-interest change across the print: buy+OI up opens longs, buy+OI down closes shorts, sell is the mirror. priceAfterPct and oiAfterPct are the move 15 minutes later when samples exist. Each row also has average price, time, size, and a small-cap flag. Omit symbol to scan the top liquid USDT coins. Tape is the newest ~1000 prints per coin, not 24h. Not financial advice."),
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
		mcp.WithDescription("On-chain holder count, concentration (top 10/50/100 %), and top wallets for a crypto base asset (BTC, ETH, or BTCUSDT). Each wallet may include a public label (Binance, Bitfinex, Satoshi genesis) when the address is widely published — not identity proof. Tries CoinMarketCap, Coin Metrics, GeckoTerminal, Ethplorer, Routescan, Tronscan, then CryptoID for some UTXO coins. 404 if every source misses. Informational only."),
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

	addTool(mcp.NewTool("get_asset_profile",
		mcp.WithDescription("Logo URL, listing date, and published token contracts for a crypto base asset. Uses Binance marketing CMC id and CoinMarketCap public data. Informational only."),
		mcp.WithString("asset", mcp.Required(), mcp.Description("Base asset ticker e.g. BTC (pairs like BTCUSDT also work)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		asset, err := req.RequireString("asset")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetAssetProfile(ctx, asset)
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
		mcp.WithDescription("Spot FX rates (units per 1 USD) plus venueQuotes, marketCapQuotes, and stable aliases for display conversion. USDT is treated as USD. Display only."),
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
		mcp.WithDescription("List spot delistings for a venue (last 30 days and next ~31 days). Each item has delistTime (halt) and announcedAt (when the notice was published) when known. Binance official schedule plus CMS titles; Bybit announcements. Empty if none or unsupported."),
		mcp.WithString("exchange", mcp.Description("Venue id; default binance")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ListDelistSchedule(ctx, req.GetString("exchange", "binance"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_post_delist",
		mcp.WithDescription("After a venue has delisted a pair, fetch off-venue movement (another listed venue or CoinGecko USD). Informational only — not this exchange's book. Listed/upcoming pairs return available=false."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. VICUSDT")),
		mcp.WithString("exchange", mcp.Description("Home venue that delisted the pair; default binance")),
		mcp.WithString("interval", mcp.Description("Candle interval hint; default 1d")),
		mcp.WithNumber("limit", mcp.Description("Max candles (1-200); default 30")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPostDelist(ctx, req.GetString("exchange", "binance"), symbol, req.GetString("interval", "1d"), req.GetInt("limit", 30))
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
		mcp.WithDescription("List price alerts for a clientId (active and triggered). A situation alert includes situationWatch (stored, blank, or flat) and, after it fires, situationFire: the clauses that left, the entry kind, the later kind, and any missing venue. That fire stays until the next fire replaces it. Repeating re-arm does not clear it. The alert still follows the newest open buy."),
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

	addTool(mcp.NewTool("create_situation_alert",
		mcp.WithDescription("Alert when the live coin read leaves the read stored on the newest open paper buy for this exchange and symbol. Compares only the clauses that were stored at the fill. A clause that appears later does not fire. No open crypto position stays quiet. A newest buy with no stored read does not fire. mode=one_time (default) fires once. mode=repeating re-arms when the live read matches the entry again. The last fire is kept on the alert, including which clauses left and what they became, until the next fire replaces it. Not for Nasdaq or BIST."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair on the paper book, e.g. BTCUSDT or BTC-USD")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit (default binance)")),
		mcp.WithString("mode", mcp.Description("one_time (default) | repeating")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateSituationAlert(ctx, clientID, req.GetString("exchange", "binance"), symbol, req.GetString("mode", "one_time"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_tape_alert",
		mcp.WithDescription("Alert when funding, open interest, taker buy share, or fear and greed crosses a level. kind=funding (percent, e.g. 0.05 or -0.01), open_interest (USDT notional), taker (5m buy share 0-100), fear_greed (score 0-100, no symbol). condition=above|below. mode=one_time or repeating. Funding exchange=all fires when any of Binance, Bybit, or OKX crosses the level and names that venue."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("kind", mcp.Required(), mcp.Description("funding | open_interest | taker | fear_greed")),
		mcp.WithString("condition", mcp.Required(), mcp.Description("above | below")),
		mcp.WithNumber("target", mcp.Required(), mcp.Description("Level to cross")),
		mcp.WithString("symbol", mcp.Description("Pair e.g. BTCUSDT. Omit for fear_greed")),
		mcp.WithString("exchange", mcp.Description("binance|bybit|okx|all (default all)")),
		mcp.WithString("mode", mcp.Description("one_time | repeating (default repeating)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
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
		target, err := req.RequireFloat("target")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateTapeAlert(ctx, clientID, req.GetString("exchange", "all"), req.GetString("symbol", ""), kind, condition, target, req.GetString("mode", "repeating"))
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

	addTool(mcp.NewTool("create_liquidation_feed_alert",
		mcp.WithDescription("Alert when Binance or Bybit stop sending liquidation data (down or stalled) longer than minDownSeconds (default 300). Repeating by default: fires once, stays quiet while the feed is still bad, re-arms when it is live again. Does not keep sending the same outage. exchange=binance|bybit|all."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithNumber("minDownSeconds", mcp.Description("How long the feed may stay down before fire (default 300, min 30)")),
		mcp.WithString("mode", mcp.Description("repeating (default) | one_time")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateLiquidationFeedAlert(ctx, clientID, req.GetString("exchange", "all"), req.GetFloat("minDownSeconds", 0), req.GetString("mode", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_liquidation_cascade_alert",
		mcp.WithDescription("Alert when a coin has a big liquidation cascade. Payload includes the exchange and coin. symbol=BTCUSDT or all (any bursting coin). minGrade=cascade (default) or extreme. Repeating: fires once per wave, does not re-fire while the same cascade stays on, re-arms when it goes quiet."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT, or all for a market scan")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("minGrade", mcp.Description("cascade (default) | extreme | elevated")),
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
		raw, err := api.CreateLiquidationCascadeAlert(ctx, clientID, req.GetString("exchange", "all"), symbol, req.GetString("minGrade", ""), req.GetString("mode", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_liquidation_level_alert",
		mcp.WithDescription("Alert when last price or later prints hit a predicted 10x/25x/50x/100x liquidation band. minReliability=any (default) | fair | high. Low-confidence bands that keep missing later prints do not fire the same alert as bands that keep working. minConfidence 0–1 overrides the default floor (any=0.35, fair=0.45, high=0.80). Payload includes importance, leverage, side, and confidence. Repeating: one fire while in/printing in a qualifying band, re-arms when none qualify."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | okx | all (default all)")),
		mcp.WithString("minReliability", mcp.Description("any (default) | fair | high")),
		mcp.WithNumber("minConfidence", mcp.Description("0–1 floor; 0 uses the default for minReliability")),
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
		raw, err := api.CreateLiquidationLevelAlert(ctx, clientID, req.GetString("exchange", "all"), symbol, req.GetString("minReliability", "any"), req.GetFloat("minConfidence", 0), req.GetString("mode", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_liquidation_notional_alert",
		mcp.WithDescription("Alert when a coin's liquidations in a window exceed a USDT amount. side=long|short|both (total). exchange=binance|bybit|all (all sums both). window=1m|5m|15m|1h (default 5m). Creating the alert subscribes the coin and fills the last window from history (no double-count with live). Repeating: fires once when the wave crosses the line, stays quiet while that wave stays above, re-arms when the window drops so a new wave can fire."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("notional", mcp.Required(), mcp.Description("USDT threshold e.g. 20000000")),
		mcp.WithString("side", mcp.Description("long | short | both (default both)")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithString("window", mcp.Description("1m | 5m | 15m | 1h (default 5m)")),
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
		notional, err := req.RequireFloat("notional")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateLiquidationNotionalAlert(ctx, clientID, req.GetString("exchange", "all"), symbol, req.GetString("side", "both"), notional, req.GetString("window", "5m"), req.GetString("mode", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_setup_radar_alert",
		mcp.WithDescription("Alert when a coin's current tape looks similar enough to the setup before that coin's own important past moves (Setup Radar / Forensics Now). minSimilarity is 40–100 (0 = 60). lean=any (default) or up/down for the usual after-path. symbol=BTCUSDT or all (any scanned top-volume coin). exchange=binance|bybit|all. Repeating: fires once when the match appears, stays quiet while still similar, re-arms when it clears. New coins while a market-wide alert is still hot do not re-fire."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Description("Pair e.g. BTCUSDT, or all (default) for any scanned coin")),
		mcp.WithString("exchange", mcp.Description("binance | bybit | all (default all)")),
		mcp.WithNumber("minSimilarity", mcp.Description("Min closeness 40–100 (0 = 60)")),
		mcp.WithString("lean", mcp.Description("any (default) | up | down")),
		mcp.WithString("mode", mcp.Description("repeating (default) | one_time")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateSetupRadarAlert(ctx, clientID, req.GetString("exchange", "all"), req.GetString("symbol", "all"), req.GetString("lean", ""), req.GetFloat("minSimilarity", 0), req.GetString("mode", ""))
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

	addTool(mcp.NewTool("get_fill_pocket",
		mcp.WithDescription("Where an open paper fill sits against the nearer estimated liquidation pocket. fillPrice is the newest open spot buy, or the margin entry when that book has no spot lot. pocketPrice is that pocket on the combined Binance, Bybit, and OKX map. gapPct is the percent from the fill to the pocket. pocketFromFill is above or below. marginLiqPrice is the paper liquidation price, and marginGapPct is the percent from the pocket to that price. A missing venue leaves the pocket blank and is named. It is not zero. open is false when the book has no position. Equities set equity and return no prices. Not financial advice."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("exchange", mcp.Required(), mcp.Description("binance | coinbase | bybit | nasdaq | bist")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name; required if multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		exchange, err := req.RequireString("exchange")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.GetFillPocket(ctx, clientID, exchange, symbol)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio",
		mcp.WithDescription("Get paper portfolio cash, positions, realized/unrealized P&L (mark-to-market). Each open crypto position includes entrySituation, the coin read stored when the newest buy still held filled, and openReads for every remaining buy (size, fill time, and that fill's stored read). Equities omit both. Every open position includes moves: the 1h, 4h, and 24h venue price change after the newest open lot, including equities. pending means that minute has not closed. missing means the close is absent and is not zero. ready pct is the price change, not profit. The live read is get_coin_situation. What changed on each buy is get_situation_change. Pass portfolioId when the client has more than one book. Works for shared books if you have access."),
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

	addTool(mcp.NewTool("get_situation_change",
		mcp.WithDescription("Compare a stored coin situation with a later one. Pass symbol for an open crypto position: entry is the read on the newest buy still held, live is the current combined read, and comparison names the stored clauses whose kind changed. openReads lists every remaining buy, newest first, with its size, fill time, stored read, and that buy's comparison when liveOk is true. A buy that stored no read keeps its size and time and has no clauses. entry and the top-level comparison stay on the newest buy. A situation alert still follows that buy. status missing means the later read left that clause blank and names the venue. A blank clause is not a new opinion. A clause the entry did not store is omitted. same is true when every stored clause still matches. liveOk false means the live read did not load, and comparison is omitted on the headline and on each buy. Pass tradeId instead of symbol to compare a sell with each buy it closed (left). Equities stay empty. Not financial advice."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name; required if multiple portfolios exist")),
		mcp.WithString("exchange", mcp.Description("binance | coinbase | bybit (default binance). Ignored when tradeId is set.")),
		mcp.WithString("symbol", mcp.Description("Pair e.g. BTCUSDT. Required unless tradeId is set.")),
		mcp.WithString("tradeId", mcp.Description("A paper fill id. When set, the sell is compared with each buy it closed.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.GetSituationChange(ctx, clientID, req.GetString("exchange", ""), req.GetString("symbol", ""), req.GetString("tradeId", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_situation_record",
		mcp.WithDescription("What this paper book's closed crypto lots made, grouped by the situation stored on the buy. Each row is one entry kind: count, how many of those closes were profitable, and the sum of realized profit. closes lists those slices, newest first, with symbol, sell time, and that slice's realized profit. The same close can appear on more than one row, so the rows are not one total. counted is each close once. skipped is a crypto close whose buy stored no read. Equities are left out. Open positions are not included. Not financial advice."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name; required if multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.GetSituationRecord(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_situation_leave",
		mcp.WithDescription("Whether this paper book's closed crypto lots kept the kind stored on the buy or sold after it left. Each row is one entry kind. still is the closes whose sell read kept that kind. left is the closes whose sell read changed that kind or left it blank. A change in print dollars with the same side stays in still. Each group has count, how many were profitable, the sum of realized profit, and closes, newest first, with symbol, sell time, and that slice's realized profit. The same close can appear on more than one row, so the rows are not one total. counted is each comparable close once. skipped is a crypto close whose buy stored no read. unreadable is a close whose buy stored a read and whose sell stored none, and it stays out of both groups. Equities are left out. Open positions are not included. Not financial advice."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name; required if multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.GetSituationLeave(ctx, clientID)
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
		mcp.WithDescription("Move virtual cash from one of your paper portfolios to another. Owner only. Recorded as transfer_out/transfer_in, not deposit/withdrawal. Pass idempotencyKey so a dropped retry returns the original pair instead of a second move. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("fromPortfolioId", mcp.Description("Source book id or name")),
		mcp.WithString("toPortfolioId", mcp.Required(), mcp.Description("Destination book id or name")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Positive cash to move")),
		mcp.WithString("note", mcp.Description("Optional label")),
		mcp.WithString("idempotencyKey", mcp.Description("Retry key; same key + same request returns the original transfer")),
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
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
		raw, err := api.TransferPortfolioCash(ctx, clientID, req.GetString("fromPortfolioId", ""), toID, amt, req.GetString("note", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_transfer_attempts",
		mcp.WithDescription("List paper transfer requests for this owner: completed (cash moved), replayed (same key came again), or conflict (same key, different amount/books)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListTransferAttempts(ctx, clientID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_portfolio_cash_movements",
		mcp.WithDescription("List paper portfolio deposits, withdrawals, and internal transfers (newest first), including the opening balance."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithNumber("limit", mcp.Description("Max rows (default 50)")),
		mcp.WithNumber("offset", mcp.Description("Pagination offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListPortfolioCashMovements(ctx, clientID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio_performance",
		mcp.WithDescription("Paper portfolio equity over 1d/1w/1m/3m plus trading P&L amount and percent from the start of that window. Deposits, withdrawals, and transfers are netFlow, not profit or loss. maxDrawdown is the largest drop of that performance line. Simulated only."),
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

	addTool(mcp.NewTool("stress_portfolio",
		mcp.WithDescription("Preview paper equity, total P&L, and margin equity if named coins move, or if a past window happened again. moves is BTC:-10,ETH:-15. from and to replay each open position's own close-to-close change over that window. points is the book at today's size on each close, and lowEquity is the lowest point. A coin on two venues keeps each venue's move. A date includes the end day. A position with no close at either end stays unchanged and is named. Send moves or a period, not both. Nothing is traded. A margin row that crosses liquidation loses its locked margin."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("moves", mcp.Description("Comma list asset:percent, e.g. BTC:-10,ETH:-15. Omit when from and to are set.")),
		mcp.WithString("from", mcp.Description("Start of a past window. YYYY-MM-DD or RFC3339. Use with to, instead of moves.")),
		mcp.WithString("to", mcp.Description("End of that window. A YYYY-MM-DD is included. Use with from, instead of moves.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.StressPortfolio(ctx, clientID, req.GetString("moves", ""), req.GetString("from", ""), req.GetString("to", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio_risk_limits",
		mcp.WithDescription("Get optional paper risk limits and live status (daily trading loss %, per-coin weights, whether new buys/margin opens are blocked). Deposits and withdrawals are not daily profit or loss. Limits never close positions."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.GetPortfolioRiskLimits(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("set_portfolio_risk_limits",
		mcp.WithDescription("Set optional paper risk limits. maxDailyLossPct e.g. 5 blocks new buys/margin when today's trading loss hits 5%. Deposits and withdrawals do not count toward that loss. maxAssetWeightPct e.g. 30 blocks new buys that would push one coin over 30%. maxExposurePct e.g. 60 blocks new buys once all coins together would exceed 60% of equity. Omit a value to disable that rule. Does not close positions."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithNumber("maxDailyLossPct", mcp.Description("0 or omit to disable; else 0.01-100")),
		mcp.WithNumber("maxAssetWeightPct", mcp.Description("0 or omit to disable; else 0.01-100")),
		mcp.WithNumber("maxExposurePct", mcp.Description("0 or omit to disable; else 0.01-1000. Total coin exposure as a percent of equity")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		var daily, weight *float64
		if v := req.GetFloat("maxDailyLossPct", 0); v > 0 {
			daily = &v
		}
		if v := req.GetFloat("maxAssetWeightPct", 0); v > 0 {
			weight = &v
		}
		var exposure *float64
		if v := req.GetFloat("maxExposurePct", 0); v > 0 {
			exposure = &v
		}
		raw, err := api.PutPortfolioRiskLimits(ctx, clientID, daily, weight, exposure)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("clear_portfolio_risk_limits",
		mcp.WithDescription("Remove all paper risk limits for a client."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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

	addTool(mcp.NewTool("preview_portfolio_order",
		mcp.WithDescription("Preview a paper order before it is placed. Returns the slipped fill, taker fee, cash in or out, reserved cash or quantity, estimated sell profit, and whether a risk limit or the borrow cap would block it. Pass riskPercent and stopLossPrice to size a spot buy, bracket, or margin open from portfolio equity (quantity is then ignored). product=spot (default) or margin. When reason names clauses and the live read is ready, reasonHistory is how this book's past closes of each selected live kind did: still kept the kind, left changed or went blank. Both counts at zero means no past close stored that kind. The rows are not one total. A failed history read omits reasonHistory and still returns the price. Nothing is written. Not real money."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("quantity", mcp.Description("Base asset quantity. Required unless riskPercent is set. Ignored when riskPercent is set.")),
		mcp.WithNumber("riskPercent", mcp.Description("Percent of portfolio equity to lose if the stop fills, for example 1. Sizes a spot buy, bracket, or margin open from stopLossPrice. Greater than 0 and at most 100.")),
		mcp.WithString("side", mcp.Description("Spot market: buy|sell. Margin: long|short")),
		mcp.WithString("type", mcp.Description("Spot: market|limit_buy|limit_sell|stop_loss|trailing_stop|oco|bracket. Margin: market|limit")),
		mcp.WithString("product", mcp.Description("spot (default) or margin")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithNumber("triggerPrice", mcp.Description("Limit, stop, or bracket entry price")),
		mcp.WithNumber("takeProfitPrice", mcp.Description("OCO, bracket, or margin take-profit")),
		mcp.WithNumber("stopLossPrice", mcp.Description("Stop used to size the order. Also the OCO, bracket, or margin stop.")),
		mcp.WithString("trailType", mcp.Description("trailing_stop: percent|offset")),
		mcp.WithNumber("trailValue", mcp.Description("trailing_stop distance")),
		mcp.WithString("lotMethod", mcp.Description("fifo (default) or lifo")),
		mcp.WithNumber("limitPrice", mcp.Description("Margin limit price")),
		mcp.WithNumber("leverage", mcp.Description("Margin leverage 1-10")),
		mcp.WithString("reason", mcp.Description("Spot buy, limit_buy, or bracket only. Comma-separated clauses to store: priceCvd, openInterest, liquidation, prints, funding, crowd, size. Omit to store the whole read. Pass none to store no read. A sell ignores this and keeps the whole read.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		qty := req.GetFloat("quantity", 0)
		riskPct := req.GetFloat("riskPercent", 0)
		if qty <= 0 && riskPct <= 0 {
			return mcp.NewToolResultError("quantity or riskPercent is required"), nil
		}
		ctx, err = withToolReason(ctx, req.GetString("reason", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.PreviewPortfolioOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol,
			req.GetString("side", ""), req.GetString("type", ""), req.GetString("product", ""),
			req.GetString("trailType", ""), req.GetString("lotMethod", ""),
			qty, req.GetFloat("triggerPrice", 0), req.GetFloat("takeProfitPrice", 0), req.GetFloat("stopLossPrice", 0),
			req.GetFloat("trailValue", 0), req.GetFloat("limitPrice", 0), int(req.GetFloat("leverage", 0)), riskPct)
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
		mcp.WithString("reason", mcp.Description("Buy only. Comma-separated clauses to store: priceCvd, openInterest, liquidation, prints, funding, crowd, size. Omit to store the whole read. Pass none to store no read. A sell ignores this.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
		ctx, err = withToolReason(ctx, req.GetString("reason", ""))
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
		raw, err := api.PlacePortfolioOrder(ctx, clientID, req.GetString("exchange", "binance"), symbol, side, qty, req.GetString("lotMethod", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_portfolio_trades",
		mcp.WithDescription("List paper trade history. Each fill includes slipped price, lastPrice, taker fee, and moves (1h, 4h, 24h venue price change after the fill). pending means that minute has not closed. missing means the close is absent and is not zero. ready pct is the price change, not realized profit. A sell with a positive pct means price rose after the sell."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
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
		mcp.WithString("reason", mcp.Description("limit_buy only. Comma-separated clauses to store. Omit to store the whole read. Pass none to store no read. A sell ignores this.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
		ctx, err = withToolReason(ctx, req.GetString("reason", ""))
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
			req.GetString("trailType", ""), req.GetFloat("trailValue", 0), req.GetString("lotMethod", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("place_portfolio_oco_order",
		mcp.WithDescription("Place a paper OCO: take-profit limit sell + stop-loss for the same quantity. Full fill of one cancels the other; partial fill shrinks both remainings. Same tick fills at most one leg. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
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
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("quantity", mcp.Required(), mcp.Description("Entry size")),
		mcp.WithNumber("entryPrice", mcp.Required(), mcp.Description("Limit buy price")),
		mcp.WithNumber("takeProfitPrice", mcp.Required(), mcp.Description("Limit sell above entry")),
		mcp.WithNumber("stopLossPrice", mcp.Required(), mcp.Description("Stop below entry")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithString("expiresAt", mcp.Description("RFC3339 expiry")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original bracket")),
		mcp.WithString("reason", mcp.Description("Clauses the entry buy stores, comma-separated. Omit to store the whole read. Pass none to store no read.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx = WithIdempotencyKey(ctx, req.GetString("idempotencyKey", ""))
		var err error
		ctx, err = withToolReason(ctx, req.GetString("reason", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("status", mcp.Description("open|filled|canceled|rejected|all")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListPortfolioOrders(ctx, clientID, req.GetString("status", "open"), req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio_order",
		mcp.WithDescription("Get one paper pending order plus last price and amend hints (editable, max remaining, available cash/qty including this order's reservation)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Pending order id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Pending order id")),
		mcp.WithNumber("triggerPrice", mcp.Description("New limit/stop price")),
		mcp.WithNumber("remainingQuantity", mcp.Description("New remaining size (must stay > 0)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("symbol", mcp.Description("When set, cancel only this pair (e.g. BTCUSDT); omit for all markets")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist; default binance when symbol is set")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.CancelAllPortfolioOrders(ctx, clientID, req.GetString("exchange", ""), req.GetString("symbol", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("cancel_portfolio_order",
		mcp.WithDescription("Cancel an open paper pending order. Canceled orders never fill."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Pending order id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithDescription("Create a named paper recurring buy: cash amount at market on daily|weekly|monthly|interval. Use weekday (monday) for weekly, dayOfMonth (1-31) for salary day, intervalHours (e.g. 12) for interval. timeZone (Europe/Istanbul) + hour/minute for a DST-aware local clock. maxPrice skips a run if last or fee+slippage unit would exceed it. budget is a total cash cap; endDate (YYYY-MM-DD) or endsAt is the last inclusive run. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Cash notional per run e.g. 500")),
		mcp.WithString("frequency", mcp.Required(), mcp.Description("daily | weekly | monthly | interval")),
		mcp.WithString("name", mcp.Description("Label e.g. Salary Day Buy")),
		mcp.WithString("weekday", mcp.Description("Weekly: monday..sunday")),
		mcp.WithNumber("dayOfMonth", mcp.Description("Monthly salary day 1-31")),
		mcp.WithNumber("intervalHours", mcp.Description("Interval frequency: 1-168 hours")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist")),
		mcp.WithString("startAt", mcp.Description("RFC3339 first run; default now")),
		mcp.WithString("timeZone", mcp.Description("IANA timezone e.g. Europe/Istanbul")),
		mcp.WithNumber("hour", mcp.Description("Local hour 0-23")),
		mcp.WithNumber("minute", mcp.Description("Local minute 0-59")),
		mcp.WithNumber("maxPrice", mcp.Description("Skip if last or fee+slippage unit exceeds this; 0 = no cap")),
		mcp.WithNumber("budget", mcp.Description("Total cash cap across succeeded runs; 0 = no cap")),
		mcp.WithString("endDate", mcp.Description("Last inclusive local day YYYY-MM-DD")),
		mcp.WithString("endsAt", mcp.Description("RFC3339 inclusive last allowed scheduled instant")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		hour := domain.RecurringHourUnset
		if _, ok := req.GetArguments()["hour"]; ok {
			hour = int(req.GetFloat("hour", 0))
		}
		raw, err := api.CreateRecurringBuyPlan(ctx, clientID, req.GetString("exchange", "binance"), symbol, amount, freq, req.GetString("startAt", ""),
			req.GetString("name", ""), req.GetString("weekday", ""), int(req.GetFloat("dayOfMonth", 0)), int(req.GetFloat("intervalHours", 0)),
			req.GetString("timeZone", ""), hour, int(req.GetFloat("minute", 0)), req.GetFloat("maxPrice", 0),
			req.GetFloat("budget", 0), req.GetString("endDate", ""), req.GetString("endsAt", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("update_recurring_buy",
		mcp.WithDescription("Update a paper recurring buy name, amount, schedule, timeZone/hour/minute, maxPrice, budget, or endDate/endsAt. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
		mcp.WithString("name", mcp.Description("New label")),
		mcp.WithNumber("amount", mcp.Description("New cash notional per run")),
		mcp.WithString("frequency", mcp.Description("daily | weekly | monthly | interval")),
		mcp.WithString("weekday", mcp.Description("monday..sunday")),
		mcp.WithNumber("dayOfMonth", mcp.Description("1-31")),
		mcp.WithNumber("intervalHours", mcp.Description("1-168")),
		mcp.WithString("startAt", mcp.Description("RFC3339 to reschedule first/next run")),
		mcp.WithString("timeZone", mcp.Description("IANA timezone e.g. Europe/Istanbul")),
		mcp.WithNumber("hour", mcp.Description("Local hour 0-23")),
		mcp.WithNumber("minute", mcp.Description("Local minute 0-59")),
		mcp.WithNumber("maxPrice", mcp.Description("Skip if last or fee+slippage unit exceeds this; 0 = no cap")),
		mcp.WithNumber("budget", mcp.Description("Total cash cap; 0 = no cap")),
		mcp.WithString("endDate", mcp.Description("Last inclusive local day YYYY-MM-DD; empty clears")),
		mcp.WithString("endsAt", mcp.Description("RFC3339 inclusive last instant; empty clears")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		hour := domain.RecurringHourUnset
		if _, ok := req.GetArguments()["hour"]; ok {
			hour = int(req.GetFloat("hour", 0))
		}
		maxP := -1.0
		if _, ok := req.GetArguments()["maxPrice"]; ok {
			maxP = req.GetFloat("maxPrice", 0)
		}
		budget := -1.0
		if _, ok := req.GetArguments()["budget"]; ok {
			budget = req.GetFloat("budget", 0)
		}
		endDate := ""
		endsAt := ""
		clearEnds := false
		if _, ok := req.GetArguments()["endDate"]; ok {
			endDate = req.GetString("endDate", "")
			if endDate == "" {
				clearEnds = true
			}
		}
		if _, ok := req.GetArguments()["endsAt"]; ok {
			endsAt = req.GetString("endsAt", "")
			if endsAt == "" && endDate == "" {
				clearEnds = true
			} else if endsAt != "" {
				clearEnds = false
			}
		}
		raw, err := api.UpdateRecurringBuyPlan(ctx, clientID, planID, req.GetString("name", ""), req.GetString("frequency", ""),
			req.GetString("weekday", ""), req.GetString("startAt", ""), req.GetFloat("amount", 0), int(req.GetFloat("dayOfMonth", 0)), int(req.GetFloat("intervalHours", 0)),
			req.GetString("timeZone", ""), hour, int(req.GetFloat("minute", 0)), maxP, budget, endDate, endsAt, clearEnds)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_recurring_buys",
		mcp.WithDescription("List paper recurring buy plans for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListRecurringBuyPlans(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_recurring_buy",
		mcp.WithDescription("Get one paper recurring buy plan by id."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithDescription("List execution history for a paper recurring buy plan (succeeded, failed, and missed slots). Succeeded rows include timing on_time or late."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 50")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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

	addTool(mcp.NewTool("complete_recurring_buy_run",
		mcp.WithDescription("Buy a missed paper recurring slot now at the current last price. Marks the fill as late. Does not move the plan's next run. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("planId", mcp.Required(), mcp.Description("Plan id")),
		mcp.WithString("runId", mcp.Required(), mcp.Description("Missed run id from list_recurring_buy_runs")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		planID, err := req.RequireString("planId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		runID, err := req.RequireString("runId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CompleteRecurringBuyRun(ctx, clientID, planID, runID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_portfolio_basket",
		mcp.WithDescription("Create a named paper allocation basket (target percent mix). targetsJSON example: [{\"asset\":\"BTC\",\"weightPct\":50},{\"asset\":\"ETH\",\"weightPct\":30},{\"asset\":\"USDT\",\"weightPct\":20}]. Does not trade. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Basket label e.g. Core 50/30/20")),
		mcp.WithString("targetsJSON", mcp.Required(), mcp.Description("JSON array of {asset,weightPct,exchange?}")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListPortfolioBaskets(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio_basket",
		mcp.WithDescription("Get a paper allocation basket with live actual vs target weights and proposed rebalance legs (preview, no trades)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
		mcp.WithString("name", mcp.Description("New label")),
		mcp.WithString("targetsJSON", mcp.Description("JSON array of {asset,weightPct,exchange?}")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithDescription("USER-TRIGGERED paper rebalance toward a basket (sell overweight, buy underweight at last price). Stores a result with filled/failed legs and leftover mix drift. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("basketId", mcp.Required(), mcp.Description("Basket id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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

	addTool(mcp.NewTool("list_portfolio_rebalances",
		mcp.WithDescription("List stored paper rebalance results for a book (optional basket). Shows filled vs failed legs and leftover mix drift. Does not trade."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("basketId", mcp.Description("Optional basket id")),
		mcp.WithNumber("limit", mcp.Description("Max rows default 20")),
		mcp.WithNumber("offset", mcp.Description("Offset default 0")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListPortfolioRebalances(ctx, clientID, req.GetString("basketId", ""), req.GetInt("limit", 20), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("continue_portfolio_rebalance",
		mcp.WithDescription("Continue a stored paper rebalance. Re-plans leftover mix from the current book (does not replay filled legs) and appends a new attempt. Use after a partial, failed, or interrupted run."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("rebalanceId", mcp.Required(), mcp.Description("Rebalance result id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		id, err := req.RequireString("rebalanceId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ContinuePortfolioRebalance(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_portfolio_rebalance",
		mcp.WithDescription("Get one stored paper rebalance result: which legs filled or failed, leftover drift vs target, sleeve before/after, paper fee plus slippage, and each attempt if the run was continued."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("rebalanceId", mcp.Required(), mcp.Description("Rebalance result id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		id, err := req.RequireString("rebalanceId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetPortfolioRebalance(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("set_margin_mode",
		mcp.WithDescription("Set paper portfolio margin mode isolated|cross. Blocked while open margin positions or pending margin orders exist."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("mode", mcp.Required(), mcp.Description("isolated | cross")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("delta", mcp.Required(), mcp.Description("Positive add, negative remove")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("amount", mcp.Required(), mcp.Description("Debt units to repay")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithDescription("Paper margin open: long|short, leverage 1-10, market or limit (uses account isolated|cross mode). Stores the coin read from placement. A limit fill keeps that read. Optional stopLoss/takeProfit. Simulated only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
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
		mcp.WithDescription("List open paper margin positions with mark, unrealized PnL, liquidation price, and entrySituation (the coin read stored when the order was placed)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListMarginPositions(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("close_margin_position",
		mcp.WithDescription("Close all or part of a paper margin position at market. quantity 0 = full close."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("positionId", mcp.Required(), mcp.Description("Margin position id")),
		mcp.WithNumber("quantity", mcp.Description("Partial size; omit/0 for full")),
		mcp.WithString("idempotencyKey", mcp.Description("Optional retry key; same key + same request returns the original close")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
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
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("status", mcp.Description("open|filled|canceled|rejected|all")),
		mcp.WithNumber("limit", mcp.Description("Max rows")),
		mcp.WithNumber("offset", mcp.Description("Offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListMarginOrders(ctx, clientID, req.GetString("status", "open"), req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("cancel_margin_order",
		mcp.WithDescription("Cancel an open paper margin limit order."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithString("orderId", mcp.Required(), mcp.Description("Margin order id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
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
		mcp.WithString("portfolioId", mcp.Description("Book id or name when multiple portfolios exist")),
		mcp.WithNumber("limit", mcp.Description("Max rows")),
		mcp.WithNumber("offset", mcp.Description("Offset")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		ctx = WithPortfolioID(ctx, req.GetString("portfolioId", ""))
		raw, err := api.ListMarginTrades(ctx, clientID, req.GetInt("limit", 50), req.GetInt("offset", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_scanner_rule",
		mcp.WithDescription("Create a technical scanner rule for the client's watchlist. Select one or more of rsi, ma_crossover, volume_increase via conditions (comma-separated). matchMode=all requires every selected condition; matchMode=any needs only one. Legacy type=rsi|ma_crossover|volume_increase still works."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("type", mcp.Description("rsi | ma_crossover | volume_increase | combo (optional if conditions set)")),
		mcp.WithString("conditions", mcp.Description("Comma-separated rsi,ma_crossover,volume_increase")),
		mcp.WithString("matchMode", mcp.Description("all (every condition) | any (one is enough)")),
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
		args := map[string]any{
			"clientId": clientID, "type": req.GetString("type", ""), "interval": req.GetString("interval", "1h"),
			"conditions": req.GetString("conditions", ""), "matchMode": req.GetString("matchMode", ""),
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

	addTool(mcp.NewTool("update_scanner_rule",
		mcp.WithDescription("Enable/disable a scanner rule or edit its conditions, matchMode, interval, periods, and thresholds without deleting it."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("ruleId", mcp.Required(), mcp.Description("Rule id")),
		mcp.WithBoolean("enabled", mcp.Description("true to run the rule, false to pause it")),
		mcp.WithString("interval", mcp.Description("Candle interval")),
		mcp.WithString("conditions", mcp.Description("Comma-separated rsi,ma_crossover,volume_increase")),
		mcp.WithString("matchMode", mcp.Description("all | any")),
		mcp.WithNumber("rsiPeriod", mcp.Description("RSI period")),
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
		ruleID, err := req.RequireString("ruleId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		args := map[string]any{"clientId": clientID, "id": ruleID}
		rawArgs := req.GetArguments()
		copyIfPresent := func(key string) {
			if v, ok := rawArgs[key]; ok {
				args[key] = v
			}
		}
		for _, key := range []string{
			"enabled", "interval", "conditions", "matchMode",
			"rsiPeriod", "rsiCondition", "rsiThreshold",
			"maFastPeriod", "maSlowPeriod", "maDirection",
			"volumeLookback", "volumeMinRatio",
		} {
			copyIfPresent(key)
		}
		raw, err := api.UpdateScannerRule(ctx, args)
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

	addTool(mcp.NewTool("start_backtest",
		mcp.WithDescription("Start a historical backtest of a scanner rule or a custom signal profile. Send ruleId or profileId (not both). profileId=built_in uses the built-in swing mix. For a custom mix, scope=watchlist (or symbol=watchlist) replays every coin on the client's watchlist in one job; omit that to run one symbol. Custom jobs still score the factors that have data. If a selected factor has no history at that bar, that timestamp is skipped (no signal); the job reports skippedBars and missingFactors. A signal fires when the mix first crosses 62 on a complete bar. Each hit stores side, leverage, entry, TP, and SL and records whether later candles hit take-profit or stop-loss first. If one bar tags both, a smaller interval is used when available so outcome and PnL follow the first print. Informational only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("rangeStart", mcp.Required(), mcp.Description("RFC3339 start")),
		mcp.WithString("rangeEnd", mcp.Required(), mcp.Description("RFC3339 end")),
		mcp.WithString("symbol", mcp.Description("Pair e.g. ETHUSDT. Use watchlist or all with profileId to run every watchlist coin")),
		mcp.WithString("scope", mcp.Description("symbol (default) or watchlist. watchlist requires profileId")),
		mcp.WithString("ruleId", mcp.Description("Scanner rule id")),
		mcp.WithString("profileId", mcp.Description("Custom signal profile id, or built_in")),
		mcp.WithString("exchange", mcp.Description("binance|coinbase|bybit|nasdaq|bist, default binance")),
		mcp.WithNumber("startingBalance", mcp.Description("Shared cash pot for a watchlist job (default 10000).")),
		mcp.WithNumber("riskPct", mcp.Description("Share of current mark-to-market equity lost on a full stop (default 1). Open fills are marked at their last print; size is not only the last closed cash.")),
		mcp.WithNumber("maxOpenRiskPct", mcp.Description("Max combined stop risk of open fills (default 3). A new signal is skipped until a fill closes and frees risk.")),
		mcp.WithBoolean("useTp2", mcp.Description("When true, close tp1ClosePct at TP1 and leave the rest for TP2. Default false (TP1 only).")),
		mcp.WithNumber("tp1ClosePct", mcp.Description("Share of the position closed at TP1 when useTp2 is true (default 50).")),
		mcp.WithString("stopAfterTp1", mcp.Description("After TP1: keep the original stop, entry to move it to the fill, or trail to ratchet with price and never give it back (default entry).")),
		mcp.WithNumber("feePct", mcp.Description("Taker fee percent on each fill, including TP1 slices (e.g. 0.10 = 0.10%). Default 0.")),
		mcp.WithNumber("slipPct", mcp.Description("Adverse slippage percent used when that minute has no stored order book (e.g. 0.05 = 0.05%). A stored ladder is walked for the trade size instead. Default 0.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		start, err := req.RequireString("rangeStart")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		end, err := req.RequireString("rangeEnd")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol := req.GetString("symbol", "")
		if scope := strings.TrimSpace(req.GetString("scope", "")); strings.EqualFold(scope, domain.BacktestScopeWatchlist) && symbol == "" {
			symbol = domain.BacktestWatchlistSymbol
		}
		raw, err := api.StartBacktest(ctx, clientID, req.GetString("ruleId", ""), req.GetString("profileId", ""), req.GetString("exchange", "binance"), symbol, start, end, req.GetFloat("startingBalance", 0), req.GetFloat("riskPct", 0), req.GetFloat("maxOpenRiskPct", 0), req.GetBool("useTp2", false), req.GetFloat("tp1ClosePct", 0), req.GetString("stopAfterTp1", ""), req.GetFloat("feePct", 0), req.GetFloat("slipPct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_backtests",
		mcp.WithDescription("List historical backtest jobs (scanner rules and custom signal profiles) for a client."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithNumber("limit", mcp.Description("Max jobs, default 50")),
		mcp.WithNumber("offset", mcp.Description("Skip this many jobs")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListBacktests(ctx, clientID, int(req.GetFloat("limit", 50)), int(req.GetFloat("offset", 0)))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("get_backtest",
		mcp.WithDescription("Get one backtest job's progress and summary."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Backtest job id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.GetBacktest(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("cancel_backtest",
		mcp.WithDescription("Cancel a pending or running backtest job."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Backtest job id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CancelBacktest(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_backtest_signals",
		mcp.WithDescription("List backtest signals with 1/5/20-day forward returns, first-touch TP/SL trade results (side, leverage, entry, stop, target, outcome, leveraged PnL), and a performance summary (average, median, win rate, sample count per horizon plus trade TP/SL counts). Watchlist jobs include exchange/symbol on each signal and performance.markets per coin. Same-bar TP+SL uses a smaller interval when available. Custom-mix hits include factors (id, score, weight, status) for why the signal fired."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Backtest job id")),
		mcp.WithNumber("limit", mcp.Description("Max signals, default 100")),
		mcp.WithNumber("offset", mcp.Description("Skip this many signals")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListBacktestSignals(ctx, clientID, id, int(req.GetFloat("limit", 100)), int(req.GetFloat("offset", 0)))
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

	addTool(mcp.NewTool("list_custom_signal_factors",
		mcp.WithDescription("List mixable custom-signal factors (swing patterns + optional hunt_up, forensics, similar_setup) and built-in percents. Tape factors score whether forensics/similar setups support a long or go against it."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := api.ListCustomSignalFactors(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_custom_signal_profiles",
		mcp.WithDescription("List saved custom-signal weight mixes for a clientId."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ListCustomSignalProfiles(ctx, clientID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("create_custom_signal_profile",
		mcp.WithDescription("Save a named custom-signal mix. weights is factor id → percent and must sum to 100. Omitted factors are 0."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("name", mcp.Required(), mcp.Description("Profile name")),
		mcp.WithString("weightsJson", mcp.Required(), mcp.Description("JSON object of factor id to percent, e.g. {\"hunt_up\":40,\"rsi_recovery\":60}")),
		mcp.WithBoolean("isDefault", mcp.Description("If true, Custom tab opens this mix")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		weights, err := parseCustomSignalWeightsJSON(req.GetString("weightsJson", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreateCustomSignalProfile(ctx, clientID, name, weights, req.GetBool("isDefault", false))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("update_custom_signal_profile",
		mcp.WithDescription("Update a saved custom-signal mix (name and/or weights). weightsJson must sum to 100 when set."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Profile id")),
		mcp.WithString("name", mcp.Description("New name")),
		mcp.WithString("weightsJson", mcp.Description("JSON object of factor id to percent")),
		mcp.WithBoolean("isDefault", mcp.Description("Set true to make this the Custom tab default")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var weights map[string]float64
		if rawW := req.GetString("weightsJson", ""); rawW != "" {
			weights, err = parseCustomSignalWeightsJSON(rawW)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
		}
		var def *bool
		if _, ok := req.GetArguments()["isDefault"]; ok {
			v := req.GetBool("isDefault", false)
			def = &v
		}
		raw, err := api.UpdateCustomSignalProfile(ctx, clientID, id, req.GetString("name", ""), weights, def)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("duplicate_custom_signal_profile",
		mcp.WithDescription("Copy a saved custom-signal mix. Optional name; default is \"{name} (copy)\"."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Profile id to copy")),
		mcp.WithString("name", mcp.Description("Name for the copy")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DuplicateCustomSignalProfile(ctx, clientID, id, req.GetString("name", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("set_default_custom_signal_profile",
		mcp.WithDescription("Mark a saved mix as the one the Custom tab opens with."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Profile id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.SetDefaultCustomSignalProfile(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("delete_custom_signal_profile",
		mcp.WithDescription("Delete a saved custom-signal mix."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("id", mcp.Required(), mcp.Description("Profile id")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.DeleteCustomSignalProfile(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_custom_signals",
		mcp.WithDescription("Scan the watchlist with a custom factor mix (swing patterns + optional hunt_up, forensics, similar_setup). Optional weight* must sum to 100. Missing optional factors add 0. Forensics and similar setups score whether the tape supports a long custom signal or goes against it, using the same range as the swing bar (4h signal → 4h after-path). Each item reports tapeSetups / tapeUsed / tapeThin, tapeFresh, and custom.evidence (strong/fair/weak plus fresh/old/missing factor counts). Evidence score is quality, not completeness. scoreCompare is default vs applied. Informational only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("profileId", mcp.Description("Saved mix id; ignored when any weight* is set")),
		mcp.WithString("exchange", mcp.Description("Optional venue filter")),
		mcp.WithNumber("limit", mcp.Description("Max symbols, default 25")),
		mcp.WithNumber("weightEmaCrossover", mcp.Description("Custom mix % for EMA 9/21. If any weight* is set, omitted factors are 0 and the mix must sum to 100.")),
		mcp.WithNumber("weightEmaTrend", mcp.Description("Custom mix % for EMA 50/200 pullback")),
		mcp.WithNumber("weightSupertrend", mcp.Description("Custom mix % for SuperTrend")),
		mcp.WithNumber("weightRsiRecovery", mcp.Description("Custom mix % for RSI recovery")),
		mcp.WithNumber("weightMacd", mcp.Description("Custom mix % for MACD histogram")),
		mcp.WithNumber("weightAdx", mcp.Description("Custom mix % for ADX trend")),
		mcp.WithNumber("weightAccumulation", mcp.Description("Custom mix % for BB squeeze / dry-up")),
		mcp.WithNumber("weightVolumeBreakout", mcp.Description("Custom mix % for volume breakout")),
		mcp.WithNumber("weightHuntUp", mcp.Description("Custom mix % for liquidation-hunt up score")),
		mcp.WithNumber("weightForensics", mcp.Description("Custom mix % for forensics tape (supports or goes against a long)")),
		mcp.WithNumber("weightSimilarSetup", mcp.Description("Custom mix % for similar-setup after-path (supports or goes against a long)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		argStr := func(k string) string {
			args := req.GetArguments()
			if args == nil {
				return ""
			}
			v, ok := args[k]
			if !ok || v == nil {
				return ""
			}
			return fmt.Sprint(v)
		}
		weights, err := domain.ParseCustomSignalWeights(argStr)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		hasW := false
		for _, id := range domain.CustomSignalFactorIDs {
			if argStr(domain.CustomSignalQueryKeys[id]) != "" {
				hasW = true
				break
			}
		}
		raw, err := api.ScanCustomSignals(ctx, clientID, req.GetString("profileId", ""), req.GetString("exchange", ""), int(req.GetFloat("limit", 25)), weights, hasW)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("list_scanner_results",
		mcp.WithDescription("List saved scanner match history for a clientId (deduped by rule/symbol/bar), plus confluence setups (grade A/B) and hits24h."),
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
		mcp.WithDescription("Start a background export of the user's watchlist, shares, alerts, backtests, paper portfolios (including allocation baskets, rebalance history, and risk limits), and/or custom signal profiles as json or csv. Only one active export per client. Poll get_export for progress; download via HTTP when completed."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("format", mcp.Description("json (default) or csv")),
		mcp.WithString("sections", mcp.Description("Comma-separated: watchlist,shares,alerts,backtests,portfolios,customSignals (default all)")),
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
		mcp.WithDescription("Preview restoring a prior JSON export of watchlist/shares/alerts/backtests/portfolios/customSignals. Paper books include allocation baskets, rebalance history, and risk limits. Returns valid/invalid/willAdd/willUpdate/willConflict counts without applying. customSignals includes addNames, updateNames, and conflicts (same name on more than one file row or saved mix). Unresolved conflicts are skipped. Pass export JSON as content. Then call confirm_import."),
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
		mcp.WithDescription("Start applying a previewed import. mode=merge (skip duplicates) or replace (clear then import). For custom-signal name conflicts pass customSignalResolutions JSON [{name, incomingIndex, profileId}]. Omitted conflicts are not imported. If saved mixes changed after preview, returns import_preview_stale — preview again. One active apply per client."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Owner client id")),
		mcp.WithString("importId", mcp.Required(), mcp.Description("Import job id from preview")),
		mcp.WithString("mode", mcp.Required(), mcp.Description("merge or replace")),
		mcp.WithString("customSignalResolutions", mcp.Description("JSON array of {name, incomingIndex?, profileId?} for same-name custom signal conflicts")),
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
		var resolutions []domain.CustomSignalResolution
		if rawRes := strings.TrimSpace(req.GetString("customSignalResolutions", "")); rawRes != "" {
			if err := json.Unmarshal([]byte(rawRes), &resolutions); err != nil {
				return mcp.NewToolResultError("customSignalResolutions must be a JSON array"), nil
			}
		}
		raw, err := api.ConfirmImport(ctx, clientID, id, mode, resolutions)
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
		mcp.WithDescription("Track a coin across chosen spot venues (default Binance+Coinbase+Bybit). Opens an opportunity only when that notional can be bought and sold on fresh live books and after-fee profit (including slippage) is at least minProfit. minDurationSec requires the fill to stay qualifying that long; a break resets the timer. exchanges is a CSV such as binance,bybit (at least two)."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("notional", mcp.Required(), mcp.Description("Quote size to walk on the buy book e.g. 10000")),
		mcp.WithNumber("minProfit", mcp.Required(), mcp.Description("Minimum after-fee profit in quote currency e.g. 20")),
		mcp.WithNumber("minDurationSec", mcp.Description("Seconds the fill must stay qualifying before opening; 0 = first tick")),
		mcp.WithNumber("minNetDiffPct", mcp.Description("Optional extra profit % floor")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance fee % e.g. 0.1")),
		mcp.WithNumber("feeCoinbasePct", mcp.Description("Coinbase fee % e.g. 0.6")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit fee % e.g. 0.1")),
		mcp.WithString("exchanges", mcp.Description("CSV of venues to walk, e.g. binance,bybit. Default all three.")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		notional, err := req.RequireFloat("notional")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		minProfit, err := req.RequireFloat("minProfit")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.CreatePriceDiffWatch(ctx, clientID, symbol, notional, minProfit,
			req.GetFloat("minNetDiffPct", 0), req.GetFloat("minDurationSec", 0),
			req.GetFloat("feeBinancePct", 0), req.GetFloat("feeCoinbasePct", 0), req.GetFloat("feeBybitPct", 0),
			mcpCSVList(req, "exchanges"))
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

	addTool(mcp.NewTool("update_price_diff_watch",
		mcp.WithDescription("Change notional, minProfit, minDurationSec, fees, or exchanges on an existing price-diff watch. Resets the duration timer. Does not change pause/active status."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("watchId", mcp.Required(), mcp.Description("Watch id")),
		mcp.WithNumber("notional", mcp.Description("New quote size to walk")),
		mcp.WithNumber("minProfit", mcp.Description("New after-fee profit floor")),
		mcp.WithNumber("minDurationSec", mcp.Description("Seconds the fill must stay qualifying; 0 = first tick")),
		mcp.WithNumber("minNetDiffPct", mcp.Description("Optional extra profit % floor")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance fee %")),
		mcp.WithNumber("feeCoinbasePct", mcp.Description("Coinbase fee %")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit fee %")),
		mcp.WithString("exchanges", mcp.Description("CSV of venues to walk, e.g. binance,bybit (at least two)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("watchId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.UpdatePriceDiffWatch(ctx, clientID, id,
			optionalMCPFee(req, "notional"), optionalMCPFee(req, "minProfit"),
			optionalMCPFee(req, "minNetDiffPct"), optionalMCPFee(req, "minDurationSec"),
			optionalMCPFee(req, "feeBinancePct"), optionalMCPFee(req, "feeCoinbasePct"), optionalMCPFee(req, "feeBybitPct"),
			mcpCSVList(req, "exchanges"))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("pause_price_diff_watch",
		mcp.WithDescription("Pause a price-diff watch. Marks it paused first so an in-flight tick cannot open. Closes any open opportunity. Resume starts the duration timer from zero."),
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
		raw, err := api.PausePriceDiffWatch(ctx, clientID, id)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("resume_price_diff_watch",
		mcp.WithDescription("Resume a paused price-diff watch. Does not continue a duration wait from before pause."),
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
		raw, err := api.ResumePriceDiffWatch(ctx, clientID, id)
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
		mcp.WithDescription("List cross-exchange price opportunities (status open|closed|all). Each row includes open/peak/last edge, duration, fade start, close reason, and a review summary. Open opportunities persist across restarts."),
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
		mcp.WithDescription("Get one price-diff opportunity by id, including how long it stayed open, the peak after-fee edge, when it started fading, why it closed, and the sampled path (ticks)."),
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

	addTool(mcp.NewTool("quote_price_diff",
		mcp.WithDescription("Walk live spot books on two venues for a size (notional USDT or quantity). Returns average buy/sell, slippage, profit after fees, and the largest still-profitable size. Provide notional or quantity, not both. Informational only."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithString("buyExchange", mcp.Required(), mcp.Description("Venue to buy on: binance | coinbase | bybit")),
		mcp.WithString("sellExchange", mcp.Required(), mcp.Description("Venue to sell on: binance | coinbase | bybit")),
		mcp.WithNumber("notional", mcp.Description("Quote currency to spend on the buy book before the buy fee e.g. 10000")),
		mcp.WithNumber("quantity", mcp.Description("Base size to buy and sell (use instead of notional)")),
		mcp.WithNumber("feeBuyPct", mcp.Description("Buy venue taker fee % e.g. 0.1")),
		mcp.WithNumber("feeSellPct", mcp.Description("Sell venue taker fee % e.g. 0.1")),
		mcp.WithNumber("minNetDiffPct", mcp.Description("Optional threshold to flag meetsMinNet")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		buyEx, err := req.RequireString("buyExchange")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		sellEx, err := req.RequireString("sellExchange")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.QuotePriceDiff(ctx, symbol, buyEx, sellEx,
			req.GetFloat("notional", 0), req.GetFloat("quantity", 0),
			req.GetFloat("feeBuyPct", 0), req.GetFloat("feeSellPct", 0), req.GetFloat("minNetDiffPct", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("scan_price_diff_quotes",
		mcp.WithDescription("Compare every Binance/Coinbase/Bybit buy-sell route at one size. Walks both live books with per-venue fees and ranks by after-fee profit. Shows how much of the entered money can actually be used. Provide notional or quantity, not both. Informational only."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Pair e.g. BTCUSDT")),
		mcp.WithNumber("notional", mcp.Description("USDT to spend on the buy book before the buy fee e.g. 10000")),
		mcp.WithNumber("quantity", mcp.Description("Base size to buy and sell (use instead of notional)")),
		mcp.WithNumber("feeBinancePct", mcp.Description("Binance taker fee % e.g. 0.1")),
		mcp.WithNumber("feeCoinbasePct", mcp.Description("Coinbase taker fee % e.g. 0.6")),
		mcp.WithNumber("feeBybitPct", mcp.Description("Bybit taker fee % e.g. 0.1")),
		mcp.WithNumber("minNetDiffPct", mcp.Description("Optional threshold to flag meetsMinNet")),
		mcp.WithNumber("minProfitPct", mcp.Description("Only list routes whose after-fee profit % is at least this")),
		mcp.WithNumber("minProfitAmount", mcp.Description("Only list routes whose after-fee profit (USDT) is at least this")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.ScanPriceDiffQuotes(ctx, symbol,
			req.GetFloat("notional", 0), req.GetFloat("quantity", 0),
			req.GetFloat("feeBinancePct", 0), req.GetFloat("feeCoinbasePct", 0), req.GetFloat("feeBybitPct", 0),
			req.GetFloat("minNetDiffPct", 0), req.GetFloat("minProfitPct", 0), req.GetFloat("minProfitAmount", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("quote_price_diff_watch",
		mcp.WithDescription("Scan all venue pairs for a stored watch at a size using that watch's fees. Ranked by after-fee profit and usable money. Provide notional or quantity. Informational only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("watchId", mcp.Required(), mcp.Description("Watch id")),
		mcp.WithNumber("notional", mcp.Description("USDT to spend on the buy book before the buy fee e.g. 10000")),
		mcp.WithNumber("quantity", mcp.Description("Base size to buy and sell (use instead of notional)")),
		mcp.WithNumber("minProfitPct", mcp.Description("Only list routes whose after-fee profit % is at least this")),
		mcp.WithNumber("minProfitAmount", mcp.Description("Only list routes whose after-fee profit (USDT) is at least this")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("watchId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.QuotePriceDiffWatch(ctx, clientID, id, req.GetFloat("notional", 0), req.GetFloat("quantity", 0),
			req.GetFloat("minProfitPct", 0), req.GetFloat("minProfitAmount", 0))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(PrettyJSON(raw)), nil
	})

	addTool(mcp.NewTool("quote_price_diff_opportunity",
		mcp.WithDescription("Quote a stored price-diff opportunity at a size using that watch's fees and the live books. Provide notional (USDT on the buy book) or quantity. Informational only."),
		mcp.WithString("clientId", mcp.Required(), mcp.Description("Opaque client id")),
		mcp.WithString("opportunityId", mcp.Required(), mcp.Description("Opportunity id")),
		mcp.WithNumber("notional", mcp.Description("Quote currency to spend on the buy book before the buy fee e.g. 10000")),
		mcp.WithNumber("quantity", mcp.Description("Base size to buy and sell (use instead of notional)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientID, err := req.RequireString("clientId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id, err := req.RequireString("opportunityId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		raw, err := api.QuotePriceDiffOpportunity(ctx, clientID, id, req.GetFloat("notional", 0), req.GetFloat("quantity", 0))
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
			"auth":        "REST: Bearer / X-API-Key. Browsers: POST /api/v1/realtime/ticket then ?ticket= on the WebSocket. Long-lived secrets are not accepted on the query string.",
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

func optionalMCPFee(req mcp.CallToolRequest, name string) *float64 {
	args := req.GetArguments()
	if args == nil {
		return nil
	}
	if _, ok := args[name]; !ok {
		return nil
	}
	v := req.GetFloat(name, 0)
	return &v
}

func mcpCSVList(req mcp.CallToolRequest, name string) []string {
	raw := strings.TrimSpace(req.GetString(name, ""))
	if raw == "" {
		if args := req.GetArguments(); args != nil {
			if arr, ok := args[name].([]any); ok {
				out := make([]string, 0, len(arr))
				for _, v := range arr {
					s := strings.TrimSpace(fmt.Sprint(v))
					if s != "" && s != "<nil>" {
						out = append(out, s)
					}
				}
				return out
			}
		}
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
