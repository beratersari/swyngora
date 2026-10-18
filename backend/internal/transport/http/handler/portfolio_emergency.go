package handler

import (
	"fmt"
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type emergencyStopBody struct {
	Enabled bool    `json:"enabled"`
	LossPct float64 `json:"lossPct"`
}

// GetEmergencyStop handles GET /api/v1/portfolio/emergency-stop.
func (h *PortfolioHandler) GetEmergencyStop(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.GetEmergencyStop(r.Context(), clientIDFrom(r), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolio.EmergencyStopPayload(v))
}

// PutEmergencyStop handles PUT /api/v1/portfolio/emergency-stop.
func (h *PortfolioHandler) PutEmergencyStop(w http.ResponseWriter, r *http.Request) {
	var body emergencyStopBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	v, err := h.svc.SetEmergencyStop(r.Context(), portfolio.EmergencyStopInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r),
		Enabled: body.Enabled, LossPct: body.LossPct,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolio.EmergencyStopPayload(v))
}
