package handler

import (
	"fmt"
	"net/http"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type allocationTargetDTO struct {
	Asset     string  `json:"asset"`
	Exchange  string  `json:"exchange,omitempty"`
	WeightPct float64 `json:"weightPct"`
}

type allocationBasketDTO struct {
	ID        string                 `json:"id"`
	ClientID  string                 `json:"clientId"`
	Name      string                 `json:"name"`
	Targets   []allocationTargetDTO  `json:"targets"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
}

type allocationLineDTO struct {
	Asset        string  `json:"asset"`
	Exchange     string  `json:"exchange,omitempty"`
	Symbol       string  `json:"symbol,omitempty"`
	IsCash       bool    `json:"isCash"`
	TargetPct    float64 `json:"targetPct"`
	ActualPct    float64 `json:"actualPct"`
	CurrentValue float64 `json:"currentValue"`
	TargetValue  float64 `json:"targetValue"`
	DeltaValue   float64 `json:"deltaValue"`
	MarkPrice    float64 `json:"markPrice,omitempty"`
}

type rebalanceLegDTO struct {
	Side     string  `json:"side"`
	Asset    string  `json:"asset"`
	Exchange string  `json:"exchange"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Notional float64 `json:"notional"`
	Reason   string  `json:"reason"`
}

func allocationBasketToDTO(b *domain.AllocationBasket) allocationBasketDTO {
	tg := make([]allocationTargetDTO, 0, len(b.Targets))
	for _, t := range b.Targets {
		tg = append(tg, allocationTargetDTO{Asset: t.Asset, Exchange: string(t.Exchange), WeightPct: t.WeightPct})
	}
	return allocationBasketDTO{
		ID: b.ID, ClientID: b.ClientID, Name: b.Name, Targets: tg,
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func allocationViewDTO(v *portfolio.AllocationBasketView, trades []domain.Trade) map[string]any {
	lines := make([]allocationLineDTO, 0, len(v.Plan.Lines))
	for _, ln := range v.Plan.Lines {
		lines = append(lines, allocationLineDTO{
			Asset: ln.Asset, Exchange: string(ln.Exchange), Symbol: ln.Symbol, IsCash: ln.IsCash,
			TargetPct: ln.TargetPct, ActualPct: ln.ActualPct,
			CurrentValue: ln.CurrentValue, TargetValue: ln.TargetValue, DeltaValue: ln.DeltaValue,
			MarkPrice: ln.MarkPrice,
		})
	}
	legs := make([]rebalanceLegDTO, 0, len(v.Plan.Legs))
	for _, l := range v.Plan.Legs {
		legs = append(legs, rebalanceLegDTO{
			Side: string(l.Side), Asset: l.Asset, Exchange: string(l.Exchange), Symbol: l.Symbol,
			Quantity: l.Quantity, Price: l.Price, Notional: l.Notional, Reason: l.Reason,
		})
	}
	out := map[string]any{
		"basket":        allocationBasketToDTO(&v.Basket),
		"currency":      v.Plan.Currency,
		"equity":        v.Plan.Equity,
		"cash":          v.Plan.Cash,
		"availableCash": v.Plan.AvailableCash,
		"allocations":   lines,
		"legs":          legs,
		"note":          v.Note,
	}
	if trades != nil {
		items := make([]tradeDTO, 0, len(trades))
		for i := range trades {
			items = append(items, tradeToDTO(&trades[i]))
		}
		out["trades"] = items
		out["tradeCount"] = len(items)
	}
	return out
}

func parseTargetBodies(in []allocationTargetDTO) []domain.AllocationTarget {
	out := make([]domain.AllocationTarget, 0, len(in))
	for _, t := range in {
		out = append(out, domain.AllocationTarget{
			Asset: t.Asset, Exchange: domain.Exchange(t.Exchange), WeightPct: t.WeightPct,
		})
	}
	return out
}

type createBasketBody struct {
	ClientID string                 `json:"clientId"`
	Name     string                 `json:"name"`
	Targets  []allocationTargetDTO  `json:"targets"`
}

type updateBasketBody struct {
	Name    *string                `json:"name"`
	Targets *[]allocationTargetDTO `json:"targets"`
}

// CreateBasket handles POST /api/v1/portfolio/baskets
func (h *PortfolioHandler) CreateBasket(w http.ResponseWriter, r *http.Request) {
	var body createBasketBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	b, err := h.svc.CreateAllocationBasket(r.Context(), portfolio.AllocationBasketCreateInput{
		ClientID: clientID, Name: body.Name, Targets: parseTargetBodies(body.Targets),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, allocationBasketToDTO(b))
}

// ListBaskets handles GET /api/v1/portfolio/baskets
func (h *PortfolioHandler) ListBaskets(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAllocationBaskets(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]allocationBasketDTO, 0, len(list))
	for i := range list {
		items = append(items, allocationBasketToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "baskets": items, "count": len(items),
	})
}

// GetBasket handles GET /api/v1/portfolio/baskets/{id} — includes live drift preview.
func (h *PortfolioHandler) GetBasket(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.PreviewAllocationRebalance(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, allocationViewDTO(v, nil))
}

// UpdateBasket handles PATCH /api/v1/portfolio/baskets/{id}
func (h *PortfolioHandler) UpdateBasket(w http.ResponseWriter, r *http.Request) {
	var body updateBasketBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	var targets []domain.AllocationTarget
	if body.Targets != nil {
		targets = parseTargetBodies(*body.Targets)
	}
	in := portfolio.AllocationBasketUpdateInput{
		ClientID: clientIDFrom(r), BasketID: r.PathValue("id"), Name: body.Name,
	}
	if body.Targets != nil {
		in.Targets = targets
	}
	b, err := h.svc.UpdateAllocationBasket(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, allocationBasketToDTO(b))
}

// DeleteBasket handles DELETE /api/v1/portfolio/baskets/{id}
func (h *PortfolioHandler) DeleteBasket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteAllocationBasket(r.Context(), clientIDFrom(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

// PreviewBasketRebalance handles GET /api/v1/portfolio/baskets/{id}/preview
func (h *PortfolioHandler) PreviewBasketRebalance(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.PreviewAllocationRebalance(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, allocationViewDTO(v, nil))
}

// RebalanceBasket handles POST /api/v1/portfolio/baskets/{id}/rebalance
func (h *PortfolioHandler) RebalanceBasket(w http.ResponseWriter, r *http.Request) {
	v, trades, err := h.svc.ExecuteAllocationRebalance(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, allocationViewDTO(v, trades))
}
