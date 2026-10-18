package handler

import (
	"fmt"
	"net/http"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

// CompareExecution handles POST /api/v1/portfolio/orders/execution-compare.
// It prices one venue, best cost, and a time split. It does not place an order.
func (h *PortfolioHandler) CompareExecution(w http.ResponseWriter, r *http.Request) {
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
	got, err := h.svc.CompareExecutions(r.Context(), portfolio.ExecutionCompareInput{
		ClientID: clientID, PortfolioID: pfID, OwnerClientID: ownerID,
		Exchange: body.Exchange, Symbol: body.Symbol, Side: body.Side, Quantity: body.Quantity,
		MaxSlippagePct: body.MaxSlippagePct, SpreadMinutes: body.SpreadMinutes, SliceCount: body.SliceCount,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolio.ExecutionComparePayload(got))
}
