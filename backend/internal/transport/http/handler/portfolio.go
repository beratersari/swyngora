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

// PortfolioHandler is the HTTP adapter for paper trading.
type PortfolioHandler struct {
	svc *portfolio.Service
}

// NewPortfolioHandler constructs the handler.
func NewPortfolioHandler(svc *portfolio.Service) *PortfolioHandler {
	return &PortfolioHandler{svc: svc}
}

type portfolioDTO struct {
	ClientID            string              `json:"clientId"`
	Currency            string              `json:"currency"`
	StartingBalance     float64             `json:"startingBalance"`
	CashBalance         float64             `json:"cashBalance"`
	ReservedCash        float64             `json:"reservedCash"`
	ReservedMargin      float64             `json:"reservedMargin"`
	AvailableCash       float64             `json:"availableCash"`
	PositionsValue      float64             `json:"positionsValue"`
	MarginMode          string              `json:"marginMode"`
	MarginLocked        float64             `json:"marginLocked"`
	MarginUnrealizedPnL float64             `json:"marginUnrealizedPnL"`
	MarginEquity        float64             `json:"marginEquity"`
	Equity              float64             `json:"equity"`
	UnrealizedPnL       float64             `json:"unrealizedPnL"`
	RealizedPnLTotal    float64             `json:"realizedPnLTotal"`
	TotalPnL            float64             `json:"totalPnL"`
	Positions           []positionDTO       `json:"positions"`
	MarginPositions     []marginPositionDTO `json:"marginPositions"`
	Note                string              `json:"note"`
	CreatedAt           string              `json:"createdAt"`
	UpdatedAt           string              `json:"updatedAt"`
}

type positionDTO struct {
	Exchange          string  `json:"exchange"`
	Symbol            string  `json:"symbol"`
	Quantity          float64 `json:"quantity"`
	ReservedQuantity  float64 `json:"reservedQuantity"`
	AvailableQuantity float64 `json:"availableQuantity"`
	AvgCost           float64 `json:"avgCost"`
	MarkPrice         float64 `json:"markPrice"`
	MarketValue       float64 `json:"marketValue"`
	UnrealizedPnL     float64 `json:"unrealizedPnL"`
	CostBasis         float64 `json:"costBasis"`
}

type tradeDTO struct {
	ID             string  `json:"id"`
	Exchange       string  `json:"exchange"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	Quantity       float64 `json:"quantity"`
	Price          float64 `json:"price"`
	Notional       float64 `json:"notional"`
	RealizedPnL    float64 `json:"realizedPnL"`
	PendingOrderID string  `json:"pendingOrderId,omitempty"`
	CreatedAt      string  `json:"createdAt"`
}

func portfolioViewDTO(v *domain.PortfolioView) portfolioDTO {
	pos := make([]positionDTO, 0, len(v.Positions))
	for _, p := range v.Positions {
		pos = append(pos, positionDTO{
			Exchange: string(p.Exchange), Symbol: p.Symbol, Quantity: p.Quantity,
			ReservedQuantity: p.ReservedQuantity, AvailableQuantity: p.AvailableQuantity,
			AvgCost: p.AvgCost, MarkPrice: p.MarkPrice, MarketValue: p.MarketValue,
			UnrealizedPnL: p.UnrealizedPnL, CostBasis: p.CostBasis,
		})
	}
	mpos := make([]marginPositionDTO, 0, len(v.MarginPositions))
	for i := range v.MarginPositions {
		mpos = append(mpos, marginPosDTO(&v.MarginPositions[i]))
	}
	return portfolioDTO{
		ClientID: v.ClientID, Currency: v.Currency, StartingBalance: v.StartingBalance,
		CashBalance: v.CashBalance, ReservedCash: v.ReservedCash, ReservedMargin: v.ReservedMargin,
		AvailableCash: v.AvailableCash, PositionsValue: v.PositionsValue, MarginMode: string(v.MarginMode),
		MarginLocked: v.MarginLocked, MarginUnrealizedPnL: v.MarginUnrealizedPnL, MarginEquity: v.MarginEquity,
		Equity: v.Equity, UnrealizedPnL: v.UnrealizedPnL, RealizedPnLTotal: v.RealizedPnLTotal, TotalPnL: v.TotalPnL,
		Positions: pos, MarginPositions: mpos, Note: v.Note,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func tradeToDTO(t *domain.Trade) tradeDTO {
	return tradeDTO{
		ID: t.ID, Exchange: string(t.Exchange), Symbol: t.Symbol, Side: string(t.Side),
		Quantity: t.Quantity, Price: t.Price, Notional: t.Notional, RealizedPnL: t.RealizedPnL,
		PendingOrderID: t.PendingOrderID,
		CreatedAt:      t.CreatedAt.UTC().Format(time.RFC3339Nano),
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
	ClientID        string  `json:"clientId"`
	Exchange        string  `json:"exchange"`
	Symbol          string  `json:"symbol"`
	Side            string  `json:"side"`
	Quantity        float64 `json:"quantity"`
	Type            string  `json:"type"`         // market | limit_buy | limit_sell | stop_loss | oco
	TriggerPrice    float64 `json:"triggerPrice"` // pending single-leg
	TakeProfitPrice float64 `json:"takeProfitPrice"` // oco limit_sell
	StopLossPrice   float64 `json:"stopLossPrice"`   // oco stop_loss
	TimeInForce     string  `json:"timeInForce"`     // gtc | ioc | fok
	ExpiresAt       string  `json:"expiresAt"`       // RFC3339; GTC only
}

type pendingOrderDTO struct {
	ID                string  `json:"id"`
	ClientID          string  `json:"clientId"`
	Exchange          string  `json:"exchange"`
	Symbol            string  `json:"symbol"`
	Type              string  `json:"type"`
	Side              string  `json:"side"`
	Quantity          float64 `json:"quantity"`
	FilledQuantity    float64 `json:"filledQuantity"`
	RemainingQuantity float64 `json:"remainingQuantity"`
	TriggerPrice      float64 `json:"triggerPrice"`
	ReservedCash      float64 `json:"reservedCash"`
	ReservedQuantity  float64 `json:"reservedQuantity"`
	TimeInForce       string  `json:"timeInForce"`
	ExpiresAt         *string `json:"expiresAt,omitempty"`
	Status            string  `json:"status"`
	OCOGroupID        string  `json:"ocoGroupId,omitempty"`
	OCOPeerID         string  `json:"ocoPeerId,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	FilledAt          *string `json:"filledAt,omitempty"`
	CanceledAt        *string `json:"canceledAt,omitempty"`
	FillTradeID       string  `json:"fillTradeId,omitempty"`
	FillPrice         float64 `json:"fillPrice,omitempty"`
	RejectReason      string  `json:"rejectReason,omitempty"`
	CancelReason      string  `json:"cancelReason,omitempty"`
}

func pendingOrderToDTO(o *domain.PendingOrder) pendingOrderDTO {
	tif := string(o.TimeInForce)
	if tif == "" {
		tif = string(domain.TimeInForceGTC)
	}
	d := pendingOrderDTO{
		ID: o.ID, ClientID: o.ClientID, Exchange: string(o.Exchange), Symbol: o.Symbol,
		Type: string(o.Type), Side: string(o.Side), Quantity: o.Quantity,
		FilledQuantity: o.FilledQuantity, RemainingQuantity: o.RemainingQuantity,
		TriggerPrice: o.TriggerPrice, ReservedCash: o.ReservedCash, ReservedQuantity: o.ReservedQuantity,
		TimeInForce: tif,
		Status:      string(o.Status),
		OCOGroupID:  o.OCOGroupID, OCOPeerID: o.OCOPeerID,
		CreatedAt:   o.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   o.UpdatedAt.UTC().Format(time.RFC3339Nano),
		FillTradeID: o.FillTradeID, FillPrice: o.FillPrice, RejectReason: o.RejectReason, CancelReason: o.CancelReason,
	}
	if o.ExpiresAt != nil {
		s := o.ExpiresAt.UTC().Format(time.RFC3339Nano)
		d.ExpiresAt = &s
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

// PlaceOrder handles POST /api/v1/portfolio/orders
// type=market (default): immediate fill; limit_buy|limit_sell|stop_loss: resting order.
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
	typ := body.Type
	if typ == "" {
		typ = "market"
	}
	switch typ {
	case "market":
		tr, view, err := h.svc.PlaceOrder(r.Context(), portfolio.OrderInput{
			ClientID: clientID, Exchange: body.Exchange, Symbol: body.Symbol,
			Side: body.Side, Quantity: body.Quantity,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"type":      "market",
			"trade":     tradeToDTO(tr),
			"portfolio": portfolioViewDTO(view),
			"note":      view.Note,
		})
	case "limit_buy", "limit_sell", "stop_loss":
		var exp *time.Time
		if body.ExpiresAt != "" {
			t, perr := time.Parse(time.RFC3339Nano, body.ExpiresAt)
			if perr != nil {
				t, perr = time.Parse(time.RFC3339, body.ExpiresAt)
			}
			if perr != nil {
				writeError(w, fmt.Errorf("%w: expiresAt must be RFC3339", domain.ErrInvalidArgument))
				return
			}
			tu := t.UTC()
			exp = &tu
		}
		o, err := h.svc.PlacePendingOrder(r.Context(), portfolio.PendingOrderInput{
			ClientID: clientID, Exchange: body.Exchange, Symbol: body.Symbol,
			Type: typ, Quantity: body.Quantity, TriggerPrice: body.TriggerPrice,
			TimeInForce: body.TimeInForce, ExpiresAt: exp,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"type":  typ,
			"order": pendingOrderToDTO(o),
			"note":  "Paper pending order (GTC/IOC/FOK) with reservations. GTC may expire; IOC/FOK act on first try. Not real money.",
		})
	case "oco":
		var exp *time.Time
		if body.ExpiresAt != "" {
			t, perr := time.Parse(time.RFC3339Nano, body.ExpiresAt)
			if perr != nil {
				t, perr = time.Parse(time.RFC3339, body.ExpiresAt)
			}
			if perr != nil {
				writeError(w, fmt.Errorf("%w: expiresAt must be RFC3339", domain.ErrInvalidArgument))
				return
			}
			tu := t.UTC()
			exp = &tu
		}
		tp, sl, err := h.svc.PlaceOCOOrder(r.Context(), portfolio.OCOOrderInput{
			ClientID: clientID, Exchange: body.Exchange, Symbol: body.Symbol,
			Quantity: body.Quantity, TakeProfitPrice: body.TakeProfitPrice, StopLossPrice: body.StopLossPrice,
			ExpiresAt: exp,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"type":       "oco",
			"ocoGroupId": tp.OCOGroupID,
			"takeProfit": pendingOrderToDTO(tp),
			"stopLoss":   pendingOrderToDTO(sl),
			"note":       "Paper OCO: take-profit limit sell + stop-loss for the same size. One fill cancels or shrinks the other. Not real money.",
		})
	default:
		writeError(w, fmt.Errorf("%w: type must be market, limit_buy, limit_sell, stop_loss, or oco", domain.ErrInvalidArgument))
	}
}

// ListOrders handles GET /api/v1/portfolio/orders — open pending orders by default.
func (h *PortfolioHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	status := q.Get("status")
	list, err := h.svc.ListPendingOrders(r.Context(), clientIDFrom(r), status, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]pendingOrderDTO, 0, len(list))
	for i := range list {
		items = append(items, pendingOrderToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r),
		"orders":   items,
		"count":    len(items),
		"status":   statusOrDefault(status),
	})
}

func statusOrDefault(s string) string {
	if s == "" {
		return "open"
	}
	return s
}

// CancelOrder handles DELETE /api/v1/portfolio/orders/{id}
func (h *PortfolioHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	o, err := h.svc.CancelPendingOrder(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"order": pendingOrderToDTO(o),
		"note":  "Order canceled; unused reservation released; it will not execute.",
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