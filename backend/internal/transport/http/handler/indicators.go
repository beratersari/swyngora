package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

// GetIndicators handles GET /api/v1/market/indicators
func (h *MarketHandler) GetIndicators(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	exchange := q.Get("exchange")
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
	rsiPeriod := 0
	if raw := q.Get("rsiPeriod"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: rsiPeriod must be an integer", domain.ErrInvalidArgument))
			return
		}
		rsiPeriod = n
	}
	emaPeriods := market.ParseEMAPeriodsCSV(q.Get("emaPeriods"))

	ser, err := h.svc.GetIndicators(r.Context(), exchange, symbol, interval, limit, rsiPeriod, emaPeriods)
	if err != nil {
		writeError(w, err)
		return
	}

	type pointDTO struct {
		OpenTime string             `json:"openTime"`
		Close    float64            `json:"close"`
		RSI      *float64           `json:"rsi"`
		EMA      map[string]float64 `json:"ema"`
	}
	points := make([]pointDTO, 0, len(ser.Points))
	for _, p := range ser.Points {
		ema := map[string]float64{}
		for k, v := range p.EMA {
			if v != nil {
				ema[strconv.Itoa(k)] = *v
			}
		}
		points = append(points, pointDTO{
			OpenTime: p.OpenTime.UTC().Format(time.RFC3339Nano),
			Close:    p.Close,
			RSI:      p.RSI,
			EMA:      ema,
		})
	}
	latestEMA := map[string]float64{}
	for k, v := range ser.LatestEMA {
		if v != nil {
			latestEMA[strconv.Itoa(k)] = *v
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"exchange":   string(ser.Exchange),
		"symbol":     ser.Symbol,
		"interval":   string(ser.Interval),
		"rsiPeriod":  ser.RSIPeriod,
		"emaPeriods": ser.EMAPeriods,
		"latest": map[string]any{
			"rsi": ser.LatestRSI,
			"ema": latestEMA,
		},
		"points": points,
		"note":   "Informational analysis only — not financial advice. RSI uses Wilder's smoothing; EMA seeded with SMA.",
	})
}

// PostIndicatorsBatch handles POST /api/v1/market/indicators/batch
// Body: { exchange, interval, symbols: [], rsiPeriod?, emaPeriods?: "12,26" }
func (h *MarketHandler) PostIndicatorsBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Exchange   string   `json:"exchange"`
		Interval   string   `json:"interval"`
		Symbols    []string `json:"symbols"`
		RSIPeriod  int      `json:"rsiPeriod"`
		EMAPeriods string   `json:"emaPeriods"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	if body.Interval == "" {
		body.Interval = "1h"
	}
	emaPeriods := market.ParseEMAPeriodsCSV(body.EMAPeriods)
	snaps, err := h.svc.GetIndicatorsBatch(r.Context(), body.Exchange, body.Interval, body.Symbols, body.RSIPeriod, emaPeriods)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(snaps))
	for _, s := range snaps {
		ema := map[string]float64{}
		for k, v := range s.EMA {
			if v != nil {
				ema[strconv.Itoa(k)] = *v
			}
		}
		item := map[string]any{
			"symbol": s.Symbol,
			"rsi":    s.RSI,
			"ema":    ema,
		}
		if s.Error != "" {
			item["error"] = "unavailable"
		}
		items = append(items, item)
	}
	ex, _ := h.svc.ResolveExchange(body.Exchange)
	writeJSON(w, http.StatusOK, map[string]any{
		"exchange": string(ex),
		"interval": body.Interval,
		"items":    items,
		"note":     "Informational only — not financial advice.",
	})
}
