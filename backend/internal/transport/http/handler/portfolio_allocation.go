package handler

import (
	"fmt"
	"net/http"
	"strconv"
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

type rebalanceLegOutcomeDTO struct {
	Side            string  `json:"side"`
	Asset           string  `json:"asset"`
	Exchange        string  `json:"exchange"`
	Symbol          string  `json:"symbol"`
	Reason          string  `json:"reason,omitempty"`
	PlannedQty      float64 `json:"plannedQuantity"`
	PlannedPrice    float64 `json:"plannedPrice"`
	PlannedNotional float64 `json:"plannedNotional"`
	Status          string  `json:"status"`
	TradeID         string  `json:"tradeId,omitempty"`
	FilledQty       float64 `json:"filledQuantity,omitempty"`
	FillPrice       float64 `json:"fillPrice,omitempty"`
	FillNotional    float64 `json:"fillNotional,omitempty"`
	LastPrice       float64 `json:"lastPrice,omitempty"`
	Fee             float64 `json:"fee,omitempty"`
	Slippage        float64 `json:"slippage,omitempty"`
	Error           string  `json:"error,omitempty"`
	Attempt         int     `json:"attempt,omitempty"`
}

type rebalanceLineDTO struct {
	Asset       string  `json:"asset"`
	Exchange    string  `json:"exchange,omitempty"`
	Symbol      string  `json:"symbol,omitempty"`
	IsCash      bool    `json:"isCash"`
	TargetPct   float64 `json:"targetPct"`
	BeforePct   float64 `json:"beforePct"`
	AfterPct    float64 `json:"afterPct"`
	BeforeValue float64 `json:"beforeValue"`
	AfterValue  float64 `json:"afterValue"`
	TargetValue float64 `json:"targetValue"`
	AfterDelta  float64 `json:"afterDelta"`
}

type rebalanceResultDTO struct {
	ID                   string                   `json:"id"`
	ClientID             string                   `json:"clientId"`
	BasketID             string                   `json:"basketId"`
	BasketName           string                   `json:"basketName"`
	Currency             string                   `json:"currency"`
	Status               string                   `json:"status"`
	PlannedLegs          int                      `json:"plannedLegs"`
	FilledLegs           int                      `json:"filledLegs"`
	FailedLegs           int                      `json:"failedLegs"`
	BeforeEquity         float64                  `json:"beforeEquity"`
	AfterEquity          float64                  `json:"afterEquity"`
	BeforeCash           float64                  `json:"beforeCash"`
	AfterCash            float64                  `json:"afterCash"`
	RemainingAbsDriftPct float64                  `json:"remainingAbsDriftPct"`
	MaxSleeveDriftPct    float64                  `json:"maxSleeveDriftPct"`
	RemainingNotional    float64                  `json:"remainingNotional"`
	TotalFee             float64                  `json:"totalFee"`
	TotalSlippage        float64                  `json:"totalSlippage"`
	TotalCost            float64                  `json:"totalCost"`
	Summary              string                   `json:"summary"`
	Note                 string                   `json:"note"`
	CreatedAt            string                   `json:"createdAt"`
	UpdatedAt            string                   `json:"updatedAt,omitempty"`
	AttemptCount         int                      `json:"attemptCount"`
	CanContinue          bool                     `json:"canContinue"`
	Legs                 []rebalanceLegOutcomeDTO `json:"legs,omitempty"`
	Lines                []rebalanceLineDTO       `json:"lines,omitempty"`
	Attempts             []rebalanceAttemptDTO    `json:"attempts,omitempty"`
}

type rebalanceAttemptDTO struct {
	Seq           int                      `json:"seq"`
	Status        string                   `json:"status"`
	PlannedLegs   int                      `json:"plannedLegs"`
	FilledLegs    int                      `json:"filledLegs"`
	FailedLegs    int                      `json:"failedLegs"`
	TotalFee      float64                  `json:"totalFee"`
	TotalSlippage float64                  `json:"totalSlippage"`
	TotalCost     float64                  `json:"totalCost"`
	Summary       string                   `json:"summary"`
	CreatedAt     string                   `json:"createdAt"`
	Legs          []rebalanceLegOutcomeDTO `json:"legs,omitempty"`
	Lines         []rebalanceLineDTO       `json:"lines,omitempty"`
}

func rebalanceResultToDTO(r *domain.RebalanceResult, includeKids bool) rebalanceResultDTO {
	d := rebalanceResultDTO{
		ID: r.ID, ClientID: r.ClientID, BasketID: r.BasketID, BasketName: r.BasketName,
		Currency: r.Currency, Status: r.Status,
		PlannedLegs: r.PlannedLegs, FilledLegs: r.FilledLegs, FailedLegs: r.FailedLegs,
		BeforeEquity: r.BeforeEquity, AfterEquity: r.AfterEquity,
		BeforeCash: r.BeforeCash, AfterCash: r.AfterCash,
		RemainingAbsDriftPct: r.RemainingAbsDriftPct, MaxSleeveDriftPct: r.MaxSleeveDriftPct,
		RemainingNotional: r.RemainingNotional,
		TotalFee: r.TotalFee, TotalSlippage: r.TotalSlippage, TotalCost: r.TotalCost,
		Summary: r.Summary, Note: r.Note,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano),
		AttemptCount: r.AttemptCount, CanContinue: r.CanContinue,
	}
	if !r.UpdatedAt.IsZero() {
		d.UpdatedAt = r.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	if !includeKids {
		return d
	}
	d.Legs = make([]rebalanceLegOutcomeDTO, 0, len(r.Legs))
	for _, l := range r.Legs {
		d.Legs = append(d.Legs, rebalanceLegOutcomeDTO{
			Side: string(l.Side), Asset: l.Asset, Exchange: string(l.Exchange), Symbol: l.Symbol, Reason: l.Reason,
			PlannedQty: l.PlannedQty, PlannedPrice: l.PlannedPrice, PlannedNotional: l.PlannedNotional,
			Status: l.Status, TradeID: l.TradeID, FilledQty: l.FilledQty, FillPrice: l.FillPrice,
			FillNotional: l.FillNotional, LastPrice: l.LastPrice, Fee: l.Fee, Slippage: l.Slippage, Error: l.Error,
			Attempt: l.Attempt,
		})
	}
	d.Lines = make([]rebalanceLineDTO, 0, len(r.Lines))
	for _, ln := range r.Lines {
		d.Lines = append(d.Lines, rebalanceLineDTO{
			Asset: ln.Asset, Exchange: string(ln.Exchange), Symbol: ln.Symbol, IsCash: ln.IsCash,
			TargetPct: ln.TargetPct, BeforePct: ln.BeforePct, AfterPct: ln.AfterPct,
			BeforeValue: ln.BeforeValue, AfterValue: ln.AfterValue, TargetValue: ln.TargetValue,
			AfterDelta: ln.AfterDelta,
		})
	}
	d.Attempts = make([]rebalanceAttemptDTO, 0, len(r.Attempts))
	for _, att := range r.Attempts {
		ad := rebalanceAttemptDTO{
			Seq: att.Seq, Status: att.Status,
			PlannedLegs: att.PlannedLegs, FilledLegs: att.FilledLegs, FailedLegs: att.FailedLegs,
			TotalFee: att.TotalFee, TotalSlippage: att.TotalSlippage, TotalCost: att.TotalCost,
			Summary: att.Summary, CreatedAt: att.CreatedAt.UTC().Format(time.RFC3339Nano),
		}
		for _, l := range att.Legs {
			ad.Legs = append(ad.Legs, rebalanceLegOutcomeDTO{
				Side: string(l.Side), Asset: l.Asset, Exchange: string(l.Exchange), Symbol: l.Symbol, Reason: l.Reason,
				PlannedQty: l.PlannedQty, PlannedPrice: l.PlannedPrice, PlannedNotional: l.PlannedNotional,
				Status: l.Status, TradeID: l.TradeID, FilledQty: l.FilledQty, FillPrice: l.FillPrice,
				FillNotional: l.FillNotional, LastPrice: l.LastPrice, Fee: l.Fee, Slippage: l.Slippage, Error: l.Error,
				Attempt: l.Attempt,
			})
		}
		d.Attempts = append(d.Attempts, ad)
	}
	return d
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
	ClientID    string                `json:"clientId"`
	PortfolioID string                `json:"portfolioId"`
	Name        string                `json:"name"`
	Targets     []allocationTargetDTO `json:"targets"`
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
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	b, err := h.svc.CreateAllocationBasket(r.Context(), portfolio.AllocationBasketCreateInput{
		ClientID: clientID, PortfolioID: coalescePortfolioID(r, body.PortfolioID), Name: body.Name, Targets: parseTargetBodies(body.Targets),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, allocationBasketToDTO(b))
}

// ListBaskets handles GET /api/v1/portfolio/baskets
func (h *PortfolioHandler) ListBaskets(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListAllocationBaskets(r.Context(), clientIDFrom(r), portfolioIDFrom(r))
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
	v, err := h.svc.PreviewAllocationRebalance(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
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
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), BasketID: r.PathValue("id"), Name: body.Name,
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
	if err := h.svc.DeleteAllocationBasket(r.Context(), clientIDFrom(r), id, portfolioIDFrom(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

// PreviewBasketRebalance handles GET /api/v1/portfolio/baskets/{id}/preview
func (h *PortfolioHandler) PreviewBasketRebalance(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.PreviewAllocationRebalance(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, allocationViewDTO(v, nil))
}

// RebalanceBasket handles POST /api/v1/portfolio/baskets/{id}/rebalance
func (h *PortfolioHandler) RebalanceBasket(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ExecuteAllocationRebalance(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rebalanceResultToDTO(res, true))
}

// ListBasketRebalances handles GET /api/v1/portfolio/baskets/{id}/rebalances
func (h *PortfolioHandler) ListBasketRebalances(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.ListRebalanceResults(r.Context(), clientIDFrom(r), r.PathValue("id"), limit, offset, portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]rebalanceResultDTO, 0, len(list))
	for i := range list {
		items = append(items, rebalanceResultToDTO(&list[i], false))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "basketId": r.PathValue("id"),
		"rebalances": items, "count": len(items),
	})
}

// ListRebalances handles GET /api/v1/portfolio/rebalances
func (h *PortfolioHandler) ListRebalances(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.ListRebalanceResults(r.Context(), clientIDFrom(r), r.URL.Query().Get("basketId"), limit, offset, portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]rebalanceResultDTO, 0, len(list))
	for i := range list {
		items = append(items, rebalanceResultToDTO(&list[i], false))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "rebalances": items, "count": len(items),
	})
}

// GetRebalance handles GET /api/v1/portfolio/rebalances/{id}
func (h *PortfolioHandler) GetRebalance(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.GetRebalanceResult(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rebalanceResultToDTO(res, true))
}

// ContinueRebalance handles POST /api/v1/portfolio/rebalances/{id}/continue
func (h *PortfolioHandler) ContinueRebalance(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.ContinueAllocationRebalance(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rebalanceResultToDTO(res, true))
}
