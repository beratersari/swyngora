package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

// PortfolioHandler is the HTTP adapter for paper trading.
type PortfolioHandler struct {
	svc *portfolio.Service
}

// NewPortfolioHandler constructs the handler.
func NewPortfolioHandler(svc *portfolio.Service) *PortfolioHandler {
	return &PortfolioHandler{svc: svc}
}

type portfolioDTO struct {
	ClientID         string         `json:"clientId"`
	Currency         string         `json:"currency"`
	StartingBalance  float64        `json:"startingBalance"`
	CashBalance      float64        `json:"cashBalance"`
	PositionsValue   float64        `json:"positionsValue"`
	Equity           float64        `json:"equity"`
	UnrealizedPnL    float64        `json:"unrealizedPnL"`
	RealizedPnLTotal float64        `json:"realizedPnLTotal"`
	TotalPnL         float64        `json:"totalPnL"`
	Positions        []positionDTO  `json:"positions"`
	Note             string         `json:"note"`
	CreatedAt        string         `json:"createdAt"`
	UpdatedAt        string         `json:"updatedAt"`
}

type positionDTO struct {
	Exchange      string  `json:"exchange"`
	Symbol        string  `json:"symbol"`
	Quantity      float64 `json:"quantity"`
	AvgCost       float64 `json:"avgCost"`
	MarkPrice     float64 `json:"markPrice"`
	MarketValue   float64 `json:"marketValue"`
	UnrealizedPnL float64 `json:"unrealizedPnL"`
	CostBasis     float64 `json:"costBasis"`
}

type tradeDTO struct {
	ID          string  `json:"id"`
	Exchange    string  `json:"exchange"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	Notional    float64 `json:"notional"`
	RealizedPnL float64 `json:"realizedPnL"`
	CreatedAt   string  `json:"createdAt"`
}

func portfolioViewDTO(v *domain.PortfolioView) portfolioDTO {
	pos := make([]positionDTO, 0, len(v.Positions))
	for _, p := range v.Positions {
		pos = append(pos, positionDTO{
			Exchange: string(p.Exchange), Symbol: p.Symbol, Quantity: p.Quantity,
			AvgCost: p.AvgCost, MarkPrice: p.MarkPrice, MarketValue: p.MarketValue,
			UnrealizedPnL: p.UnrealizedPnL, CostBasis: p.CostBasis,
		})
	}
	return portfolioDTO{
		ClientID: v.ClientID, Currency: v.Currency, StartingBalance: v.StartingBalance,
		CashBalance: v.CashBalance, PositionsValue: v.PositionsValue, Equity: v.Equity,
		UnrealizedPnL: v.UnrealizedPnL, RealizedPnLTotal: v.RealizedPnLTotal, TotalPnL: v.TotalPnL,
		Positions: pos, Note: v.Note,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func tradeToDTO(t *domain.Trade) tradeDTO {
	return tradeDTO{
		ID: t.ID, Exchange: string(t.Exchange), Symbol: t.Symbol, Side: string(t.Side),
		Quantity: t.Quantity, Price: t.Price, Notional: t.Notional, RealizedPnL: t.RealizedPnL,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type createPortfolioBody struct {
	ClientID        string  `json:"clientId"`
	StartingBalance float64 `json:"startingBalance"`
	Currency        string  `json:"currency"`
}

// Create handles POST /api/v1/portfolio
func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createPortfolioBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	p, err := h.svc.Create(r.Context(), portfolio.CreateInput{
		ClientID: clientID, StartingBalance: body.StartingBalance, Currency: body.Currency,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	view, err := h.svc.View(r.Context(), p.ClientID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, portfolioViewDTO(view))
}

// Get handles GET /api/v1/portfolio
func (h *PortfolioHandler) Get(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.View(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolioViewDTO(view))
}

type orderBody struct {
	ClientID string  `json:"clientId"`
	Exchange string  `json:"exchange"`
	Symbol   string  `json:"symbol"`
	Side     string  `json:"side"`
	Quantity float64 `json:"quantity"`
}

// PlaceOrder handles POST /api/v1/portfolio/orders
func (h *PortfolioHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var body orderBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	tr, view, err := h.svc.PlaceOrder(r.Context(), portfolio.OrderInput{
		ClientID: clientID, Exchange: body.Exchange, Symbol: body.Symbol,
		Side: body.Side, Quantity: body.Quantity,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"trade":     tradeToDTO(tr),
		"portfolio": portfolioViewDTO(view),
		"note":      view.Note,
	})
}

// ListTrades handles GET /api/v1/portfolio/trades
func (h *PortfolioHandler) ListTrades(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.svc.ListTrades(r.Context(), clientIDFrom(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]tradeDTO, 0, len(list))
	for i := range list {
		items = append(items, tradeToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r),
		"trades":   items,
		"count":    len(items),
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}