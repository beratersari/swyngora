package handler

import (
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetCoinSituation handles GET /api/v1/market/situation?symbol=
func (h *MarketHandler) GetCoinSituation(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, domain.ErrUpstream)
		return
	}
	got, err := h.svc.GetCoinSituation(r.Context(), r.URL.Query().Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}
