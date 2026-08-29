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

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: endsAt must be RFC3339", domain.ErrInvalidArgument)
	}
	u := t.UTC()
	return &u, nil
}

type recurringBuyPlanDTO struct {
	ID              string   `json:"id"`
	ClientID        string   `json:"clientId"`
	Exchange        string   `json:"exchange"`
	Symbol          string   `json:"symbol"`
	Name            string   `json:"name"`
	Amount          float64  `json:"amount"`
	Frequency       string   `json:"frequency"`
	Weekday         string   `json:"weekday,omitempty"`
	DayOfMonth      int      `json:"dayOfMonth,omitempty"`
	IntervalHours   int      `json:"intervalHours,omitempty"`
	TimeZone        string   `json:"timeZone,omitempty"`
	Hour            *int     `json:"hour,omitempty"`
	Minute          *int     `json:"minute,omitempty"`
	MaxPrice        float64  `json:"maxPrice,omitempty"`
	Budget          float64  `json:"budget,omitempty"`
	Spent           float64  `json:"spent,omitempty"`
	RemainingBudget *float64 `json:"remainingBudget,omitempty"`
	EndsAt          string   `json:"endsAt,omitempty"`
	EndDate         string   `json:"endDate,omitempty"`
	Status          string   `json:"status"`
	NextRunAt       string   `json:"nextRunAt"`
	LastRunAt       *string  `json:"lastRunAt,omitempty"`
	LastPeriodKey   string   `json:"lastPeriodKey,omitempty"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

type recurringBuyRunDTO struct {
	ID           string  `json:"id"`
	PlanID       string  `json:"planId"`
	PeriodKey    string  `json:"periodKey"`
	Status       string  `json:"status"`
	Amount       float64 `json:"amount"`
	Quantity     float64 `json:"quantity,omitempty"`
	Price        float64 `json:"price,omitempty"`
	TradeID      string  `json:"tradeId,omitempty"`
	FailReason   string  `json:"failReason,omitempty"`
	ScheduledFor string  `json:"scheduledFor"`
	ExecutedAt   string  `json:"executedAt"`
}

func recurringPlanDTO(p *domain.RecurringBuyPlan) recurringBuyPlanDTO {
	d := recurringBuyPlanDTO{
		ID: p.ID, ClientID: p.ClientID, Exchange: string(p.Exchange), Symbol: p.Symbol, Name: p.Name,
		Amount: p.Amount, Frequency: string(p.Frequency), Weekday: p.Weekday, DayOfMonth: p.DayOfMonth,
		IntervalHours: p.IntervalHours, TimeZone: p.TimeZone, MaxPrice: p.MaxPrice,
		Budget: p.Budget, Spent: p.Spent,
		Status:    string(p.Status),
		NextRunAt: p.NextRunAt.UTC().Format(time.RFC3339Nano), LastPeriodKey: p.LastPeriodKey,
		CreatedAt: p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if p.Budget > 0 {
		rem := domain.RecurringRemainingBudget(*p)
		d.RemainingBudget = &rem
	}
	if p.EndsAt != nil && !p.EndsAt.IsZero() {
		d.EndsAt = p.EndsAt.UTC().Format(time.RFC3339Nano)
		d.EndDate = domain.RecurringEndDate(*p)
	}
	if p.HasLocalTime || p.TimeZone != "" {
		h, m := p.Hour, p.Minute
		d.Hour, d.Minute = &h, &m
	}
	if p.LastRunAt != nil {
		s := p.LastRunAt.UTC().Format(time.RFC3339Nano)
		d.LastRunAt = &s
	}
	return d
}

func recurringRunDTO(r *domain.RecurringBuyRun) recurringBuyRunDTO {
	return recurringBuyRunDTO{
		ID: r.ID, PlanID: r.PlanID, PeriodKey: r.PeriodKey, Status: string(r.Status),
		Amount: r.Amount, Quantity: r.Quantity, Price: r.Price, TradeID: r.TradeID,
		FailReason:   r.FailReason,
		ScheduledFor: r.ScheduledFor.UTC().Format(time.RFC3339Nano),
		ExecutedAt:   r.ExecutedAt.UTC().Format(time.RFC3339Nano),
	}
}

type createRecurringBody struct {
	ClientID      string  `json:"clientId"`
	PortfolioID   string  `json:"portfolioId"`
	Exchange      string  `json:"exchange"`
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Amount        float64 `json:"amount"`
	Frequency     string  `json:"frequency"`
	Weekday       string  `json:"weekday"`
	DayOfMonth    int     `json:"dayOfMonth"`
	IntervalHours int     `json:"intervalHours"`
	TimeZone      string  `json:"timeZone"`
	Hour          *int    `json:"hour"`
	Minute        *int    `json:"minute"`
	MaxPrice      float64 `json:"maxPrice"`
	Budget        float64 `json:"budget"`
	EndDate       string  `json:"endDate"`
	EndsAt        string  `json:"endsAt"`
	StartAt       string  `json:"startAt"`
}

type updateRecurringBody struct {
	Name          *string  `json:"name"`
	Amount        *float64 `json:"amount"`
	Frequency     *string  `json:"frequency"`
	Weekday       *string  `json:"weekday"`
	DayOfMonth    *int     `json:"dayOfMonth"`
	IntervalHours *int     `json:"intervalHours"`
	TimeZone      *string  `json:"timeZone"`
	Hour          *int     `json:"hour"`
	Minute        *int     `json:"minute"`
	MaxPrice      *float64 `json:"maxPrice"`
	Budget        *float64 `json:"budget"`
	EndDate       *string  `json:"endDate"`
	EndsAt        *string  `json:"endsAt"`
	StartAt       string   `json:"startAt"`
}

// CreateRecurringBuy handles POST /api/v1/portfolio/recurring-buys
func (h *PortfolioHandler) CreateRecurringBuy(w http.ResponseWriter, r *http.Request) {
	var body createRecurringBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	var start *time.Time
	if body.StartAt != "" {
		t, err := time.Parse(time.RFC3339Nano, body.StartAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, body.StartAt)
		}
		if err != nil {
			writeError(w, fmt.Errorf("%w: startAt must be RFC3339", domain.ErrInvalidArgument))
			return
		}
		u := t.UTC()
		start = &u
	}
	endsAt, err := parseOptionalRFC3339(body.EndsAt)
	if err != nil {
		writeError(w, err)
		return
	}
	plan, err := h.svc.CreateRecurringBuyPlan(r.Context(), portfolio.RecurringBuyCreateInput{
		ClientID: clientID, PortfolioID: coalescePortfolioID(r, body.PortfolioID), Exchange: body.Exchange, Symbol: body.Symbol, Name: body.Name,
		Amount: body.Amount, Frequency: body.Frequency, Weekday: body.Weekday,
		DayOfMonth: body.DayOfMonth, IntervalHours: body.IntervalHours,
		TimeZone: body.TimeZone, Hour: body.Hour, Minute: body.Minute, MaxPrice: body.MaxPrice,
		Budget: body.Budget, EndDate: body.EndDate, EndsAt: endsAt,
		StartAt: start,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, recurringPlanDTO(plan))
}

// ListRecurringBuys handles GET /api/v1/portfolio/recurring-buys
func (h *PortfolioHandler) ListRecurringBuys(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListRecurringBuyPlans(r.Context(), clientIDFrom(r), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]recurringBuyPlanDTO, 0, len(list))
	for i := range list {
		items = append(items, recurringPlanDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "plans": items, "count": len(items),
	})
}

// UpdateRecurringBuy handles PATCH /api/v1/portfolio/recurring-buys/{id}
func (h *PortfolioHandler) UpdateRecurringBuy(w http.ResponseWriter, r *http.Request) {
	var body updateRecurringBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	var start *time.Time
	if body.StartAt != "" {
		t, err := time.Parse(time.RFC3339Nano, body.StartAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, body.StartAt)
		}
		if err != nil {
			writeError(w, fmt.Errorf("%w: startAt must be RFC3339", domain.ErrInvalidArgument))
			return
		}
		u := t.UTC()
		start = &u
	}
	in := portfolio.RecurringBuyUpdateInput{
		ClientID: clientIDFrom(r), PortfolioID: portfolioIDFrom(r), PlanID: r.PathValue("id"),
		Name: body.Name, Amount: body.Amount, Frequency: body.Frequency,
		Weekday: body.Weekday, DayOfMonth: body.DayOfMonth, IntervalHours: body.IntervalHours,
		TimeZone: body.TimeZone, Hour: body.Hour, Minute: body.Minute, MaxPrice: body.MaxPrice,
		Budget: body.Budget, EndDate: body.EndDate, StartAt: start,
	}
	if body.EndsAt != nil {
		if strings.TrimSpace(*body.EndsAt) == "" {
			if body.EndDate == nil || strings.TrimSpace(*body.EndDate) == "" {
				in.ClearEnds = true
			}
		} else {
			t, err := parseOptionalRFC3339(*body.EndsAt)
			if err != nil {
				writeError(w, err)
				return
			}
			in.EndsAt = t
		}
	} else if body.EndDate != nil && strings.TrimSpace(*body.EndDate) == "" {
		in.ClearEnds = true
	}
	plan, err := h.svc.UpdateRecurringBuyPlan(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recurringPlanDTO(plan))
}

// GetRecurringBuy handles GET /api/v1/portfolio/recurring-buys/{id}
func (h *PortfolioHandler) GetRecurringBuy(w http.ResponseWriter, r *http.Request) {
	plan, err := h.svc.GetRecurringBuyPlan(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recurringPlanDTO(plan))
}

// PauseRecurringBuy handles POST /api/v1/portfolio/recurring-buys/{id}/pause
func (h *PortfolioHandler) PauseRecurringBuy(w http.ResponseWriter, r *http.Request) {
	plan, err := h.svc.PauseRecurringBuyPlan(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recurringPlanDTO(plan))
}

// ResumeRecurringBuy handles POST /api/v1/portfolio/recurring-buys/{id}/resume
func (h *PortfolioHandler) ResumeRecurringBuy(w http.ResponseWriter, r *http.Request) {
	plan, err := h.svc.ResumeRecurringBuyPlan(r.Context(), clientIDFrom(r), r.PathValue("id"), portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, recurringPlanDTO(plan))
}

// DeleteRecurringBuy handles DELETE /api/v1/portfolio/recurring-buys/{id}
func (h *PortfolioHandler) DeleteRecurringBuy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DeleteRecurringBuyPlan(r.Context(), clientIDFrom(r), id, portfolioIDFrom(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "id": id})
}

// ListRecurringBuyRuns handles GET /api/v1/portfolio/recurring-buys/{id}/runs
func (h *PortfolioHandler) ListRecurringBuyRuns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.ListRecurringBuyRuns(r.Context(), clientIDFrom(r), r.PathValue("id"), limit, offset, portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]recurringBuyRunDTO, 0, len(list))
	for i := range list {
		items = append(items, recurringRunDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"planId": r.PathValue("id"), "runs": items, "count": len(items),
	})
}
