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
	Kind           string  `json:"kind"`
	Condition      string  `json:"condition"`
	TargetPrice    float64 `json:"targetPrice"`
	RangePct       float64 `json:"rangePct,omitempty"`
	Mode           string  `json:"mode"`
	Armed          bool    `json:"armed,omitempty"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	TriggeredAt    *string `json:"triggeredAt,omitempty"`
	TriggeredPrice float64 `json:"triggeredPrice,omitempty"`
}

func alertToDTO(a *domain.PriceAlert) alertDTO {
	mode := string(a.Mode)
	if mode == "" {
		mode = string(domain.AlertModeOneTime)
	}
	kind := string(domain.EffectiveAlertKind(*a))
	dto := alertDTO{
		ID:          a.ID,
		ClientID:    a.ClientID,
		Exchange:    string(a.Exchange),
		Symbol:      a.Symbol,
		Kind:        kind,
		Condition:   string(a.Condition),
		TargetPrice: a.TargetPrice,
		Mode:        mode,
		Status:      string(a.Status),
		CreatedAt:   a.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if (domain.IsBookAlert(a.Kind) || domain.IsLiqNotionalAlert(a.Kind)) && a.RangePct > 0 {
		dto.RangePct = a.RangePct
	}
	if a.Mode == domain.AlertModeRepeating {
		dto.Armed = a.Armed
	}
	if a.TriggeredAt != nil && !a.TriggeredAt.IsZero() {
		s := a.TriggeredAt.UTC().Format(time.RFC3339Nano)
		dto.TriggeredAt = &s
	}
	if a.TriggeredPrice > 0 {
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
	Kind        string  `json:"kind"` // price | imbalance | wall | liquidation_feed | liquidation_cascade | liquidation_notional
	Condition   string  `json:"condition"`
	TargetPrice float64 `json:"targetPrice"`
	RangePct    float64 `json:"rangePct"`
	Window      string  `json:"window"` // liquidation_notional: 1m|5m|15m|1h
	Mode        string  `json:"mode"`   // one_time | repeating
}

// Create handles POST /api/v1/alerts
func (h *AlertHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createAlertBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	kind := strings.ToLower(strings.TrimSpace(body.Kind))
	if body.TargetPrice == 0 && kind != "wall" && kind != "liquidation_feed" && kind != "liquidation_cascade" && kind != "liquidation_notional" {
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
		Kind:        body.Kind,
		Condition:   body.Condition,
		TargetPrice: body.TargetPrice,
		RangePct:    body.RangePct,
		Window:      body.Window,
		Mode:        body.Mode,
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

type quietHoursDTO struct {
	Enabled bool   `json:"enabled"`
	Start   string `json:"start,omitempty"`
	End     string `json:"end,omitempty"`
}

type webhookDTO struct {
	ClientID     string        `json:"clientId"`
	URL          string        `json:"url"`
	DeliveryMode string        `json:"deliveryMode"`
	TimeZone     string        `json:"timeZone"`
	QuietHours   quietHoursDTO `json:"quietHours"`
	UpdatedAt    string        `json:"updatedAt,omitempty"`
	Configured   bool          `json:"configured"`
}

func webhookToDTO(w *domain.ClientWebhook) webhookDTO {
	mode := string(w.DeliveryMode)
	if mode == "" {
		mode = string(domain.DeliveryImmediate)
	}
	tz := w.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	dto := webhookDTO{
		ClientID:     w.ClientID,
		URL:          w.URL,
		DeliveryMode: mode,
		TimeZone:     tz,
		QuietHours: quietHoursDTO{
			Enabled: w.QuietHoursEnabled,
			Start:   w.QuietStart,
			End:     w.QuietEnd,
		},
		Configured: strings.TrimSpace(w.URL) != "",
	}
	if !w.UpdatedAt.IsZero() {
		dto.UpdatedAt = w.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return dto
}

// GetWebhook handles GET /api/v1/alerts/webhook
func (h *AlertHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	wh, err := h.svc.GetWebhook(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, webhookToDTO(wh))
}

type setWebhookBody struct {
	ClientID     string `json:"clientId"`
	URL          string `json:"url"`
	DeliveryMode string `json:"deliveryMode"` // immediate | hourly_digest
	TimeZone     string `json:"timeZone"`
	QuietHours   *struct {
		Enabled bool   `json:"enabled"`
		Start   string `json:"start"`
		End     string `json:"end"`
	} `json:"quietHours"`
}

// PutWebhook handles PUT /api/v1/alerts/webhook
func (h *AlertHandler) PutWebhook(w http.ResponseWriter, r *http.Request) {
	var body setWebhookBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	in := domain.WebhookSettings{
		URL:          body.URL,
		DeliveryMode: body.DeliveryMode,
		TimeZone:     body.TimeZone,
	}
	if body.QuietHours != nil {
		in.QuietHoursEnabled = body.QuietHours.Enabled
		in.QuietStart = body.QuietHours.Start
		in.QuietEnd = body.QuietHours.End
	}
	wh, err := h.svc.SetWebhook(r.Context(), clientID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, webhookToDTO(wh))
}

// DeleteWebhook handles DELETE /api/v1/alerts/webhook
func (h *AlertHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	clientID := clientIDFrom(r)
	if err := h.svc.DeleteWebhook(r.Context(), clientID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId":   clientID,
		"deleted":    true,
		"configured": false,
	})
}
