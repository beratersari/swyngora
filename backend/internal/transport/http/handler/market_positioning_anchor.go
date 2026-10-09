package handler

import (
	"net/http"
	"time"
)

// GetPositioningAnchor handles GET /api/v1/market/positioning-anchor.
func (h *MarketHandler) GetPositioningAnchor(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var fromPtr, toPtr *time.Time
	if raw := q.Get("from"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		fromPtr = &t
	}
	if raw := q.Get("to"); raw != "" {
		t, err := parseTimeParam(raw)
		if err != nil {
			writeError(w, err)
			return
		}
		toPtr = &t
	}
	got, err := h.svc.GetPositioningAnchor(r.Context(), q.Get("exchange"), q.Get("symbol"), q.Get("anchor"), q.Get("bucket"), fromPtr, toPtr)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, got)
}
