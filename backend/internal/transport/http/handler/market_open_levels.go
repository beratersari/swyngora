package handler

import (
	"net/http"
)

// GetLiquidationOpenLevels handles GET /api/v1/market/liquidation-open-levels.
func (h *MarketHandler) GetLiquidationOpenLevels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetLiquidationOpenLevels(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("range"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}
