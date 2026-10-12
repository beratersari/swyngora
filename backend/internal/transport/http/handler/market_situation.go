package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCoinSituation handles GET /api/v1/market/situation?symbol=
// Optional at (RFC3339) reads the stored minute at a candle close.
// An empty at is the live read. A past read leaves the estimated pocket off.
func (h *MarketHandler) GetCoinSituation(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUpstream)
		return
	}
	symbol := r.URL.Query().Get("symbol")
	rawAt := strings.TrimSpace(r.URL.Query().Get("at"))
	if rawAt == "" {
		got, err := h.svc.GetCoinSituation(r.Context(), symbol)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, got)
		return
	}
	at, err := parseSituationAt(rawAt)
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.GetCoinSituationAt(r.Context(), symbol, at)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}

func parseSituationAt(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("%w: at must be RFC3339", domain.ErrInvalidArgument)
}
