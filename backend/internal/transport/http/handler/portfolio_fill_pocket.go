package handler

import (
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetFillPocket handles GET /api/v1/portfolio/fill-pocket?exchange=&symbol=
func (h *PortfolioHandler) GetFillPocket(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc == nil {
		writeError(w, domain.ErrUpstream)
		return
	}
	q := r.URL.Query()
	got, err := h.svc.FillPocket(r.Context(), clientIDFrom(r), q.Get("exchange"), q.Get("symbol"), portfolioIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}
