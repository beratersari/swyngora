package fundingarb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
)

// QuoteFetcher evaluates live funding-arb quotes and scans (usually *market.Service).
type QuoteFetcher interface {
	GetFundingArb(ctx context.Context, in market.FundingArbParams) (*domain.FundingArbReport, error)
	ScanFundingArb(ctx context.Context, in market.FundingArbScanParams) (*domain.FundingArbScan, error)
}

// AccountChecker reports whether a tenant is closed.
type AccountChecker interface {
	IsClosed(ctx context.Context, clientID string) (bool, *domain.Account, error)
}

// Notifier delivers a webhook payload using the client's alert webhook.
type Notifier interface {
	NotifyClient(ctx context.Context, clientID, sourceID, payloadJSON string) error
}

// CreateInput creates a follow-watch.
type CreateInput struct {
	ClientID      string
	Symbol        string
	Notional      float64
	HoldHours     float64
	MinProfit     float64
	Quote         string
	SymbolLimit   int
	FeeBinancePct *float64
	FeeBybitPct   *float64
}

// Service orchestrates funding-arb watches and threshold notifies.
type Service struct {
	store   domain.FundingArbWatchPort
	quotes  QuoteFetcher
	account AccountChecker
	notify  Notifier
}

// New constructs the service.
func New(store domain.FundingArbWatchPort, quotes QuoteFetcher) *Service {
	return &Service{store: store, quotes: quotes}
}

// SetAccountChecker skips closed tenants on the checker.
func (s *Service) SetAccountChecker(a AccountChecker) {
	if s != nil {
		s.account = a
	}
}

// SetNotifier attaches webhook delivery (usually the price-alert service).
func (s *Service) SetNotifier(n Notifier) {
	if s != nil {
		s.notify = n
	}
}

// PurgeClient deletes watches for a tenant.
func (s *Service) PurgeClient(ctx context.Context, clientID string) error {
	if s == nil || s.store == nil {
		return nil
	}
	list, err := s.store.ListWatches(ctx, clientID)
	if err != nil {
		return err
	}
	for i := range list {
		_ = s.store.DeleteWatch(ctx, clientID, list[i].ID)
	}
	return nil
}

// CreateWatch validates and stores an active watch.
func (s *Service) CreateWatch(ctx context.Context, in CreateInput) (*domain.FundingArbWatch, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: funding-arb store not configured", domain.ErrUpstream)
	}
	clientID, err := domain.NormalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	symbol, err := domain.ResolveFundingArbWatchSymbol(in.Symbol)
	if err != nil {
		return nil, err
	}
	notional, err := domain.ResolveFundingArbNotional(in.Notional)
	if err != nil {
		return nil, err
	}
	hold, err := domain.ResolveFundingArbHoldHours(in.HoldHours)
	if err != nil {
		return nil, err
	}
	minP, err := domain.ResolveFundingArbMinProfit(in.MinProfit)
	if err != nil {
		return nil, err
	}
	fb, err := domain.ResolveFundingArbFeeRate(domain.ExchangeBinance, in.FeeBinancePct)
	if err != nil {
		return nil, err
	}
	fy, err := domain.ResolveFundingArbFeeRate(domain.ExchangeBybit, in.FeeBybitPct)
	if err != nil {
		return nil, err
	}
	quote := strings.ToUpper(strings.TrimSpace(in.Quote))
	if quote == "" {
		quote = "USDT"
	}
	limit := domain.ClampFundingArbScanLimit(in.SymbolLimit)
	n, err := s.store.CountWatches(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxFundingArbWatchesPerClient {
		return nil, fmt.Errorf("%w: max %d funding-arb watches per client", domain.ErrInvalidArgument, domain.MaxFundingArbWatchesPerClient)
	}
	if symbol == domain.FundingArbWatchScanSymbol {
		existing, err := s.store.ListWatches(ctx, clientID)
		if err != nil {
			return nil, err
		}
		for i := range existing {
			if existing[i].IsScan() {
				return nil, fmt.Errorf("%w: a scan follow already exists (delete it first)", domain.ErrInvalidArgument)
			}
		}
	}
	now := time.Now().UTC()
	return s.store.CreateWatch(ctx, domain.FundingArbWatch{
		ID: uuid.NewString(), ClientID: clientID, Symbol: symbol,
		Quote: quote, SymbolLimit: limit,
		Notional: notional, HoldHours: hold, MinProfit: minP,
		FeeBinancePct: fb * 100, FeeBybitPct: fy * 100,
		Status: domain.FundingArbWatchActive, Armed: true,
		CreatedAt: now, UpdatedAt: now,
	})
}

// ListWatches lists watches for a client.
func (s *Service) ListWatches(ctx context.Context, clientID string) ([]domain.FundingArbWatch, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: funding-arb store not configured", domain.ErrUpstream)
	}
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListWatches(ctx, clientID)
}

// GetWatch returns one watch.
func (s *Service) GetWatch(ctx context.Context, clientID, id string) (*domain.FundingArbWatch, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: funding-arb store not configured", domain.ErrUpstream)
	}
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: watch id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetWatch(ctx, clientID, id)
}

// UpdateInput patches watch settings. Nil fields stay unchanged.
type UpdateInput struct {
	ClientID      string
	ID            string
	Notional      *float64
	HoldHours     *float64
	MinProfit     *float64
	Quote         *string
	SymbolLimit   *int
	FeeBinancePct *float64
	FeeBybitPct   *float64
}

// UpdateWatch changes min profit and other settings without deleting the follow.
func (s *Service) UpdateWatch(ctx context.Context, in UpdateInput) (*domain.FundingArbWatch, error) {
	w, err := s.GetWatch(ctx, in.ClientID, in.ID)
	if err != nil {
		return nil, err
	}
	if in.Notional != nil {
		n, err := domain.ResolveFundingArbNotional(*in.Notional)
		if err != nil {
			return nil, err
		}
		w.Notional = n
	}
	if in.HoldHours != nil {
		h, err := domain.ResolveFundingArbHoldHours(*in.HoldHours)
		if err != nil {
			return nil, err
		}
		w.HoldHours = h
	}
	if in.MinProfit != nil {
		p, err := domain.ResolveFundingArbMinProfit(*in.MinProfit)
		if err != nil {
			return nil, err
		}
		w.MinProfit = p
	}
	if in.Quote != nil {
		q := strings.ToUpper(strings.TrimSpace(*in.Quote))
		if q == "" {
			q = "USDT"
		}
		w.Quote = q
	}
	if in.SymbolLimit != nil {
		w.SymbolLimit = domain.ClampFundingArbScanLimit(*in.SymbolLimit)
	}
	if in.FeeBinancePct != nil {
		fb, err := domain.ResolveFundingArbFeeRate(domain.ExchangeBinance, in.FeeBinancePct)
		if err != nil {
			return nil, err
		}
		w.FeeBinancePct = fb * 100
	}
	if in.FeeBybitPct != nil {
		fy, err := domain.ResolveFundingArbFeeRate(domain.ExchangeBybit, in.FeeBybitPct)
		if err != nil {
			return nil, err
		}
		w.FeeBybitPct = fy * 100
	}
	w.UpdatedAt = time.Now().UTC()
	return s.store.UpdateWatch(ctx, *w)
}

// PauseWatch stops evaluation without deleting the follow or its signals.
func (s *Service) PauseWatch(ctx context.Context, clientID, id string) (*domain.FundingArbWatch, error) {
	return s.setWatchStatus(ctx, clientID, id, domain.FundingArbWatchPaused)
}

// ResumeWatch starts evaluating a paused follow again.
func (s *Service) ResumeWatch(ctx context.Context, clientID, id string) (*domain.FundingArbWatch, error) {
	return s.setWatchStatus(ctx, clientID, id, domain.FundingArbWatchActive)
}

func (s *Service) setWatchStatus(ctx context.Context, clientID, id string, st domain.FundingArbWatchStatus) (*domain.FundingArbWatch, error) {
	w, err := s.GetWatch(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	if w.Status == st {
		return w, nil
	}
	w.Status = st
	w.UpdatedAt = time.Now().UTC()
	return s.store.UpdateWatch(ctx, *w)
}

// DeleteWatch removes a watch.
func (s *Service) DeleteWatch(ctx context.Context, clientID, id string) error {
	if s.store == nil {
		return fmt.Errorf("%w: funding-arb store not configured", domain.ErrUpstream)
	}
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: watch id is required", domain.ErrInvalidArgument)
	}
	return s.store.DeleteWatch(ctx, clientID, id)
}

// ListSignals lists open/closed crossings for a client.
func (s *Service) ListSignals(ctx context.Context, clientID, status string, limit int) ([]domain.FundingArbSignal, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: funding-arb store not configured", domain.ErrUpstream)
	}
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	var st domain.FundingArbSignalStatus
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "all":
		st = ""
	case "open":
		st = domain.FundingArbSignalOpen
	case "closed":
		st = domain.FundingArbSignalClosed
	default:
		return nil, fmt.Errorf("%w: status must be open, closed, or all", domain.ErrInvalidArgument)
	}
	return s.store.ListSignals(ctx, clientID, st, limit)
}

func tenantClosed(ctx context.Context, accounts AccountChecker, clientID string) bool {
	if accounts == nil || clientID == "" {
		return false
	}
	closed, _, err := accounts.IsClosed(ctx, clientID)
	return err == nil && closed
}

// ProcessActiveWatches evaluates every active watch. Returns opened/closed counts.
func (s *Service) ProcessActiveWatches(ctx context.Context, now time.Time) (opened, closed, touched int, err error) {
	if s == nil || s.store == nil || s.quotes == nil {
		return 0, 0, 0, nil
	}
	now = now.UTC()
	watches, err := s.store.ListActiveWatches(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	for i := range watches {
		w := watches[i]
		if tenantClosed(ctx, s.account, w.ClientID) {
			continue
		}
		touched++
		o, c, err := s.processWatch(ctx, w, now)
		if err != nil {
			return opened, closed, touched, err
		}
		opened += o
		closed += c
	}
	return opened, closed, touched, nil
}

func (s *Service) processWatch(ctx context.Context, w domain.FundingArbWatch, now time.Time) (opened, closed int, err error) {
	fb, fy := w.FeeBinancePct, w.FeeBybitPct
	if w.IsScan() {
		scan, qerr := s.quotes.ScanFundingArb(ctx, market.FundingArbScanParams{
			Quote: w.Quote, Notional: w.Notional, HoldHours: w.HoldHours,
			FeeBinancePct: &fb, FeeBybitPct: &fy, SymbolLimit: w.SymbolLimit,
		})
		if qerr != nil || scan == nil {
			return 0, 0, nil
		}
		above := map[string]domain.FundingArbHit{}
		for _, h := range scan.Hits {
			if h.RankScore >= w.MinProfit && h.LongExchange != "" && h.ShortExchange != "" {
				above[h.Symbol] = h
			}
		}
		openList, lerr := s.store.ListOpenSignals(ctx, w.ID)
		if lerr != nil {
			return 0, 0, lerr
		}
		openBy := map[string]domain.FundingArbSignal{}
		for _, sig := range openList {
			openBy[sig.Symbol] = sig
		}
		for _, h := range above {
			o, c, aerr := s.applyHit(ctx, w, h.Symbol, domain.Exchange(h.LongExchange), domain.Exchange(h.ShortExchange), h.RankScore, h.Summary, now)
			if aerr != nil {
				return opened, closed, aerr
			}
			opened += o
			closed += c
			delete(openBy, h.Symbol)
		}
		for _, leftover := range openBy {
			// Missing from the volume-limited scan is not enough to close.
			// Re-quote that pair and close only when net is really below min.
			o, c, rerr := s.recheckSymbol(ctx, w, leftover.Symbol, now)
			if rerr != nil {
				return opened, closed, rerr
			}
			opened += o
			closed += c
		}
		return opened, closed, nil
	}
	rep, qerr := s.quotes.GetFundingArb(ctx, market.FundingArbParams{
		Symbol: w.Symbol, Notional: w.Notional, HoldHours: w.HoldHours,
		FeeBinancePct: &fb, FeeBybitPct: &fy,
	})
	if qerr != nil || rep == nil {
		return 0, 0, nil
	}
	if rep.HorizonNet >= w.MinProfit && rep.Trade != nil {
		return s.applyHit(ctx, w, w.Symbol, domain.Exchange(rep.Trade.LongExchange), domain.Exchange(rep.Trade.ShortExchange), rep.HorizonNet, rep.Summary, now)
	}
	open, oerr := s.store.GetOpenSignal(ctx, w.ID, w.Symbol)
	if oerr != nil && !errors.Is(oerr, domain.ErrNotFound) {
		return 0, 0, oerr
	}
	if open != nil {
		if err := s.store.CloseSignal(ctx, open.ID, now); err != nil {
			return 0, 0, err
		}
		closed = 1
	}
	if !w.Armed {
		_ = s.store.SetWatchArmed(ctx, w.ID, true, now)
	}
	return 0, closed, nil
}

func (s *Service) recheckSymbol(ctx context.Context, w domain.FundingArbWatch, symbol string, now time.Time) (opened, closed int, err error) {
	fb, fy := w.FeeBinancePct, w.FeeBybitPct
	rep, qerr := s.quotes.GetFundingArb(ctx, market.FundingArbParams{
		Symbol: symbol, Notional: w.Notional, HoldHours: w.HoldHours,
		FeeBinancePct: &fb, FeeBybitPct: &fy,
	})
	if qerr != nil || rep == nil {
		return 0, 0, nil
	}
	if rep.HorizonNet >= w.MinProfit && rep.Trade != nil {
		return s.applyHit(ctx, w, symbol, domain.Exchange(rep.Trade.LongExchange), domain.Exchange(rep.Trade.ShortExchange), rep.HorizonNet, rep.Summary, now)
	}
	open, oerr := s.store.GetOpenSignal(ctx, w.ID, symbol)
	if oerr != nil && !errors.Is(oerr, domain.ErrNotFound) {
		return 0, 0, oerr
	}
	if open == nil {
		return 0, 0, nil
	}
	if err := s.store.CloseSignal(ctx, open.ID, now); err != nil {
		return 0, 0, err
	}
	return 0, 1, nil
}

func (s *Service) applyHit(ctx context.Context, w domain.FundingArbWatch, symbol string, long, short domain.Exchange, net float64, summary string, now time.Time) (opened, closed int, err error) {
	open, oerr := s.store.GetOpenSignal(ctx, w.ID, symbol)
	if oerr != nil && !errors.Is(oerr, domain.ErrNotFound) {
		return 0, 0, oerr
	}
	if open != nil && (open.LongExchange != long || open.ShortExchange != short) {
		if err := s.store.CloseSignal(ctx, open.ID, now); err != nil {
			return 0, 0, err
		}
		closed = 1
		open = nil
	}
	if open == nil {
		sig := domain.FundingArbSignal{
			ID: uuid.NewString(), WatchID: w.ID, ClientID: w.ClientID, Symbol: symbol,
			LongExchange: long, ShortExchange: short,
			NetAfterFees: net, MinProfit: w.MinProfit,
			Status: domain.FundingArbSignalOpen, OpenedAt: now, LastSeenAt: now,
		}
		if _, err := s.store.CreateSignal(ctx, sig); err != nil {
			return opened, closed, err
		}
		_ = s.store.SetWatchArmed(ctx, w.ID, false, now)
		s.fire(ctx, w, symbol, string(long), string(short), net, summary)
		return opened + 1, closed, nil
	}
	_ = s.store.TouchSignal(ctx, open.ID, net, now)
	return 0, closed, nil
}

func (s *Service) fire(ctx context.Context, w domain.FundingArbWatch, symbol, long, short string, net float64, summary string) {
	if s.notify == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type": "funding_arb.triggered", "watchId": w.ID, "symbol": symbol,
		"longExchange": long, "shortExchange": short,
		"notional": w.Notional, "holdHours": w.HoldHours,
		"minProfit": w.MinProfit, "netAfterFees": net,
		"summary": summary, "scan": w.IsScan(),
	})
	if err != nil {
		return
	}
	_ = s.notify.NotifyClient(ctx, w.ClientID, w.ID+":"+symbol, string(payload))
}
