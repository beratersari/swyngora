package handler

import (
	"net/http"
	"time"
)

// HealthHandler serves liveness probes.
type HealthHandler struct{}

// NewHealthHandler constructs a health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// ServeHTTP returns a simple OK payload.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339Nano),
	})
}
