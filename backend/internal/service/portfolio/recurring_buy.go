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
	PortfolioID   string
	OwnerClientID string
	Exchange      string
	Symbol        string
	Name          string  // optional label; default "<symbol> <frequency>"
	Amount        float64 // cash notional per run
	Frequency     string  // daily | weekly | monthly | interval
	Weekday       string  // monday..sunday; weekly
	DayOfMonth    int     // 1-31; monthly salary day
	IntervalHours int     // 1-168; interval frequency
	TimeZone      string  // IANA e.g. Europe/Istanbul; empty = UTC
	Hour          *int    // 0-23 local; nil = inherit startAt/now clock unless TimeZone set
	Minute        *int    // 0-59
	MaxPrice      float64 // 0 = no cap
	Budget        float64 // 0 = no cap
	EndDate       string  // YYYY-MM-DD inclusive last local day
	EndsAt        *time.Time
	// StartAt is the first scheduled run (default: now — first worker tick after create).
	StartAt *time.Time
}

// RecurringBuyUpdateInput patches name / amount / schedule of an existing plan.
// Nil / empty optional fields keep the current value. Frequency set to non-empty replaces schedule extras.
type RecurringBuyUpdateInput struct {
	ClientID      string
	PortfolioID   string
	OwnerClientID string
	PlanID        string
	Name          *string
	Amount        *float64
	Frequency     *string
	Weekday       *string
	DayOfMonth    *int
	IntervalHours *int
	TimeZone      *string
	Hour          *int
	Minute        *int
	MaxPrice      *float64
	Budget        *float64
	EndDate       *string // YYYY-MM-DD; empty string clears the end with EndsAt
	EndsAt        *time.Time
	ClearEnds     bool
	StartAt       *time.Time // if set, next run is recomputed from this instant
}

// CreateRecurringBuyPlan validates and stores an active plan.
func (s *Service) CreateRecurringBuyPlan(ctx context.Context, in RecurringBuyCreateInput) (*domain.RecurringBuyPlan, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	if err := domain.RequireQuoteMatchesCurrency(ex, sym, p.Currency); err != nil {
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
	tz, err := domain.NormalizeRecurringTimeZone(in.TimeZone)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	hour, minute := domain.RecurringHourUnset, 0
	hasLocal := false
	if in.Hour != nil {
		min := 0
		if in.Minute != nil {
			min = *in.Minute
		}
		hour, minute, err = domain.NormalizeRecurringClock(*in.Hour, min)
		if err != nil {
			return nil, err
		}
		hasLocal = true
	} else if in.Minute != nil {
		hour, minute, err = domain.NormalizeRecurringClock(0, *in.Minute)
		if err != nil {
			return nil, err
		}
		hasLocal = true
	}
	if tz != "" {
		hasLocal = true
		if hour == domain.RecurringHourUnset {
			anchor := now
			if in.StartAt != nil && !in.StartAt.IsZero() {
				anchor = in.StartAt.UTC()
			}
			lt := anchor.In(domain.RecurringLocation(tz))
			hour, minute = lt.Hour(), lt.Minute()
		}
	}
	if hour == domain.RecurringHourUnset {
		hour, minute = 0, 0
	}
	maxP, err := domain.ResolveRecurringMaxPrice(in.MaxPrice)
	if err != nil {
		return nil, err
	}
	budget, err := domain.ResolveRecurringBudget(in.Budget)
	if err != nil {
		return nil, err
	}
	endsAt, err := domain.ResolveRecurringEndsAt(in.EndsAt, in.EndDate, tz)
	if err != nil {
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
	draft := domain.RecurringBuyPlan{
		Frequency: freq, Weekday: weekday, DayOfMonth: in.DayOfMonth, IntervalHours: in.IntervalHours,
		TimeZone: tz, HasLocalTime: hasLocal, Hour: hour, Minute: minute,
	}
	next := domain.FirstRecurringRunAt(now, in.StartAt, draft)
	status := domain.RecurringBuyActive
	if endsAt != nil && !domain.RecurringPlanAllowsRun(domain.RecurringBuyPlan{EndsAt: endsAt}, next) {
		status = domain.RecurringBuyEnded
	}
	plan := domain.RecurringBuyPlan{
		ID: uuid.NewString(), ClientID: clientID, Exchange: ex, Symbol: sym, Name: name,
		Amount: in.Amount, Frequency: freq, Weekday: weekday, DayOfMonth: in.DayOfMonth, IntervalHours: in.IntervalHours,
		TimeZone: tz, HasLocalTime: hasLocal, Hour: hour, Minute: minute, MaxPrice: maxP,
		Budget: budget, EndsAt: endsAt,
		Status: status, NextRunAt: next, CreatedAt: now, UpdatedAt: now,
	}
	return s.store.CreateRecurringBuyPlan(ctx, plan)
}

// UpdateRecurringBuyPlan changes name, amount, and/or schedule of a plan.
func (s *Service) UpdateRecurringBuyPlan(ctx context.Context, in RecurringBuyUpdateInput) (*domain.RecurringBuyPlan, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireAccessErr(ctx, in.ClientID, domain.PortfolioRoleTrader, in.PortfolioID, in.OwnerClientID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
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
	tz := cur.TimeZone
	hasLocal := cur.HasLocalTime
	hour, minute := cur.Hour, cur.Minute
	if in.TimeZone != nil {
		tz, err = domain.NormalizeRecurringTimeZone(*in.TimeZone)
		if err != nil {
			return nil, err
		}
		if tz != "" && !hasLocal && in.Hour == nil {
			lt := cur.NextRunAt.In(domain.RecurringLocation(tz))
			hour, minute = lt.Hour(), lt.Minute()
		}
		if tz != "" {
			hasLocal = true
		}
	}
	if in.Hour != nil || in.Minute != nil {
		h, m := hour, minute
		if in.Hour != nil {
			h = *in.Hour
		}
		if in.Minute != nil {
			m = *in.Minute
		}
		hour, minute, err = domain.NormalizeRecurringClock(h, m)
		if err != nil {
			return nil, err
		}
		hasLocal = true
	}
	maxP := cur.MaxPrice
	if in.MaxPrice != nil {
		maxP, err = domain.ResolveRecurringMaxPrice(*in.MaxPrice)
		if err != nil {
			return nil, err
		}
	}
	budget := cur.Budget
	if in.Budget != nil {
		budget, err = domain.ResolveRecurringBudget(*in.Budget)
		if err != nil {
			return nil, err
		}
	}
	endsAt := cur.EndsAt
	if in.ClearEnds {
		endsAt = nil
	} else if in.EndDate != nil || in.EndsAt != nil {
		endDate := ""
		if in.EndDate != nil {
			endDate = *in.EndDate
		}
		endsAt, err = domain.ResolveRecurringEndsAt(in.EndsAt, endDate, tz)
		if err != nil {
			return nil, err
		}
	}
	name := cur.Name
	if in.Name != nil {
		name, err = domain.NormalizeRecurringBuyName(*in.Name, cur.Symbol, freq)
		if err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	draft := domain.RecurringBuyPlan{
		Frequency: freq, Weekday: weekday, DayOfMonth: dom, IntervalHours: ih,
		TimeZone: tz, HasLocalTime: hasLocal, Hour: hour, Minute: minute,
	}
	next := cur.NextRunAt
	if in.Frequency != nil || in.Weekday != nil || in.DayOfMonth != nil || in.IntervalHours != nil ||
		in.StartAt != nil || in.TimeZone != nil || in.Hour != nil || in.Minute != nil {
		next = domain.FirstRecurringRunAt(now, in.StartAt, draft)
	}
	updated := *cur
	updated.Name = name
	updated.Amount = amount
	updated.Frequency = freq
	updated.Weekday = weekday
	updated.DayOfMonth = dom
	updated.IntervalHours = ih
	updated.TimeZone = tz
	updated.HasLocalTime = hasLocal
	updated.Hour = hour
	updated.Minute = minute
	updated.MaxPrice = maxP
	updated.Budget = budget
	updated.EndsAt = endsAt
	updated.NextRunAt = next
	updated.UpdatedAt = now
	if updated.Status == domain.RecurringBuyEnded && (endsAt == nil || domain.RecurringPlanAllowsRun(updated, now)) {
		updated.Status = domain.RecurringBuyActive
	}
	if endsAt != nil && !domain.RecurringPlanAllowsRun(updated, next) {
		updated.Status = domain.RecurringBuyEnded
	}
	return s.store.UpdateRecurringBuyPlan(ctx, clientID, id, updated)
}

// ListRecurringBuyPlans lists plans for a client.
func (s *Service) ListRecurringBuyPlans(ctx context.Context, clientID string, portfolioID ...string) ([]domain.RecurringBuyPlan, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	return s.store.ListRecurringBuyPlans(ctx, p.BookID())
}

// GetRecurringBuyPlan returns one plan.
func (s *Service) GetRecurringBuyPlan(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.RecurringBuyPlan, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: plan id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetRecurringBuyPlan(ctx, p.BookID(), id)
}

// PauseRecurringBuyPlan sets status paused.
func (s *Service) PauseRecurringBuyPlan(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.RecurringBuyPlan, error) {
	return s.setRecurringStatus(ctx, clientID, id, domain.RecurringBuyPaused, false, portfolioID...)
}

// ResumeRecurringBuyPlan sets status active; if next_run_at is in the past, sets it to now.
func (s *Service) ResumeRecurringBuyPlan(ctx context.Context, clientID, id string, portfolioID ...string) (*domain.RecurringBuyPlan, error) {
	return s.setRecurringStatus(ctx, clientID, id, domain.RecurringBuyActive, true, portfolioID...)
}

func (s *Service) setRecurringStatus(ctx context.Context, clientID, id string, status domain.RecurringBuyPlanStatus, bumpPastNext bool, portfolioID ...string) (*domain.RecurringBuyPlan, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleTrader, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
	plan, err := s.store.GetRecurringBuyPlan(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if status == domain.RecurringBuyActive && plan.Status == domain.RecurringBuyEnded {
		if plan.EndsAt != nil && now.After(*plan.EndsAt) {
			return nil, fmt.Errorf("%w: plan ended; extend endDate or endsAt first", domain.ErrInvalidArgument)
		}
	}
	next := plan.NextRunAt
	if bumpPastNext && next.Before(now) {
		next = now
	}
	return s.store.UpdateRecurringBuyPlanStatus(ctx, clientID, id, status, next, now)
}

// DeleteRecurringBuyPlan removes a plan and its run history.
func (s *Service) DeleteRecurringBuyPlan(ctx context.Context, clientID, id string, portfolioID ...string) error {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleTrader, portfolioID...)
	if err != nil {
		return err
	}
	clientID = p.BookID()
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: plan id is required", domain.ErrInvalidArgument)
	}
	return s.store.DeleteRecurringBuyPlan(ctx, clientID, id)
}

// ListRecurringBuyRuns lists execution history for a plan.
func (s *Service) ListRecurringBuyRuns(ctx context.Context, clientID, planID string, limit, offset int, portfolioID ...string) ([]domain.RecurringBuyRun, error) {
	p, err := s.requireAccessErr(ctx, clientID, domain.PortfolioRoleViewer, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
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
		if s.ownerClosed(ctx, due[i].ClientID) {
			continue
		}
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

	// Reload after claim so a PATCH (maxPrice, budget, end, pause) wins over the
	// snapshot this worker started with.
	fresh, err := s.store.GetRecurringBuyPlan(ctx, plan.ClientID, plan.ID)
	if err != nil || fresh == nil {
		final := *run
		final.Status = domain.RecurringBuyRunFailed
		final.FailReason = "plan unavailable"
		final.ExecutedAt = now
		return s.store.FinishRecurringBuyRun(ctx, plan.ID, final, nextAfter, periodKey, now)
	}
	final := *run
	final.ExecutedAt = now
	if fresh.Status != domain.RecurringBuyActive {
		final.Status = domain.RecurringBuyRunFailed
		final.FailReason = "plan paused"
		if fresh.Status == domain.RecurringBuyEnded {
			final.FailReason = "plan ended"
			final.PlanStatus = domain.RecurringBuyEnded
		}
		return s.store.FinishRecurringBuyRun(ctx, plan.ID, final, nextAfter, periodKey, now)
	}
	if !domain.RecurringPlanAllowsRun(*fresh, scheduledFor) {
		final.Status = domain.RecurringBuyRunFailed
		final.FailReason = "plan ended"
		final.PlanStatus = domain.RecurringBuyEnded
		return s.store.FinishRecurringBuyRun(ctx, plan.ID, final, nextAfter, periodKey, now)
	}

	tr, failReason := s.executeRecurringCashBuy(ctx, fresh, scheduledFor, now)
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
		final.Amount = fresh.Amount
		if failReason == "plan ended" {
			final.PlanStatus = domain.RecurringBuyEnded
		}
	}
	if fresh.EndsAt != nil && nextAfter.After(*fresh.EndsAt) {
		final.PlanStatus = domain.RecurringBuyEnded
	}
	return s.store.FinishRecurringBuyRun(ctx, plan.ID, final, nextAfter, periodKey, now)
}

// executeRecurringCashBuy spends up to plan.Amount cash at last price (qty = amount/price).
// It reloads the plan after the last print and again immediately before PlaceOrder so a
// concurrent maxPrice / budget / pause / end PATCH cannot fill with stale limits.
func (s *Service) executeRecurringCashBuy(ctx context.Context, plan *domain.RecurringBuyPlan, scheduledFor, now time.Time) (*domain.Trade, string) {
	price, err := s.lastPrice(ctx, string(plan.Exchange), plan.Symbol)
	if err != nil || price <= 0 {
		return nil, "market price unavailable"
	}
	fresh, reason := s.recurringPlanReadyToBuy(ctx, plan.ClientID, plan.ID, scheduledFor, price)
	if reason != "" {
		return nil, reason
	}
	cost := s.paperCost(fresh.Exchange)
	fill := domain.ApplySlippage(price, domain.TradeSideBuy, cost.SlippageRate)
	unit := domain.BuyUnitCost(fill, cost.FeeRate)
	if unit <= 0 {
		return nil, "market price unavailable"
	}
	spend, reason := domain.RecurringSpendAmount(*fresh)
	if reason != "" {
		return nil, reason
	}
	qty := spend / unit
	if qty < domain.MinTradeQuantity {
		return nil, "buy quantity too small for amount"
	}
	// Re-check immediately before the fill so a PATCH during sizing still applies.
	fresh, reason = s.recurringPlanReadyToBuy(ctx, plan.ClientID, plan.ID, scheduledFor, price)
	if reason != "" {
		return nil, reason
	}
	spend, reason = domain.RecurringSpendAmount(*fresh)
	if reason != "" {
		return nil, reason
	}
	qty = spend / unit
	if qty < domain.MinTradeQuantity {
		return nil, "buy quantity too small for amount"
	}
	// Plans store ClientID = book id. PlaceOrder treats ClientID as the owner
	// and requires portfolioId once more than one book exists.
	book, err := s.store.GetPortfolio(ctx, plan.ClientID)
	if err != nil || book == nil {
		return nil, "order failed"
	}
	tr, _, err := s.PlaceOrder(ctx, OrderInput{
		ClientID: book.ClientID, PortfolioID: book.ID,
		Exchange: string(plan.Exchange), Symbol: plan.Symbol,
		Side: "buy", Quantity: qty, MarkPrice: price,
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

func (s *Service) recurringPlanReadyToBuy(ctx context.Context, bookID, planID string, scheduledFor time.Time, last float64) (*domain.RecurringBuyPlan, string) {
	fresh, err := s.store.GetRecurringBuyPlan(ctx, bookID, planID)
	if err != nil || fresh == nil {
		return nil, "plan unavailable"
	}
	if fresh.Status != domain.RecurringBuyActive {
		if fresh.Status == domain.RecurringBuyEnded {
			return nil, "plan ended"
		}
		return nil, "plan paused"
	}
	if !domain.RecurringPlanAllowsRun(*fresh, scheduledFor) {
		return nil, "plan ended"
	}
	cost := s.paperCost(fresh.Exchange)
	if reason := domain.RecurringMaxPriceBlocks(last, cost.SlippageRate, cost.FeeRate, fresh.MaxPrice); reason != "" {
		return nil, reason
	}
	return fresh, ""
}
