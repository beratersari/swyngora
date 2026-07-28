package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricealert"
)

// AlertHandler is the transport adapter for price alerts.
type AlertHandler struct {
	svc *pricealert.Service
}

// NewAlertHandler constructs the handler.
func NewAlertHandler(svc *pricealert.Service) *AlertHandler {
	return &AlertHandler{svc: svc}
}

type alertDTO struct {
	ID             string  `json:"id"`
	ClientID       string  `json:"clientId"`
	Exchange       string  `json:"exchange"`
	Symbol         string  `json:"symbol"`
	Condition      string  `json:"condition"`
	TargetPrice    float64 `json:"targetPrice"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	TriggeredAt    *string `json:"triggeredAt,omitempty"`
	TriggeredPrice float64 `json:"triggeredPrice,omitempty"`
}

func alertToDTO(a *domain.PriceAlert) alertDTO {
	dto := alertDTO{
		ID:          a.ID,
		ClientID:    a.ClientID,
		Exchange:    string(a.Exchange),
		Symbol:      a.Symbol,
		Condition:   string(a.Condition),
		TargetPrice: a.TargetPrice,
		Status:      string(a.Status),
		CreatedAt:   a.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if a.TriggeredAt != nil && !a.TriggeredAt.IsZero() {
		s := a.TriggeredAt.UTC().Format(time.RFC3339Nano)
		dto.TriggeredAt = &s
	}
	if a.Status == domain.AlertStatusTriggered {
		dto.TriggeredPrice = a.TriggeredPrice
	}
	return dto
}

// List handles GET /api/v1/alerts
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]alertDTO, 0, len(list))
	for i := range list {
		items = append(items, alertToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r),
		"alerts":   items,
		"count":    len(items),
	})
}

// Get handles GET /api/v1/alerts/{id}
func (h *AlertHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	a, err := h.svc.Get(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, alertToDTO(a))
}

type createAlertBody struct {
	ClientID    string  `json:"clientId"`
	Exchange    string  `json:"exchange"`
	Symbol      string  `json:"symbol"`
	Condition   string  `json:"condition"`
	TargetPrice float64 `json:"targetPrice"`
	// TargetPrice may also arrive as string from some clients — handled via raw if needed.
}

// Create handles POST /api/v1/alerts
func (h *AlertHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createAlertBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	if body.TargetPrice == 0 {
		// Try query fallback only if body zero and query set (optional convenience).
		if raw := r.URL.Query().Get("targetPrice"); raw != "" {
			if f, err := strconv.ParseFloat(raw, 64); err == nil {
				body.TargetPrice = f
			}
		}
	}
	a, err := h.svc.Create(r.Context(), pricealert.CreateInput{
		ClientID:    clientID,
		Exchange:    body.Exchange,
		Symbol:      body.Symbol,
		Condition:   body.Condition,
		TargetPrice: body.TargetPrice,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, alertToDTO(a))
}

// Delete handles DELETE /api/v1/alerts/{id}
func (h *AlertHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.svc.Delete(r.Context(), clientIDFrom(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}