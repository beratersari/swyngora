package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/scanner"
)

// ScannerHandler is the HTTP adapter for the indicator scanner.
type ScannerHandler struct {
	svc *scanner.Service
}

// NewScannerHandler constructs the handler.
func NewScannerHandler(svc *scanner.Service) *ScannerHandler {
	return &ScannerHandler{svc: svc}
}

type scannerRuleDTO struct {
	ID             string   `json:"id"`
	ClientID       string   `json:"clientId"`
	Type           string   `json:"type"`
	Conditions     []string `json:"conditions"`
	MatchMode      string   `json:"matchMode"`
	Interval       string   `json:"interval"`
	Enabled        bool     `json:"enabled"`
	RSIPeriod      int      `json:"rsiPeriod,omitempty"`
	RSICondition   string   `json:"rsiCondition,omitempty"`
	RSIThreshold   float64  `json:"rsiThreshold,omitempty"`
	MAFastPeriod   int      `json:"maFastPeriod,omitempty"`
	MASlowPeriod   int      `json:"maSlowPeriod,omitempty"`
	MADirection    string   `json:"maDirection,omitempty"`
	VolumeLookback int      `json:"volumeLookback,omitempty"`
	VolumeMinRatio float64  `json:"volumeMinRatio,omitempty"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

type scannerResultDTO struct {
	ID            string             `json:"id"`
	ClientID      string             `json:"clientId"`
	RuleID        string             `json:"ruleId"`
	Exchange      string             `json:"exchange"`
	Symbol        string             `json:"symbol"`
	RuleType      string             `json:"ruleType"`
	Interval      string             `json:"interval"`
	MarketDataKey string             `json:"marketDataKey"`
	MatchedAt     string             `json:"matchedAt"`
	Summary       string             `json:"summary"`
	Metrics       map[string]float64 `json:"metrics,omitempty"`
}

func ruleToDTO(r *domain.ScannerRule) scannerRuleDTO {
	conds := r.SelectedConditions()
	names := make([]string, 0, len(conds))
	for _, c := range conds {
		names = append(names, string(c))
	}
	mode := string(r.MatchMode)
	if mode == "" {
		mode = string(domain.ScannerMatchAll)
	}
	d := scannerRuleDTO{
		ID: r.ID, ClientID: r.ClientID, Type: string(r.Type),
		Conditions: names, MatchMode: mode,
		Interval: r.Interval, Enabled: r.Enabled,
		CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	for _, c := range conds {
		switch c {
		case domain.ScannerRuleRSI:
			d.RSIPeriod, d.RSICondition, d.RSIThreshold = r.RSIPeriod, string(r.RSICondition), r.RSIThreshold
		case domain.ScannerRuleMACrossover:
			d.MAFastPeriod, d.MASlowPeriod, d.MADirection = r.MAFastPeriod, r.MASlowPeriod, r.MADirection
		case domain.ScannerRuleVolumeIncrease:
			d.VolumeLookback, d.VolumeMinRatio = r.VolumeLookback, r.VolumeMinRatio
		}
	}
	return d
}

func resultToDTO(r *domain.ScannerResult) scannerResultDTO {
	return scannerResultDTO{
		ID: r.ID, ClientID: r.ClientID, RuleID: r.RuleID, Exchange: string(r.Exchange), Symbol: r.Symbol,
		RuleType: string(r.RuleType), Interval: r.Interval, MarketDataKey: r.MarketDataKey,
		MatchedAt: r.MatchedAt.UTC().Format(time.RFC3339Nano), Summary: r.Summary, Metrics: r.Metrics,
	}
}

type createScannerRuleBody struct {
	ClientID       string   `json:"clientId"`
	Type           string   `json:"type"`
	Conditions     []string `json:"conditions"`
	MatchMode      string   `json:"matchMode"`
	Interval       string   `json:"interval"`
	RSIPeriod      int      `json:"rsiPeriod"`
	RSICondition   string   `json:"rsiCondition"`
	RSIThreshold   float64  `json:"rsiThreshold"`
	MAFastPeriod   int      `json:"maFastPeriod"`
	MASlowPeriod   int      `json:"maSlowPeriod"`
	MADirection    string   `json:"maDirection"`
	VolumeLookback int      `json:"volumeLookback"`
	VolumeMinRatio float64  `json:"volumeMinRatio"`
}

// Create handles POST /api/v1/scanner/rules
func (h *ScannerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createScannerRuleBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	rule, err := h.svc.Create(r.Context(), scanner.CreateInput{
		ClientID: clientID, Type: body.Type, Conditions: body.Conditions, MatchMode: body.MatchMode, Interval: body.Interval,
		RSIPeriod: body.RSIPeriod, RSICondition: body.RSICondition, RSIThreshold: body.RSIThreshold,
		MAFastPeriod: body.MAFastPeriod, MASlowPeriod: body.MASlowPeriod, MADirection: body.MADirection,
		VolumeLookback: body.VolumeLookback, VolumeMinRatio: body.VolumeMinRatio,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ruleToDTO(rule))
}

type updateScannerRuleBody struct {
	ClientID       string   `json:"clientId"`
	Enabled        *bool    `json:"enabled"`
	Interval       *string  `json:"interval"`
	Conditions     []string `json:"conditions"`
	MatchMode      *string  `json:"matchMode"`
	RSIPeriod      *int     `json:"rsiPeriod"`
	RSICondition   *string  `json:"rsiCondition"`
	RSIThreshold   *float64 `json:"rsiThreshold"`
	MAFastPeriod   *int     `json:"maFastPeriod"`
	MASlowPeriod   *int     `json:"maSlowPeriod"`
	MADirection    *string  `json:"maDirection"`
	VolumeLookback *int     `json:"volumeLookback"`
	VolumeMinRatio *float64 `json:"volumeMinRatio"`
}

// Update handles PATCH /api/v1/scanner/rules/{id}
func (h *ScannerHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body updateScannerRuleBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	rule, err := h.svc.Update(r.Context(), scanner.UpdateInput{
		ClientID: clientID, ID: strings.TrimSpace(r.PathValue("id")),
		Enabled: body.Enabled, Interval: body.Interval, Conditions: body.Conditions, MatchMode: body.MatchMode,
		RSIPeriod: body.RSIPeriod, RSICondition: body.RSICondition, RSIThreshold: body.RSIThreshold,
		MAFastPeriod: body.MAFastPeriod, MASlowPeriod: body.MASlowPeriod, MADirection: body.MADirection,
		VolumeLookback: body.VolumeLookback, VolumeMinRatio: body.VolumeMinRatio,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ruleToDTO(rule))
}

// ListRules handles GET /api/v1/scanner/rules
func (h *ScannerHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), clientIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]scannerRuleDTO, 0, len(list))
	for i := range list {
		items = append(items, ruleToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "rules": items, "count": len(items),
	})
}

// GetRule handles GET /api/v1/scanner/rules/{id}
func (h *ScannerHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	rule, err := h.svc.Get(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ruleToDTO(rule))
}

// DeleteRule handles DELETE /api/v1/scanner/rules/{id}
func (h *ScannerHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if err := h.svc.Delete(r.Context(), clientIDFrom(r), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

// ListResults handles GET /api/v1/scanner/results
func (h *ScannerHandler) ListResults(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.svc.ListResults(r.Context(), clientIDFrom(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]scannerResultDTO, 0, len(list))
	for i := range list {
		items = append(items, resultToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "results": items, "count": len(items), "total": total,
		"limit": limit, "offset": offset,
	})
}

type scannerBacktestDTO struct {
	ID            string  `json:"id"`
	ClientID      string  `json:"clientId"`
	RuleID        string  `json:"ruleId"`
	Exchange      string  `json:"exchange"`
	Symbol        string  `json:"symbol"`
	Interval      string  `json:"interval"`
	RangeStart    string  `json:"rangeStart"`
	RangeEnd      string  `json:"rangeEnd"`
	Status        string  `json:"status"`
	ProgressPct   float64 `json:"progressPct"`
	ProcessedBars int     `json:"processedBars"`
	TotalBars     int     `json:"totalBars"`
	SignalCount   int     `json:"signalCount"`
	ErrorMessage  string  `json:"errorMessage,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	StartedAt     *string `json:"startedAt,omitempty"`
	FinishedAt    *string `json:"finishedAt,omitempty"`
}

type scannerBacktestSignalDTO struct {
	ID         string             `json:"id"`
	BacktestID string             `json:"backtestId"`
	SignalAt   string             `json:"signalAt"`
	ClosePrice float64            `json:"closePrice"`
	Summary    string             `json:"summary"`
	Return1d   *float64           `json:"return1d,omitempty"`
	Return5d   *float64           `json:"return5d,omitempty"`
	Return20d  *float64           `json:"return20d,omitempty"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

func backtestToDTO(b *domain.ScannerBacktest) scannerBacktestDTO {
	d := scannerBacktestDTO{
		ID: b.ID, ClientID: b.ClientID, RuleID: b.RuleID, Exchange: string(b.Exchange), Symbol: b.Symbol,
		Interval: b.Interval, RangeStart: b.RangeStart.UTC().Format(time.RFC3339Nano),
		RangeEnd: b.RangeEnd.UTC().Format(time.RFC3339Nano), Status: string(b.Status),
		ProgressPct: b.ProgressPct, ProcessedBars: b.ProcessedBars, TotalBars: b.TotalBars,
		SignalCount: b.SignalCount, ErrorMessage: b.ErrorMessage,
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if b.StartedAt != nil {
		s := b.StartedAt.UTC().Format(time.RFC3339Nano)
		d.StartedAt = &s
	}
	if b.FinishedAt != nil {
		s := b.FinishedAt.UTC().Format(time.RFC3339Nano)
		d.FinishedAt = &s
	}
	return d
}

func backtestSignalToDTO(s *domain.ScannerBacktestSignal) scannerBacktestSignalDTO {
	return scannerBacktestSignalDTO{
		ID: s.ID, BacktestID: s.BacktestID, SignalAt: s.SignalAt.UTC().Format(time.RFC3339Nano),
		ClosePrice: s.ClosePrice, Summary: s.Summary, Return1d: s.Return1d, Return5d: s.Return5d,
		Return20d: s.Return20d, Metrics: s.Metrics,
	}
}

type startBacktestBody struct {
	ClientID   string `json:"clientId"`
	RuleID     string `json:"ruleId"`
	Exchange   string `json:"exchange"`
	Symbol     string `json:"symbol"`
	RangeStart string `json:"rangeStart"`
	RangeEnd   string `json:"rangeEnd"`
}

// StartBacktest handles POST /api/v1/scanner/backtests
func (h *ScannerHandler) StartBacktest(w http.ResponseWriter, r *http.Request) {
	var body startBacktestBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	start, err := parseRFC3339(body.RangeStart)
	if err != nil {
		writeError(w, fmt.Errorf("%w: rangeStart must be RFC3339", domain.ErrInvalidArgument))
		return
	}
	end, err := parseRFC3339(body.RangeEnd)
	if err != nil {
		writeError(w, fmt.Errorf("%w: rangeEnd must be RFC3339", domain.ErrInvalidArgument))
		return
	}
	job, err := h.svc.StartBacktest(r.Context(), scanner.StartBacktestInput{
		ClientID: clientID, RuleID: body.RuleID, Exchange: body.Exchange, Symbol: body.Symbol,
		RangeStart: start, RangeEnd: end,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, backtestToDTO(job))
}

func parseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	return t.UTC(), err
}

// ListBacktests handles GET /api/v1/scanner/backtests
func (h *ScannerHandler) ListBacktests(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.svc.ListBacktests(r.Context(), clientIDFrom(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]scannerBacktestDTO, 0, len(list))
	for i := range list {
		items = append(items, backtestToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "backtests": items, "count": len(items), "total": total,
	})
}

// GetBacktest handles GET /api/v1/scanner/backtests/{id}
func (h *ScannerHandler) GetBacktest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	job, err := h.svc.GetBacktest(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, backtestToDTO(job))
}

// CancelBacktest handles POST /api/v1/scanner/backtests/{id}/cancel
func (h *ScannerHandler) CancelBacktest(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	job, err := h.svc.CancelBacktest(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, backtestToDTO(job))
}

// ListBacktestSignals handles GET /api/v1/scanner/backtests/{id}/signals
func (h *ScannerHandler) ListBacktestSignals(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.svc.ListBacktestSignals(r.Context(), clientIDFrom(r), id, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]scannerBacktestSignalDTO, 0, len(list))
	for i := range list {
		items = append(items, backtestSignalToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"backtestId": id, "signals": items, "count": len(items), "total": total,
		"signalCount": total,
	})
}
