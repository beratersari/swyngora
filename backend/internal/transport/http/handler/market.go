package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

// MarketHandler is the transport adapter for market use cases.
type MarketHandler struct {
	svc *market.Service
}

// NewMarketHandler constructs the market HTTP handler.
func NewMarketHandler(svc *market.Service) *MarketHandler {
	return &MarketHandler{svc: svc}
}

type candleDTO struct {
	OpenTime    string `json:"openTime"`
	Open        string `json:"open"`
	High        string `json:"high"`
	Low         string `json:"low"`
	Close       string `json:"close"`
	Volume      string `json:"volume"`
	CloseTime   string `json:"closeTime"`
	QuoteVolume string `json:"quoteVolume"`
	TradeCount  int64  `json:"tradeCount"`
}

type candlesResponse struct {
	Symbol   string      `json:"symbol"`
	Interval string      `json:"interval"`
	Exchange string      `json:"exchange"`
	Candles  []candleDTO `json:"candles"`
}

// GetCandles handles GET /api/v1/market/candles
func (h *MarketHandler) GetCandles(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	symbol := q.Get("symbol")
	interval := q.Get("interval")
	if interval == "" {
		interval = "1h"
	}
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}

	var startPtr, endPtr *time.Time
	if raw := q.Get("startTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		startPtr = &t
	}
	if raw := q.Get("endTime"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		endPtr = &t
	}

	candles, err := h.svc.GetCandles(r.Context(), symbol, interval, limit, startPtr, endPtr)
	if err != nil {
		writeError(w, err)
		return
	}

	out := candlesResponse{
		Symbol:   symbol,
		Interval: interval,
		Exchange: "binance",
		Candles:  make([]candleDTO, 0, len(candles)),
	}
	for _, c := range candles {
		out.Candles = append(out.Candles, candleDTO{
			OpenTime:    c.OpenTime.UTC().Format(time.RFC3339Nano),
			Open:        c.Open,
			High:        c.High,
			Low:         c.Low,
			Close:       c.Close,
			Volume:      c.Volume,
			CloseTime:   c.CloseTime.UTC().Format(time.RFC3339Nano),
			QuoteVolume: c.QuoteVolume,
			TradeCount:  c.TradeCount,
		})
	}
	out.Symbol = strings.ToUpper(strings.TrimSpace(symbol))
	writeJSON(w, http.StatusOK, out)
}

type tickerResponse struct {
	Exchange           string `json:"exchange"`
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice          string `json:"lastPrice"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           string `json:"openTime"`
	CloseTime          string `json:"closeTime"`
	TradeCount         int64  `json:"tradeCount"`
}

// GetTicker24h handles GET /api/v1/market/ticker/24h
func (h *MarketHandler) GetTicker24h(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	tkr, err := h.svc.GetTicker24h(r.Context(), symbol)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tickerResponse{
		Exchange:           "binance",
		Symbol:             tkr.Symbol,
		PriceChange:        tkr.PriceChange,
		PriceChangePercent: tkr.PriceChangePercent,
		LastPrice:          tkr.LastPrice,
		OpenPrice:          tkr.OpenPrice,
		HighPrice:          tkr.HighPrice,
		LowPrice:           tkr.LowPrice,
		Volume:             tkr.Volume,
		QuoteVolume:        tkr.QuoteVolume,
		OpenTime:           tkr.OpenTime.UTC().Format(time.RFC3339Nano),
		CloseTime:          tkr.CloseTime.UTC().Format(time.RFC3339Nano),
		TradeCount:         tkr.TradeCount,
	})
}

type supplyResponse struct {
	Asset             string   `json:"asset"`
	Name              string   `json:"name"`
	ProviderID        string   `json:"providerId"`
	CirculatingSupply *float64 `json:"circulatingSupply"`
	TotalSupply       *float64 `json:"totalSupply"`
	MaxSupply         *float64 `json:"maxSupply"`
	CurrentPriceUSD   *float64 `json:"currentPriceUsd"`
	AsOf              string   `json:"asOf"`
	Source            string   `json:"source"`
	Note              string   `json:"note"`
}

// GetSupply handles GET /api/v1/market/supply
func (h *MarketHandler) GetSupply(w http.ResponseWriter, r *http.Request) {
	asset := r.URL.Query().Get("asset")
	if asset == "" {
		// Convenience: also accept symbol=BTCUSDT and strip to base.
		asset = r.URL.Query().Get("symbol")
	}
	sup, err := h.svc.GetSupply(r.Context(), asset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, supplyResponse{
		Asset:             sup.Asset,
		Name:              sup.Name,
		ProviderID:        sup.ProviderID,
		CirculatingSupply: sup.CirculatingSupply,
		TotalSupply:       sup.TotalSupply,
		MaxSupply:         sup.MaxSupply,
		CurrentPriceUSD:   sup.CurrentPriceUSD,
		AsOf:              sup.AsOf.UTC().Format(time.RFC3339Nano),
		Source:            sup.Source,
		Note:              "Supply metrics are not available from Binance public market APIs; sourced from free CoinGecko metadata.",
	})
}

type intervalsResponse struct {
	Intervals []string `json:"intervals"`
	Exchange  string   `json:"exchange"`
}

// GetIntervals handles GET /api/v1/market/intervals
func (h *MarketHandler) GetIntervals(w http.ResponseWriter, r *http.Request) {
	ivs := h.svc.ListIntervals()
	out := make([]string, len(ivs))
	for i, iv := range ivs {
		out[i] = string(iv)
	}
	writeJSON(w, http.StatusOK, intervalsResponse{
		Intervals: out,
		Exchange:  "binance",
	})
}

func parseTimeParam(raw string) (time.Time, error) {
	// Accept RFC3339 or Unix milliseconds.
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.UnixMilli(ms).UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: time must be RFC3339 or unix milliseconds", domain.ErrInvalidArgument)
}

