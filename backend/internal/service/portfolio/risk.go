package portfolio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const riskNote = "Risk limits only block new spot buys and new margin opens. Existing positions are never closed or reduced. You can change or remove limits anytime. Paper trading — not real money. Not financial advice."

// RiskLimitsInput replaces both optional rules. Nil field = that rule off.
type RiskLimitsInput struct {
	ClientID          string
	PortfolioID       string
	MaxDailyLossPct   *float64
	MaxAssetWeightPct *float64
}

// RiskLimitsView is GET payload for the settings screen (limits + live status).
type RiskLimitsView struct {
	Limits domain.RiskLimits
	Status domain.RiskStatus
	Note   string
}

// GetRiskLimitsView loads rules (or empty) and live block status for the UI.
func (s *Service) GetRiskLimitsView(ctx context.Context, clientID string, portfolioID ...string) (*RiskLimitsView, error) {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return nil, err
	}
	clientID = p.BookID()
	lim, snap, err := s.loadRiskWithDaySnapshot(ctx, clientID)
	if err != nil {
		return nil, err
	}
	st, err := s.buildRiskStatus(ctx, clientID, lim, snap)
	if err != nil {
		return nil, err
	}
	return &RiskLimitsView{Limits: lim, Status: st, Note: riskNote}, nil
}

// SetRiskLimits saves or updates optional rules. Nil percents disable that rule.
func (s *Service) SetRiskLimits(ctx context.Context, in RiskLimitsInput) (*RiskLimitsView, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: portfolio store not configured", domain.ErrUpstream)
	}
	p, err := s.requireBook(ctx, in.ClientID, in.PortfolioID)
	if err != nil {
		return nil, err
	}
	clientID := p.BookID()
	if err := domain.ValidateOptionalRiskPct(in.MaxDailyLossPct, "maxDailyLossPct"); err != nil {
		return nil, err
	}
	if err := domain.ValidateOptionalRiskPct(in.MaxAssetWeightPct, "maxAssetWeightPct"); err != nil {
		return nil, err
	}
	_, snap, err := s.loadRiskWithDaySnapshot(ctx, clientID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	saved, err := s.store.UpsertRiskLimits(ctx, domain.RiskLimits{
		ClientID: clientID, MaxDailyLossPct: in.MaxDailyLossPct, MaxAssetWeightPct: in.MaxAssetWeightPct,
		DayKey: snap.DayKey, DayStartEquity: snap.DayStartEquity, UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	st, err := s.buildRiskStatus(ctx, clientID, *saved, *saved)
	if err != nil {
		return nil, err
	}
	return &RiskLimitsView{Limits: *saved, Status: st, Note: riskNote}, nil
}

// ClearRiskLimits removes all rules (idempotent).
func (s *Service) ClearRiskLimits(ctx context.Context, clientID string, portfolioID ...string) error {
	p, err := s.requireBook(ctx, clientID, portfolioID...)
	if err != nil {
		return err
	}
	return s.store.DeleteRiskLimits(ctx, p.BookID())
}

// guardNewRisk blocks new spot buys / new margin opens when a user limit is hit.
// additionalNotional is the cash/notional the new order would add to `asset` (base ticker).
func (s *Service) guardNewRisk(ctx context.Context, clientID, asset string, additionalNotional float64) error {
	existing, err := s.store.GetRiskLimits(ctx, clientID)
	if err == domain.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.MaxDailyLossPct == nil && existing.MaxAssetWeightPct == nil {
		return nil
	}
	lim, snap, err := s.loadRiskWithDaySnapshot(ctx, clientID)
	if err != nil {
		return err
	}
	st, err := s.buildRiskStatus(ctx, clientID, lim, snap)
	if err != nil {
		return err
	}
	var assetVal float64
	asset = domain.NormalizeAllocationAsset(asset)
	for _, a := range st.Assets {
		if a.Asset == asset {
			assetVal = a.Value
			break
		}
	}
	reasons := domain.RiskLimitMessages(lim, st.StartOfDayEquity, st.Equity, asset, assetVal, additionalNotional)
	if len(reasons) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", domain.ErrForbidden, strings.Join(reasons, "; "))
}

func (s *Service) loadRiskWithDaySnapshot(ctx context.Context, clientID string) (domain.RiskLimits, domain.RiskLimits, error) {
	lim, err := s.store.GetRiskLimits(ctx, clientID)
	if err == domain.ErrNotFound {
		lim = &domain.RiskLimits{ClientID: clientID}
	} else if err != nil {
		return domain.RiskLimits{}, domain.RiskLimits{}, err
	}
	view, err := s.View(ctx, clientID)
	if err != nil {
		return domain.RiskLimits{}, domain.RiskLimits{}, err
	}
	today := domain.UTCDayKey(time.Now().UTC())
	if lim.DayKey != today || lim.DayStartEquity <= 0 {
		lim.DayKey = today
		lim.DayStartEquity = view.Equity
		now := time.Now().UTC()
		lim.UpdatedAt = now
		// Persist snapshot even when no rules yet so the day baseline is stable.
		saved, uerr := s.store.UpsertRiskLimits(ctx, *lim)
		if uerr != nil {
			return domain.RiskLimits{}, domain.RiskLimits{}, uerr
		}
		lim = saved
	}
	return *lim, *lim, nil
}

func (s *Service) buildRiskStatus(ctx context.Context, clientID string, lim, snap domain.RiskLimits) (domain.RiskStatus, error) {
	view, err := s.View(ctx, clientID)
	if err != nil {
		return domain.RiskStatus{}, err
	}
	weights := s.assetExposureWeights(view)
	st := domain.RiskStatus{
		DayKey:           snap.DayKey,
		StartOfDayEquity: snap.DayStartEquity,
		Equity:           view.Equity,
		DailyPnL:         view.Equity - snap.DayStartEquity,
		DailyPnLPct:      domain.DailyPnLPct(snap.DayStartEquity, view.Equity),
		CanOpenSpotBuy:   true,
		CanOpenMargin:    true,
	}
	if lim.MaxDailyLossPct != nil && domain.DailyLossLimitHit(snap.DayStartEquity, view.Equity, *lim.MaxDailyLossPct) {
		st.DailyLossLimitHit = true
		st.CanOpenSpotBuy = false
		st.CanOpenMargin = false
		st.BlockReasons = append(st.BlockReasons, fmt.Sprintf("daily loss limit reached (%.2f%% loss >= %.2f%%)",
			mathAbs(st.DailyPnLPct), *lim.MaxDailyLossPct))
	}
	maxW := 0.0
	if lim.MaxAssetWeightPct != nil {
		maxW = *lim.MaxAssetWeightPct
	}
	for _, w := range weights {
		if maxW > 0 && w.WeightPct > maxW+1e-9 {
			w.AtOrOverLimit = true
		}
		st.Assets = append(st.Assets, w)
	}
	if !st.DailyLossLimitHit {
		// Concentration does not globally disable all buys — only the heavy coin (checked per order).
		st.CanOpenSpotBuy = true
		st.CanOpenMargin = true
	}
	return st, nil
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func (s *Service) assetExposureWeights(view *domain.PortfolioView) []domain.RiskAssetWeight {
	if view == nil || view.Equity <= domain.PositionEpsilon {
		return nil
	}
	quote := domain.NormalizeAllocationAsset(view.Currency)
	by := map[string]float64{}
	for _, pos := range view.Positions {
		base, q := domain.SplitBaseQuote(pos.Exchange, pos.Symbol)
		if base == "" || (q != "" && q != quote) {
			continue
		}
		by[base] += pos.MarketValue
	}
	for _, mp := range view.MarginPositions {
		if mp.Status != domain.MarginPositionOpen {
			continue
		}
		base, q := domain.SplitBaseQuote(mp.Exchange, mp.Symbol)
		if base == "" || (q != "" && q != quote) {
			continue
		}
		by[base] += mp.Quantity * mp.MarkPrice
	}
	out := make([]domain.RiskAssetWeight, 0, len(by))
	for a, v := range by {
		out = append(out, domain.RiskAssetWeight{
			Asset: a, Value: v, WeightPct: 100 * v / view.Equity,
		})
	}
	return out
}
