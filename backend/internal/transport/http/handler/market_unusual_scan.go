package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ScanUnusual handles GET /api/v1/market/unusual/scan.
func (h *MarketHandler) ScanUnusual(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	got, err := h.svc.ScanUnusual(r.Context(), q.Get("exchange"), q.Get("metric"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}
