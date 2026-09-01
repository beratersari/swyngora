// Package mcp implements an MCP (Model Context Protocol) server that exposes
// Swyngora market and watchlist capabilities as tools for AI agents.
// Tools call the HTTP API (same contracts as OpenAPI) so the MCP process stays
// a thin adapter over application services.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// APIClient is a minimal HTTP client for the Swyngora backend API.
type APIClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewAPIClient constructs a client. baseURL is e.g. http://localhost:8080.
func NewAPIClient(baseURL string, timeout time.Duration) *APIClient {
	return NewAPIClientWithAuth(baseURL, timeout, "")
}

// NewAPIClientWithAuth is NewAPIClient plus a Bearer / X-API-Key secret for
// locked APIs (stdio MCP: API_AUTH_TOKEN or SWYNGORA_API_TOKEN).
func NewAPIClientWithAuth(baseURL string, timeout time.Duration, authToken string) *APIClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &APIClient{
		baseURL:    baseURL,
		authToken:  strings.TrimSpace(authToken),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *APIClient) applyAuth(req *http.Request) {
	if c == nil || req == nil || c.authToken == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("X-API-Key", c.authToken)
}

func applyPortfolioID(ctx context.Context, req *http.Request) {
	id := PortfolioIDFrom(ctx)
	if id == "" || req == nil {
		return
	}
	req.Header.Set("X-Portfolio-Id", id)
	q := req.URL.Query()
	q.Set("portfolioId", id)
	req.URL.RawQuery = q.Encode()
}

func (c *APIClient) get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	c.applyAuth(req)
	applyPortfolioID(ctx, req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, truncate(string(body), 400))
	}
	return json.RawMessage(body), nil
}

func (c *APIClient) sendJSON(ctx context.Context, method, path string, payload any) (json.RawMessage, error) {
	var rdr io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rdr = strings.NewReader(string(b))
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.applyAuth(req)
	applyPortfolioID(ctx, req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, truncate(string(body), 400))
	}
	return json.RawMessage(body), nil
}

// GetTicker returns 24h ticker JSON.
func (c *APIClient) GetTicker(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/ticker/24h", q)
}

// GetOrderBook returns a grouped spot order book plus ±rangePct analysis.
func (c *APIClient) GetOrderBook(ctx context.Context, exchange, symbol, group string, limit int, rangePct float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if group != "" {
		q.Set("group", group)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if rangePct > 0 {
		q.Set("rangePct", strconv.FormatFloat(rangePct, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/market/orderbook", q)
}

// GetBookHistory returns a stored book at a time, or a newest-first list.
func (c *APIClient) GetBookHistory(ctx context.Context, exchange, symbol, at, from, to string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if at != "" {
		q.Set("at", at)
	}
	if from != "" {
		q.Set("from", from)
	}
	if to != "" {
		q.Set("to", to)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/orderbook/history", q)
}

// CompareBookHistory diffs stored books nearest to two times.
func (c *APIClient) CompareBookHistory(ctx context.Context, exchange, symbol, from, to string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("from", from)
	q.Set("to", to)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/orderbook/history/compare", q)
}

// GetIcebergs returns prices where visible size was eaten and then refilled.
func (c *APIClient) GetIcebergs(ctx context.Context, exchange, symbol string, minNotional float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if minNotional > 0 {
		q.Set("minNotional", strconv.FormatFloat(minNotional, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/market/orderbook/icebergs", q)
}

// EstimateOrderBookImpact walks live depth for a simulated market order.
func (c *APIClient) EstimateOrderBookImpact(ctx context.Context, exchange, symbol, side string, quantity, notional float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if side != "" {
		q.Set("side", side)
	}
	if quantity > 0 {
		q.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/market/orderbook/impact", q)
}

// GetLiquidations returns rolling 5m/1h/4h/24h futures liquidation totals.
func (c *APIClient) GetLiquidations(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/liquidations", q)
}

// GetLiquidationOverview returns market-wide 1h/4h/12h/24h totals plus ranked coins.
func (c *APIClient) GetLiquidationOverview(ctx context.Context, exchange, window string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if window != "" {
		q.Set("window", window)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/liquidations/overview", q)
}

// GetLiquidationLevels returns price-level bars or market-wide time bars.
func (c *APIClient) GetLiquidationLevels(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if symbol != "" {
		q.Set("symbol", symbol)
	}
	if rawRange != "" {
		q.Set("range", rawRange)
	}
	return c.get(ctx, "/api/v1/market/liquidation-levels", q)
}

// GetLiquidationCascade scores a short-burst long/short liquidation cascade.
func (c *APIClient) GetLiquidationCascade(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if symbol != "" {
		q.Set("symbol", symbol)
	}
	return c.get(ctx, "/api/v1/market/liquidation-cascade", q)
}

// ScanLiquidationCascades scores market-wide cascade risk and lists bursting coins.
func (c *APIClient) ScanLiquidationCascades(ctx context.Context, exchange string) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/liquidation-cascade/scan", q)
}

// GetOrderBookHeatmap returns recent resting bid/ask size over time.
func (c *APIClient) GetOrderBookHeatmap(ctx context.Context, exchange, symbol, group string, windowSec int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if group != "" {
		q.Set("group", group)
	}
	if windowSec > 0 {
		q.Set("window", strconv.Itoa(windowSec))
	}
	return c.get(ctx, "/api/v1/market/orderbook/heatmap", q)
}

// GetOpenInterest returns current futures open interest and windowed change.
func (c *APIClient) GetOpenInterest(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/open-interest", q)
}

// GetFundingRate returns the predicted next funding rate plus recent settlements.
func (c *APIClient) GetFundingRate(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/funding-rate", q)
}

// GetFundingArb sizes a Binance/Bybit long-short funding trade.
func (c *APIClient) GetFundingArb(ctx context.Context, symbol string, notional, holdHours float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	if holdHours > 0 {
		q.Set("holdHours", strconv.FormatFloat(holdHours, 'f', -1, 64))
	}
	if feeBinancePct != nil {
		q.Set("feeBinancePct", strconv.FormatFloat(*feeBinancePct, 'f', -1, 64))
	}
	if feeBybitPct != nil {
		q.Set("feeBybitPct", strconv.FormatFloat(*feeBybitPct, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/market/funding-arb", q)
}

// ScanFundingArb ranks liquid coins by after-fee funding spread.
func (c *APIClient) ScanFundingArb(ctx context.Context, quote string, notional, holdHours float64, feeBinancePct, feeBybitPct *float64, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if quote != "" {
		q.Set("quote", quote)
	}
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	if holdHours > 0 {
		q.Set("holdHours", strconv.FormatFloat(holdHours, 'f', -1, 64))
	}
	if feeBinancePct != nil {
		q.Set("feeBinancePct", strconv.FormatFloat(*feeBinancePct, 'f', -1, 64))
	}
	if feeBybitPct != nil {
		q.Set("feeBybitPct", strconv.FormatFloat(*feeBybitPct, 'f', -1, 64))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/funding-arb/scan", q)
}

// GetFundingArbHistory lists past after-fee winning stretches.
func (c *APIClient) GetFundingArbHistory(ctx context.Context, symbol, from, to string, notional float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("from", from)
	q.Set("to", to)
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	if feeBinancePct != nil {
		q.Set("feeBinancePct", strconv.FormatFloat(*feeBinancePct, 'f', -1, 64))
	}
	if feeBybitPct != nil {
		q.Set("feeBybitPct", strconv.FormatFloat(*feeBybitPct, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/market/funding-arb/history", q)
}

// CreateFundingArbWatch creates a min-profit follow (omit symbol for the scan).
func (c *APIClient) CreateFundingArbWatch(ctx context.Context, clientID, symbol string, notional, holdHours, minProfit float64, quote string, limit int, durationHours float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "minProfit": minProfit,
	}
	if symbol != "" {
		body["symbol"] = symbol
	}
	if notional > 0 {
		body["notional"] = notional
	}
	if holdHours > 0 {
		body["holdHours"] = holdHours
	}
	if quote != "" {
		body["quote"] = quote
	}
	if limit > 0 {
		body["limit"] = limit
	}
	if durationHours > 0 {
		body["durationHours"] = durationHours
	}
	if feeBinancePct != nil {
		body["feeBinancePct"] = *feeBinancePct
	}
	if feeBybitPct != nil {
		body["feeBybitPct"] = *feeBybitPct
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/funding-arb/watches", body)
}

// ListFundingArbWatches lists follow watches.
func (c *APIClient) ListFundingArbWatches(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/funding-arb/watches", q)
}

// GetFundingArbWatch gets one follow watch.
func (c *APIClient) GetFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/funding-arb/watches/"+url.PathEscape(id), q)
}

// UpdateFundingArbWatch patches follow settings.
func (c *APIClient) UpdateFundingArbWatch(ctx context.Context, clientID, id string, notional, holdHours, minProfit *float64, quote *string, limit *int, durationHours *float64, feeBinancePct, feeBybitPct *float64) (json.RawMessage, error) {
	body := map[string]any{"clientId": clientID}
	if notional != nil {
		body["notional"] = *notional
	}
	if holdHours != nil {
		body["holdHours"] = *holdHours
	}
	if minProfit != nil {
		body["minProfit"] = *minProfit
	}
	if quote != nil {
		body["quote"] = *quote
	}
	if limit != nil {
		body["limit"] = *limit
	}
	if durationHours != nil {
		body["durationHours"] = *durationHours
	}
	if feeBinancePct != nil {
		body["feeBinancePct"] = *feeBinancePct
	}
	if feeBybitPct != nil {
		body["feeBybitPct"] = *feeBybitPct
	}
	return c.sendJSON(ctx, http.MethodPatch, "/api/v1/funding-arb/watches/"+url.PathEscape(id), body)
}

// PauseFundingArbWatch pauses a follow without deleting it.
func (c *APIClient) PauseFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/funding-arb/watches/" + url.PathEscape(id) + "/pause"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, nil)
}

// ResumeFundingArbWatch resumes a paused follow.
func (c *APIClient) ResumeFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/funding-arb/watches/" + url.PathEscape(id) + "/resume"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, nil)
}

// DeleteFundingArbWatch deletes a follow watch.
func (c *APIClient) DeleteFundingArbWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/funding-arb/watches/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodDelete, path, nil)
}

// ListFundingArbSignals lists min-profit crossings.
func (c *APIClient) ListFundingArbSignals(ctx context.Context, clientID, status string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if status != "" {
		q.Set("status", status)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/funding-arb/signals", q)
}

// GetLongShortRatio returns the latest account long/short ratio plus recent history.
func (c *APIClient) GetLongShortRatio(ctx context.Context, exchange, symbol string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/long-short-ratio", q)
}

// GetLiquidationHunt returns the hypothetical per-venue liquidation-hunt model.
func (c *APIClient) GetLiquidationHunt(ctx context.Context, exchange, symbol string, weights domain.HuntScoreWeights) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if weights.Source == "custom" {
		for id, key := range domain.HuntScoreQueryKeys {
			if v, ok := weights.Pct[id]; ok {
				q.Set(key, strconv.FormatFloat(v, 'f', -1, 64))
			}
		}
	}
	return c.get(ctx, "/api/v1/market/liquidation-hunt", q)
}

// GetLiquidationHuntHeatmap returns a price × time liquidation intensity grid.
func (c *APIClient) GetLiquidationHuntHeatmap(ctx context.Context, exchange, symbol, rawRange string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if rawRange != "" {
		q.Set("range", rawRange)
	}
	return c.get(ctx, "/api/v1/market/liquidation-hunt/heatmap", q)
}

// GetSqueezeRisk returns long/short squeeze risk scores per venue and combined.
func (c *APIClient) GetSqueezeRisk(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/squeeze-risk", q)
}

// GetPositioning returns price+OI buildup/unwinding regimes per venue and combined.
func (c *APIClient) GetPositioning(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/positioning", q)
}

// GetVenueDivergence compares Binance vs Bybit signals for one coin.
func (c *APIClient) GetVenueDivergence(ctx context.Context, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	return c.get(ctx, "/api/v1/market/venue-divergence", q)
}

// GetTakerFlow returns aggressive futures buy vs sell volume windows.
func (c *APIClient) GetTakerFlow(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/taker-flow", q)
}

// GetCVD returns cumulative taker buy−sell versus price.
func (c *APIClient) GetCVD(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/cvd", q)
}

// GetVolumeSurge returns current vs typical volume (5m / 15m / 1h, buy/sell).
func (c *APIClient) GetVolumeSurge(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/volume-surge", q)
}

// ScanVolumeSurges ranks coins by how far above typical their volume is.
func (c *APIClient) ScanVolumeSurges(ctx context.Context, exchange, quote string, minRatio float64, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if quote != "" {
		q.Set("quote", quote)
	}
	if minRatio > 0 {
		q.Set("minRatio", strconv.FormatFloat(minRatio, 'f', -1, 64))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/volume-surge/scan", q)
}

// GetRSIHeatmap returns a ranked Wilder RSI scatter for top listed pairs.
func (c *APIClient) GetRSIHeatmap(ctx context.Context, exchange, quote, interval, sort string, limit, period int) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if quote != "" {
		q.Set("quote", quote)
	}
	if interval != "" {
		q.Set("interval", interval)
	}
	if sort != "" {
		q.Set("sort", sort)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if period > 0 {
		q.Set("period", strconv.Itoa(period))
	}
	return c.get(ctx, "/api/v1/market/rsi-heatmap", q)
}

// GetLiquiditySweeps returns pokes through a prior high/low that came back.
func (c *APIClient) GetLiquiditySweeps(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/liquidity-sweeps", q)
}

// GetAbsorption returns aggressive volume versus price hold (who is absorbing).
func (c *APIClient) GetAbsorption(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/absorption", q)
}

// GetAround returns before/during/after tape around an event time.
func (c *APIClient) GetAround(ctx context.Context, exchange, symbol, at, window, during string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("at", at)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if window != "" {
		q.Set("window", window)
	}
	if during != "" {
		q.Set("during", during)
	}
	return c.get(ctx, "/api/v1/market/around", q)
}

// CompareAround diffs two around-the-move tapes for the same coin.
func (c *APIClient) CompareAround(ctx context.Context, exchange, symbol, from, to, window, during string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("from", from)
	q.Set("to", to)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if window != "" {
		q.Set("window", window)
	}
	if during != "" {
		q.Set("during", during)
	}
	return c.get(ctx, "/api/v1/market/around/compare", q)
}

// FindAroundMoves finds strong up/down legs and attaches the around tape.
func (c *APIClient) FindAroundMoves(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if lookback != "" {
		q.Set("lookback", lookback)
	}
	if interval != "" {
		q.Set("interval", interval)
	}
	if direction != "" {
		q.Set("direction", direction)
	}
	if minReturnPct > 0 {
		q.Set("minReturnPct", strconv.FormatFloat(minReturnPct, 'f', -1, 64))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if window != "" {
		q.Set("window", window)
	}
	if during != "" {
		q.Set("during", during)
	}
	return c.get(ctx, "/api/v1/market/around/moves", q)
}

// GetAroundPrecursors compares the tape before many important moves.
func (c *APIClient) GetAroundPrecursors(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if lookback != "" {
		q.Set("lookback", lookback)
	}
	if interval != "" {
		q.Set("interval", interval)
	}
	if direction != "" {
		q.Set("direction", direction)
	}
	if minReturnPct > 0 {
		q.Set("minReturnPct", strconv.FormatFloat(minReturnPct, 'f', -1, 64))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if window != "" {
		q.Set("window", window)
	}
	if during != "" {
		q.Set("during", during)
	}
	return c.get(ctx, "/api/v1/market/around/precursors", q)
}

// GetAroundSimilar ranks past important-move setups against the current tape.
func (c *APIClient) GetAroundSimilar(ctx context.Context, exchange, symbol, lookback, interval, direction string, minReturnPct float64, limit int, window, during, fields, weights, minCoverage, horizons string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if lookback != "" {
		q.Set("lookback", lookback)
	}
	if interval != "" {
		q.Set("interval", interval)
	}
	if direction != "" {
		q.Set("direction", direction)
	}
	if minReturnPct > 0 {
		q.Set("minReturnPct", strconv.FormatFloat(minReturnPct, 'f', -1, 64))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if window != "" {
		q.Set("window", window)
	}
	if during != "" {
		q.Set("during", during)
	}
	if fields != "" {
		q.Set("fields", fields)
	}
	if weights != "" {
		q.Set("weights", weights)
	}
	if minCoverage != "" {
		q.Set("minCoverage", minCoverage)
	}
	if horizons != "" {
		q.Set("horizons", horizons)
	}
	return c.get(ctx, "/api/v1/market/around/similar", q)
}

// GetVWAP returns volume-weighted average price from startTime (or window) to now.
func (c *APIClient) GetVWAP(ctx context.Context, exchange, symbol, window, startTime, endTime string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if window != "" {
		q.Set("window", window)
	}
	if startTime != "" {
		q.Set("startTime", startTime)
	}
	if endTime != "" {
		q.Set("endTime", endTime)
	}
	return c.get(ctx, "/api/v1/market/vwap", q)
}

// GetVolumeProfile returns traded volume by price (POC + value area).
func (c *APIClient) GetVolumeProfile(ctx context.Context, exchange, symbol, window, startTime, endTime string, tickSize float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if window != "" {
		q.Set("window", window)
	}
	if startTime != "" {
		q.Set("startTime", startTime)
	}
	if endTime != "" {
		q.Set("endTime", endTime)
	}
	if tickSize > 0 {
		q.Set("tickSize", strconv.FormatFloat(tickSize, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/market/volume-profile", q)
}

// GetBasis returns futures-vs-spot premium/discount per venue.
func (c *APIClient) GetBasis(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/basis", q)
}

// GetCorrelation returns how similarly a coin has been moving with BTC and ETH.
func (c *APIClient) GetCorrelation(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/correlation", q)
}

// GetBreadth returns how many followed coins are up or down over 1h / 4h / 24h.
func (c *APIClient) GetBreadth(ctx context.Context, exchange string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/breadth", q)
}

// GetVolatility returns how much a coin has been moving vs its history and vs BTC/ETH.
func (c *APIClient) GetVolatility(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/volatility", q)
}

// GetSnapshot returns price, volume, mcap, OI, funding, long/short, and taker flow together.
func (c *APIClient) GetSnapshot(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/snapshot", q)
}

// GetLevels returns support/resistance areas and breakout strength.
func (c *APIClient) GetLevels(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/levels", q)
}

// GetWhales returns the largest recent aggressive trades and liquidations.
func (c *APIClient) GetWhales(ctx context.Context, exchange, symbol string, minNotional float64, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if symbol != "" {
		q.Set("symbol", symbol)
	}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if minNotional > 0 {
		q.Set("minNotional", strconv.FormatFloat(minNotional, 'f', -1, 64))
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/whales", q)
}

// GetFuturesHistory returns durable stored futures samples or liquidation events.
func (c *APIClient) GetFuturesHistory(ctx context.Context, metric, exchange, symbol, from, to string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("metric", metric)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if from != "" {
		q.Set("from", from)
	}
	if to != "" {
		q.Set("to", to)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/futures-history", q)
}

// GetMarketLiquidity scores ±0.1 / ±0.5 / ±1% depth per venue and market-wide.
func (c *APIClient) GetMarketLiquidity(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/orderbook/liquidity", q)
}

// AnalyzeCombinedOrderBook sums live books from all venues in one price band.
func (c *APIClient) AnalyzeCombinedOrderBook(ctx context.Context, symbol string, rangePct float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if rangePct > 0 {
		q.Set("rangePct", strconv.FormatFloat(rangePct, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/market/orderbook/combined", q)
}

// AnalyzeOrderBook returns pressure/imbalance/walls from live depth in ±rangePct of mid.
func (c *APIClient) AnalyzeOrderBook(ctx context.Context, exchange, symbol string, rangePct float64) (json.RawMessage, error) {
	raw, err := c.GetOrderBook(ctx, exchange, symbol, "", 5, rangePct)
	if err != nil {
		return nil, err
	}
	var full map[string]any
	if err := json.Unmarshal(raw, &full); err != nil {
		return raw, nil
	}
	keep := map[string]any{}
	for _, k := range []string{"exchange", "symbol", "lastPrice", "bestBid", "bestAsk", "spread", "spreadPct", "live", "source", "updatedAt", "analysis"} {
		if v, ok := full[k]; ok {
			keep[k] = v
		}
	}
	out, err := json.Marshal(keep)
	if err != nil {
		return raw, nil
	}
	return out, nil
}

// GetCandles returns OHLCV candles.
func (c *APIClient) GetCandles(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if interval != "" {
		q.Set("interval", interval)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/candles", q)
}

// GetSupply returns asset supply snapshot.
func (c *APIClient) GetSupply(ctx context.Context, asset string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("asset", asset)
	return c.get(ctx, "/api/v1/market/supply", q)
}

// GetAssetProfile returns logo, listing date, and contracts for a crypto asset.
func (c *APIClient) GetAssetProfile(ctx context.Context, asset string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("asset", asset)
	return c.get(ctx, "/api/v1/market/asset-profile", q)
}

// GetHolders returns an on-chain holder snapshot for a crypto asset.
func (c *APIClient) GetHolders(ctx context.Context, asset string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("asset", asset)
	return c.get(ctx, "/api/v1/market/holders", q)
}

// ListSpot returns spot markets page.
func (c *APIClient) ListSpot(ctx context.Context, exchange, query, quote, sort, order, tag string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if query != "" {
		q.Set("q", query)
	}
	if quote != "" {
		q.Set("quote", quote)
	}
	if sort != "" {
		q.Set("sort", sort)
	}
	if order != "" {
		q.Set("order", order)
	}
	if tag != "" {
		q.Set("tag", tag)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/market/spot", q)
}

// GetIndicators returns RSI/EMA series.
func (c *APIClient) GetIndicators(ctx context.Context, exchange, symbol, interval string, limit, rsiPeriod int, emaPeriods string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if interval != "" {
		q.Set("interval", interval)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if rsiPeriod > 0 {
		q.Set("rsiPeriod", strconv.Itoa(rsiPeriod))
	}
	if emaPeriods != "" {
		q.Set("emaPeriods", emaPeriods)
	}
	return c.get(ctx, "/api/v1/market/indicators", q)
}

// ListExchanges returns configured venues.
func (c *APIClient) ListExchanges(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/v1/market/exchanges", nil)
}

// GetFxRates returns USD-based spot FX for display conversion.
func (c *APIClient) GetFxRates(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/api/v1/market/fx", nil)
}

// GetWatchlist fetches a client watchlist.

func (c *APIClient) ListDelistSchedule(ctx context.Context, exchange string) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/delist-schedule", q)
}

// GetPostDelist returns off-venue movement after this exchange halted the pair.
func (c *APIClient) GetPostDelist(ctx context.Context, exchange, symbol, interval string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if symbol != "" {
		q.Set("symbol", symbol)
	}
	if interval != "" {
		q.Set("interval", interval)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/market/post-delist", q)
}

func (c *APIClient) GetWatchlist(ctx context.Context, clientID string) (json.RawMessage, error) {
	return c.GetWatchlistOwned(ctx, clientID, "")
}

// GetWatchlistOwned fetches own or shared list (ownerClientID).
func (c *APIClient) GetWatchlistOwned(ctx context.Context, actorClientID, ownerClientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", actorClientID)
	if ownerClientID != "" {
		q.Set("ownerClientId", ownerClientID)
	}
	return c.get(ctx, "/api/v1/watchlist", q)
}

// AddWatchlistItem adds a symbol to a watchlist.
func (c *APIClient) AddWatchlistItem(ctx context.Context, clientID, exchange, symbol, note string) (json.RawMessage, error) {
	return c.AddWatchlistItemOwned(ctx, clientID, "", exchange, symbol, note)
}

// AddWatchlistItemOwned adds a symbol; ownerClientID empty = actor's list.
func (c *APIClient) AddWatchlistItemOwned(ctx context.Context, actorClientID, ownerClientID, exchange, symbol, note string) (json.RawMessage, error) {
	body := map[string]string{
		"clientId": actorClientID,
		"exchange": exchange,
		"symbol":   symbol,
		"note":     note,
	}
	if ownerClientID != "" {
		body["ownerClientId"] = ownerClientID
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/watchlist/items", body)
}

// RemoveWatchlistItem removes a symbol.
func (c *APIClient) RemoveWatchlistItem(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error) {
	return c.RemoveWatchlistItemOwned(ctx, clientID, "", exchange, symbol)
}

// RemoveWatchlistItemOwned removes a symbol from own or shared list.
func (c *APIClient) RemoveWatchlistItemOwned(ctx context.Context, actorClientID, ownerClientID, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", actorClientID)
	q.Set("exchange", exchange)
	q.Set("symbol", symbol)
	if ownerClientID != "" {
		q.Set("ownerClientId", ownerClientID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/v1/watchlist/items?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, truncate(string(body), 400))
	}
	return json.RawMessage(body), nil
}

// ShareWatchlist grants viewer/editor access (owner only).
func (c *APIClient) ShareWatchlist(ctx context.Context, ownerClientID, granteeClientID, role string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/watchlist/shares", map[string]string{
		"clientId": ownerClientID, "granteeClientId": granteeClientID, "role": role,
	})
}

// UpdateWatchlistShare changes an existing share role.
func (c *APIClient) UpdateWatchlistShare(ctx context.Context, ownerClientID, granteeClientID, role string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPatch, "/api/v1/watchlist/shares", map[string]string{
		"clientId": ownerClientID, "granteeClientId": granteeClientID, "role": role,
	})
}

// RevokeWatchlistShare removes access.
func (c *APIClient) RevokeWatchlistShare(ctx context.Context, ownerClientID, granteeClientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", ownerClientID)
	q.Set("granteeClientId", granteeClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/v1/watchlist/shares?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, truncate(string(body), 400))
	}
	return json.RawMessage(body), nil
}

// ListWatchlistShares lists shares granted by owner.
func (c *APIClient) ListWatchlistShares(ctx context.Context, ownerClientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", ownerClientID)
	return c.get(ctx, "/api/v1/watchlist/shares", q)
}

// ListSharedWatchlists lists lists shared with grantee.
func (c *APIClient) ListSharedWatchlists(ctx context.Context, granteeClientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", granteeClientID)
	return c.get(ctx, "/api/v1/watchlist/shared", q)
}

// ListWatchlistAudit returns change history for owner's list.
func (c *APIClient) ListWatchlistAudit(ctx context.Context, ownerClientID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", ownerClientID)
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	return c.get(ctx, "/api/v1/watchlist/audit", q)
}

// ListPriceAlerts lists alerts for a client.
func (c *APIClient) ListPriceAlerts(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/alerts", q)
}

// CreatePriceAlert creates a price alert (one_time or repeating).
func (c *APIClient) CreatePriceAlert(ctx context.Context, clientID, exchange, symbol, condition string, targetPrice float64, mode string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/alerts", map[string]any{
		"clientId":    clientID,
		"exchange":    exchange,
		"symbol":      symbol,
		"condition":   condition,
		"targetPrice": targetPrice,
		"mode":        mode,
	})
}

// CreateOrderBookAlert creates an imbalance or wall alert on the live local book.
func (c *APIClient) CreateOrderBookAlert(ctx context.Context, clientID, exchange, symbol, kind, condition string, threshold, rangePct float64, mode string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/alerts", map[string]any{
		"clientId":    clientID,
		"exchange":    exchange,
		"symbol":      symbol,
		"kind":        kind,
		"condition":   condition,
		"targetPrice": threshold,
		"rangePct":    rangePct,
		"mode":        mode,
	})
}

// CreateLiquidationFeedAlert alerts when a venue's liquidation socket stays down.
func (c *APIClient) CreateLiquidationFeedAlert(ctx context.Context, clientID, exchange string, minDownSeconds float64, mode string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/alerts", map[string]any{
		"clientId":    clientID,
		"exchange":    exchange,
		"kind":        "liquidation_feed",
		"targetPrice": minDownSeconds,
		"mode":        mode,
	})
}

// CreateLiquidationCascadeAlert alerts when a coin hits a cascade grade.
func (c *APIClient) CreateLiquidationCascadeAlert(ctx context.Context, clientID, exchange, symbol, minGrade, mode string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/alerts", map[string]any{
		"clientId":  clientID,
		"exchange":  exchange,
		"symbol":    symbol,
		"kind":      "liquidation_cascade",
		"condition": minGrade,
		"mode":      mode,
	})
}

// CreateLiquidationNotionalAlert alerts when window notional crosses a USDT line.
func (c *APIClient) CreateLiquidationNotionalAlert(ctx context.Context, clientID, exchange, symbol, side string, notional float64, window, mode string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/alerts", map[string]any{
		"clientId":    clientID,
		"exchange":    exchange,
		"symbol":      symbol,
		"kind":        "liquidation_notional",
		"condition":   side,
		"targetPrice": notional,
		"window":      window,
		"mode":        mode,
	})
}

// DeletePriceAlert deletes an alert by id.
func (c *APIClient) DeletePriceAlert(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/v1/alerts/"+url.PathEscape(id)+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, truncate(string(body), 400))
	}
	return json.RawMessage(body), nil
}

// GetAlertWebhook fetches the client's webhook settings.
func (c *APIClient) GetAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/alerts/webhook", q)
}

// SetAlertWebhook sets the client's webhook URL (immediate delivery).
func (c *APIClient) SetAlertWebhook(ctx context.Context, clientID, webhookURL string) (json.RawMessage, error) {
	return c.SetAlertWebhookWithMode(ctx, clientID, webhookURL, "immediate")
}

// SetAlertWebhookWithMode sets webhook URL and delivery mode.
func (c *APIClient) SetAlertWebhookWithMode(ctx context.Context, clientID, webhookURL, deliveryMode string) (json.RawMessage, error) {
	return c.SetAlertWebhookSettings(ctx, clientID, webhookURL, deliveryMode, "UTC", false, "", "")
}

// SetAlertWebhookSettings sets full webhook notification preferences.
func (c *APIClient) SetAlertWebhookSettings(ctx context.Context, clientID, webhookURL, deliveryMode, timeZone string, quietEnabled bool, quietStart, quietEnd string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPut, "/api/v1/alerts/webhook", map[string]any{
		"clientId":     clientID,
		"url":          webhookURL,
		"deliveryMode": deliveryMode,
		"timeZone":     timeZone,
		"quietHours": map[string]any{
			"enabled": quietEnabled,
			"start":   quietStart,
			"end":     quietEnd,
		},
	})
}

// DeleteAlertWebhook clears the client's webhook URL.
func (c *APIClient) DeleteAlertWebhook(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"/api/v1/alerts/webhook?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, truncate(string(body), 400))
	}
	return json.RawMessage(body), nil
}

// CreateAPIKey creates a named user API key.
func (c *APIClient) CreateAPIKey(ctx context.Context, clientID, name, permission string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/account/api-keys", map[string]any{
		"clientId": clientID, "name": name, "permission": permission,
	})
}

// ListAPIKeys lists named keys (no secrets).
func (c *APIClient) ListAPIKeys(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/account/api-keys", q)
}

// RevokeAPIKey revokes a named key.
func (c *APIClient) RevokeAPIKey(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.sendJSON(ctx, http.MethodDelete, "/api/v1/account/api-keys/"+url.PathEscape(id)+"?"+q.Encode(), nil)
}

// CreatePortfolio creates a paper portfolio.
func (c *APIClient) CreatePortfolio(ctx context.Context, clientID string, startingBalance float64, currency string) (json.RawMessage, error) {
	return c.CreateNamedPortfolio(ctx, clientID, startingBalance, currency, "")
}

func (c *APIClient) CreateNamedPortfolio(ctx context.Context, clientID string, startingBalance float64, currency, name string) (json.RawMessage, error) {
	body := map[string]any{"clientId": clientID, "startingBalance": startingBalance, "currency": currency}
	if strings.TrimSpace(name) != "" {
		body["name"] = name
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio", body)
}

func (c *APIClient) ListPortfolios(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolios", q)
}

func (c *APIClient) RenamePortfolio(ctx context.Context, clientID, id, name string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.sendJSON(ctx, http.MethodPatch, "/api/v1/portfolios/"+url.PathEscape(id)+"?"+q.Encode(), map[string]any{"name": name})
}

func (c *APIClient) DeletePortfolio(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.sendJSON(ctx, http.MethodDelete, "/api/v1/portfolios/"+url.PathEscape(id)+"?"+q.Encode(), nil)
}

func (c *APIClient) SharePortfolio(ctx context.Context, clientID, portfolioID, granteeClientID, role string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/shares", map[string]any{
		"clientId": clientID, "portfolioId": portfolioID, "granteeClientId": granteeClientID, "role": role,
	})
}

func (c *APIClient) UpdatePortfolioShare(ctx context.Context, clientID, portfolioID, granteeClientID, role string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPatch, "/api/v1/portfolio/shares", map[string]any{
		"clientId": clientID, "portfolioId": portfolioID, "granteeClientId": granteeClientID, "role": role,
	})
}

func (c *APIClient) RevokePortfolioShare(ctx context.Context, clientID, portfolioID, granteeClientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	q.Set("portfolioId", portfolioID)
	q.Set("granteeClientId", granteeClientID)
	return c.sendJSON(ctx, http.MethodDelete, "/api/v1/portfolio/shares?"+q.Encode(), nil)
}

func (c *APIClient) ListPortfolioShares(ctx context.Context, clientID, portfolioID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if strings.TrimSpace(portfolioID) != "" {
		q.Set("portfolioId", portfolioID)
	}
	return c.get(ctx, "/api/v1/portfolio/shares", q)
}

func (c *APIClient) ListSharedPortfolios(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolios/shared", q)
}

// DepositPortfolioCash adds virtual cash.
func (c *APIClient) DepositPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/deposits", map[string]any{
		"clientId": clientID, "amount": amount, "note": note,
	})
}

// WithdrawPortfolioCash removes available virtual cash.
func (c *APIClient) WithdrawPortfolioCash(ctx context.Context, clientID string, amount float64, note string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/withdrawals", map[string]any{
		"clientId": clientID, "amount": amount, "note": note,
	})
}

func (c *APIClient) TransferPortfolioCash(ctx context.Context, clientID, fromPortfolioID, toPortfolioID string, amount float64, note string) (json.RawMessage, error) {
	body := map[string]any{"clientId": clientID, "toPortfolioId": toPortfolioID, "amount": amount}
	if strings.TrimSpace(fromPortfolioID) != "" {
		body["fromPortfolioId"] = fromPortfolioID
	}
	if note != "" {
		body["note"] = note
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/transfers", body)
}

// ListPortfolioCashMovements lists deposit/withdraw history.
func (c *APIClient) ListPortfolioCashMovements(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/portfolio/cash-movements", q)
}

// GetPortfolioPerformance fetches equity history + period P&L.
func (c *APIClient) GetPortfolioPerformance(ctx context.Context, clientID, period string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if strings.TrimSpace(period) != "" {
		q.Set("period", period)
	}
	return c.get(ctx, "/api/v1/portfolio/performance", q)
}

// GetPortfolio fetches paper portfolio view.
func (c *APIClient) GetPortfolio(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio", q)
}

func (c *APIClient) GetPaperTradingCosts(ctx context.Context, exchange string) (json.RawMessage, error) {
	q := url.Values{}
	if strings.TrimSpace(exchange) != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/portfolio/trading-costs", q)
}

func (c *APIClient) GetPortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/risk-limits", q)
}

func (c *APIClient) PutPortfolioRiskLimits(ctx context.Context, clientID string, maxDailyLossPct, maxAssetWeightPct *float64) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/risk-limits"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	body := map[string]any{}
	if maxDailyLossPct != nil {
		body["maxDailyLossPct"] = *maxDailyLossPct
	}
	if maxAssetWeightPct != nil {
		body["maxAssetWeightPct"] = *maxAssetWeightPct
	}
	return c.sendJSON(ctx, http.MethodPut, path, body)
}

func (c *APIClient) DeletePortfolioRiskLimits(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.sendJSON(ctx, http.MethodDelete, "/api/v1/portfolio/risk-limits?"+q.Encode(), nil)
}

// PlacePortfolioOrder places a paper market order.
func (c *APIClient) PlacePortfolioOrder(ctx context.Context, clientID, exchange, symbol, side string, quantity float64, lotMethod string) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol, "side": side, "quantity": quantity,
	}
	if lotMethod != "" {
		body["lotMethod"] = lotMethod
	}
	if k := IdempotencyKeyFrom(ctx); k != "" {
		body["idempotencyKey"] = k
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/orders", body)
}

func (c *APIClient) ListPortfolioLots(ctx context.Context, clientID, exchange, symbol, status string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if symbol != "" {
		q.Set("symbol", symbol)
	}
	if status != "" {
		q.Set("status", status)
	}
	return c.get(ctx, "/api/v1/portfolio/lots", q)
}

// PlacePortfolioPendingOrder places a limit/stop/trailing paper order.
func (c *APIClient) PlacePortfolioPendingOrder(ctx context.Context, clientID, exchange, symbol, orderType string, quantity, triggerPrice float64, timeInForce, expiresAt, trailType string, trailValue float64, lotMethod string) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol, "type": orderType,
		"quantity": quantity, "triggerPrice": triggerPrice,
	}
	if lotMethod != "" {
		body["lotMethod"] = lotMethod
	}
	if timeInForce != "" {
		body["timeInForce"] = timeInForce
	}
	if expiresAt != "" {
		body["expiresAt"] = expiresAt
	}
	if trailType != "" {
		body["trailType"] = trailType
	}
	if trailValue > 0 {
		body["trailValue"] = trailValue
	}
	if k := IdempotencyKeyFrom(ctx); k != "" {
		body["idempotencyKey"] = k
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/orders", body)
}

// PlacePortfolioBracketOrder places limit-buy entry with pending TP/SL exits.
func (c *APIClient) PlacePortfolioBracketOrder(ctx context.Context, clientID, exchange, symbol string, quantity, entryPrice, takeProfitPrice, stopLossPrice float64, expiresAt string) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol, "type": "bracket",
		"quantity": quantity, "triggerPrice": entryPrice,
		"takeProfitPrice": takeProfitPrice, "stopLossPrice": stopLossPrice,
	}
	if expiresAt != "" {
		body["expiresAt"] = expiresAt
	}
	if k := IdempotencyKeyFrom(ctx); k != "" {
		body["idempotencyKey"] = k
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/orders", body)
}

// PlacePortfolioOCOOrder places linked take-profit + stop-loss paper orders.
func (c *APIClient) PlacePortfolioOCOOrder(ctx context.Context, clientID, exchange, symbol string, quantity, takeProfitPrice, stopLossPrice float64, expiresAt string) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol, "type": "oco",
		"quantity": quantity, "takeProfitPrice": takeProfitPrice, "stopLossPrice": stopLossPrice,
	}
	if expiresAt != "" {
		body["expiresAt"] = expiresAt
	}
	if k := IdempotencyKeyFrom(ctx); k != "" {
		body["idempotencyKey"] = k
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/orders", body)
}

// GetPortfolioOrder returns one pending order plus amend hints.
func (c *APIClient) GetPortfolioOrder(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	return c.get(ctx, "/api/v1/portfolio/orders/"+url.PathEscape(id), q)
}

// AmendPortfolioOrder patches triggerPrice and/or remainingQuantity of an open pending order.
func (c *APIClient) AmendPortfolioOrder(ctx context.Context, clientID, id string, triggerPrice, remainingQuantity *float64) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/orders/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	body := map[string]any{}
	if triggerPrice != nil {
		body["triggerPrice"] = *triggerPrice
	}
	if remainingQuantity != nil {
		body["remainingQuantity"] = *remainingQuantity
	}
	return c.sendJSON(ctx, http.MethodPatch, path, body)
}

// ListPortfolioOrders lists paper pending orders.
func (c *APIClient) ListPortfolioOrders(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	if status != "" {
		q.Set("status", status)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/portfolio/orders", q)
}

// CancelAllPortfolioOrders cancels all open pending orders, or one market when symbol is set.
func (c *APIClient) CancelAllPortfolioOrders(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error) {
	body := map[string]any{"clientId": clientID}
	if exchange != "" {
		body["exchange"] = exchange
	}
	if symbol != "" {
		body["symbol"] = symbol
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/orders/cancel-all", body)
}

// CancelPortfolioOrder cancels an open pending paper order.
func (c *APIClient) CancelPortfolioOrder(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/orders/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodDelete, path, nil)
}

// ListPortfolioTrades lists paper trades.
func (c *APIClient) ListPortfolioTrades(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if limit > 0 {
		q.Set("limit", fmt.Sprint(limit))
	}
	if offset > 0 {
		q.Set("offset", fmt.Sprint(offset))
	}
	return c.get(ctx, "/api/v1/portfolio/trades", q)
}

// CreateRecurringBuyPlan creates a paper recurring buy plan.
func (c *APIClient) CreateRecurringBuyPlan(ctx context.Context, clientID, exchange, symbol string, amount float64, frequency, startAt, name, weekday string, dayOfMonth, intervalHours int, timeZone string, hour, minute int, maxPrice, budget float64, endDate, endsAt string) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol,
		"amount": amount, "frequency": frequency,
	}
	if startAt != "" {
		body["startAt"] = startAt
	}
	if name != "" {
		body["name"] = name
	}
	if weekday != "" {
		body["weekday"] = weekday
	}
	if dayOfMonth > 0 {
		body["dayOfMonth"] = dayOfMonth
	}
	if intervalHours > 0 {
		body["intervalHours"] = intervalHours
	}
	if timeZone != "" {
		body["timeZone"] = timeZone
	}
	if hour != domain.RecurringHourUnset {
		body["hour"] = hour
		body["minute"] = minute
	}
	if maxPrice > 0 {
		body["maxPrice"] = maxPrice
	}
	if budget > 0 {
		body["budget"] = budget
	}
	if endDate != "" {
		body["endDate"] = endDate
	}
	if endsAt != "" {
		body["endsAt"] = endsAt
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/recurring-buys", body)
}

// UpdateRecurringBuyPlan patches name/amount/schedule. Zero/empty optional fields are omitted.
func (c *APIClient) UpdateRecurringBuyPlan(ctx context.Context, clientID, id, name, frequency, weekday, startAt string, amount float64, dayOfMonth, intervalHours int, timeZone string, hour, minute int, maxPrice, budget float64, endDate, endsAt string, clearEnds bool) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/recurring-buys/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if frequency != "" {
		body["frequency"] = frequency
	}
	if weekday != "" {
		body["weekday"] = weekday
	}
	if startAt != "" {
		body["startAt"] = startAt
	}
	if amount > 0 {
		body["amount"] = amount
	}
	if dayOfMonth > 0 {
		body["dayOfMonth"] = dayOfMonth
	}
	if intervalHours > 0 {
		body["intervalHours"] = intervalHours
	}
	if timeZone != "" {
		body["timeZone"] = timeZone
	}
	if hour != domain.RecurringHourUnset {
		body["hour"] = hour
		body["minute"] = minute
	}
	if maxPrice >= 0 {
		body["maxPrice"] = maxPrice
	}
	if budget >= 0 {
		body["budget"] = budget
	}
	if clearEnds {
		empty := ""
		body["endDate"] = empty
		body["endsAt"] = empty
	} else {
		if endDate != "" {
			body["endDate"] = endDate
		}
		if endsAt != "" {
			body["endsAt"] = endsAt
		}
	}
	return c.sendJSON(ctx, http.MethodPatch, path, body)
}

// ListRecurringBuyPlans lists paper recurring buy plans.
func (c *APIClient) ListRecurringBuyPlans(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/recurring-buys", q)
}

// GetRecurringBuyPlan fetches one plan.
func (c *APIClient) GetRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/recurring-buys/"+url.PathEscape(id), q)
}

// PauseRecurringBuyPlan pauses a plan.
func (c *APIClient) PauseRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/recurring-buys/" + url.PathEscape(id) + "/pause"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, nil)
}

// ResumeRecurringBuyPlan resumes a paused plan.
func (c *APIClient) ResumeRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/recurring-buys/" + url.PathEscape(id) + "/resume"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, nil)
}

// CreatePortfolioBasket creates a named allocation basket.
func (c *APIClient) CreatePortfolioBasket(ctx context.Context, clientID, name, targetsJSON string) (json.RawMessage, error) {
	var targets any
	if err := json.Unmarshal([]byte(targetsJSON), &targets); err != nil {
		return nil, err
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/baskets", map[string]any{
		"clientId": clientID, "name": name, "targets": targets,
	})
}

func (c *APIClient) ListPortfolioBaskets(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/baskets", q)
}

func (c *APIClient) GetPortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/baskets/"+url.PathEscape(id), q)
}

func (c *APIClient) UpdatePortfolioBasket(ctx context.Context, clientID, id, name, targetsJSON string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/baskets/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if strings.TrimSpace(targetsJSON) != "" {
		var targets any
		if err := json.Unmarshal([]byte(targetsJSON), &targets); err != nil {
			return nil, err
		}
		body["targets"] = targets
	}
	return c.sendJSON(ctx, http.MethodPatch, path, body)
}

func (c *APIClient) DeletePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	path := "/api/v1/portfolio/baskets/" + url.PathEscape(id) + "?" + q.Encode()
	return c.sendJSON(ctx, http.MethodDelete, path, nil)
}

func (c *APIClient) PreviewPortfolioRebalance(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/baskets/"+url.PathEscape(id)+"/preview", q)
}

func (c *APIClient) RebalancePortfolioBasket(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/baskets/" + url.PathEscape(id) + "/rebalance"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, map[string]any{})
}

// DeleteRecurringBuyPlan deletes a plan.
func (c *APIClient) DeleteRecurringBuyPlan(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/recurring-buys/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodDelete, path, nil)
}

// ListRecurringBuyRuns lists execution history for a plan.
func (c *APIClient) ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/portfolio/recurring-buys/"+url.PathEscape(planID)+"/runs", q)
}

// SetMarginMode sets isolated|cross.
func (c *APIClient) SetMarginMode(ctx context.Context, clientID, mode string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPut, "/api/v1/portfolio/margin/mode", map[string]any{
		"clientId": clientID, "mode": mode,
	})
}

// AdjustMargin adds/removes isolated position margin.
func (c *APIClient) AdjustMargin(ctx context.Context, clientID, positionID string, delta float64) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/margin/positions/" + url.PathEscape(positionID) + "/margin"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, map[string]any{"delta": delta})
}

// RepayMarginDebt repays interest then principal.
func (c *APIClient) RepayMarginDebt(ctx context.Context, clientID, positionID string, amount float64) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/margin/positions/" + url.PathEscape(positionID) + "/repay"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, map[string]any{"amount": amount})
}

// PlaceMarginOrder opens paper margin long/short.
func (c *APIClient) PlaceMarginOrder(ctx context.Context, clientID, exchange, symbol, side, orderType string, quantity float64, leverage int, limitPrice float64, stopLoss, takeProfit *float64) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol, "side": side, "type": orderType,
		"quantity": quantity, "leverage": leverage,
	}
	if limitPrice > 0 {
		body["limitPrice"] = limitPrice
	}
	if stopLoss != nil {
		body["stopLoss"] = *stopLoss
	}
	if takeProfit != nil {
		body["takeProfit"] = *takeProfit
	}
	if k := IdempotencyKeyFrom(ctx); k != "" {
		body["idempotencyKey"] = k
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/margin/orders", body)
}

// ListMarginPositions lists open margin positions.
func (c *APIClient) ListMarginPositions(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/margin/positions", q)
}

// GetMarginPosition gets one margin position.
func (c *APIClient) GetMarginPosition(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio/margin/positions/"+url.PathEscape(id), q)
}

// CloseMarginPosition closes full or partial margin position.
func (c *APIClient) CloseMarginPosition(ctx context.Context, clientID, id string, quantity float64) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/margin/positions/" + url.PathEscape(id) + "/close"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	body := map[string]any{}
	if quantity > 0 {
		body["quantity"] = quantity
	}
	if k := IdempotencyKeyFrom(ctx); k != "" {
		body["idempotencyKey"] = k
	}
	return c.sendJSON(ctx, http.MethodPost, path, body)
}

// SetMarginBrackets sets SL/TP on a margin position.
func (c *APIClient) SetMarginBrackets(ctx context.Context, clientID, id string, stopLoss, takeProfit *float64, clearSL, clearTP bool) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/margin/positions/" + url.PathEscape(id) + "/brackets"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	body := map[string]any{"clearStopLoss": clearSL, "clearTakeProfit": clearTP}
	if stopLoss != nil {
		body["stopLoss"] = *stopLoss
	}
	if takeProfit != nil {
		body["takeProfit"] = *takeProfit
	}
	return c.sendJSON(ctx, http.MethodPut, path, body)
}

// ListMarginOrders lists margin orders.
func (c *APIClient) ListMarginOrders(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if status != "" {
		q.Set("status", status)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/portfolio/margin/orders", q)
}

// CancelMarginOrder cancels a margin limit order.
func (c *APIClient) CancelMarginOrder(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/portfolio/margin/orders/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodDelete, path, nil)
}

// ListMarginTrades lists margin trades.
func (c *APIClient) ListMarginTrades(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/portfolio/margin/trades", q)
}

// CreatePriceDiffWatch creates a cross-exchange price difference watch.
func (c *APIClient) CreatePriceDiffWatch(ctx context.Context, clientID, symbol string, notional, minProfit, minNetDiffPct, minDurationSec, feeBinance, feeCoinbase, feeBybit float64, exchanges []string) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "symbol": symbol, "notional": notional, "minProfit": minProfit,
		"minNetDiffPct": minNetDiffPct, "minDurationSec": minDurationSec,
		"feeBinancePct": feeBinance, "feeCoinbasePct": feeCoinbase, "feeBybitPct": feeBybit,
	}
	if len(exchanges) > 0 {
		body["exchanges"] = exchanges
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/price-diff/watches", body)
}

// ListPriceDiffWatches lists watches.
func (c *APIClient) ListPriceDiffWatches(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/price-diff/watches", q)
}

// GetPriceDiffWatch gets one watch.
func (c *APIClient) GetPriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/price-diff/watches/"+url.PathEscape(id), q)
}

// UpdatePriceDiffWatch patches watch settings.
func (c *APIClient) UpdatePriceDiffWatch(ctx context.Context, clientID, id string, notional, minProfit, minNetDiffPct, minDurationSec, feeBinance, feeCoinbase, feeBybit *float64, exchanges []string) (json.RawMessage, error) {
	body := map[string]any{"clientId": clientID}
	if notional != nil {
		body["notional"] = *notional
	}
	if minProfit != nil {
		body["minProfit"] = *minProfit
	}
	if minNetDiffPct != nil {
		body["minNetDiffPct"] = *minNetDiffPct
	}
	if minDurationSec != nil {
		body["minDurationSec"] = *minDurationSec
	}
	if feeBinance != nil {
		body["feeBinancePct"] = *feeBinance
	}
	if feeCoinbase != nil {
		body["feeCoinbasePct"] = *feeCoinbase
	}
	if feeBybit != nil {
		body["feeBybitPct"] = *feeBybit
	}
	if len(exchanges) > 0 {
		body["exchanges"] = exchanges
	}
	return c.sendJSON(ctx, http.MethodPatch, "/api/v1/price-diff/watches/"+url.PathEscape(id), body)
}

// PausePriceDiffWatch pauses a watch and closes open opportunities.
func (c *APIClient) PausePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/price-diff/watches/" + url.PathEscape(id) + "/pause"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, nil)
}

// ResumePriceDiffWatch resumes a paused watch with a fresh duration timer.
func (c *APIClient) ResumePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/price-diff/watches/" + url.PathEscape(id) + "/resume"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, nil)
}

// DeletePriceDiffWatch deletes a watch.
func (c *APIClient) DeletePriceDiffWatch(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/price-diff/watches/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodDelete, path, nil)
}

// ListPriceDiffOpportunities lists opportunities.
func (c *APIClient) ListPriceDiffOpportunities(ctx context.Context, clientID, status string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if status != "" {
		q.Set("status", status)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/price-diff/opportunities", q)
}

// GetPriceDiffOpportunity gets one opportunity.
func (c *APIClient) GetPriceDiffOpportunity(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/price-diff/opportunities/"+url.PathEscape(id), q)
}

// QuotePriceDiff walks two live books for a size.
func (c *APIClient) QuotePriceDiff(ctx context.Context, symbol, buyExchange, sellExchange string, notional, quantity, feeBuyPct, feeSellPct, minNetDiffPct float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("buyExchange", buyExchange)
	q.Set("sellExchange", sellExchange)
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	if quantity > 0 {
		q.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if feeBuyPct != 0 {
		q.Set("feeBuyPct", strconv.FormatFloat(feeBuyPct, 'f', -1, 64))
	}
	if feeSellPct != 0 {
		q.Set("feeSellPct", strconv.FormatFloat(feeSellPct, 'f', -1, 64))
	}
	if minNetDiffPct != 0 {
		q.Set("minNetDiffPct", strconv.FormatFloat(minNetDiffPct, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/price-diff/quote", q)
}

// ScanPriceDiffQuotes compares every venue pair at one size.
func (c *APIClient) ScanPriceDiffQuotes(ctx context.Context, symbol string, notional, quantity, feeBinance, feeCoinbase, feeBybit, minNetDiffPct, minProfitPct, minProfitAmount float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("symbol", symbol)
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	if quantity > 0 {
		q.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if feeBinance != 0 {
		q.Set("feeBinancePct", strconv.FormatFloat(feeBinance, 'f', -1, 64))
	}
	if feeCoinbase != 0 {
		q.Set("feeCoinbasePct", strconv.FormatFloat(feeCoinbase, 'f', -1, 64))
	}
	if feeBybit != 0 {
		q.Set("feeBybitPct", strconv.FormatFloat(feeBybit, 'f', -1, 64))
	}
	if minNetDiffPct != 0 {
		q.Set("minNetDiffPct", strconv.FormatFloat(minNetDiffPct, 'f', -1, 64))
	}
	if minProfitPct != 0 {
		q.Set("minProfitPct", strconv.FormatFloat(minProfitPct, 'f', -1, 64))
	}
	if minProfitAmount != 0 {
		q.Set("minProfitAmount", strconv.FormatFloat(minProfitAmount, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/price-diff/quote/scan", q)
}

// QuotePriceDiffWatch scans all pairs for a stored watch.
func (c *APIClient) QuotePriceDiffWatch(ctx context.Context, clientID, watchID string, notional, quantity, minProfitPct, minProfitAmount float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	if quantity > 0 {
		q.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	if minProfitPct != 0 {
		q.Set("minProfitPct", strconv.FormatFloat(minProfitPct, 'f', -1, 64))
	}
	if minProfitAmount != 0 {
		q.Set("minProfitAmount", strconv.FormatFloat(minProfitAmount, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/price-diff/watches/"+url.PathEscape(watchID)+"/quote", q)
}

// QuotePriceDiffOpportunity quotes a stored opportunity at a size.
func (c *APIClient) QuotePriceDiffOpportunity(ctx context.Context, clientID, id string, notional, quantity float64) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if notional > 0 {
		q.Set("notional", strconv.FormatFloat(notional, 'f', -1, 64))
	}
	if quantity > 0 {
		q.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	}
	return c.get(ctx, "/api/v1/price-diff/opportunities/"+url.PathEscape(id)+"/quote", q)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Health checks the remote API liveness (HTTP adapter mode).
func (c *APIClient) Health(ctx context.Context) (json.RawMessage, error) {
	return c.get(ctx, "/health", nil)
}

// DetectPumpEvents calls GET /api/v1/market/pumps.
func (c *APIClient) DetectPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	q := url.Values{}
	setQ := func(k string) {
		if v, ok := args[k]; ok && v != nil {
			s := fmt.Sprint(v)
			if s != "" && s != "0" && s != "0.0" {
				q.Set(k, s)
			}
		}
	}
	// Always set required/defaults carefully
	if s, ok := args["symbol"].(string); ok {
		q.Set("symbol", s)
	} else if v, ok := args["symbol"]; ok {
		q.Set("symbol", fmt.Sprint(v))
	}
	for _, k := range []string{
		"exchange", "interval", "mode", "direction", "startTime", "endTime",
		"lookbackHours", "limit", "minReturnPct", "windowBars", "minVolumeRatio", "maxEvents",
	} {
		setQ(k)
	}
	// allow 0 for some numeric? minReturnPct 0 is invalid server-side; skip empty
	return c.get(ctx, "/api/v1/market/pumps", q)
}

// ScanPumpEvents calls GET /api/v1/market/pumps/scan.
func (c *APIClient) ScanPumpEvents(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	q := url.Values{}
	for _, k := range []string{
		"exchange", "quote", "interval", "mode", "direction",
		"lookbackHours", "minReturnPct", "windowBars", "minVolumeRatio", "symbolLimit", "maxTotalEvents",
	} {
		if v, ok := args[k]; ok && v != nil {
			s := fmt.Sprint(v)
			if s != "" {
				q.Set(k, s)
			}
		}
	}
	return c.get(ctx, "/api/v1/market/pumps/scan", q)
}

// PrettyJSON returns indented JSON for tool results.
func PrettyJSON(raw json.RawMessage) string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(b)
}

// CreateScannerRule creates a scanner rule via HTTP.
func (c *APIClient) CreateScannerRule(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/scanner/rules", args)
}

// ListScannerRules lists scanner rules.
func (c *APIClient) ListScannerRules(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	return c.get(ctx, "/api/v1/scanner/rules", q)
}

// UpdateScannerRule patches a scanner rule via HTTP.
func (c *APIClient) UpdateScannerRule(ctx context.Context, args map[string]any) (json.RawMessage, error) {
	id, _ := args["id"].(string)
	if id == "" {
		id, _ = args["ruleId"].(string)
	}
	return c.sendJSON(ctx, http.MethodPatch, "/api/v1/scanner/rules/"+url.PathEscape(id), args)
}

// DeleteScannerRule deletes a scanner rule.
func (c *APIClient) DeleteScannerRule(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	path := "/api/v1/scanner/rules/" + url.PathEscape(id)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodDelete, path, nil)
}

// AnalyzeSwing runs the swing engine on one symbol.
func (c *APIClient) AnalyzeSwing(ctx context.Context, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	q.Set("symbol", symbol)
	return c.get(ctx, "/api/v1/market/swing", q)
}

// ScanSwingSetups scans the client watchlist for swing setups.
func (c *APIClient) ScanSwingSetups(ctx context.Context, clientID, exchange string, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	return c.get(ctx, "/api/v1/swing/setups", q)
}

// ListScannerResults lists scanner match history.
func (c *APIClient) ListScannerResults(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	if clientID != "" {
		q.Set("clientId", clientID)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/scanner/results", q)
}

// StartExport queues a user data export job.
func (c *APIClient) StartExport(ctx context.Context, clientID, format string, sections []string) (json.RawMessage, error) {
	body := map[string]any{"clientId": clientID, "format": format}
	if len(sections) > 0 {
		body["sections"] = sections
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/export", body)
}

// GetExport returns export job status.
func (c *APIClient) GetExport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/export/"+url.PathEscape(id), q)
}

// ListExports lists export jobs for a client.
func (c *APIClient) ListExports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/export", q)
}

// CancelExport cancels a pending/running export.
func (c *APIClient) CancelExport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	path := "/api/v1/export/" + url.PathEscape(id) + "/cancel"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return c.sendJSON(ctx, http.MethodPost, path, map[string]any{})
}

// PreviewImport uploads raw export bytes for preview (JSON or CSV body).
func (c *APIClient) PreviewImport(ctx context.Context, clientID, fileName, format string, fileBytes []byte) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if format != "" {
		q.Set("format", format)
	}
	if fileName != "" {
		q.Set("fileName", fileName)
	}
	ct := "application/json"
	if format == "csv" || strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		ct = "text/csv"
	}
	path := "/api/v1/import/preview?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(fileBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ct)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API %s: %s", resp.Status, truncate(string(body), 400))
	}
	return json.RawMessage(body), nil
}

// ConfirmImport starts applying a previewed import.
func (c *APIClient) ConfirmImport(ctx context.Context, clientID, id, mode string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/import/"+url.PathEscape(id)+"/confirm", map[string]string{
		"clientId": clientID, "mode": mode,
	})
}

// GetImport returns import job status.
func (c *APIClient) GetImport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/import/"+url.PathEscape(id), q)
}

// ListImports lists import jobs.
func (c *APIClient) ListImports(ctx context.Context, clientID string, limit, offset int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	return c.get(ctx, "/api/v1/import", q)
}

// CancelImport cancels a previewed/pending/running import.
func (c *APIClient) CancelImport(ctx context.Context, clientID, id string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	path := "/api/v1/import/" + url.PathEscape(id) + "/cancel?" + q.Encode()
	return c.sendJSON(ctx, http.MethodPost, path, map[string]any{})
}
