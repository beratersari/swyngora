package swing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// CustomScanInput is a watchlist custom-signal pass.
type CustomScanInput struct {
	ClientID  string
	Exchange  string
	Limit     int
	ProfileID string
	Weights   domain.CustomSignalWeights
}

// CustomScanResult is the desk payload for one mix.
type CustomScanResult struct {
	ProfileID string
	Weights   domain.CustomSignalWeights
	Items     []domain.CustomSignalItem
	Count     int
	Note      string
}

// ListFactors returns the mixable catalog.
func (s *Service) ListFactors() []domain.CustomSignalCatalogRow {
	return domain.CustomSignalCatalog()
}

// ListProfiles returns saved mixes for the client.
func (s *Service) ListProfiles(ctx context.Context, clientID string) ([]domain.CustomSignalProfile, error) {
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s == nil || s.profiles == nil {
		return nil, fmt.Errorf("%w: custom signal store not configured", domain.ErrUpstream)
	}
	return s.profiles.ListProfiles(ctx, clientID)
}

// GetProfile loads one saved mix.
func (s *Service) GetProfile(ctx context.Context, clientID, id string) (*domain.CustomSignalProfile, error) {
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if s == nil || s.profiles == nil {
		return nil, fmt.Errorf("%w: custom signal store not configured", domain.ErrUpstream)
	}
	if id == "" {
		return nil, fmt.Errorf("%w: profile id is required", domain.ErrInvalidArgument)
	}
	return s.profiles.GetProfile(ctx, clientID, id)
}

// CreateProfileInput creates a named mix.
type CreateProfileInput struct {
	ClientID string
	Name     string
	Weights  map[string]float64
}

// CreateProfile validates and stores a mix.
func (s *Service) CreateProfile(ctx context.Context, in CreateProfileInput) (*domain.CustomSignalProfile, error) {
	clientID, err := domain.NormalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	if s == nil || s.profiles == nil {
		return nil, fmt.Errorf("%w: custom signal store not configured", domain.ErrUpstream)
	}
	name, err := domain.NormalizeCustomSignalProfileName(in.Name)
	if err != nil {
		return nil, err
	}
	w, err := domain.ParseCustomSignalWeightMap(in.Weights)
	if err != nil {
		return nil, err
	}
	n, err := s.profiles.CountProfiles(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if n >= domain.MaxCustomSignalProfilesPerClient {
		return nil, fmt.Errorf("%w: at most %d custom signal profiles", domain.ErrInvalidArgument, domain.MaxCustomSignalProfilesPerClient)
	}
	now := time.Now().UTC()
	return s.profiles.CreateProfile(ctx, domain.CustomSignalProfile{
		ID: uuid.NewString(), ClientID: clientID, Name: name, Weights: w,
		CreatedAt: now, UpdatedAt: now,
	})
}

// UpdateProfileInput patches name and/or weights.
type UpdateProfileInput struct {
	ClientID   string
	ID         string
	Name       *string
	Weights    map[string]float64
	HasWeights bool
}

// UpdateProfile saves an existing mix.
func (s *Service) UpdateProfile(ctx context.Context, in UpdateProfileInput) (*domain.CustomSignalProfile, error) {
	clientID, err := domain.NormalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	if s == nil || s.profiles == nil {
		return nil, fmt.Errorf("%w: custom signal store not configured", domain.ErrUpstream)
	}
	cur, err := s.profiles.GetProfile(ctx, clientID, in.ID)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		name, err := domain.NormalizeCustomSignalProfileName(*in.Name)
		if err != nil {
			return nil, err
		}
		cur.Name = name
	}
	if in.HasWeights {
		w, err := domain.ParseCustomSignalWeightMap(in.Weights)
		if err != nil {
			return nil, err
		}
		cur.Weights = w
	}
	cur.UpdatedAt = time.Now().UTC()
	return s.profiles.UpdateProfile(ctx, *cur)
}

// DeleteProfile removes a saved mix.
func (s *Service) DeleteProfile(ctx context.Context, clientID, id string) error {
	clientID, err := domain.NormalizeClientID(clientID)
	if err != nil {
		return err
	}
	if s == nil || s.profiles == nil {
		return fmt.Errorf("%w: custom signal store not configured", domain.ErrUpstream)
	}
	if id == "" {
		return fmt.Errorf("%w: profile id is required", domain.ErrInvalidArgument)
	}
	return s.profiles.DeleteProfile(ctx, clientID, id)
}

// ResolveCustomWeights picks inline weights, else a saved profile, else default.
func (s *Service) ResolveCustomWeights(ctx context.Context, clientID, profileID string, inline domain.CustomSignalWeights, hasInline bool) (domain.CustomSignalWeights, string, error) {
	if hasInline {
		if inline.Pct == nil {
			return domain.CustomSignalWeights{}, "", fmt.Errorf("%w: custom signal weights required", domain.ErrInvalidArgument)
		}
		// Re-validate sum so a half-built map cannot sneak through.
		w, err := domain.ParseCustomSignalWeightMap(inline.Pct)
		if err != nil {
			return domain.CustomSignalWeights{}, "", err
		}
		return w, profileID, nil
	}
	if profileID != "" {
		p, err := s.GetProfile(ctx, clientID, profileID)
		if err != nil {
			return domain.CustomSignalWeights{}, "", err
		}
		return p.Weights, p.ID, nil
	}
	return domain.DefaultCustomSignalWeights(), "", nil
}

// ScanCustom evaluates the watchlist with a custom mix.
func (s *Service) ScanCustom(ctx context.Context, in CustomScanInput) (*CustomScanResult, error) {
	if s == nil || s.market == nil || s.watch == nil {
		return nil, fmt.Errorf("%w: swing service not configured", domain.ErrUpstream)
	}
	clientID, err := domain.NormalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	w := in.Weights
	if w.Pct == nil {
		w = domain.DefaultCustomSignalWeights()
	}
	if in.Limit <= 0 || in.Limit > maxScan {
		in.Limit = maxScan
	}
	wl, err := s.watch.Get(ctx, clientID, "")
	if err != nil {
		return nil, err
	}
	var items []domain.WatchlistItem
	wantEx := in.Exchange
	for _, it := range wl.Items {
		if wantEx != "" && string(it.Exchange) != wantEx {
			continue
		}
		items = append(items, it)
		if len(items) >= in.Limit {
			break
		}
	}

	out := make([]domain.CustomSignalItem, 0, len(items))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	needHunt := w.NeedsHunt()
	for _, it := range items {
		it := it
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			row, err := s.scoreOne(ctx, it.Exchange, it.Symbol, w, needHunt)
			if err != nil {
				return
			}
			mu.Lock()
			out = append(out, row)
			mu.Unlock()
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	domain.SortCustomSignalItems(out)
	return &CustomScanResult{
		ProfileID: in.ProfileID,
		Weights:   w,
		Items:     out,
		Count:     len(out),
		Note:      "Informational only — not financial advice. Custom mix does not apply engine quality gates.",
	}, nil
}

func (s *Service) scoreOne(ctx context.Context, ex domain.Exchange, symbol string, w domain.CustomSignalWeights, needHunt bool) (domain.CustomSignalItem, error) {
	dec, err := s.analyzeResolved(ctx, ex, symbol)
	if err != nil {
		return domain.CustomSignalItem{}, err
	}
	var hunt *domain.CustomSignalRaw
	var huntDetail string
	if needHunt {
		raw := s.loadHuntRaw(ctx, symbol)
		hunt = &raw
		huntDetail = raw.Detail
	}
	raws := domain.BuildCustomSignalRaws(dec.Patterns, hunt)
	custom := domain.ScoreCustomSignal(raws, w)
	return domain.CustomSignalItem{
		Exchange:   ex,
		Symbol:     symbol,
		Interval:   dec.Interval,
		Price:      dec.Price,
		BarTime:    dec.BarTime,
		Engine:     dec,
		Custom:     custom,
		Compare:    domain.CompareCustomSignal(raws, w),
		HuntDetail: huntDetail,
	}, nil
}

func (s *Service) loadHuntRaw(ctx context.Context, symbol string) domain.CustomSignalRaw {
	if s.hunt == nil {
		return domain.CustomSignalRaw{Score: 50, Status: domain.CustomFactorMissing, Detail: "hunt not configured"}
	}
	rep, err := s.hunt.GetLiquidationHunt(ctx, "all", symbol)
	if err != nil {
		return domain.CustomSignalRaw{Score: 50, Status: domain.CustomFactorMissing, Detail: "hunt unavailable"}
	}
	return domain.HuntUpRaw(rep)
}
