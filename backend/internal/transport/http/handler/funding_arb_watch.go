package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/fundingarb"
)

// FundingArbWatchHandler is the tenant transport for funding-arb follows.
type FundingArbWatchHandler struct {
	svc *fundingarb.Service
}

// NewFundingArbWatchHandler constructs the handler.
func NewFundingArbWatchHandler(svc *fundingarb.Service) *FundingArbWatchHandler {
	return &FundingArbWatchHandler{svc: svc}
}

type fundingArbWatchDTO struct {
	ID            string  `json:"id"`
	ClientID      string  `json:"clientId"`
	Scope         string  `json:"scope"`
	Symbol        string  `json:"symbol"`
	Quote         string  `json:"quote,omitempty"`
	SymbolLimit   int     `json:"symbolLimit,omitempty"`
	Notional      float64 `json:"notional"`
	HoldHours     float64 `json:"holdHours"`
	MinProfit     float64 `json:"minProfit"`
	FeeBinancePct float64 `json:"feeBinancePct"`
	FeeBybitPct   float64 `json:"feeBybitPct"`
	Status        string  `json:"status"`
	Armed         bool    `json:"armed"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type fundingArbSignalDTO struct {
	ID            string  `json:"id"`
	WatchID       string  `json:"watchId"`
	ClientID      string  `json:"clientId"`
	Symbol        string  `json:"symbol"`
	LongExchange  string  `json:"longExchange"`
	ShortExchange string  `json:"shortExchange"`
	NetAfterFees  float64 `json:"netAfterFees"`
	MinProfit     float64 `json:"minProfit"`
	Status        string  `json:"status"`
	OpenedAt      string  `json:"openedAt"`
	LastSeenAt    string  `json:"lastSeenAt"`
	ClosedAt      *string `json:"closedAt,omitempty"`
}

func faWatchDTO(w *domain.FundingArbWatch) fundingArbWatchDTO {
	scope := "symbol"
	if w.IsScan() {
		scope = "scan"
	}
	return fundingArbWatchDTO{
		ID: w.ID, ClientID: w.ClientID, Scope: scope, Symbol: w.Symbol,
		Quote: w.Quote, SymbolLimit: w.SymbolLimit,
		Notional: w.Notional, HoldHours: w.HoldHours, MinProfit: w.MinProfit,
		FeeBinancePct: w.FeeBinancePct, FeeBybitPct: w.FeeBybitPct,
		Status: string(w.Status), Armed: w.Armed,
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: w.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func faSignalDTO(s *domain.FundingArbSignal) fundingArbSignalDTO {
	d := fundingArbSignalDTO{
		ID: s.ID, WatchID: s.WatchID, ClientID: s.ClientID, Symbol: s.Symbol,
		LongExchange: string(s.LongExchange), ShortExchange: string(s.ShortExchange),
		NetAfterFees: s.NetAfterFees, MinProfit: s.MinProfit, Status: string(s.Status),
		OpenedAt:   s.OpenedAt.UTC().Format(time.RFC3339Nano),
		LastSeenAt: s.LastSeenAt.UTC().Format(time.RFC3339Nano),
	}
	if s.ClosedAt != nil {
		t := s.ClosedAt.UTC().Format(time.RFC3339Nano)
		d.ClosedAt = &t
	}
	return d
}

type createFundingArbWatchBody struct {
	ClientID      string   `json:"clientId"`
	Symbol        string   `json:"symbol"`
	Notional      float64  `json:"notional"`
	HoldHours     float64  `json:"holdHours"`
	MinProfit     float64  `json:"minProfit"`
	Quote         string   `json:"quote"`
	SymbolLimit   int      `json:"limit"`
	FeeBinancePct *float64 `json:"feeBinancePct"`
	FeeBybitPct   *float64 `json:"feeBybitPct"`
}

// CreateWatch handles POST /api/v1/funding-arb/watches
func (h *FundingArbWatchHandler) CreateWatch(w http.ResponseWriter, r *http.Request) {
	var body createFundingArbWatchBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid json", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	got, err := h.svc.CreateWatch(r.Context(), fundingarb.CreateInput{
		ClientID: clientID,
		Symbol:   body.Symbol, Notional: body.Notional, HoldHours: body.HoldHours, MinProfit: body.MinProfit,
		Quote: body.Quote, SymbolLimit: body.SymbolLimit,
		FeeBinancePct: body.FeeBinancePct, FeeBybitPct: body.FeeBybitPct,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, faWatchDTO(got))
}

// ListWatches handles GET /api/v1/funding-arb/watches
func (h *FundingArbWatchHandler) ListWatches(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListWatches(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]fundingArbWatchDTO, 0, len(list))
	for i := range list {
		out = append(out, faWatchDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"watches": out})
}

// GetWatch handles GET /api/v1/funding-arb/watches/{id}
func (h *FundingArbWatchHandler) GetWatch(w http.ResponseWriter, r *http.Request) {
	got, err := h.svc.GetWatch(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, faWatchDTO(got))
}

// DeleteWatch handles DELETE /api/v1/funding-arb/watches/{id}
func (h *FundingArbWatchHandler) DeleteWatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteWatch(r.Context(), clientIDFrom(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

// ListSignals handles GET /api/v1/funding-arb/signals
func (h *FundingArbWatchHandler) ListSignals(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 0
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, fmt.Errorf("%w: limit must be an integer", domain.ErrInvalidArgument))
			return
		}
		limit = n
	}
	list, err := h.svc.ListSignals(r.Context(), clientIDFrom(r), q.Get("status"), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]fundingArbSignalDTO, 0, len(list))
	for i := range list {
		out = append(out, faSignalDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"signals": out})
}
