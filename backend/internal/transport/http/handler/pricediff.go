package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/pricediff"
)

// PriceDiffHandler is the transport adapter for cross-exchange price difference tracking.
type PriceDiffHandler struct {
	svc *pricediff.Service
}

// NewPriceDiffHandler constructs the handler.
func NewPriceDiffHandler(svc *pricediff.Service) *PriceDiffHandler {
	return &PriceDiffHandler{svc: svc}
}

type priceDiffWatchDTO struct {
	ID             string  `json:"id"`
	ClientID       string  `json:"clientId"`
	Symbol         string  `json:"symbol"`
	MinNetDiffPct  float64 `json:"minNetDiffPct"`
	FeeBinancePct  float64 `json:"feeBinancePct"`
	FeeCoinbasePct float64 `json:"feeCoinbasePct"`
	FeeBybitPct    float64 `json:"feeBybitPct"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

type priceDiffOppDTO struct {
	ID            string  `json:"id"`
	WatchID       string  `json:"watchId"`
	ClientID      string  `json:"clientId"`
	Symbol        string  `json:"symbol"`
	BuyExchange   string  `json:"buyExchange"`
	SellExchange  string  `json:"sellExchange"`
	BuyPrice      float64 `json:"buyPrice"`
	SellPrice     float64 `json:"sellPrice"`
	GrossDiffPct  float64 `json:"grossDiffPct"`
	NetDiffPct    float64 `json:"netDiffPct"`
	MinNetDiffPct float64 `json:"minNetDiffPct"`
	Status        string  `json:"status"`
	OpenedAt      string  `json:"openedAt"`
	LastSeenAt    string  `json:"lastSeenAt"`
	ClosedAt      *string `json:"closedAt,omitempty"`
}

func watchDTO(w *domain.PriceDiffWatch) priceDiffWatchDTO {
	return priceDiffWatchDTO{
		ID: w.ID, ClientID: w.ClientID, Symbol: w.Symbol,
		MinNetDiffPct: w.MinNetDiffPct,
		FeeBinancePct: w.FeeBinancePct, FeeCoinbasePct: w.FeeCoinbasePct, FeeBybitPct: w.FeeBybitPct,
		Status: string(w.Status),
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: w.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func oppDTO(o *domain.PriceDiffOpportunity) priceDiffOppDTO {
	d := priceDiffOppDTO{
		ID: o.ID, WatchID: o.WatchID, ClientID: o.ClientID, Symbol: o.Symbol,
		BuyExchange: string(o.BuyExchange), SellExchange: string(o.SellExchange),
		BuyPrice: o.BuyPrice, SellPrice: o.SellPrice,
		GrossDiffPct: o.GrossDiffPct, NetDiffPct: o.NetDiffPct, MinNetDiffPct: o.MinNetDiffPct,
		Status: string(o.Status),
		OpenedAt:   o.OpenedAt.UTC().Format(time.RFC3339Nano),
		LastSeenAt: o.LastSeenAt.UTC().Format(time.RFC3339Nano),
	}
	if o.ClosedAt != nil {
		s := o.ClosedAt.UTC().Format(time.RFC3339Nano)
		d.ClosedAt = &s
	}
	return d
}

type createWatchBody struct {
	ClientID       string  `json:"clientId"`
	Symbol         string  `json:"symbol"`
	MinNetDiffPct  float64 `json:"minNetDiffPct"`
	FeeBinancePct  float64 `json:"feeBinancePct"`
	FeeCoinbasePct float64 `json:"feeCoinbasePct"`
	FeeBybitPct    float64 `json:"feeBybitPct"`
}

// CreateWatch handles POST /api/v1/price-diff/watches
func (h *PriceDiffHandler) CreateWatch(w http.ResponseWriter, r *http.Request) {
	var body createWatchBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	watch, err := h.svc.CreateWatch(r.Context(), pricediff.CreateInput{
		ClientID: clientID, Symbol: body.Symbol, MinNetDiffPct: body.MinNetDiffPct,
		FeeBinancePct: body.FeeBinancePct, FeeCoinbasePct: body.FeeCoinbasePct, FeeBybitPct: body.FeeBybitPct,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, watchDTO(watch))
}

// ListWatches handles GET /api/v1/price-diff/watches
func (h *PriceDiffHandler) ListWatches(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListWatches(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]priceDiffWatchDTO, 0, len(list))
	for i := range list {
		items = append(items, watchDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "watches": items, "count": len(items),
	})
}

// GetWatch handles GET /api/v1/price-diff/watches/{id}
func (h *PriceDiffHandler) GetWatch(w http.ResponseWriter, r *http.Request) {
	watch, err := h.svc.GetWatch(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, watchDTO(watch))
}

// DeleteWatch handles DELETE /api/v1/price-diff/watches/{id}
func (h *PriceDiffHandler) DeleteWatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteWatch(r.Context(), clientIDFrom(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

// ListOpportunities handles GET /api/v1/price-diff/opportunities
func (h *PriceDiffHandler) ListOpportunities(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.ListOpportunities(r.Context(), clientIDFrom(r), q.Get("status"), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]priceDiffOppDTO, 0, len(list))
	for i := range list {
		items = append(items, oppDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "opportunities": items, "count": len(items),
	})
}

// GetOpportunity handles GET /api/v1/price-diff/opportunities/{id}
func (h *PriceDiffHandler) GetOpportunity(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.GetOpportunity(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, oppDTO(o))
}
