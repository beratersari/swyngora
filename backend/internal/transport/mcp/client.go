// Package mcp implements an MCP (Model Context Protocol) server that exposes
// Swyngora market and watchlist capabilities as tools for AI agents.
// Tools call the HTTP API (same contracts as OpenAPI) so the MCP process stays
// a thin adapter over application services.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// APIClient is a minimal HTTP client for the Swyngora backend API.
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient constructs a client. baseURL is e.g. http://localhost:8080.
func NewAPIClient(baseURL string, timeout time.Duration) *APIClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
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

// GetWatchlist fetches a client watchlist.

func (c *APIClient) ListDelistSchedule(ctx context.Context, exchange string) (json.RawMessage, error) {
	q := url.Values{}
	if exchange != "" {
		q.Set("exchange", exchange)
	}
	return c.get(ctx, "/api/v1/market/delist-schedule", q)
}

func (c *APIClient) GetWatchlist(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/watchlist", q)
}

// AddWatchlistItem adds a symbol to a watchlist.
func (c *APIClient) AddWatchlistItem(ctx context.Context, clientID, exchange, symbol, note string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/watchlist/items", map[string]string{
		"clientId": clientID,
		"exchange": exchange,
		"symbol":   symbol,
		"note":     note,
	})
}

// RemoveWatchlistItem removes a symbol.
func (c *APIClient) RemoveWatchlistItem(ctx context.Context, clientID, exchange, symbol string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	q.Set("exchange", exchange)
	q.Set("symbol", symbol)
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

// CreatePortfolio creates a paper portfolio.
func (c *APIClient) CreatePortfolio(ctx context.Context, clientID string, startingBalance float64, currency string) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio", map[string]any{
		"clientId": clientID, "startingBalance": startingBalance, "currency": currency,
	})
}

// GetPortfolio fetches paper portfolio view.
func (c *APIClient) GetPortfolio(ctx context.Context, clientID string) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("clientId", clientID)
	return c.get(ctx, "/api/v1/portfolio", q)
}

// PlacePortfolioOrder places a paper market order.
func (c *APIClient) PlacePortfolioOrder(ctx context.Context, clientID, exchange, symbol, side string, quantity float64) (json.RawMessage, error) {
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/orders", map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol, "side": side, "quantity": quantity,
	})
}

// PlacePortfolioPendingOrder places a limit/stop paper order.
func (c *APIClient) PlacePortfolioPendingOrder(ctx context.Context, clientID, exchange, symbol, orderType string, quantity, triggerPrice float64, timeInForce, expiresAt string) (json.RawMessage, error) {
	body := map[string]any{
		"clientId": clientID, "exchange": exchange, "symbol": symbol, "type": orderType,
		"quantity": quantity, "triggerPrice": triggerPrice,
	}
	if timeInForce != "" {
		body["timeInForce"] = timeInForce
	}
	if expiresAt != "" {
		body["expiresAt"] = expiresAt
	}
	return c.sendJSON(ctx, http.MethodPost, "/api/v1/portfolio/orders", body)
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
