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
		Status:    string(w.Status),
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
		Status:     string(o.Status),
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

type priceDiffQuoteDTO struct {
	Symbol              string  `json:"symbol"`
	BuyExchange         string  `json:"buyExchange"`
	SellExchange        string  `json:"sellExchange"`
	RequestedNotional   string  `json:"requestedNotional,omitempty"`
	RequestedQuantity   string  `json:"requestedQuantity,omitempty"`
	FilledQuantity      string  `json:"filledQuantity"`
	FilledRequested     bool    `json:"filledRequested"`
	AverageBuyPrice     string  `json:"averageBuyPrice"`
	AverageSellPrice    string  `json:"averageSellPrice"`
	BestAsk             string  `json:"bestAsk,omitempty"`
	BestBid             string  `json:"bestBid,omitempty"`
	BuySlippagePct      float64 `json:"buySlippagePct"`
	SellSlippagePct     float64 `json:"sellSlippagePct"`
	SlippagePct         float64 `json:"slippagePct"`
	BuyNotional         string  `json:"buyNotional"`
	SellNotional        string  `json:"sellNotional"`
	BuyFeePct           float64 `json:"buyFeePct"`
	SellFeePct          float64 `json:"sellFeePct"`
	BuyFee              string  `json:"buyFee"`
	SellFee             string  `json:"sellFee"`
	CostAfterFees       string  `json:"costAfterFees"`
	ProceedsAfterFees   string  `json:"proceedsAfterFees"`
	ProfitAfterFees     string  `json:"profitAfterFees"`
	ProfitPct           float64 `json:"profitPct"`
	GrossProfit         string  `json:"grossProfit"`
	GrossPct            float64 `json:"grossPct"`
	BuyExhausted        bool    `json:"buyExhausted"`
	SellExhausted       bool    `json:"sellExhausted"`
	Profitable          bool    `json:"profitable"`
	Executable          bool    `json:"executable"`
	MeetsMinNet         bool    `json:"meetsMinNet"`
	MinNetDiffPct       float64 `json:"minNetDiffPct,omitempty"`
	Live                bool    `json:"live"`
	BuyLive             bool    `json:"buyLive"`
	SellLive            bool    `json:"sellLive"`
	AsOf                string  `json:"asOf,omitempty"`
	VisibleBuyQty       string  `json:"visibleBuyQuantity"`
	VisibleBuyNotional  string  `json:"visibleBuyNotional"`
	VisibleSellQty      string  `json:"visibleSellQuantity"`
	VisibleSellNotional string  `json:"visibleSellNotional"`
	MaxQuantity         string  `json:"maxQuantity,omitempty"`
	MaxNotional         string  `json:"maxNotional,omitempty"`
	MaxAverageBuyPrice  string  `json:"maxAverageBuyPrice,omitempty"`
	MaxAverageSellPrice string  `json:"maxAverageSellPrice,omitempty"`
	MaxProfitAfterFees  string  `json:"maxProfitAfterFees,omitempty"`
	MaxProfitPct        float64 `json:"maxProfitPct,omitempty"`
	MaxLimitedBy        string  `json:"maxLimitedBy"`
	Note                string  `json:"note"`
}

func quoteDTO(q *domain.PriceDiffQuote) priceDiffQuoteDTO {
	if q == nil {
		return priceDiffQuoteDTO{}
	}
	d := priceDiffQuoteDTO{
		Symbol: q.Symbol, BuyExchange: string(q.BuyExchange), SellExchange: string(q.SellExchange),
		RequestedNotional: q.RequestedNotional, RequestedQuantity: q.RequestedQuantity,
		FilledQuantity: q.FilledQuantity, FilledRequested: q.FilledRequested,
		AverageBuyPrice: q.AverageBuyPrice, AverageSellPrice: q.AverageSellPrice,
		BestAsk: q.BestAsk, BestBid: q.BestBid,
		BuySlippagePct: q.BuySlippagePct, SellSlippagePct: q.SellSlippagePct, SlippagePct: q.SlippagePct,
		BuyNotional: q.BuyNotional, SellNotional: q.SellNotional,
		BuyFeePct: q.BuyFeePct, SellFeePct: q.SellFeePct, BuyFee: q.BuyFee, SellFee: q.SellFee,
		CostAfterFees: q.CostAfterFees, ProceedsAfterFees: q.ProceedsAfterFees,
		ProfitAfterFees: q.ProfitAfterFees, ProfitPct: q.ProfitPct,
		GrossProfit: q.GrossProfit, GrossPct: q.GrossPct,
		BuyExhausted: q.BuyExhausted, SellExhausted: q.SellExhausted,
		Profitable: q.Profitable, Executable: q.Executable, MeetsMinNet: q.MeetsMinNet,
		MinNetDiffPct: q.MinNetDiffPct, Live: q.Live, BuyLive: q.BuyLive, SellLive: q.SellLive,
		VisibleBuyQty: q.VisibleBuyQty, VisibleBuyNotional: q.VisibleBuyNotional,
		VisibleSellQty: q.VisibleSellQty, VisibleSellNotional: q.VisibleSellNotional,
		MaxQuantity: q.MaxQuantity, MaxNotional: q.MaxNotional,
		MaxAverageBuyPrice: q.MaxAverageBuyPrice, MaxAverageSellPrice: q.MaxAverageSellPrice,
		MaxProfitAfterFees: q.MaxProfitAfterFees, MaxProfitPct: q.MaxProfitPct,
		MaxLimitedBy: q.MaxLimitedBy, Note: q.Note,
	}
	if !q.AsOf.IsZero() {
		d.AsOf = q.AsOf.UTC().Format(time.RFC3339Nano)
	}
	return d
}

func parseQuoteSize(r *http.Request) (notional, quantity float64, err error) {
	q := r.URL.Query()
	if raw := q.Get("notional"); raw != "" {
		n, e := strconv.ParseFloat(raw, 64)
		if e != nil {
			return 0, 0, fmt.Errorf("%w: notional must be a number", domain.ErrInvalidArgument)
		}
		notional = n
	}
	if raw := q.Get("quantity"); raw != "" {
		n, e := strconv.ParseFloat(raw, 64)
		if e != nil {
			return 0, 0, fmt.Errorf("%w: quantity must be a number", domain.ErrInvalidArgument)
		}
		quantity = n
	}
	return notional, quantity, nil
}

func parseOptionalFloat(raw string) (float64, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: must be a number", domain.ErrInvalidArgument)
	}
	return n, nil
}

// QuoteRoute handles GET /api/v1/price-diff/quote
func (h *PriceDiffHandler) QuoteRoute(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	notional, quantity, err := parseQuoteSize(r)
	if err != nil {
		writeError(w, err)
		return
	}
	buyFee, err := parseOptionalFloat(q.Get("feeBuyPct"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: feeBuyPct must be a number", domain.ErrInvalidArgument))
		return
	}
	sellFee, err := parseOptionalFloat(q.Get("feeSellPct"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: feeSellPct must be a number", domain.ErrInvalidArgument))
		return
	}
	minNet, err := parseOptionalFloat(q.Get("minNetDiffPct"))
	if err != nil {
		writeError(w, fmt.Errorf("%w: minNetDiffPct must be a number", domain.ErrInvalidArgument))
		return
	}
	got, err := h.svc.Quote(r.Context(), pricediff.QuoteInput{
		Symbol: q.Get("symbol"), BuyExchange: q.Get("buyExchange"), SellExchange: q.Get("sellExchange"),
		BuyFeePct: buyFee, SellFeePct: sellFee, Notional: notional, Quantity: quantity, MinNetDiffPct: minNet,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, quoteDTO(got))
}

// QuoteOpportunity handles GET /api/v1/price-diff/opportunities/{id}/quote
func (h *PriceDiffHandler) QuoteOpportunity(w http.ResponseWriter, r *http.Request) {
	notional, quantity, err := parseQuoteSize(r)
	if err != nil {
		writeError(w, err)
		return
	}
	got, err := h.svc.QuoteOpportunity(r.Context(), clientIDFrom(r), r.PathValue("id"), notional, quantity)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, quoteDTO(got))
}
