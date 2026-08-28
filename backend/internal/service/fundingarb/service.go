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

// QuoteFetcher evaluates a live funding-arb quote (usually *market.Service).
type QuoteFetcher interface {
	GetFundingArb(ctx context.Context, in market.FundingArbParams) (*domain.FundingArbReport, error)
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
	symbol, err := domain.ValidateOpenInterestSymbol(in.Symbol)
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
	n, err := s.store.CountWatches(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxFundingArbWatchesPerClient {
		return nil, fmt.Errorf("%w: max %d funding-arb watches per client", domain.ErrInvalidArgument, domain.MaxFundingArbWatchesPerClient)
	}
	now := time.Now().UTC()
	return s.store.CreateWatch(ctx, domain.FundingArbWatch{
		ID: uuid.NewString(), ClientID: clientID, Symbol: symbol,
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
		fb, fy := w.FeeBinancePct, w.FeeBybitPct
		rep, qerr := s.quotes.GetFundingArb(ctx, market.FundingArbParams{
			Symbol: w.Symbol, Notional: w.Notional, HoldHours: w.HoldHours,
			FeeBinancePct: &fb, FeeBybitPct: &fy,
		})
		if qerr != nil || rep == nil {
			continue
		}
		net := rep.HorizonNet
		above := net >= w.MinProfit && rep.Trade != nil
		open, oerr := s.store.GetOpenSignal(ctx, w.ID)
		if oerr != nil && !errors.Is(oerr, domain.ErrNotFound) {
			return opened, closed, touched, oerr
		}
		if above {
			long := domain.Exchange(rep.Trade.LongExchange)
			short := domain.Exchange(rep.Trade.ShortExchange)
			if open != nil && (open.LongExchange != long || open.ShortExchange != short) {
				if err := s.store.CloseSignal(ctx, open.ID, now); err != nil {
					return opened, closed, touched, err
				}
				closed++
				open = nil
				w.Armed = true
			}
			if open == nil {
				if !w.Armed {
					continue
				}
				sig := domain.FundingArbSignal{
					ID: uuid.NewString(), WatchID: w.ID, ClientID: w.ClientID, Symbol: w.Symbol,
					LongExchange: long, ShortExchange: short,
					NetAfterFees: net, MinProfit: w.MinProfit,
					Status: domain.FundingArbSignalOpen, OpenedAt: now, LastSeenAt: now,
				}
				if _, err := s.store.CreateSignal(ctx, sig); err != nil {
					return opened, closed, touched, err
				}
				_ = s.store.SetWatchArmed(ctx, w.ID, false, now)
				s.fire(ctx, w, rep, net)
				opened++
			} else {
				_ = s.store.TouchSignal(ctx, open.ID, net, now)
			}
			continue
		}
		if open != nil {
			_ = s.store.CloseSignal(ctx, open.ID, now)
			closed++
		}
		if !w.Armed {
			_ = s.store.SetWatchArmed(ctx, w.ID, true, now)
		}
	}
	return opened, closed, touched, nil
}

func (s *Service) fire(ctx context.Context, w domain.FundingArbWatch, rep *domain.FundingArbReport, net float64) {
	if s.notify == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"type": "funding_arb.triggered", "watchId": w.ID, "symbol": w.Symbol,
		"longExchange": rep.Trade.LongExchange, "shortExchange": rep.Trade.ShortExchange,
		"notional": w.Notional, "holdHours": w.HoldHours,
		"minProfit": w.MinProfit, "netAfterFees": net,
		"summary": rep.Summary,
	})
	if err != nil {
		return
	}
	_ = s.notify.NotifyClient(ctx, w.ClientID, w.ID, string(payload))
}
