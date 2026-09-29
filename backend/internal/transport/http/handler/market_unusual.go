package handler

import "net/http"

// GetUnusual handles GET /api/v1/market/unusual.
func (h *MarketHandler) GetUnusual(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	got, err := h.svc.GetUnusual(r.Context(), q.Get("exchange"), q.Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}
