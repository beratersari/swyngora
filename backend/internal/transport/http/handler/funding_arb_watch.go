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
	ExpiresAt     *string `json:"expiresAt,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

type fundingArbSignalDTO struct {
	ID            string                 `json:"id"`
	WatchID       string                 `json:"watchId"`
	ClientID      string                 `json:"clientId"`
	Symbol        string                 `json:"symbol"`
	LongExchange  string                 `json:"longExchange"`
	ShortExchange string                 `json:"shortExchange"`
	NetAfterFees  float64                `json:"netAfterFees"`
	MinProfit     float64                `json:"minProfit"`
	Status        string                 `json:"status"`
	OpenedAt      string                 `json:"openedAt"`
	LastSeenAt    string                 `json:"lastSeenAt"`
	ClosedAt      *string                `json:"closedAt,omitempty"`
	Expected      *fundingArbExpectedDTO `json:"expected,omitempty"`
	Review        *fundingArbReviewDTO   `json:"review,omitempty"`
}

type fundingArbExpectedDTO struct {
	Notional       float64                      `json:"notional"`
	HoldHours      float64                      `json:"holdHours"`
	HorizonFunding float64                      `json:"horizonFunding"`
	RoundTripFee   float64                      `json:"roundTripFee"`
	ExpectedNet    float64                      `json:"expectedNet"`
	Payments       []fundingArbReviewPaymentDTO `json:"payments"`
}

type fundingArbReviewPaymentDTO struct {
	Time            string `json:"time"`
	Exchange        string `json:"exchange"`
	ExpectedRate    string `json:"expectedRate,omitempty"`
	ExpectedRatePct string `json:"expectedRatePct,omitempty"`
	ActualRate      string `json:"actualRate,omitempty"`
	ActualRatePct   string `json:"actualRatePct,omitempty"`
	ExpectedAmount  string `json:"expectedAmount,omitempty"`
	ActualAmount    string `json:"actualAmount,omitempty"`
	Status          string `json:"status"`
}

type fundingArbReviewDTO struct {
	AsOf            string                       `json:"asOf"`
	PendingCount    int                          `json:"pendingCount"`
	SettledCount    int                          `json:"settledCount"`
	MissingCount    int                          `json:"missingCount"`
	Complete        bool                         `json:"complete"`
	ExpectedFunding float64                      `json:"expectedFunding"`
	ActualFunding   float64                      `json:"actualFunding"`
	ExpectedNet     float64                      `json:"expectedNet"`
	ActualNet       float64                      `json:"actualNet"`
	DeltaNet        float64                      `json:"deltaNet"`
	StillWorthIt    bool                         `json:"stillWorthIt"`
	Payments        []fundingArbReviewPaymentDTO `json:"payments"`
	Summary         string                       `json:"summary"`
}

func faWatchDTO(w *domain.FundingArbWatch) fundingArbWatchDTO {
	scope := "symbol"
	if w.IsScan() {
		scope = "scan"
	}
	d := fundingArbWatchDTO{
		ID: w.ID, ClientID: w.ClientID, Scope: scope, Symbol: w.Symbol,
		Quote: w.Quote, SymbolLimit: w.SymbolLimit,
		Notional: w.Notional, HoldHours: w.HoldHours, MinProfit: w.MinProfit,
		FeeBinancePct: w.FeeBinancePct, FeeBybitPct: w.FeeBybitPct,
		Status: string(w.Status), Armed: w.Armed,
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: w.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if w.ExpiresAt != nil {
		t := w.ExpiresAt.UTC().Format(time.RFC3339Nano)
		d.ExpiresAt = &t
	}
	return d
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
	if s.Expected != nil {
		d.Expected = faExpectedDTO(s.Expected)
	}
	if s.Review != nil {
		d.Review = faReviewDTO(s.Review)
	}
	return d
}

func faExpectedDTO(e *domain.FundingArbExpected) *fundingArbExpectedDTO {
	if e == nil {
		return nil
	}
	pays := make([]fundingArbReviewPaymentDTO, 0, len(e.Payments))
	for _, p := range e.Payments {
		dec, pct := domain.FormatFundingRate(p.Rate)
		pays = append(pays, fundingArbReviewPaymentDTO{
			Time: p.Time.UTC().Format(time.RFC3339Nano), Exchange: string(p.Exchange),
			ExpectedRate: dec, ExpectedRatePct: pct, ExpectedAmount: domain.FormatSignedQty(p.Amount),
			Status: "expected",
		})
	}
	return &fundingArbExpectedDTO{
		Notional: e.Notional, HoldHours: e.HoldHours,
		HorizonFunding: e.HorizonFunding, RoundTripFee: e.RoundTripFee,
		ExpectedNet: e.ExpectedNet, Payments: pays,
	}
}

func faReviewDTO(r *domain.FundingArbReview) *fundingArbReviewDTO {
	if r == nil {
		return nil
	}
	pays := make([]fundingArbReviewPaymentDTO, 0, len(r.Payments))
	for _, p := range r.Payments {
		ed, ep := domain.FormatFundingRate(p.ExpectedRate)
		row := fundingArbReviewPaymentDTO{
			Time: p.Time.UTC().Format(time.RFC3339Nano), Exchange: string(p.Exchange),
			ExpectedRate: ed, ExpectedRatePct: ep,
			ExpectedAmount: domain.FormatSignedQty(p.ExpectedAmount),
			Status:         p.Status,
		}
		if p.Status == domain.FundingArbReviewSettled {
			ad, ap := domain.FormatFundingRate(p.ActualRate)
			row.ActualRate, row.ActualRatePct = ad, ap
			row.ActualAmount = domain.FormatSignedQty(p.ActualAmount)
		}
		pays = append(pays, row)
	}
	return &fundingArbReviewDTO{
		AsOf:         r.AsOf.UTC().Format(time.RFC3339Nano),
		PendingCount: r.PendingCount, SettledCount: r.SettledCount, MissingCount: r.MissingCount,
		Complete: r.Complete, ExpectedFunding: r.ExpectedFunding, ActualFunding: r.ActualFunding,
		ExpectedNet: r.ExpectedNet, ActualNet: r.ActualNet, DeltaNet: r.DeltaNet,
		StillWorthIt: r.StillWorthIt, Payments: pays, Summary: r.Summary,
	}
}

type createFundingArbWatchBody struct {
	ClientID      string   `json:"clientId"`
	Symbol        string   `json:"symbol"`
	Notional      float64  `json:"notional"`
	HoldHours     float64  `json:"holdHours"`
	MinProfit     float64  `json:"minProfit"`
	Quote         string   `json:"quote"`
	SymbolLimit   int      `json:"limit"`
	DurationHours float64  `json:"durationHours"`
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
		Quote: body.Quote, SymbolLimit: body.SymbolLimit, DurationHours: body.DurationHours,
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

type updateFundingArbWatchBody struct {
	ClientID      string   `json:"clientId"`
	Notional      *float64 `json:"notional"`
	HoldHours     *float64 `json:"holdHours"`
	MinProfit     *float64 `json:"minProfit"`
	Quote         *string  `json:"quote"`
	SymbolLimit   *int     `json:"limit"`
	DurationHours *float64 `json:"durationHours"`
	FeeBinancePct *float64 `json:"feeBinancePct"`
	FeeBybitPct   *float64 `json:"feeBybitPct"`
}

// UpdateWatch handles PATCH /api/v1/funding-arb/watches/{id}
func (h *FundingArbWatchHandler) UpdateWatch(w http.ResponseWriter, r *http.Request) {
	var body updateFundingArbWatchBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid json", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	got, err := h.svc.UpdateWatch(r.Context(), fundingarb.UpdateInput{
		ClientID: clientID, ID: r.PathValue("id"),
		Notional: body.Notional, HoldHours: body.HoldHours, MinProfit: body.MinProfit,
		Quote: body.Quote, SymbolLimit: body.SymbolLimit, DurationHours: body.DurationHours,
		FeeBinancePct: body.FeeBinancePct, FeeBybitPct: body.FeeBybitPct,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, faWatchDTO(got))
}

// PauseWatch handles POST /api/v1/funding-arb/watches/{id}/pause
func (h *FundingArbWatchHandler) PauseWatch(w http.ResponseWriter, r *http.Request) {
	got, err := h.svc.PauseWatch(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, faWatchDTO(got))
}

// ResumeWatch handles POST /api/v1/funding-arb/watches/{id}/resume
func (h *FundingArbWatchHandler) ResumeWatch(w http.ResponseWriter, r *http.Request) {
	got, err := h.svc.ResumeWatch(r.Context(), clientIDFrom(r), r.PathValue("id"))
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

// GetSignal handles GET /api/v1/funding-arb/signals/{id}
func (h *FundingArbWatchHandler) GetSignal(w http.ResponseWriter, r *http.Request) {
	got, err := h.svc.GetSignal(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, faSignalDTO(got))
}
