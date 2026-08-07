package portfolio

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// RecurringBuyCreateInput creates a paper recurring buy plan.
type RecurringBuyCreateInput struct {
	ClientID      string
	Exchange      string
	Symbol        string
	Name          string // optional label; default "<symbol> <frequency>"
	Amount        float64 // cash notional per run
	Frequency     string  // daily | weekly | monthly | interval
	Weekday       string  // monday..sunday; weekly
	DayOfMonth    int     // 1-31; monthly salary day
	IntervalHours int     // 1-168; interval frequency
	// StartAt is the first scheduled run (default: now — first worker tick after create).
	StartAt *time.Time
}

// RecurringBuyUpdateInput patches name / amount / schedule of an existing plan.
// Nil / empty optional fields keep the current value. Frequency set to non-empty replaces schedule extras.
type RecurringBuyUpdateInput struct {
	ClientID      string
	PlanID        string
	Name          *string
	Amount        *float64
	Frequency     *string
	Weekday       *string
	DayOfMonth    *int
	IntervalHours *int
	StartAt       *time.Time // if set, next run is recomputed from this instant
}

// CreateRecurringBuyPlan validates and stores an active plan.
func (s *Service) CreateRecurringBuyPlan(ctx context.Context, in RecurringBuyCreateInput) (*domain.RecurringBuyPlan, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.GetPortfolio(ctx, clientID); err != nil {
		return nil, err
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	if in.Amount < domain.MinRecurringBuyAmount || in.Amount > domain.MaxRecurringBuyAmount ||
		math.IsNaN(in.Amount) || math.IsInf(in.Amount, 0) {
		return nil, fmt.Errorf("%w: amount must be between %g and %g", domain.ErrInvalidArgument,
			domain.MinRecurringBuyAmount, domain.MaxRecurringBuyAmount)
	}
	freq, err := domain.NormalizeRecurringBuyFrequency(in.Frequency)
	if err != nil {
		return nil, err
	}
	weekday := strings.ToLower(strings.TrimSpace(in.Weekday))
	if err := domain.ValidateRecurringSchedule(freq, weekday, in.DayOfMonth, in.IntervalHours); err != nil {
		return nil, err
	}
	name, err := domain.NormalizeRecurringBuyName(in.Name, sym, freq)
	if err != nil {
		return nil, err
	}
	n, err := s.store.CountRecurringBuyPlans(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxRecurringBuyPlansPerClient {
		return nil, fmt.Errorf("%w: max %d recurring buy plans per client", domain.ErrInvalidArgument, domain.MaxRecurringBuyPlansPerClient)
	}
	now := time.Now().UTC()
	draft := domain.RecurringBuyPlan{
		Frequency: freq, Weekday: weekday, DayOfMonth: in.DayOfMonth, IntervalHours: in.IntervalHours,
	}
	next := domain.FirstRecurringRunAt(now, in.StartAt, draft)
	plan := domain.RecurringBuyPlan{
		ID: uuid.NewString(), ClientID: clientID, Exchange: ex, Symbol: sym, Name: name,
		Amount: in.Amount, Frequency: freq, Weekday: weekday, DayOfMonth: in.DayOfMonth, IntervalHours: in.IntervalHours,
		Status: domain.RecurringBuyActive, NextRunAt: next, CreatedAt: now, UpdatedAt: now,
	}
	return s.store.CreateRecurringBuyPlan(ctx, plan)
}

// UpdateRecurringBuyPlan changes name, amount, and/or schedule of a plan.
func (s *Service) UpdateRecurringBuyPlan(ctx context.Context, in RecurringBuyUpdateInput) (*domain.RecurringBuyPlan, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(in.PlanID)
	if id == "" {
		return nil, fmt.Errorf("%w: plan id is required", domain.ErrInvalidArgument)
	}
	cur, err := s.store.GetRecurringBuyPlan(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	amount := cur.Amount
	if in.Amount != nil {
		if *in.Amount < domain.MinRecurringBuyAmount || *in.Amount > domain.MaxRecurringBuyAmount ||
			math.IsNaN(*in.Amount) || math.IsInf(*in.Amount, 0) {
			return nil, fmt.Errorf("%w: amount must be between %g and %g", domain.ErrInvalidArgument,
				domain.MinRecurringBuyAmount, domain.MaxRecurringBuyAmount)
		}
		amount = *in.Amount
	}
	freq := cur.Frequency
	weekday := cur.Weekday
	dom := cur.DayOfMonth
	ih := cur.IntervalHours
	if in.Frequency != nil {
		freq, err = domain.NormalizeRecurringBuyFrequency(*in.Frequency)
		if err != nil {
			return nil, err
		}
	}
	if in.Weekday != nil {
		weekday = strings.ToLower(strings.TrimSpace(*in.Weekday))
	}
	if in.DayOfMonth != nil {
		dom = *in.DayOfMonth
	}
	if in.IntervalHours != nil {
		ih = *in.IntervalHours
	}
	if err := domain.ValidateRecurringSchedule(freq, weekday, dom, ih); err != nil {
		return nil, err
	}
	name := cur.Name
	if in.Name != nil {
		name, err = domain.NormalizeRecurringBuyName(*in.Name, cur.Symbol, freq)
		if err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	draft := domain.RecurringBuyPlan{Frequency: freq, Weekday: weekday, DayOfMonth: dom, IntervalHours: ih}
	next := cur.NextRunAt
	if in.Frequency != nil || in.Weekday != nil || in.DayOfMonth != nil || in.IntervalHours != nil || in.StartAt != nil {
		next = domain.FirstRecurringRunAt(now, in.StartAt, draft)
	}
	updated := *cur
	updated.Name = name
	updated.Amount = amount
	updated.Frequency = freq
	updated.Weekday = weekday
	updated.DayOfMonth = dom
	updated.IntervalHours = ih
	updated.NextRunAt = next
	updated.UpdatedAt = now
	return s.store.UpdateRecurringBuyPlan(ctx, clientID, id, updated)
}

// ListRecurringBuyPlans lists plans for a client.
func (s *Service) ListRecurringBuyPlans(ctx context.Context, clientID string) ([]domain.RecurringBuyPlan, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListRecurringBuyPlans(ctx, clientID)
}

// GetRecurringBuyPlan returns one plan.
func (s *Service) GetRecurringBuyPlan(ctx context.Context, clientID, id string) (*domain.RecurringBuyPlan, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: plan id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetRecurringBuyPlan(ctx, clientID, id)
}

// PauseRecurringBuyPlan sets status paused.
func (s *Service) PauseRecurringBuyPlan(ctx context.Context, clientID, id string) (*domain.RecurringBuyPlan, error) {
	return s.setRecurringStatus(ctx, clientID, id, domain.RecurringBuyPaused, false)
}

// ResumeRecurringBuyPlan sets status active; if next_run_at is in the past, sets it to now.
func (s *Service) ResumeRecurringBuyPlan(ctx context.Context, clientID, id string) (*domain.RecurringBuyPlan, error) {
	return s.setRecurringStatus(ctx, clientID, id, domain.RecurringBuyActive, true)
}

func (s *Service) setRecurringStatus(ctx context.Context, clientID, id string, status domain.RecurringBuyPlanStatus, bumpPastNext bool) (*domain.RecurringBuyPlan, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	plan, err := s.store.GetRecurringBuyPlan(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	next := plan.NextRunAt
	if bumpPastNext && next.Before(now) {
		next = now
	}
	return s.store.UpdateRecurringBuyPlanStatus(ctx, clientID, id, status, next, now)
}

// DeleteRecurringBuyPlan removes a plan and its run history.
func (s *Service) DeleteRecurringBuyPlan(ctx context.Context, clientID, id string) error {
	if s.store == nil {
		return fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: plan id is required", domain.ErrInvalidArgument)
	}
	return s.store.DeleteRecurringBuyPlan(ctx, clientID, id)
}

// ListRecurringBuyRuns lists execution history for a plan.
func (s *Service) ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int) ([]domain.RecurringBuyRun, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.ListRecurringBuyRuns(ctx, clientID, planID, limit, offset)
}

// ProcessDueRecurringBuys runs due active plans once (worker).
// For each plan: only the latest missed schedule slot is claimed (UNIQUE period_key);
// concurrent workers / restarts cannot double-buy the same period.
func (s *Service) ProcessDueRecurringBuys(ctx context.Context, now time.Time) (int, error) {
	if s.store == nil || s.market == nil {
		return 0, nil
	}
	now = now.UTC()
	due, err := s.store.ListDueRecurringBuyPlans(ctx, now, 100)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range due {
		if err := s.processOneRecurringBuy(ctx, &due[i], now); err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) processOneRecurringBuy(ctx context.Context, plan *domain.RecurringBuyPlan, now time.Time) error {
	if plan.Status != domain.RecurringBuyActive {
		return nil
	}
	// Latest missed slot only (skip intermediate catch-up buys).
	scheduledFor := domain.LatestDueRunAtPlan(plan.NextRunAt, now, *plan)
	if scheduledFor.After(now) {
		return nil
	}
	periodKey := domain.RecurringPeriodKeyPlan(scheduledFor, *plan)
	nextAfter := domain.AdvanceRecurringSchedule(scheduledFor, *plan)

	runID := uuid.NewString()
	claim := domain.RecurringBuyRun{
		ID: runID, PlanID: plan.ID, ClientID: plan.ClientID, PeriodKey: periodKey,
		Status: domain.RecurringBuyRunFailed, FailReason: "in_progress",
		Amount: plan.Amount, ScheduledFor: scheduledFor, ExecutedAt: now,
	}
	claimed, run, err := s.store.ClaimRecurringBuyRun(ctx, claim)
	if err != nil {
		return err
	}
	if !claimed {
		// Another worker already handled this period — still advance schedule past it.
		return s.store.AdvanceRecurringBuyPlan(ctx, plan.ID, nextAfter, periodKey, now)
	}

	// Execute market buy for cash amount.
	final := *run
	final.ExecutedAt = now
	tr, failReason := s.executeRecurringCashBuy(ctx, plan, now)
	if tr != nil {
		final.Status = domain.RecurringBuyRunSucceeded
		final.FailReason = ""
		final.TradeID = tr.ID
		final.Price = tr.Price
		final.Quantity = tr.Quantity
		final.Amount = tr.Notional
	} else {
		final.Status = domain.RecurringBuyRunFailed
		final.FailReason = failReason
		final.Amount = plan.Amount
	}
	return s.store.FinishRecurringBuyRun(ctx, plan.ID, final, nextAfter, periodKey, now)
}

// executeRecurringCashBuy spends up to plan.Amount cash at last price (qty = amount/price).
func (s *Service) executeRecurringCashBuy(ctx context.Context, plan *domain.RecurringBuyPlan, now time.Time) (*domain.Trade, string) {
	price, err := s.lastPrice(ctx, string(plan.Exchange), plan.Symbol)
	if err != nil || price <= 0 {
		return nil, "market price unavailable"
	}
	qty := plan.Amount / price
	if qty < domain.MinTradeQuantity {
		return nil, "buy quantity too small for amount"
	}
	tr, _, err := s.PlaceOrder(ctx, OrderInput{
		ClientID: plan.ClientID, Exchange: string(plan.Exchange), Symbol: plan.Symbol,
		Side: "buy", Quantity: qty,
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "insufficient cash") {
			return nil, "insufficient cash balance"
		}
		// strip sentinel prefix when present
		const p = "invalid argument: "
		if strings.HasPrefix(msg, p) {
			return nil, msg[len(p):]
		}
		return nil, "order failed"
	}
	_ = now
	return tr, ""
}
