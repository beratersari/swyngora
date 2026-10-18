package portfolio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const emergencyNote = "When today's trading loss reaches the percent you set, open orders are canceled, recurring buys pause, and open spot and margin positions are closed at the paper market. Cash you add or remove is not part of that loss. A step that fails is named and tried again while the loss is still past the line. Turning it off stops further tries and does not undo a close. Paper only. Not real money. Not financial advice."

// EmergencyStopInput is the switch and the loss percent. 5 means 5 percent.
type EmergencyStopInput struct {
	ClientID    string
	PortfolioID string
	Enabled     bool
	LossPct     float64
}

// EmergencyStopView is the settings screen: the switch, today's loss, and the last trip.
type EmergencyStopView struct {
	Enabled     bool
	LossPct     float64
	DailyPnLPct float64
	Hit         bool
	Tripped     bool
	Run         *domain.EmergencyStopRun
	Note        string
}

// GetEmergencyStop loads the switch and the latest trip. It does not close anything.
func (s *Service) GetEmergencyStop(ctx context.Context, clientID string, portfolioID ...string) (*EmergencyStopView, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return nil, err
	}
	return s.emergencyView(ctx, p, false)
}

// SetEmergencyStop saves the switch. Turning it on checks the loss immediately.
func (s *Service) SetEmergencyStop(ctx context.Context, in EmergencyStopInput) (*EmergencyStopView, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireBook(ctx, in.ClientID, in.PortfolioID)
	if err != nil {
		return nil, err
	}
	if in.LossPct != 0 {
		pct := in.LossPct
		if err := domain.ValidateOptionalRiskPct(&pct, "lossPct"); err != nil {
			return nil, err
		}
	}
	if in.Enabled && in.LossPct == 0 {
		return nil, fmt.Errorf("%w: lossPct is required when the emergency stop is on", domain.ErrInvalidArgument)
	}
	bookID := p.BookID()
	row, err := s.store.GetEmergencyStop(ctx, bookID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if row == nil {
		row = &domain.EmergencyStop{ClientID: bookID}
	}
	row.Enabled = in.Enabled
	row.LossPct = in.LossPct
	row.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if !in.Enabled {
		row.Tripped = false
	}
	if err := s.store.UpsertEmergencyStop(ctx, *row); err != nil {
		return nil, err
	}
	if in.Enabled {
		if err := s.evaluateEmergencyStop(ctx, bookID); err != nil {
			return nil, err
		}
	}
	return s.emergencyView(ctx, p, false)
}

// ProcessEmergencyStops checks every armed book. One book failing does not stop the others.
func (s *Service) ProcessEmergencyStops(ctx context.Context) (int, error) {
	if s == nil || s.store == nil {
		return 0, nil
	}
	rows, err := s.store.ListEnabledEmergencyStops(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, row := range rows {
		before := row.Tripped
		if err := s.evaluateEmergencyStop(ctx, row.ClientID); err != nil {
			slog.Default().Error("emergency stop", "book", row.ClientID, "err", err)
			continue
		}
		after, gerr := s.store.GetEmergencyStop(ctx, row.ClientID)
		if gerr == nil && after != nil && after.Tripped && !before {
			n++
		}
	}
	return n, nil
}

func (s *Service) evaluateEmergencyStop(ctx context.Context, bookID string) error {
	row, err := s.store.GetEmergencyStop(ctx, bookID)
	if errors.Is(err, domain.ErrNotFound) || row == nil || !row.Enabled || row.LossPct <= 0 {
		return nil
	}
	if err != nil {
		return err
	}
	p, err := s.store.GetPortfolio(ctx, bookID)
	if err != nil {
		return err
	}
	_, snap, err := s.loadRiskWithDaySnapshot(ctx, p)
	if err != nil {
		return err
	}
	view, err := s.View(ctx, p.ClientID, p.ID)
	if err != nil {
		return err
	}
	mark := domain.EquityExcludingFlows(view.Equity, snap.DayStartNetDeposits, view.NetDeposits, snap.DayStartFlowKnown)
	pnlPct := domain.DailyPnLPct(snap.DayStartEquity, mark)
	hit := domain.DailyLossLimitHit(snap.DayStartEquity, mark, row.LossPct)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if !hit {
		if row.Tripped {
			row.Tripped = false
			row.UpdatedAt = now
			return s.store.UpsertEmergencyStop(ctx, *row)
		}
		return nil
	}
	var run *domain.EmergencyStopRun
	if row.Tripped && row.RunID != "" {
		run, err = s.store.GetEmergencyStopRun(ctx, bookID, row.RunID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
	}
	if run == nil {
		run = &domain.EmergencyStopRun{
			ID: uuid.NewString(), ClientID: bookID, CreatedAt: now, LossPct: row.LossPct,
		}
		row.Tripped = true
		row.RunID = run.ID
	}
	run.DailyPnLPct = pnlPct
	run.LossPct = row.LossPct
	run.UpdatedAt = now
	s.flattenForStop(ctx, p, run)
	if err := s.store.SaveEmergencyStopRun(ctx, *run); err != nil {
		return err
	}
	row.UpdatedAt = now
	return s.store.UpsertEmergencyStop(ctx, *row)
}

func (s *Service) flattenForStop(ctx context.Context, p *domain.Portfolio, run *domain.EmergencyStopRun) {
	owner, book := p.ClientID, p.BookID()
	canceled, _, err := s.CancelOpenPendingOrders(ctx, CancelOpenOrdersInput{ClientID: owner, PortfolioID: book})
	if err != nil {
		run.Actions = upsertEmergencyAction(run.Actions, domain.EmergencyAction{
			Kind: domain.EmergencyCancelPending, TargetID: "all", Status: domain.EmergencyFailed, Detail: emergencyDetail(err),
		})
	} else {
		run.Actions = dropEmergencyAction(run.Actions, domain.EmergencyCancelPending, "all")
		for _, o := range canceled {
			run.Actions = upsertEmergencyAction(run.Actions, domain.EmergencyAction{
				Kind: domain.EmergencyCancelOrder, TargetID: o.ID, Exchange: string(o.Exchange), Symbol: o.Symbol, Status: domain.EmergencyOK,
			})
		}
	}
	if spreads, err := s.ListSpreadOrders(ctx, owner, string(domain.SpreadStatusOpen), 200, 0, book); err != nil {
		run.Actions = upsertEmergencyAction(run.Actions, domain.EmergencyAction{
			Kind: domain.EmergencyCancelSpread, TargetID: "all", Status: domain.EmergencyFailed, Detail: emergencyDetail(err),
		})
	} else {
		for _, o := range spreads {
			if emergencyOK(run.Actions, domain.EmergencyCancelSpread, o.ID) {
				continue
			}
			_, cerr := s.CancelSpreadOrder(ctx, owner, o.ID, book)
			run.Actions = upsertEmergencyAction(run.Actions, emergencyResult(domain.EmergencyCancelSpread, o.ID, "", o.Symbol, cerr))
		}
	}
	if plans, err := s.ListRecurringBuyPlans(ctx, owner, book); err != nil {
		run.Actions = upsertEmergencyAction(run.Actions, domain.EmergencyAction{
			Kind: domain.EmergencyPausePlan, TargetID: "all", Status: domain.EmergencyFailed, Detail: emergencyDetail(err),
		})
	} else {
		for _, plan := range plans {
			if plan.Status != domain.RecurringBuyActive {
				continue
			}
			if emergencyOK(run.Actions, domain.EmergencyPausePlan, plan.ID) {
				continue
			}
			_, perr := s.PauseRecurringBuyPlan(ctx, owner, plan.ID, book)
			run.Actions = upsertEmergencyAction(run.Actions, emergencyResult(domain.EmergencyPausePlan, plan.ID, "", plan.Symbol, perr))
		}
	}
	if positions, err := s.store.ListPositions(ctx, book); err != nil {
		run.Actions = upsertEmergencyAction(run.Actions, domain.EmergencyAction{
			Kind: domain.EmergencyCloseSpot, TargetID: "all", Status: domain.EmergencyFailed, Detail: emergencyDetail(err),
		})
	} else {
		for _, pos := range positions {
			if pos.Quantity <= domain.PositionEpsilon {
				continue
			}
			id := string(pos.Exchange) + ":" + pos.Symbol
			if emergencyOK(run.Actions, domain.EmergencyCloseSpot, id) {
				continue
			}
			_, _, serr := s.PlaceOrder(ctx, OrderInput{
				ClientID: owner, PortfolioID: book, Exchange: string(pos.Exchange), Symbol: pos.Symbol,
				Side: string(domain.TradeSideSell), Quantity: pos.Quantity,
				IdempotencyKey: emergencyKey(run.ID, "spot", id),
			})
			run.Actions = upsertEmergencyAction(run.Actions, emergencyResult(domain.EmergencyCloseSpot, id, string(pos.Exchange), pos.Symbol, serr))
		}
		if fresh, ferr := s.store.ListPositions(ctx, book); ferr == nil {
			held := map[string]float64{}
			for _, pos := range fresh {
				held[string(pos.Exchange)+":"+pos.Symbol] = pos.Quantity
			}
			for i := range run.Actions {
				a := &run.Actions[i]
				if a.Kind == domain.EmergencyCloseSpot && a.TargetID != "all" && a.Status == domain.EmergencyFailed && held[a.TargetID] <= domain.PositionEpsilon {
					a.Status = domain.EmergencyOK
					a.Detail = ""
				}
			}
		}
	}
	if margin, err := s.ListMarginPositions(ctx, owner, book); err != nil {
		run.Actions = upsertEmergencyAction(run.Actions, domain.EmergencyAction{
			Kind: domain.EmergencyCloseMargin, TargetID: "all", Status: domain.EmergencyFailed, Detail: emergencyDetail(err),
		})
	} else {
		for _, pos := range margin {
			if emergencyOK(run.Actions, domain.EmergencyCloseMargin, pos.ID) {
				continue
			}
			_, _, merr := s.CloseMarginPosition(ctx, MarginCloseInput{
				ClientID: owner, PortfolioID: book, PositionID: pos.ID,
				IdempotencyKey: emergencyKey(run.ID, "margin", pos.ID),
			})
			run.Actions = upsertEmergencyAction(run.Actions, emergencyResult(domain.EmergencyCloseMargin, pos.ID, string(pos.Exchange), pos.Symbol, merr))
		}
		open := map[string]bool{}
		for _, pos := range margin {
			open[pos.ID] = true
		}
		for i := range run.Actions {
			a := &run.Actions[i]
			if a.Kind == domain.EmergencyCloseMargin && a.TargetID != "all" && a.Status == domain.EmergencyFailed && !open[a.TargetID] {
				a.Status = domain.EmergencyOK
				a.Detail = ""
			}
		}
	}
}

func (s *Service) emergencyView(ctx context.Context, p *domain.Portfolio, _ bool) (*EmergencyStopView, error) {
	bookID := p.BookID()
	row, err := s.store.GetEmergencyStop(ctx, bookID)
	if errors.Is(err, domain.ErrNotFound) {
		row = &domain.EmergencyStop{ClientID: bookID}
	} else if err != nil {
		return nil, err
	}
	_, snap, err := s.loadRiskWithDaySnapshot(ctx, p)
	if err != nil {
		return nil, err
	}
	view, err := s.View(ctx, p.ClientID, p.ID)
	if err != nil {
		return nil, err
	}
	mark := domain.EquityExcludingFlows(view.Equity, snap.DayStartNetDeposits, view.NetDeposits, snap.DayStartFlowKnown)
	out := &EmergencyStopView{
		Enabled: row.Enabled, LossPct: row.LossPct, Tripped: row.Tripped, Note: emergencyNote,
		DailyPnLPct: domain.DailyPnLPct(snap.DayStartEquity, mark),
	}
	if row.Enabled && row.LossPct > 0 {
		out.Hit = domain.DailyLossLimitHit(snap.DayStartEquity, mark, row.LossPct)
	}
	if row.RunID != "" {
		run, rerr := s.store.GetEmergencyStopRun(ctx, bookID, row.RunID)
		if rerr == nil {
			out.Run = run
		}
	}
	return out, nil
}

// EmergencyStopPayload is the JSON body for the switch and the last trip.
func EmergencyStopPayload(v *EmergencyStopView) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"enabled": v.Enabled, "lossPct": v.LossPct, "dailyPnlPct": v.DailyPnLPct,
		"hit": v.Hit, "tripped": v.Tripped, "note": v.Note,
	}
	if v.Run == nil {
		return out
	}
	actions := make([]any, 0, len(v.Run.Actions))
	for _, a := range v.Run.Actions {
		row := map[string]any{"kind": a.Kind, "targetId": a.TargetID, "status": a.Status}
		if a.Exchange != "" {
			row["exchange"] = a.Exchange
		}
		if a.Symbol != "" {
			row["symbol"] = a.Symbol
		}
		if a.Detail != "" {
			row["detail"] = a.Detail
		}
		actions = append(actions, row)
	}
	out["run"] = map[string]any{
		"id": v.Run.ID, "createdAt": v.Run.CreatedAt, "updatedAt": v.Run.UpdatedAt,
		"dailyPnlPct": v.Run.DailyPnLPct, "lossPct": v.Run.LossPct, "actions": actions,
	}
	return out
}

func emergencyKey(runID, kind, target string) string {
	key := "es-" + runID + "-" + kind + "-" + target
	if len(key) > domain.MaxIdempotencyKeyLen {
		return key[:domain.MaxIdempotencyKeyLen]
	}
	return key
}

func emergencyDetail(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 200 {
		msg = msg[:200]
	}
	return msg
}

func emergencyResult(kind, id, exchange, symbol string, err error) domain.EmergencyAction {
	if err == nil || errors.Is(err, domain.ErrNotFound) {
		return domain.EmergencyAction{Kind: kind, TargetID: id, Exchange: exchange, Symbol: symbol, Status: domain.EmergencyOK}
	}
	return domain.EmergencyAction{Kind: kind, TargetID: id, Exchange: exchange, Symbol: symbol, Status: domain.EmergencyFailed, Detail: emergencyDetail(err)}
}

func emergencyOK(actions []domain.EmergencyAction, kind, id string) bool {
	for _, a := range actions {
		if a.Kind == kind && a.TargetID == id && a.Status == domain.EmergencyOK {
			return true
		}
	}
	return false
}

func upsertEmergencyAction(actions []domain.EmergencyAction, next domain.EmergencyAction) []domain.EmergencyAction {
	for i := range actions {
		if actions[i].Kind == next.Kind && actions[i].TargetID == next.TargetID {
			actions[i] = next
			return actions
		}
	}
	return append(actions, next)
}

func dropEmergencyAction(actions []domain.EmergencyAction, kind, id string) []domain.EmergencyAction {
	out := actions[:0]
	for _, a := range actions {
		if a.Kind == kind && a.TargetID == id {
			continue
		}
		out = append(out, a)
	}
	return out
}
