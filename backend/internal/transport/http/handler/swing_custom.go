package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/swing"
)

type customSignalProfileDTO struct {
	ID        string             `json:"id"`
	ClientID  string             `json:"clientId"`
	Name      string             `json:"name"`
	Weights   map[string]float64 `json:"weights"`
	Source    string             `json:"source"`
	IsDefault bool               `json:"isDefault"`
	CreatedAt string             `json:"createdAt"`
	UpdatedAt string             `json:"updatedAt"`
}

type customSignalItemDTO struct {
	Exchange   string                     `json:"exchange"`
	Symbol     string                     `json:"symbol"`
	Interval   string                     `json:"interval"`
	Price      float64                    `json:"price"`
	BarTime    string                     `json:"barTime,omitempty"`
	Engine     *swingDecisionDTO          `json:"engine,omitempty"`
	Custom     domain.CustomSignalScore   `json:"custom"`
	Compare    domain.CustomSignalCompare `json:"scoreCompare"`
	HuntDetail string                     `json:"huntDetail,omitempty"`
}

func customProfileToDTO(p *domain.CustomSignalProfile) customSignalProfileDTO {
	if p == nil {
		return customSignalProfileDTO{}
	}
	pct := p.Weights.Pct
	if pct == nil {
		pct = map[string]float64{}
	}
	src := p.Weights.Source
	if src == "" {
		src = domain.CustomSignalMixCustom
	}
	return customSignalProfileDTO{
		ID: p.ID, ClientID: p.ClientID, Name: p.Name, Weights: pct, Source: src,
		IsDefault: p.IsDefault,
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func customItemToDTO(it domain.CustomSignalItem) customSignalItemDTO {
	out := customSignalItemDTO{
		Exchange: string(it.Exchange), Symbol: it.Symbol, Interval: it.Interval,
		Price: it.Price, Custom: it.Custom, Compare: it.Compare, HuntDetail: it.HuntDetail,
	}
	if !it.BarTime.IsZero() {
		out.BarTime = it.BarTime.UTC().Format(time.RFC3339Nano)
	}
	if it.Engine != nil {
		d := swingToDTO(it.Engine)
		out.Engine = &d
	}
	return out
}

// ListCustomFactors handles GET /api/v1/swing/custom/factors
func (h *SwingHandler) ListCustomFactors(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"factors": h.svc.ListFactors(),
		"note":    "Built-in percents match the swing engine. Hunt is off until you give it weight. Mix must sum to 100.",
	})
}

// ListCustomProfiles handles GET /api/v1/swing/custom/profiles
func (h *SwingHandler) ListCustomProfiles(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	clientID, ok := mustResolveClientID(w, r, "")
	if !ok {
		return
	}
	list, err := h.svc.ListProfiles(r.Context(), clientID)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]customSignalProfileDTO, 0, len(list))
	for i := range list {
		items = append(items, customProfileToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": items, "count": len(items)})
}

type customProfileBody struct {
	ClientID  string             `json:"clientId"`
	Name      string             `json:"name"`
	Weights   map[string]float64 `json:"weights"`
	IsDefault bool               `json:"isDefault"`
}

// CreateCustomProfile handles POST /api/v1/swing/custom/profiles
func (h *SwingHandler) CreateCustomProfile(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	var body customProfileBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid json", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	p, err := h.svc.CreateProfile(r.Context(), swing.CreateProfileInput{
		ClientID: clientID, Name: body.Name, Weights: body.Weights, IsDefault: body.IsDefault,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, customProfileToDTO(p))
}

// GetCustomProfile handles GET /api/v1/swing/custom/profiles/{id}
func (h *SwingHandler) GetCustomProfile(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	clientID, ok := mustResolveClientID(w, r, "")
	if !ok {
		return
	}
	p, err := h.svc.GetProfile(r.Context(), clientID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, customProfileToDTO(p))
}

type customProfilePatchBody struct {
	ClientID  string             `json:"clientId"`
	Name      *string            `json:"name"`
	Weights   map[string]float64 `json:"weights"`
	IsDefault *bool              `json:"isDefault"`
}

// UpdateCustomProfile handles PATCH /api/v1/swing/custom/profiles/{id}
func (h *SwingHandler) UpdateCustomProfile(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	var body customProfilePatchBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid json", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	p, err := h.svc.UpdateProfile(r.Context(), swing.UpdateProfileInput{
		ClientID: clientID, ID: r.PathValue("id"),
		Name: body.Name, Weights: body.Weights, HasWeights: body.Weights != nil,
		IsDefault: body.IsDefault,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, customProfileToDTO(p))
}

// DuplicateCustomProfile handles POST /api/v1/swing/custom/profiles/{id}/duplicate
func (h *SwingHandler) DuplicateCustomProfile(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	var body customProfileBody
	_ = decodeJSON(r, &body, DefaultMaxJSONBody)
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	p, err := h.svc.DuplicateProfile(r.Context(), clientID, r.PathValue("id"), body.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, customProfileToDTO(p))
}

// SetDefaultCustomProfile handles POST /api/v1/swing/custom/profiles/{id}/default
func (h *SwingHandler) SetDefaultCustomProfile(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	clientID, ok := mustResolveClientID(w, r, "")
	if !ok {
		return
	}
	p, err := h.svc.SetDefaultProfile(r.Context(), clientID, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, customProfileToDTO(p))
}

// DeleteCustomProfile handles DELETE /api/v1/swing/custom/profiles/{id}
func (h *SwingHandler) DeleteCustomProfile(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	clientID, ok := mustResolveClientID(w, r, "")
	if !ok {
		return
	}
	if err := h.svc.DeleteProfile(r.Context(), clientID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ScanCustom handles GET /api/v1/swing/custom/scan
func (h *SwingHandler) ScanCustom(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		writeError(w, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream))
		return
	}
	clientID, ok := mustResolveClientID(w, r, "")
	if !ok {
		return
	}
	q := r.URL.Query()
	inline, err := domain.ParseCustomSignalWeights(q.Get)
	if err != nil {
		writeError(w, err)
		return
	}
	hasInline := false
	for _, id := range domain.CustomSignalFactorIDs {
		if q.Get(domain.CustomSignalQueryKeys[id]) != "" {
			hasInline = true
			break
		}
	}
	wts, profileID, err := h.svc.ResolveCustomWeights(r.Context(), clientID, q.Get("profileId"), inline, hasInline)
	if err != nil {
		writeError(w, err)
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	res, err := h.svc.ScanCustom(r.Context(), swing.CustomScanInput{
		ClientID: clientID, Exchange: q.Get("exchange"), Limit: limit,
		ProfileID: profileID, Weights: wts,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]customSignalItemDTO, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, customItemToDTO(it))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profileId": res.ProfileID,
		"source":    res.Weights.Source,
		"weights":   res.Weights.Pct,
		"items":     items,
		"count":     res.Count,
		"note":      res.Note,
	})
}
