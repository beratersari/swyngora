package handler

import (
	"errors"
	"fmt"
	"io"
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
	ID                  string              `json:"id"`
	ClientID            string              `json:"clientId"`
	Name                string              `json:"name"`
	Role                string              `json:"role,omitempty"`
	Currency            string              `json:"currency"`
	StartingBalance     float64             `json:"startingBalance"`
	CashBalance         float64             `json:"cashBalance"`
	NetDeposits         float64             `json:"netDeposits"`
	ContributedCapital  float64             `json:"contributedCapital"`
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
	Exchange          string      `json:"exchange"`
	Symbol            string      `json:"symbol"`
	Quantity          float64     `json:"quantity"`
	ReservedQuantity  float64     `json:"reservedQuantity"`
	AvailableQuantity float64     `json:"availableQuantity"`
	AvgCost           float64     `json:"avgCost"`
	MarkPrice         float64     `json:"markPrice"`
	MarketValue       float64     `json:"marketValue"`
	UnrealizedPnL     float64     `json:"unrealizedPnL"`
	CostBasis         float64     `json:"costBasis"`
	Lots              []taxLotDTO `json:"lots,omitempty"`
}

type taxLotDTO struct {
	ID               string  `json:"id"`
	Exchange         string  `json:"exchange"`
	Symbol           string  `json:"symbol"`
	Quantity         float64 `json:"quantity"`
	OriginalQuantity float64 `json:"originalQuantity"`
	Price            float64 `json:"price"`
	OpenedAt         string  `json:"openedAt"`
	SourceTradeID    string  `json:"sourceTradeId,omitempty"`
	ClosedAt         *string `json:"closedAt,omitempty"`
}

type taxLotFillDTO struct {
	LotID       string  `json:"lotId"`
	Quantity    float64 `json:"quantity"`
	CostPrice   float64 `json:"costPrice"`
	SellPrice   float64 `json:"sellPrice"`
	RealizedPnL float64 `json:"realizedPnL"`
}

type tradeDTO struct {
	ID             string          `json:"id"`
	Exchange       string          `json:"exchange"`
	Symbol         string          `json:"symbol"`
	Side           string          `json:"side"`
	Quantity       float64         `json:"quantity"`
	Price          float64         `json:"price"`
	Notional       float64         `json:"notional"`
	RealizedPnL    float64         `json:"realizedPnL"`
	PendingOrderID string          `json:"pendingOrderId,omitempty"`
	LotMethod      string          `json:"lotMethod,omitempty"`
	LotFills       []taxLotFillDTO `json:"lotFills,omitempty"`
	Fee            float64         `json:"fee"`
	LastPrice      float64         `json:"lastPrice,omitempty"`
	CreatedAt      string          `json:"createdAt"`
}

func portfolioViewDTO(v *domain.PortfolioView) portfolioDTO {
	pos := make([]positionDTO, 0, len(v.Positions))
	for _, p := range v.Positions {
		pos = append(pos, positionDTO{
			Exchange: string(p.Exchange), Symbol: p.Symbol, Quantity: p.Quantity,
			ReservedQuantity: p.ReservedQuantity, AvailableQuantity: p.AvailableQuantity,
			AvgCost: p.AvgCost, MarkPrice: p.MarkPrice, MarketValue: p.MarketValue,
			UnrealizedPnL: p.UnrealizedPnL, CostBasis: p.CostBasis,
			Lots: taxLotsDTO(p.Lots),
		})
	}
	mpos := make([]marginPositionDTO, 0, len(v.MarginPositions))
	for i := range v.MarginPositions {
		mpos = append(mpos, marginPosDTO(&v.MarginPositions[i]))
	}
	return portfolioDTO{
		ID: v.ID, ClientID: v.ClientID, Name: v.Name, Role: string(v.Role),
		Currency: v.Currency, StartingBalance: v.StartingBalance,
		CashBalance: v.CashBalance, NetDeposits: v.NetDeposits, ContributedCapital: v.ContributedCapital,
		ReservedCash: v.ReservedCash, ReservedMargin: v.ReservedMargin,
		AvailableCash: v.AvailableCash, PositionsValue: v.PositionsValue, MarginMode: string(v.MarginMode),
		MarginLocked: v.MarginLocked, MarginUnrealizedPnL: v.MarginUnrealizedPnL, MarginEquity: v.MarginEquity,
		Equity: v.Equity, UnrealizedPnL: v.UnrealizedPnL, RealizedPnLTotal: v.RealizedPnLTotal, TotalPnL: v.TotalPnL,
		Positions: pos, MarginPositions: mpos, Note: v.Note,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: v.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func tradeToDTO(t *domain.Trade) tradeDTO {
	fills := make([]taxLotFillDTO, 0, len(t.LotFills))
	for _, f := range t.LotFills {
		fills = append(fills, taxLotFillDTO{
			LotID: f.LotID, Quantity: f.Quantity, CostPrice: f.CostPrice,
			SellPrice: f.SellPrice, RealizedPnL: f.RealizedPnL,
		})
	}
	d := tradeDTO{
		ID: t.ID, Exchange: string(t.Exchange), Symbol: t.Symbol, Side: string(t.Side),
		Quantity: t.Quantity, Price: t.Price, Notional: t.Notional, RealizedPnL: t.RealizedPnL,
		PendingOrderID: t.PendingOrderID, LotMethod: string(t.LotMethod),
		Fee: t.Fee, LastPrice: t.LastPrice,
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if len(fills) > 0 {
		d.LotFills = fills
	}
	return d
}

func taxLotsDTO(lots []domain.TaxLot) []taxLotDTO {
	if len(lots) == 0 {
		return nil
	}
	out := make([]taxLotDTO, 0, len(lots))
	for _, l := range lots {
		d := taxLotDTO{
			ID: l.ID, Exchange: string(l.Exchange), Symbol: l.Symbol,
			Quantity: l.Quantity, OriginalQuantity: l.OriginalQuantity, Price: l.Price,
			OpenedAt: l.OpenedAt.UTC().Format(time.RFC3339Nano), SourceTradeID: l.SourceTradeID,
		}
		if l.ClosedAt != nil {
			s := l.ClosedAt.UTC().Format(time.RFC3339Nano)
			d.ClosedAt = &s
		}
		out = append(out, d)
	}
	return out
}

type createPortfolioBody struct {
	ClientID        string  `json:"clientId"`
	Name            string  `json:"name"`
	StartingBalance float64 `json:"startingBalance"`
	Currency        string  `json:"currency"`
}

type renamePortfolioBody struct {
	Name string `json:"name"`
}

type portfolioSummaryDTO struct {
	ID              string  `json:"id"`
	ClientID        string  `json:"clientId"`
	Name            string  `json:"name"`
	Currency        string  `json:"currency"`
	StartingBalance float64 `json:"startingBalance"`
	CashBalance     float64 `json:"cashBalance"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

func portfolioSummary(p domain.Portfolio) portfolioSummaryDTO {
	return portfolioSummaryDTO{
		ID: p.ID, ClientID: p.ClientID, Name: p.Name, Currency: p.Currency,
		StartingBalance: p.StartingBalance, CashBalance: p.CashBalance,
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
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
		ClientID: clientID, Name: body.Name, StartingBalance: body.StartingBalance, Currency: body.Currency,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	view, err := h.svc.View(r.Context(), p.ClientID, p.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, portfolioViewDTO(view))
}

// List handles GET /api/v1/portfolios
func (h *PortfolioHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]portfolioSummaryDTO, 0, len(list))
	for _, p := range list {
		items = append(items, portfolioSummary(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "portfolios": items, "count": len(items),
	})
}

// Rename handles PATCH /api/v1/portfolios/{id}
func (h *PortfolioHandler) Rename(w http.ResponseWriter, r *http.Request) {
	var body renamePortfolioBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	p, err := h.svc.Rename(r.Context(), clientIDFrom(r), r.PathValue("id"), body.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolioSummary(*p))
}

// DeleteBook handles DELETE /api/v1/portfolios/{id}
func (h *PortfolioHandler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), clientIDFrom(r), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": r.PathValue("id")})
}

// ListLots handles GET /api/v1/portfolio/lots
func (h *PortfolioHandler) ListLots(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lots, err := h.svc.ListLots(r.Context(), clientIDFrom(r), q.Get("exchange"), q.Get("symbol"), q.Get("status"),
		portfolioIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"lots":  taxLotsDTO(lots),
		"count": len(lots),
	})
}

// Get handles GET /api/v1/portfolio
func (h *PortfolioHandler) Get(w http.ResponseWriter, r *http.Request) {
	view, err := h.svc.View(r.Context(), clientIDFrom(r), portfolioIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, portfolioViewDTO(view))
}

type orderBody struct {
	ClientID        string  `json:"clientId"`
	PortfolioID     string  `json:"portfolioId"`
	OwnerClientID   string  `json:"ownerClientId"`
	Exchange        string  `json:"exchange"`
	Symbol          string  `json:"symbol"`
	Side            string  `json:"side"`
	Quantity        float64 `json:"quantity"`
	Type            string  `json:"type"`         // market | limit_buy | limit_sell | stop_loss | trailing_stop | oco | bracket
	TriggerPrice    float64 `json:"triggerPrice"` // pending single-leg / bracket entry
	TakeProfitPrice float64 `json:"takeProfitPrice"` // oco / bracket
	StopLossPrice   float64 `json:"stopLossPrice"`   // oco / bracket
	TrailType       string  `json:"trailType"`        // trailing_stop: percent | offset
	TrailValue      float64 `json:"trailValue"`       // trailing_stop distance
	TimeInForce     string  `json:"timeInForce"`      // gtc | ioc | fok
	ExpiresAt       string  `json:"expiresAt"`        // RFC3339; GTC only
	LotMethod       string  `json:"lotMethod"`        // fifo | lifo (sells)
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
	BracketID         string  `json:"bracketId,omitempty"`
	BracketRole       string  `json:"bracketRole,omitempty"`
	TrailType         string  `json:"trailType,omitempty"`
	TrailValue        float64 `json:"trailValue,omitempty"`
	TrailPeak         float64 `json:"trailPeak,omitempty"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	FilledAt          *string `json:"filledAt,omitempty"`
	CanceledAt        *string `json:"canceledAt,omitempty"`
	FillTradeID       string  `json:"fillTradeId,omitempty"`
	FillPrice         float64 `json:"fillPrice,omitempty"`
	RejectReason      string  `json:"rejectReason,omitempty"`
	CancelReason      string  `json:"cancelReason,omitempty"`
	LotMethod         string  `json:"lotMethod,omitempty"`
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
		BracketID: o.BracketID, BracketRole: o.BracketRole,
		TrailType: o.TrailType, TrailValue: o.TrailValue, TrailPeak: o.TrailPeak,
		CreatedAt:   o.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   o.UpdatedAt.UTC().Format(time.RFC3339Nano),
		FillTradeID: o.FillTradeID, FillPrice: o.FillPrice, RejectReason: o.RejectReason, CancelReason: o.CancelReason,
		LotMethod: string(o.LotMethod),
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
	pfID := coalescePortfolioID(r, body.PortfolioID)
	ownerID := ownerClientIDFrom(r)
	if body.OwnerClientID != "" {
		ownerID = body.OwnerClientID
	}
	typ := body.Type
	if typ == "" {
		typ = "market"
	}
	switch typ {
	case "market":
		tr, view, err := h.svc.PlaceOrder(r.Context(), portfolio.OrderInput{
			ClientID: clientID, PortfolioID: pfID, OwnerClientID: ownerID, Exchange: body.Exchange, Symbol: body.Symbol,
			Side: body.Side, Quantity: body.Quantity, LotMethod: body.LotMethod,
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
	case "limit_buy", "limit_sell", "stop_loss", "trailing_stop":
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
			ClientID: clientID, PortfolioID: pfID, OwnerClientID: ownerID, Exchange: body.Exchange, Symbol: body.Symbol,
			Type: typ, Quantity: body.Quantity, TriggerPrice: body.TriggerPrice,
			TrailType: body.TrailType, TrailValue: body.TrailValue,
			TimeInForce: body.TimeInForce, ExpiresAt: exp, LotMethod: body.LotMethod,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		note := "Paper pending order (GTC/IOC/FOK) with reservations. GTC may expire; IOC/FOK act on first try. Not real money."
		if typ == "trailing_stop" {
			note = "Paper trailing stop: stop ratchets up with price (percent or offset), never moves down; fires once on touch or gap. Not real money."
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"type":  typ,
			"order": pendingOrderToDTO(o),
			"note":  note,
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
			ClientID: clientID, PortfolioID: pfID, OwnerClientID: ownerID, Exchange: body.Exchange, Symbol: body.Symbol,
			Quantity: body.Quantity, TakeProfitPrice: body.TakeProfitPrice, StopLossPrice: body.StopLossPrice,
			ExpiresAt: exp, LotMethod: body.LotMethod,
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
	case "bracket":
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
		entry, tp, sl, err := h.svc.PlaceBracketOrder(r.Context(), portfolio.BracketOrderInput{
			ClientID: clientID, PortfolioID: pfID, OwnerClientID: ownerID, Exchange: body.Exchange, Symbol: body.Symbol,
			Quantity: body.Quantity, EntryPrice: body.TriggerPrice,
			TakeProfitPrice: body.TakeProfitPrice, StopLossPrice: body.StopLossPrice,
			ExpiresAt: exp, LotMethod: body.LotMethod,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"type":       "bracket",
			"bracketId":  entry.BracketID,
			"entry":      pendingOrderToDTO(entry),
			"takeProfit": pendingOrderToDTO(tp),
			"stopLoss":   pendingOrderToDTO(sl),
			"note":       "Paper bracket: limit-buy entry with take-profit + stop-loss. Exits stay pending until entry fills; size tracks filled qty; OCO exits cannot double-sell. Not real money.",
		})
	default:
		writeError(w, fmt.Errorf("%w: type must be market, limit_buy, limit_sell, stop_loss, trailing_stop, oco, or bracket", domain.ErrInvalidArgument))
	}
}

// ListOrders handles GET /api/v1/portfolio/orders — open pending orders by default.
func (h *PortfolioHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	status := q.Get("status")
	list, err := h.svc.ListPendingOrders(r.Context(), clientIDFrom(r), status, limit, offset, portfolioIDFrom(r), ownerClientIDFrom(r))
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

// GetOrder handles GET /api/v1/portfolio/orders/{id}
func (h *PortfolioHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	d, err := h.svc.GetPendingOrderDetail(r.Context(), clientIDFrom(r), id, portfolioIDFrom(r), ownerClientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pendingOrderDetailDTO(d))
}

type amendOrderBody struct {
	TriggerPrice      *float64 `json:"triggerPrice"`
	RemainingQuantity *float64 `json:"remainingQuantity"`
}

// AmendOrder handles PATCH /api/v1/portfolio/orders/{id}
func (h *PortfolioHandler) AmendOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	var body amendOrderBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	o, view, err := h.svc.AmendPendingOrder(r.Context(), portfolio.AmendPendingOrderInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), OwnerClientID: ownerClientIDFrom(r), OrderID: id,
		TriggerPrice: body.TriggerPrice, RemainingQuantity: body.RemainingQuantity,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	note := "Order amended in place; reservations updated. Not real money."
	if o.Status == domain.PendingStatusFilled {
		note = "Order amended and filled at last price. Not real money."
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"order":     pendingOrderToDTO(o),
		"portfolio": portfolioViewDTO(view),
		"note":      note,
	})
}

func pendingOrderDetailDTO(d *portfolio.PendingOrderDetail) map[string]any {
	return map[string]any{
		"order":     pendingOrderToDTO(&d.Order),
		"lastPrice": d.LastPrice,
		"editable":  d.Editable,
		"amend": map[string]any{
			"availableCashForOrder":     d.AvailableCashForOrder,
			"availableQuantityForOrder": d.AvailableQuantityForOrder,
			"maxRemainingQuantity":      d.MaxRemainingQuantity,
			"minRemainingQuantity":      d.MinRemainingQuantity,
		},
		"note": "Paper pending order. Amend keeps the same id and updates reservations. Not real money.",
	}
}

type cancelAllBody struct {
	ClientID      string `json:"clientId"`
	PortfolioID   string `json:"portfolioId"`
	OwnerClientID string `json:"ownerClientId"`
	Exchange      string `json:"exchange"`
	Symbol        string `json:"symbol"`
}

// CancelAllOrders handles POST /api/v1/portfolio/orders/cancel-all
// Empty body/symbol cancels every open (and pending bracket-exit) order; symbol scopes to one market.
func (h *PortfolioHandler) CancelAllOrders(w http.ResponseWriter, r *http.Request) {
	var body cancelAllBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	list, view, err := h.svc.CancelOpenPendingOrders(r.Context(), portfolio.CancelOpenOrdersInput{
		ClientID: clientID, PortfolioID: coalescePortfolioID(r, body.PortfolioID), OwnerClientID: coalesceOwner(r, body.OwnerClientID),
		Exchange: body.Exchange, Symbol: body.Symbol,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]pendingOrderDTO, 0, len(list))
	for i := range list {
		items = append(items, pendingOrderToDTO(&list[i]))
	}
	scope := "all"
	if strings.TrimSpace(body.Symbol) != "" {
		scope = "market"
	} else if strings.TrimSpace(body.Exchange) != "" {
		scope = "exchange"
	}
	note := "Open paper orders canceled; unused reservations released. Not real money."
	if len(items) == 0 {
		note = "No open paper orders to cancel."
	}
	out := map[string]any{
		"orders":    items,
		"canceled":  len(items),
		"scope":     scope,
		"note":      note,
	}
	if view != nil {
		out["portfolio"] = portfolioViewDTO(view)
	}
	if scope == "market" || scope == "exchange" {
		if len(list) > 0 {
			out["exchange"] = string(list[0].Exchange)
			if scope == "market" {
				out["symbol"] = list[0].Symbol
			}
		} else if strings.TrimSpace(body.Symbol) != "" {
			out["symbol"] = strings.ToUpper(strings.TrimSpace(body.Symbol))
			if body.Exchange != "" {
				out["exchange"] = strings.ToLower(strings.TrimSpace(body.Exchange))
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// CancelOrder handles DELETE /api/v1/portfolio/orders/{id}
func (h *PortfolioHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	o, err := h.svc.CancelPendingOrder(r.Context(), clientIDFrom(r), id, portfolioIDFrom(r), ownerClientIDFrom(r))
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
	list, total, err := h.svc.ListTrades(r.Context(), clientIDFrom(r), limit, offset, portfolioIDFrom(r), ownerClientIDFrom(r))
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

// GetTradingCosts handles GET /api/v1/portfolio/trading-costs
func (h *PortfolioHandler) GetTradingCosts(w http.ResponseWriter, r *http.Request) {
	ex := strings.TrimSpace(r.URL.Query().Get("exchange"))
	if ex != "" {
		v := domain.PaperTradingCostViewFor(domain.ParseExchange(ex))
		writeJSON(w, http.StatusOK, map[string]any{
			"exchange":     string(v.Exchange),
			"feeRate":      v.FeeRate,
			"slippageRate": v.SlippageRate,
			"feePct":       v.FeePct,
			"slippagePct":  v.SlippagePct,
			"note":         domain.PaperTradingCostsNote,
		})
		return
	}
	all := domain.AllPaperTradingCosts()
	items := make([]map[string]any, 0, len(all))
	for _, v := range all {
		items = append(items, map[string]any{
			"exchange":     string(v.Exchange),
			"feeRate":      v.FeeRate,
			"slippageRate": v.SlippageRate,
			"feePct":       v.FeePct,
			"slippagePct":  v.SlippagePct,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"note":  domain.PaperTradingCostsNote,
	})
}