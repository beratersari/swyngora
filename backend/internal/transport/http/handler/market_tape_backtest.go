package handler

import (
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetTapeBacktest handles GET /api/v1/market/tape-backtest.
func (h *MarketHandler) GetTapeBacktest(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc == nil {
		writeError(w, domain.ErrUpstream)
		return
	}
	q := r.URL.Query()
	var threshold float64
	if raw := q.Get("threshold"); raw != "" {
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			writeError(w, domain.ErrInvalidArgument)
			return
		}
		threshold = n
	}
	fromT, ok := parseOptionalTapeTime(w, q.Get("from"))
	if !ok {
		return
	}
	toT, ok := parseOptionalTapeTime(w, q.Get("to"))
	if !ok {
		return
	}
	got, err := h.svc.GetTapeBacktest(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("condition"), q.Get("side"), threshold, fromT, toT)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}

func parseOptionalTapeTime(w http.ResponseWriter, raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, true
	}
	t, err := parseTimeParam(raw)
	if err != nil {
		writeError(w, err)
		return time.Time{}, false
	}
	return t, true
}
