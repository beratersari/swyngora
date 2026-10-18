package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type spreadHoldDTO struct {
	Exchange string  `json:"exchange"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
}

type spreadOrderDTO struct {
	ID                string          `json:"id"`
	ClientID          string          `json:"clientId"`
	Symbol            string          `json:"symbol"`
	Side              string          `json:"side"`
	Quantity          float64         `json:"quantity"`
	FilledQuantity    float64         `json:"filledQuantity"`
	RemainingQuantity float64         `json:"remainingQuantity"`
	SliceCount        int             `json:"sliceCount"`
	SlicesDone        int             `json:"slicesDone"`
	DurationMinutes   int             `json:"durationMinutes"`
	NextSliceAt       string          `json:"nextSliceAt,omitempty"`
	EndsAt            string          `json:"endsAt,omitempty"`
	Status            string          `json:"status"`
	MaxSlippagePct    float64         `json:"maxSlippagePct,omitempty"`
	ReservedCash      float64         `json:"reservedCash"`
	FilledNotional    float64         `json:"filledNotional"`
	FilledFee         float64         `json:"filledFee"`
	LotMethod         string          `json:"lotMethod,omitempty"`
	ReasonSet         bool            `json:"reasonSet,omitempty"`
	Reason            []string        `json:"reason,omitempty"`
	CancelReason      string          `json:"cancelReason,omitempty"`
	Holds             []spreadHoldDTO `json:"holds"`
	CreatedAt         string          `json:"createdAt"`
	UpdatedAt         string          `json:"updatedAt"`
}

func spreadOrderToDTO(o *domain.SpreadOrder) spreadOrderDTO {
	if o == nil {
		return spreadOrderDTO{}
	}
	holds := make([]spreadHoldDTO, 0, len(o.Holds))
	for _, h := range o.Holds {
		holds = append(holds, spreadHoldDTO{Exchange: string(h.Exchange), Symbol: h.Symbol, Quantity: h.Quantity})
	}
	d := spreadOrderDTO{
		ID: o.ID, ClientID: o.ClientID, Symbol: o.Symbol, Side: string(o.Side),
		Quantity: o.Quantity, FilledQuantity: o.FilledQuantity, RemainingQuantity: o.RemainingQuantity,
		SliceCount: o.SliceCount, SlicesDone: o.SlicesDone, DurationMinutes: o.DurationMinutes(), Status: string(o.Status),
		MaxSlippagePct: o.MaxSlippage * 100, ReservedCash: o.ReservedCash,
		FilledNotional: o.FilledNotional, FilledFee: o.FilledFee, LotMethod: string(o.LotMethod),
		ReasonSet: o.ReasonSet, CancelReason: o.CancelReason, Holds: holds,
		CreatedAt: o.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: o.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if o.ReasonSet {
		d.Reason = append([]string{}, o.Reason...)
		if d.Reason == nil {
			d.Reason = []string{}
		}
	}
	if !o.NextSliceAt.IsZero() {
		d.NextSliceAt = o.NextSliceAt.UTC().Format(time.RFC3339Nano)
	}
	if !o.EndsAt.IsZero() {
		d.EndsAt = o.EndsAt.UTC().Format(time.RFC3339Nano)
	}
	return d
}

func (h *PortfolioHandler) placeSpread(w http.ResponseWriter, r *http.Request, clientID, pfID, ownerID string, body orderBody, reason domain.SituationReason) {
	order, view, err := h.svc.PlaceSpreadOrder(r.Context(), portfolio.OrderInput{
		ClientID: clientID, PortfolioID: pfID, OwnerClientID: ownerID, Exchange: body.Exchange, Symbol: body.Symbol,
		Side: body.Side, Quantity: body.Quantity, LotMethod: body.LotMethod,
		IdempotencyKey: idempotencyKeyFrom(r, body.IdempotencyKey),
		Reason:         reason,
		MaxSlippagePct: body.MaxSlippagePct,
		SpreadMinutes:  body.SpreadMinutes,
		SliceCount:     body.SliceCount,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	var book any
	if view != nil {
		book = portfolioViewDTO(view)
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"type":      "spread",
		"spread":    true,
		"order":     spreadOrderToDTO(order),
		"portfolio": book,
		"note":      domain.ExplainSpreadPlace(order.SliceCount, order.DurationMinutes()),
	})
}

// ListSpreadOrders handles GET /api/v1/portfolio/spread-orders.
func (h *PortfolioHandler) ListSpreadOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	status := q.Get("status")
	list, err := h.svc.ListSpreadOrders(r.Context(), clientIDFrom(r), status, limit, offset, portfolioIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]spreadOrderDTO, 0, len(list))
	for i := range list {
		items = append(items, spreadOrderToDTO(&list[i]))
	}
	shown := status
	if strings.TrimSpace(shown) == "" {
		shown = "open"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r),
		"orders":   items,
		"count":    len(items),
		"status":   shown,
	})
}

// CancelSpreadOrder handles DELETE /api/v1/portfolio/spread-orders/{id}.
func (h *PortfolioHandler) CancelSpreadOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	o, err := h.svc.CancelSpreadOrder(r.Context(), clientIDFrom(r), id, portfolioIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"order": spreadOrderToDTO(o),
		"note":  "Spread order canceled. Filled slices stay. The unused lock was released.",
	})
}

func rejectSpreadMisuse(body orderBody, typ string) error {
	if !body.Spread {
		return nil
	}
	if typ != "market" || !domain.IsPaperRouteExchange(body.Exchange) {
		return fmt.Errorf("%w: a spread order is a best-cost market order", domain.ErrInvalidArgument)
	}
	return nil
}
