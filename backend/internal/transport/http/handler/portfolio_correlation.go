package handler

import (
	"net/http"
	"strconv"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

// GetPortfolioCorrelation handles GET /api/v1/portfolio/correlation.
// It reads how the open coins move together. Optional add* query fields
// price one extra line without placing it.
func (h *PortfolioHandler) GetPortfolioCorrelation(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	add, err := corrAddFromQuery(q.Get("addExchange"), q.Get("addSymbol"), q.Get("addSide"), q.Get("addNotional"))
	if err != nil {
		writeError(w, err)
		return
	}
	rep, err := h.svc.BookCorrelation(r.Context(), clientIDFrom(r), portfolioIDFrom(r), ownerClientIDFrom(r), add)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func corrAddFromQuery(exchange, symbol, side, notional string) (portfolio.PortfolioCorrRequest, error) {
	notional = strings.TrimSpace(notional)
	symbol = strings.TrimSpace(symbol)
	if notional == "" && symbol == "" && strings.TrimSpace(side) == "" {
		return portfolio.PortfolioCorrRequest{}, nil
	}
	n, err := strconv.ParseFloat(notional, 64)
	if err != nil {
		return portfolio.PortfolioCorrRequest{}, domain.ErrInvalidArgument
	}
	ex := domain.Exchange(strings.ToLower(strings.TrimSpace(exchange)))
	return portfolio.PortfolioCorrRequest{
		Exchange: ex, Symbol: symbol, Side: side, Notional: n,
	}, nil
}
