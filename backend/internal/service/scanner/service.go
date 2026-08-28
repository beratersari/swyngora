package scanner

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// CandleFetcher loads OHLCV for scanner evaluation.
type CandleFetcher interface {
	GetCandles(ctx context.Context, exchange, symbol, interval string, limit int, start, end *time.Time) ([]domain.Candle, error)
}

// WatchlistReader loads a client's watchlist symbols (own list).
// Matches watchlist.Service.Get(actor, owner); pass empty owner for own list.
type WatchlistReader interface {
	Get(ctx context.Context, actorClientID, ownerClientID string) (*domain.WatchlistAccess, error)
}

// AccountChecker reports whether a tenant is closed so workers can skip them.
type AccountChecker interface {
	IsClosed(ctx context.Context, clientID string) (bool, *domain.Account, error)
}

// Service orchestrates scanner rules and evaluation.
type Service struct {
	store   domain.ScannerPort
	market  CandleFetcher
	watch   WatchlistReader
	account AccountChecker
}

// New constructs a scanner service.
func New(store domain.ScannerPort, market CandleFetcher, watch WatchlistReader) *Service {
	return &Service{store: store, market: market, watch: watch}
}

// SetAccountChecker wires account-closed skips for the background runner.
func (s *Service) SetAccountChecker(a AccountChecker) {
	if s != nil {
		s.account = a
	}
}

func tenantClosed(ctx context.Context, accounts AccountChecker, clientID string) bool {
	if accounts == nil || clientID == "" {
		return false
	}
	closed, _, err := accounts.IsClosed(ctx, clientID)
	return err == nil && closed
}

// CreateInput creates a scanner rule for the client's watchlist.
type CreateInput struct {
	ClientID   string
	Type       string   // rsi | ma_crossover | volume_increase | combo (optional if Conditions set)
	Conditions []string // subset of rsi, ma_crossover, volume_increase
	MatchMode  string   // all | any
	Interval   string
	// RSI
	RSIPeriod    int
	RSICondition string // above | below
	RSIThreshold float64
	// MA
	MAFastPeriod int
	MASlowPeriod int
	MADirection  string // golden_cross | death_cross
	// Volume
	VolumeLookback int
	VolumeMinRatio float64
}

// Create validates and stores a rule.
func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.ScannerRule, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	conds, typ, err := domain.ResolveScannerConditions(in.Type, in.Conditions)
	if err != nil {
		return nil, err
	}
	mode, err := domain.ResolveScannerMatchMode(in.MatchMode)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(in.Interval)
	if interval == "" {
		interval = domain.DefaultScannerInterval
	}
	if !domain.IsValidInterval(interval) {
		return nil, fmt.Errorf("%w: invalid interval", domain.ErrInvalidArgument)
	}
	n, err := s.store.CountRules(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxScannerRulesPerClient {
		return nil, fmt.Errorf("%w: max %d scanner rules per client", domain.ErrInvalidArgument, domain.MaxScannerRulesPerClient)
	}

	rule := domain.ScannerRule{
		ID:         uuid.NewString(),
		ClientID:   clientID,
		Type:       typ,
		Conditions: conds,
		MatchMode:  mode,
		Interval:   interval,
		Enabled:    true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	for _, c := range conds {
		if err := applyScannerConditionParams(&rule, c, in); err != nil {
			return nil, err
		}
	}
	return s.store.CreateRule(ctx, rule)
}

func applyScannerConditionParams(rule *domain.ScannerRule, typ domain.ScannerRuleType, in CreateInput) error {
	switch typ {
	case domain.ScannerRuleRSI:
		period := in.RSIPeriod
		if period == 0 {
			period = domain.DefaultRSIPeriod
		}
		if period < domain.MinIndicatorPeriod || period > domain.MaxIndicatorPeriod {
			return fmt.Errorf("%w: rsiPeriod out of range", domain.ErrInvalidArgument)
		}
		cond := domain.AlertCondition(strings.ToLower(strings.TrimSpace(in.RSICondition)))
		if cond == "" {
			cond = domain.AlertBelow
		}
		if !domain.IsValidAlertCondition(string(cond)) {
			return fmt.Errorf("%w: rsiCondition must be above or below", domain.ErrInvalidArgument)
		}
		if in.RSIThreshold < 0 || in.RSIThreshold > 100 || math.IsNaN(in.RSIThreshold) {
			return fmt.Errorf("%w: rsiThreshold must be between 0 and 100", domain.ErrInvalidArgument)
		}
		rule.RSIPeriod = period
		rule.RSICondition = cond
		rule.RSIThreshold = in.RSIThreshold
	case domain.ScannerRuleMACrossover:
		fast, slow := in.MAFastPeriod, in.MASlowPeriod
		if fast == 0 {
			fast = domain.DefaultEMAFast
		}
		if slow == 0 {
			slow = domain.DefaultEMASlow
		}
		if fast < domain.MinIndicatorPeriod || slow < domain.MinIndicatorPeriod ||
			fast > domain.MaxIndicatorPeriod || slow > domain.MaxIndicatorPeriod {
			return fmt.Errorf("%w: ma periods out of range", domain.ErrInvalidArgument)
		}
		if fast >= slow {
			return fmt.Errorf("%w: maFastPeriod must be < maSlowPeriod", domain.ErrInvalidArgument)
		}
		dir := strings.ToLower(strings.TrimSpace(in.MADirection))
		if dir == "" {
			dir = "golden_cross"
		}
		if !domain.IsValidMADirection(dir) {
			return fmt.Errorf("%w: maDirection must be golden_cross or death_cross", domain.ErrInvalidArgument)
		}
		rule.MAFastPeriod = fast
		rule.MASlowPeriod = slow
		rule.MADirection = dir
	case domain.ScannerRuleVolumeIncrease:
		lb := in.VolumeLookback
		if lb == 0 {
			lb = 20
		}
		if lb < domain.MinVolumeLookback || lb > domain.MaxVolumeLookback {
			return fmt.Errorf("%w: volumeLookback out of range", domain.ErrInvalidArgument)
		}
		ratio := in.VolumeMinRatio
		if ratio == 0 {
			ratio = 2
		}
		if ratio < domain.MinVolumeRatio || ratio > domain.MaxVolumeRatio ||
			math.IsNaN(ratio) || math.IsInf(ratio, 0) {
			return fmt.Errorf("%w: volumeMinRatio out of range", domain.ErrInvalidArgument)
		}
		rule.VolumeLookback = lb
		rule.VolumeMinRatio = ratio
	default:
		return fmt.Errorf("%w: unknown scanner condition", domain.ErrInvalidArgument)
	}
	return nil
}

// Get returns one rule.
func (s *Service) Get(ctx context.Context, clientID, id string) (*domain.ScannerRule, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: rule id is required", domain.ErrInvalidArgument)
	}
	return s.store.GetRule(ctx, clientID, id)
}

// List returns rules for a client.
func (s *Service) List(ctx context.Context, clientID string) ([]domain.ScannerRule, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	return s.store.ListRules(ctx, clientID)
}

// Delete removes a rule.
func (s *Service) Delete(ctx context.Context, clientID, id string) error {
	if s.store == nil {
		return fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: rule id is required", domain.ErrInvalidArgument)
	}
	return s.store.DeleteRule(ctx, clientID, id)
}

// ListResults returns match history.
func (s *Service) ListResults(ctx context.Context, clientID string, limit, offset int) ([]domain.ScannerResult, int, error) {
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
	total, err := s.store.CountResults(ctx, clientID)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.store.ListResults(ctx, clientID, limit, offset)
	return list, total, err
}

// ListEnabledRules for the background checker.
func (s *Service) ListEnabledRules(ctx context.Context) ([]domain.ScannerRule, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: scanner store not configured", domain.ErrUpstream)
	}
	return s.store.ListEnabledRules(ctx)
}

// RunOnce evaluates all enabled rules against each client's watchlist.
// Returns number of newly inserted results.
func (s *Service) RunOnce(ctx context.Context) (int, error) {
	if s.store == nil || s.market == nil || s.watch == nil {
		return 0, nil
	}
	rules, err := s.store.ListEnabledRules(ctx)
	if err != nil {
		return 0, err
	}
	if len(rules) == 0 {
		return 0, nil
	}

	// Group rules by client.
	byClient := map[string][]domain.ScannerRule{}
	for _, r := range rules {
		byClient[r.ClientID] = append(byClient[r.ClientID], r)
	}

	// Candle cache: exchange|symbol|interval
	type ck struct{ ex, sym, iv string }
	candleCache := map[ck][]domain.Candle{}

	inserted := 0
	now := time.Now().UTC()
	for clientID, clientRules := range byClient {
		if tenantClosed(ctx, s.account, clientID) {
			continue
		}
		acc, err := s.watch.Get(ctx, clientID, "")
		if err != nil {
			// Skip clients without watchlist
			continue
		}
		if acc == nil || len(acc.Items) == 0 {
			continue
		}
		for _, rule := range clientRules {
			need := candleNeed(rule)
			for _, item := range acc.Items {
				key := ck{ex: string(item.Exchange), sym: item.Symbol, iv: rule.Interval}
				candles, ok := candleCache[key]
				if !ok {
					candles, err = s.market.GetCandles(ctx, key.ex, key.sym, key.iv, need, nil, nil)
					if err != nil {
						candleCache[key] = nil
						continue
					}
					candleCache[key] = candles
				}
				if len(candles) == 0 {
					continue
				}
				match, err := domain.EvaluateScannerRule(rule, candles)
				if err != nil || match == nil {
					continue
				}
				res := domain.ScannerResult{
					ID:            uuid.NewString(),
					ClientID:      clientID,
					RuleID:        rule.ID,
					Exchange:      item.Exchange,
					Symbol:        item.Symbol,
					RuleType:      rule.Type,
					Interval:      rule.Interval,
					MarketDataKey: match.MarketDataKey,
					MatchedAt:     now,
					Summary:       match.Summary,
					Metrics:       match.Metrics,
				}
				_, okIns, err := s.store.InsertResult(ctx, res)
				if err != nil {
					continue
				}
				if okIns {
					inserted++
				}
			}
		}
	}
	return inserted, nil
}

func candleNeed(rule domain.ScannerRule) int {
	switch rule.Type {
	case domain.ScannerRuleRSI:
		p := rule.RSIPeriod
		if p <= 0 {
			p = domain.DefaultRSIPeriod
		}
		return p + 30
	case domain.ScannerRuleMACrossover:
		slow := rule.MASlowPeriod
		if slow <= 0 {
			slow = domain.DefaultEMASlow
		}
		return slow + 30
	case domain.ScannerRuleVolumeIncrease:
		lb := rule.VolumeLookback
		if lb <= 0 {
			lb = 20
		}
		return lb + 5
	default:
		return 100
	}
}

func normalizeClientID(id string) (string, error) {
	return domain.NormalizeClientID(id)
}
