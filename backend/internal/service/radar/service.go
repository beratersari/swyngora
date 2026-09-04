package radar

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Market supplies scan hits, last price, and candles for after-fills.
type Market interface {
	ScanForensics(ctx context.Context, exchange, quote, lookback, interval, direction string, minReturnPct float64, symbolLimit int, window, during, fields, weights, minCoverage, horizons string) (*domain.ForensicsScanReport, error)
	GetTicker24h(ctx context.Context, exchange, symbol string) (*domain.Ticker24h, error)
	GetCandles(ctx context.Context, exchange, symbol, interval string, limit int, start, end *time.Time) ([]domain.Candle, error)
}

// Service records Setup Radar hits and fills what price did after.
type Service struct {
	store domain.RadarCasePort
	mkt   Market
	Log   *slog.Logger
	now   func() time.Time
}

// New constructs a radar history service.
func New(store domain.RadarCasePort, mkt Market) *Service {
	return &Service{store: store, mkt: mkt, now: time.Now}
}

func (s *Service) clock() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) log() *slog.Logger {
	if s != nil && s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// RecordHits persists new scan appearances (one per coin per cooldown).
func (s *Service) RecordHits(ctx context.Context, report *domain.ForensicsScanReport) (int, error) {
	if s == nil || s.store == nil || report == nil {
		return 0, nil
	}
	now := report.AsOf
	if now.IsZero() {
		now = s.clock()
	}
	n := 0
	for _, hit := range report.Hits {
		if strings.TrimSpace(hit.Symbol) == "" {
			continue
		}
		ex := hit.Exchange
		if ex == "" {
			ex = report.Exchange
		}
		price := s.spotPrice(ctx, ex, hit.Symbol)
		c := domain.NewRadarCase(hit, now, price, report.Lookback, report.Window)
		c.ID = uuid.NewString()
		if c.Exchange == "" || c.Exchange == "all" {
			c.Exchange = strings.ToLower(strings.TrimSpace(ex))
		}
		_, inserted, err := s.store.InsertIfQuiet(ctx, c, domain.DefaultRadarCooldown)
		if err != nil {
			s.log().Debug("radar persist hit failed", "symbol", hit.Symbol, "err", err)
			continue
		}
		if inserted {
			n++
		}
	}
	return n, nil
}

// ScanAndRecord runs the live Setup Radar scan and saves new hits.
func (s *Service) ScanAndRecord(ctx context.Context) (int, error) {
	if s == nil || s.mkt == nil {
		return 0, fmt.Errorf("%w: market not configured", domain.ErrUpstream)
	}
	rep, err := s.mkt.ScanForensics(ctx, "all", "", domain.DefaultForensicsScanLookback, "", "", 0, 0, "", "", "", "", "", "")
	if err != nil {
		return 0, err
	}
	return s.RecordHits(ctx, rep)
}

// Review lists saved hits and how useful they were.
func (s *Service) Review(ctx context.Context, exchange, symbol, since string, limit int) (*domain.RadarReview, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("%w: radar store not configured", domain.ErrUpstream)
	}
	now := s.clock()
	from, sinceID, err := domain.ParseRadarSince(since, now)
	if err != nil {
		return nil, err
	}
	cases, err := s.store.List(ctx, domain.RadarListQuery{
		Exchange: exchange,
		Symbol:   symbol,
		Since:    from,
		Limit:    domain.ClampRadarListLimit(limit),
	})
	if err != nil {
		return nil, err
	}
	out := &domain.RadarReview{
		Exchange: strings.ToLower(strings.TrimSpace(exchange)),
		Symbol:   strings.ToUpper(strings.TrimSpace(symbol)),
		Since:    sinceID,
		From:     from,
		Limit:    domain.ClampRadarListLimit(limit),
		AsOf:     now,
		Cases:    cases,
	}
	if out.Exchange == "" {
		out.Exchange = "all"
	}
	domain.FinishRadarReview(out)
	return out, nil
}

// FillDue writes after-returns for horizons that have elapsed.
func (s *Service) FillDue(ctx context.Context, now time.Time, limit int) (int, error) {
	if s == nil || s.store == nil || s.mkt == nil {
		return 0, nil
	}
	if now.IsZero() {
		now = s.clock()
	}
	if limit <= 0 {
		limit = 40
	}
	due, err := s.store.ListDueAfters(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range due {
		px, ok := s.afterPrice(ctx, d)
		if !ok {
			continue
		}
		found := d.FoundPrice
		if found <= 0 {
			found = s.spotPrice(ctx, d.Exchange, d.Symbol)
			if found > 0 {
				_ = s.store.SetFoundPrice(ctx, d.CaseID, found)
			}
		}
		if found <= 0 {
			continue
		}
		fill := domain.FillRadarAfter(found, px, d.Lean, now)
		fill.Horizon = d.Horizon
		if err := s.store.FillAfter(ctx, d.CaseID, d.Horizon, fill); err != nil {
			s.log().Debug("radar fill after failed", "id", d.CaseID, "horizon", d.Horizon, "err", err)
			continue
		}
		n++
	}
	return n, nil
}

func (s *Service) spotPrice(ctx context.Context, exchange, symbol string) float64 {
	if s.mkt == nil {
		return 0
	}
	ex := exchange
	if ex == "" || ex == "all" {
		ex = string(domain.ExchangeBinance)
	}
	tkr, err := s.mkt.GetTicker24h(ctx, ex, symbol)
	if err != nil || tkr == nil {
		if ex != string(domain.ExchangeBybit) {
			tkr, err = s.mkt.GetTicker24h(ctx, string(domain.ExchangeBybit), symbol)
		}
	}
	if err != nil || tkr == nil {
		return 0
	}
	px, err := strconv.ParseFloat(strings.TrimSpace(tkr.LastPrice), 64)
	if err != nil || px <= 0 {
		return 0
	}
	return px
}

func (s *Service) afterPrice(ctx context.Context, d domain.RadarAfterDue) (float64, bool) {
	ex := d.Exchange
	if ex == "" || ex == "all" {
		ex = string(domain.ExchangeBinance)
	}
	interval := "1m"
	if d.DueAt.Sub(d.FoundAt) > time.Hour {
		interval = "15m"
	}
	start := d.FoundAt.UTC()
	end := d.DueAt.UTC().Add(2 * time.Minute)
	bars, err := s.mkt.GetCandles(ctx, ex, d.Symbol, interval, 200, &start, &end)
	if err != nil || len(bars) == 0 {
		return 0, false
	}
	return domain.AfterPriceFromCandles(bars, d.DueAt)
}
