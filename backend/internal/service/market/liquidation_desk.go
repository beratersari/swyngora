package market

import (
	"context"
	"log/slog"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const liquidationStoredNote = "Time bars are stored force-order prints for every coin on the selected venues, plus prints newer than the newest saved row. A zero bar was quiet or was not recorded. The range is the one requested, up to 30 days. Informational only."

func liquidationDeskVenues(exchange string) []domain.Exchange {
	if exchange != "" && exchange != "all" {
		return []domain.Exchange{domain.Exchange(exchange)}
	}
	return domain.FuturesVenues()
}

func (s *Service) liquidationArchive() domain.LiquidationDeskArchive {
	if s == nil || s.futHist == nil {
		return nil
	}
	arch, _ := s.futHist.(domain.LiquidationDeskArchive)
	return arch
}

// applyStoredLiquidationOverview replaces memory totals with the print archive.
// Feed health stays on the live book. A store error leaves the memory overview.
func (s *Service) applyStoredLiquidationOverview(ctx context.Context, out *domain.LiquidationOverview, exchange, window string, limit int) *domain.LiquidationOverview {
	arch := s.liquidationArchive()
	if arch == nil || out == nil {
		return out
	}
	now := time.Now().UTC()
	venues := liquidationDeskVenues(exchange)
	var tail []domain.LiquidationEvent
	if s.liq != nil {
		tail = s.liq.EventsSince(exchange, "all", now.Add(-24*time.Hour))
	}
	cov := time.Duration(0)
	if s.liq != nil {
		cov = s.liq.MarketCoverage(exchange)
	}
	windows := make([]domain.LiquidationWindowTotals, 0, len(domain.LiquidationOverviewWindows))
	var coinRows []domain.LiquidationSymbolSum
	for _, w := range domain.LiquidationOverviewWindows {
		from := now.Add(-w.Dur)
		scan, err := arch.SumLiquidationSymbols(ctx, from, now, venues)
		if err != nil {
			slog.Warn("liquidation overview archive", "window", w.ID, "err", err)
			return out
		}
		rows := domain.MergeLiquidationSymbolTail(scan.Rows, scan.Latest, tail, from, now)
		big := domain.MergeLiquidationBiggest(scan.Biggest, tail, scan.Latest, from, now)
		windows = append(windows, domain.LiquidationWindowFromSums(rows, w.ID, cov, big))
		if w.ID == window {
			coinRows = rows
		}
	}
	out.Windows = windows
	out.Coins = domain.LiquidationCoinsFromSums(coinRows, limit)
	return out
}

// storedLiquidationBars reads all-coin time bars. ok is false when this
// process has no print archive, so the caller keeps the 24-hour memory chart.
func (s *Service) storedLiquidationBars(ctx context.Context, exchange string, from, to time.Time) ([]domain.LiquidationTimeBar, time.Time, bool) {
	arch := s.liquidationArchive()
	if arch == nil {
		return nil, from, false
	}
	origin, until, step, n := domain.LiquidationBarGrid(from, to, domain.LiquidationTimeBarStep(to.Sub(from)))
	scan, err := arch.BucketLiquidations(ctx, origin, until, step, liquidationDeskVenues(exchange))
	if err != nil {
		slog.Warn("liquidation chart archive", "err", err)
		return nil, from, false
	}
	var tail []domain.LiquidationEvent
	if s.liq != nil {
		tail = s.liq.EventsSince(exchange, "all", origin)
	}
	return domain.LiquidationTimeBarsFromBuckets(scan, tail, origin, step, n), origin, true
}
