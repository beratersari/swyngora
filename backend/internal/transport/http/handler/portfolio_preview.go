package handler

import (
	"fmt"
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

// PreviewOrder handles POST /api/v1/portfolio/orders/preview.
// It prices the order and checks risk limits. It does not place the order.
func (h *PortfolioHandler) PreviewOrder(w http.ResponseWriter, r *http.Request) {
	var body orderBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	pfID := coalescePortfolioID(r, body.PortfolioID)
	ownerID := ownerClientIDFrom(r)
	if body.OwnerClientID != "" {
		ownerID = body.OwnerClientID
	}
	got, err := h.svc.PreviewOrder(r.Context(), portfolio.OrderPreviewInput{
		ClientID: clientID, PortfolioID: pfID, OwnerClientID: ownerID,
		Product: body.Product, Exchange: body.Exchange, Symbol: body.Symbol,
		Side: body.Side, Type: body.Type, Quantity: body.Quantity,
		TriggerPrice: body.TriggerPrice, TakeProfitPrice: body.TakeProfitPrice, StopLossPrice: body.StopLossPrice,
		TrailType: body.TrailType, TrailValue: body.TrailValue, LotMethod: body.LotMethod,
		Leverage: body.Leverage, LimitPrice: body.LimitPrice, RiskPercent: body.RiskPercent,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolio.OrderPreviewPayload(got))
}
