package handler

import "net/http"

// GetHistorySpan handles GET /api/v1/market/history-span.
func (h *MarketHandler) GetHistorySpan(w http.ResponseWriter, r *http.Request) {
	got, err := h.svc.HistorySpan(r.Context(), r.URL.Query().Get("symbol"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}
