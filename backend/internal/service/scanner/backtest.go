package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// StartBacktestInput starts a historical test of a rule on one symbol.
type StartBacktestInput struct {
	ClientID   string
	RuleID     string
	Exchange   string
	Symbol     string
	RangeStart time.Time
	RangeEnd   time.Time
}

// StartBacktest queues a background historical test.
// If an identical pending/running/completed job exists, it is returned (no duplicate run).
func (s *Service) StartBacktest(ctx context.Context, in StartBacktestInput) (*domain.ScannerBacktest, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	ruleID := strings.TrimSpace(in.RuleID)
	if ruleID == "" {
		return nil, fmt.Errorf("%w: ruleId is required", domain.ErrInvalidArgument)
	}
	rule, err := s.store.GetRule(ctx, clientID, ruleID)
	if err != nil {
		return nil, err
	}
	ex, sym, err := normalizeExchangeSymbol(in.Exchange, in.Symbol)
	if err != nil {
		return nil, err
	}
	if in.RangeStart.IsZero() || in.RangeEnd.IsZero() {
		return nil, fmt.Errorf("%w: rangeStart and rangeEnd are required", domain.ErrInvalidArgument)
	}
	start := in.RangeStart.UTC()
	end := in.RangeEnd.UTC()
	if !end.After(start) {
		return nil, fmt.Errorf("%w: rangeEnd must be after rangeStart", domain.ErrInvalidArgument)
	}
	if end.Sub(start) > 400*24*time.Hour {
		return nil, fmt.Errorf("%w: date range must be at most 400 days", domain.ErrInvalidArgument)
	}

	if existing, err := s.store.FindActiveBacktest(ctx, clientID, ruleID, ex, sym, start, end); err == nil && existing != nil {
		return existing, nil
	} else if err != nil && err != domain.ErrNotFound {
		return nil, err
	}

	n, err := s.store.CountBacktests(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxScannerBacktestsPerClient {
		return nil, fmt.Errorf("%w: max %d backtests per client", domain.ErrInvalidArgument, domain.MaxScannerBacktestsPerClient)
	}

	now := time.Now().UTC()
	job := domain.ScannerBacktest{
		ID: uuid.NewString(), ClientID: clientID, RuleID: rule.ID,
		Exchange: ex, Symbol: sym, Interval: rule.Interval,
		RangeStart: start, RangeEnd: end, Status: domain.BacktestPending, CreatedAt: now,
	}
	return s.store.CreateBacktest(ctx, job)
}

// GetBacktest returns one job.
func (s *Service) GetBacktest(ctx context.Context, clientID, id string) (*domain.ScannerBacktest, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: backtest id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetBacktest(ctx, clientID, id)
}

// ListBacktests lists jobs for a client.
func (s *Service) ListBacktests(ctx context.Context, clientID string, limit, offset int) ([]domain.ScannerBacktest, int, error) {
	if s.store == nil {
		return nil, 0, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	total, err := s.store.CountBacktests(ctx, clientID)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.store.ListBacktests(ctx, clientID, limit, offset)
	return list, total, err
}

// CancelBacktest cancels a pending or running job.
func (s *Service) CancelBacktest(ctx context.Context, clientID, id string) (*domain.ScannerBacktest, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: backtest id is required", domain.ErrInvalidArgument)
	}
	return s.store.CancelBacktest(ctx, clientID, id, time.Now().UTC())
}

// ListBacktestSignals returns signal rows for a job owned by the client.
func (s *Service) ListBacktestSignals(ctx context.Context, clientID, backtestID string, limit, offset int) ([]domain.ScannerBacktestSignal, int, error) {
	if s.store == nil {
		return nil, 0, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, 0, err
	}
	if _, err := s.store.GetBacktest(ctx, clientID, backtestID); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	total, err := s.store.CountBacktestSignals(ctx, backtestID)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.store.ListBacktestSignals(ctx, backtestID, limit, offset)
	return list, total, err
}

// BacktestWorker claims and executes pending backtests until ctx is done.
type BacktestWorker struct {
	Scanner  *Service
	Interval time.Duration
	Logger   *slog.Logger
	sleep    func(context.Context, time.Duration) error
}

// Start processes pending jobs on an interval.
func (w *BacktestWorker) Start(ctx context.Context) {
	if w.Logger == nil {
		w.Logger = slog.Default()
	}
	if w.sleep == nil {
		w.sleep = sleepCtx
	}
	if w.Interval <= 0 {
		w.Interval = 2 * time.Second
	}
	w.RunOnce(ctx)
	for {
		if err := w.sleep(ctx, w.Interval); err != nil {
			w.Logger.Info("scanner backtest worker stopped", "err", err)
			return
		}
		w.RunOnce(ctx)
	}
}

// RunOnce claims and runs all currently pending jobs (sequentially).
func (w *BacktestWorker) RunOnce(ctx context.Context) {
	if w.Scanner == nil || w.Scanner.store == nil {
		return
	}
	pending, err := w.Scanner.store.ListPendingBacktests(ctx)
	if err != nil {
		if w.Logger != nil {
			w.Logger.Error("list pending backtests", "err", err)
		}
		return
	}
	for _, job := range pending {
		if ctx.Err() != nil {
			return
		}
		ok, err := w.Scanner.store.ClaimBacktest(ctx, job.ID, time.Now().UTC())
		if err != nil || !ok {
			continue
		}
		job.Status = domain.BacktestRunning
		if err := w.Scanner.executeBacktestJob(ctx, job); err != nil && w.Logger != nil {
			w.Logger.Error("backtest failed", "id", job.ID, "err", err)
		}
	}
}

func (s *Service) executeBacktestJob(ctx context.Context, job domain.ScannerBacktest) error {
	id := job.ID
	rule, err := s.store.GetRule(ctx, job.ClientID, job.RuleID)
	if err != nil {
		_ = s.store.FinishBacktest(ctx, id, domain.BacktestFailed, 0, err.Error(), time.Now().UTC())
		return err
	}

	warmup := domain.ScannerCandleNeed(*rule) + 5
	ivDur := intervalApprox(rule.Interval)
	fetchStart := job.RangeStart.Add(-time.Duration(warmup) * ivDur)
	fetchEnd := job.RangeEnd.Add(21 * 24 * time.Hour)

	candles, err := s.fetchCandlesRange(ctx, string(job.Exchange), job.Symbol, rule.Interval, fetchStart, fetchEnd)
	if err != nil {
		_ = s.store.FinishBacktest(ctx, id, domain.BacktestFailed, 0, err.Error(), time.Now().UTC())
		return err
	}
	if len(candles) == 0 {
		_ = s.store.FinishBacktest(ctx, id, domain.BacktestFailed, 0, "no candles in range", time.Now().UTC())
		return fmt.Errorf("no candles")
	}

	firstIdx, lastIdx := -1, -1
	for i, c := range candles {
		ot := c.OpenTime.UTC()
		if ot.Before(job.RangeStart) {
			continue
		}
		if ot.After(job.RangeEnd) {
			break
		}
		if firstIdx < 0 {
			firstIdx = i
		}
		lastIdx = i
	}
	if firstIdx < 0 || lastIdx < firstIdx {
		_ = s.store.FinishBacktest(ctx, id, domain.BacktestCompleted, 0, "", time.Now().UTC())
		return nil
	}

	total := lastIdx - firstIdx + 1
	signalCount := 0
	for i := firstIdx; i <= lastIdx; i++ {
		if ctx.Err() != nil {
			_ = s.store.FinishBacktest(ctx, id, domain.BacktestCanceled, signalCount, "context canceled", time.Now().UTC())
			return ctx.Err()
		}
		if st, err := s.store.GetBacktestStatus(ctx, id); err == nil && st == domain.BacktestCanceled {
			return nil
		}

		match, err := domain.EvaluateScannerRule(*rule, candles[:i+1])
		if err == nil && match != nil {
			closePx, _ := strconv.ParseFloat(strings.TrimSpace(candles[i].Close), 64)
			if math.IsNaN(closePx) {
				closePx = 0
			}
			sig := domain.ScannerBacktestSignal{
				ID: uuid.NewString(), BacktestID: id, SignalAt: candles[i].OpenTime.UTC(),
				ClosePrice: closePx, Summary: match.Summary, Metrics: match.Metrics,
				Return1d: domain.ForwardReturnPct(candles, i, 1),
				Return5d: domain.ForwardReturnPct(candles, i, 5),
				Return20d: domain.ForwardReturnPct(candles, i, 20),
			}
			if err := s.store.InsertBacktestSignal(ctx, sig); err == nil {
				signalCount++
			}
		}

		processed := i - firstIdx + 1
		pct := float64(processed) / float64(total) * 100
		if processed == total || processed%25 == 0 {
			_ = s.store.UpdateBacktestProgress(ctx, id, processed, total, signalCount, pct)
		}
	}

	if st, err := s.store.GetBacktestStatus(ctx, id); err == nil && st == domain.BacktestCanceled {
		return nil
	}
	_ = s.store.UpdateBacktestProgress(ctx, id, total, total, signalCount, 100)
	return s.store.FinishBacktest(ctx, id, domain.BacktestCompleted, signalCount, "", time.Now().UTC())
}

func (s *Service) fetchCandlesRange(ctx context.Context, exchange, symbol, interval string, start, end time.Time) ([]domain.Candle, error) {
	if s.market == nil {
		return nil, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	const page = 1000
	var all []domain.Candle
	curStart := start.UTC()
	end = end.UTC()
	step := intervalApprox(interval)
	for curStart.Before(end) {
		chunkEnd := curStart.Add(time.Duration(page) * step)
		if chunkEnd.After(end) {
			chunkEnd = end
		}
		st, en := curStart, chunkEnd
		part, err := s.market.GetCandles(ctx, exchange, symbol, interval, page, &st, &en)
		if err != nil {
			return nil, err
		}
		if len(part) == 0 {
			curStart = chunkEnd.Add(step)
			continue
		}
		all = append(all, part...)
		last := part[len(part)-1].OpenTime.UTC()
		next := last.Add(step)
		if !next.After(curStart) {
			curStart = chunkEnd.Add(step)
		} else {
			curStart = next
		}
		if curStart.After(end) {
			break
		}
	}
	if len(all) == 0 {
		return all, nil
	}
	byOT := map[int64]domain.Candle{}
	for _, c := range all {
		byOT[c.OpenTime.UTC().UnixNano()] = c
	}
	out := make([]domain.Candle, 0, len(byOT))
	for _, c := range byOT {
		out = append(out, c)
	}
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j].OpenTime.Before(out[j-1].OpenTime) {
			out[j], out[j-1] = out[j-1], out[j]
			j--
		}
	}
	return out, nil
}

func intervalApprox(iv string) time.Duration {
	switch iv {
	case "1m":
		return time.Minute
	case "3m":
		return 3 * time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "2h":
		return 2 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "6h":
		return 6 * time.Hour
	case "8h":
		return 8 * time.Hour
	case "12h":
		return 12 * time.Hour
	case "1d":
		return 24 * time.Hour
	case "3d":
		return 72 * time.Hour
	case "1w":
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func normalizeExchangeSymbol(exchange, symbol string) (domain.Exchange, string, error) {
	rawEx := strings.TrimSpace(exchange)
	var ex domain.Exchange
	if rawEx == "" {
		ex = domain.DefaultExchange
	} else {
		if !domain.IsValidExchange(rawEx) {
			return "", "", fmt.Errorf("%w: exchange must be one of %v", domain.ErrInvalidArgument, domain.SupportedExchanges)
		}
		ex = domain.ParseExchange(rawEx)
	}
	sym := domain.NormalizeSymbol(ex, symbol)
	if sym == "" {
		return "", "", fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	return ex, sym, nil
}
