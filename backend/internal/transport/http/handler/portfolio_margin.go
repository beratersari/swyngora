package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type marginPositionDTO struct {
	ID               string   `json:"id"`
	ClientID         string   `json:"clientId"`
	Exchange         string   `json:"exchange"`
	Symbol           string   `json:"symbol"`
	Side             string   `json:"side"`
	Mode             string   `json:"mode"`
	Quantity         float64  `json:"quantity"`
	EntryPrice       float64  `json:"entryPrice"`
	Leverage         int      `json:"leverage"`
	Margin           float64  `json:"margin"`
	DebtPrincipal    float64  `json:"debtPrincipal"`
	DebtInterest     float64  `json:"debtInterest"`
	DebtAsset        string   `json:"debtAsset"`
	DebtNotional     float64  `json:"debtNotional,omitempty"`
	LastInterestAt   string   `json:"lastInterestAt,omitempty"`
	LiquidationPrice float64  `json:"liquidationPrice"`
	StopLoss         *float64 `json:"stopLoss,omitempty"`
	TakeProfit       *float64 `json:"takeProfit,omitempty"`
	Status           string   `json:"status"`
	MarkPrice        float64  `json:"markPrice,omitempty"`
	UnrealizedPnL    float64  `json:"unrealizedPnL,omitempty"`
	RealizedPnL      float64  `json:"realizedPnL"`
	CloseReason      string   `json:"closeReason,omitempty"`
	OpenedAt         string   `json:"openedAt"`
	UpdatedAt        string   `json:"updatedAt"`
	ClosedAt         *string  `json:"closedAt,omitempty"`
}

type marginOrderDTO struct {
	ID             string   `json:"id"`
	ClientID       string   `json:"clientId"`
	Exchange       string   `json:"exchange"`
	Symbol         string   `json:"symbol"`
	Side           string   `json:"side"`
	Type           string   `json:"type"`
	Quantity       float64  `json:"quantity"`
	Leverage       int      `json:"leverage"`
	LimitPrice     float64  `json:"limitPrice,omitempty"`
	ReservedMargin float64  `json:"reservedMargin"`
	StopLoss       *float64 `json:"stopLoss,omitempty"`
	TakeProfit     *float64 `json:"takeProfit,omitempty"`
	Status         string   `json:"status"`
	PositionID     string   `json:"positionId,omitempty"`
	RejectReason   string   `json:"rejectReason,omitempty"`
	CancelReason   string   `json:"cancelReason,omitempty"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	FilledAt       *string  `json:"filledAt,omitempty"`
	CanceledAt     *string  `json:"canceledAt,omitempty"`
}

type marginTradeDTO struct {
	ID            string  `json:"id"`
	PositionID    string  `json:"positionId"`
	Exchange      string  `json:"exchange"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Action        string  `json:"action"`
	Quantity      float64 `json:"quantity"`
	Price         float64 `json:"price"`
	Notional      float64 `json:"notional"`
	RealizedPnL   float64 `json:"realizedPnL"`
	MarginDelta   float64 `json:"marginDelta"`
	PrincipalPaid float64 `json:"principalPaid,omitempty"`
	InterestPaid  float64 `json:"interestPaid,omitempty"`
	Leverage      int     `json:"leverage"`
	CreatedAt     string  `json:"createdAt"`
}

func marginPosDTO(p *domain.MarginPosition) marginPositionDTO {
	mode := string(p.Mode)
	if mode == "" {
		mode = string(domain.MarginModeIsolated)
	}
	d := marginPositionDTO{
		ID: p.ID, ClientID: p.ClientID, Exchange: string(p.Exchange), Symbol: p.Symbol,
		Side: string(p.Side), Mode: mode, Quantity: p.Quantity, EntryPrice: p.EntryPrice, Leverage: p.Leverage,
		Margin: p.Margin, DebtPrincipal: p.DebtPrincipal, DebtInterest: p.DebtInterest,
		DebtAsset: string(p.DebtAsset), DebtNotional: p.DebtNotional,
		LiquidationPrice: p.LiquidationPrice, Status: string(p.Status),
		MarkPrice: p.MarkPrice, UnrealizedPnL: p.UnrealizedPnL, RealizedPnL: p.RealizedPnL,
		CloseReason: p.CloseReason, StopLoss: p.StopLoss, TakeProfit: p.TakeProfit,
		OpenedAt: p.OpenedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if !p.LastInterestAt.IsZero() {
		d.LastInterestAt = p.LastInterestAt.UTC().Format(time.RFC3339Nano)
	}
	if p.ClosedAt != nil {
		s := p.ClosedAt.UTC().Format(time.RFC3339Nano)
		d.ClosedAt = &s
	}
	return d
}

func marginOrdDTO(o *domain.MarginOrder) marginOrderDTO {
	d := marginOrderDTO{
		ID: o.ID, ClientID: o.ClientID, Exchange: string(o.Exchange), Symbol: o.Symbol,
		Side: string(o.Side), Type: string(o.Type), Quantity: o.Quantity, Leverage: o.Leverage,
		LimitPrice: o.LimitPrice, ReservedMargin: o.ReservedMargin, StopLoss: o.StopLoss, TakeProfit: o.TakeProfit,
		Status: string(o.Status), PositionID: o.PositionID, RejectReason: o.RejectReason, CancelReason: o.CancelReason,
		CreatedAt: o.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: o.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if o.FilledAt != nil {
		s := o.FilledAt.UTC().Format(time.RFC3339Nano)
		d.FilledAt = &s
	}
	if o.CanceledAt != nil {
		s := o.CanceledAt.UTC().Format(time.RFC3339Nano)
		d.CanceledAt = &s
	}
	return d
}

func marginTradeToDTO(t *domain.MarginTrade) marginTradeDTO {
	return marginTradeDTO{
		ID: t.ID, PositionID: t.PositionID, Exchange: string(t.Exchange), Symbol: t.Symbol,
		Side: string(t.Side), Action: t.Action, Quantity: t.Quantity, Price: t.Price, Notional: t.Notional,
		RealizedPnL: t.RealizedPnL, MarginDelta: t.MarginDelta,
		PrincipalPaid: t.PrincipalPaid, InterestPaid: t.InterestPaid, Leverage: t.Leverage,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

type placeMarginBody struct {
	ClientID    string   `json:"clientId"`
	PortfolioID string   `json:"portfolioId"`
	Exchange    string   `json:"exchange"`
	Symbol     string   `json:"symbol"`
	Side       string   `json:"side"`
	Type       string   `json:"type"`
	Quantity   float64  `json:"quantity"`
	Leverage   int      `json:"leverage"`
	LimitPrice float64  `json:"limitPrice"`
	StopLoss   *float64 `json:"stopLoss"`
	TakeProfit *float64 `json:"takeProfit"`
}

// PlaceMarginOrder handles POST /api/v1/portfolio/margin/orders
func (h *PortfolioHandler) PlaceMarginOrder(w http.ResponseWriter, r *http.Request) {
	var body placeMarginBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	pos, ord, err := h.svc.PlaceMarginOrder(r.Context(), portfolio.MarginOrderInput{
		ClientID: clientID, PortfolioID: coalescePortfolioID(r, body.PortfolioID), Exchange: body.Exchange, Symbol: body.Symbol, Side: body.Side, Type: body.Type,
		Quantity: body.Quantity, Leverage: body.Leverage, LimitPrice: body.LimitPrice,
		StopLoss: body.StopLoss, TakeProfit: body.TakeProfit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if pos != nil {
		writeJSON(w, http.StatusCreated, map[string]any{
			"type": "market", "position": marginPosDTO(pos),
			"note": "Paper margin position opened. Not real money.",
		})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"type": "limit", "order": marginOrdDTO(ord),
		"note": "Paper margin limit order resting; margin reserved. Not real money.",
	})
}

// ListMarginOrders handles GET /api/v1/portfolio/margin/orders
func (h *PortfolioHandler) ListMarginOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.ListMarginOrders(r.Context(), clientIDFrom(r), q.Get("status"), limit, offset, portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]marginOrderDTO, 0, len(list))
	for i := range list {
		items = append(items, marginOrdDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "orders": items, "count": len(items),
	})
}

// CancelMarginOrder handles DELETE /api/v1/portfolio/margin/orders/{id}
func (h *PortfolioHandler) CancelMarginOrder(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.CancelMarginOrder(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": marginOrdDTO(o), "note": "Margin order canceled; reservation released."})
}

// ListMarginPositions handles GET /api/v1/portfolio/margin/positions
func (h *PortfolioHandler) ListMarginPositions(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListMarginPositions(r.Context(), clientIDFrom(r), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]marginPositionDTO, 0, len(list))
	for i := range list {
		items = append(items, marginPosDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "positions": items, "count": len(items),
	})
}

// GetMarginPosition handles GET /api/v1/portfolio/margin/positions/{id}
func (h *PortfolioHandler) GetMarginPosition(w http.ResponseWriter, r *http.Request) {
	pos, err := h.svc.GetMarginPosition(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, marginPosDTO(pos))
}

type closeMarginBody struct {
	Quantity float64 `json:"quantity"`
}

// CloseMarginPosition handles POST /api/v1/portfolio/margin/positions/{id}/close
func (h *PortfolioHandler) CloseMarginPosition(w http.ResponseWriter, r *http.Request) {
	var body closeMarginBody
	_ = decodeJSON(r, &body, DefaultMaxJSONBody) // empty body = full close
	pos, tr, err := h.svc.CloseMarginPosition(r.Context(), portfolio.MarginCloseInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), PositionID: r.PathValue("id"), Quantity: body.Quantity,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"position": marginPosDTO(pos), "trade": marginTradeToDTO(tr),
	})
}

type bracketsBody struct {
	StopLoss        *float64 `json:"stopLoss"`
	TakeProfit      *float64 `json:"takeProfit"`
	ClearStopLoss   bool     `json:"clearStopLoss"`
	ClearTakeProfit bool     `json:"clearTakeProfit"`
}

// SetMarginBrackets handles PUT /api/v1/portfolio/margin/positions/{id}/brackets
func (h *PortfolioHandler) SetMarginBrackets(w http.ResponseWriter, r *http.Request) {
	var body bracketsBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	pos, err := h.svc.SetMarginBrackets(r.Context(), portfolio.MarginBracketsInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), PositionID: r.PathValue("id"),
		StopLoss: body.StopLoss, TakeProfit: body.TakeProfit,
		ClearStopLoss: body.ClearStopLoss, ClearTakeProfit: body.ClearTakeProfit,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, marginPosDTO(pos))
}

// ListMarginTrades handles GET /api/v1/portfolio/margin/trades
func (h *PortfolioHandler) ListMarginTrades(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.ListMarginTrades(r.Context(), clientIDFrom(r), limit, offset, portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]marginTradeDTO, 0, len(list))
	for i := range list {
		items = append(items, marginTradeToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "trades": items, "count": len(items),
	})
}

type setMarginModeBody struct {
	ClientID    string `json:"clientId"`
	PortfolioID string `json:"portfolioId"`
	Mode        string `json:"mode"`
}

// SetMarginMode handles PUT /api/v1/portfolio/margin/mode
func (h *PortfolioHandler) SetMarginMode(w http.ResponseWriter, r *http.Request) {
	var body setMarginModeBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	p, err := h.svc.SetMarginMode(r.Context(), portfolio.SetMarginModeInput{ClientID: clientID, PortfolioID: coalescePortfolioID(r, body.PortfolioID), Mode: body.Mode})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": p.ClientID, "marginMode": string(p.MarginMode),
		"updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	})
}

type adjustMarginBody struct {
	Delta float64 `json:"delta"`
}

// AdjustMargin handles POST /api/v1/portfolio/margin/positions/{id}/margin
func (h *PortfolioHandler) AdjustMargin(w http.ResponseWriter, r *http.Request) {
	var body adjustMarginBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	pos, err := h.svc.AdjustMargin(r.Context(), portfolio.MarginAdjustInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), PositionID: r.PathValue("id"), Delta: body.Delta,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, marginPosDTO(pos))
}

type repayMarginBody struct {
	Amount float64 `json:"amount"`
}

// RepayMarginDebt handles POST /api/v1/portfolio/margin/positions/{id}/repay
func (h *PortfolioHandler) RepayMarginDebt(w http.ResponseWriter, r *http.Request) {
	var body repayMarginBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	pos, tr, err := h.svc.RepayMarginDebt(r.Context(), portfolio.MarginRepayInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), PositionID: r.PathValue("id"), Amount: body.Amount,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"position": marginPosDTO(pos), "trade": marginTradeToDTO(tr),
	})
}
