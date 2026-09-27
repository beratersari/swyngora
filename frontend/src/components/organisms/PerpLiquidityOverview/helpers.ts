import type {
  MarketPerpDepthOverview,
  MarketPerpDepthOverviewBand,
  MarketPerpDepthShareWindow,
} from '@/libs/api';
import { formatCompactAmount } from '@/libs/utils';
import { PERP_LIQUIDITY_LEADERS } from './constants';

/** Bid as a percent of bid+ask. The gauge needle uses this. */
export function bidSharePct(data?: MarketPerpDepthOverview | null): number | null {
  const bid = Number(data?.bid);
  const ask = Number(data?.ask);
  if (!Number.isFinite(bid) || !Number.isFinite(ask) || bid + ask <= 0) return null;
  return (bid / (bid + ask)) * 100;
}

export function leaderCoins(data?: MarketPerpDepthOverview | null) {
  const coins = data?.coins ?? [];
  return coins.filter((c) => !c.error && Number(c.total) > 0).slice(0, PERP_LIQUIDITY_LEADERS);
}

export function shareLabel(pct?: string | null): string {
  if (!pct) return '—';
  return `${pct}%`;
}

export function mixLine(w?: MarketPerpDepthShareWindow | null): string {
  if (!w?.complete || !w.pastBidPct || !w.bidPct) return '';
  const chg = w.changePct ? ` (${w.changePct})` : '';
  return `${w.pastBidPct}% → ${w.bidPct}%${chg}`;
}

/** Past total bid+ask to now, with the percent change. */
export function depthLine(w?: MarketPerpDepthShareWindow | null): string {
  if (!w?.complete || w.pastTotal == null || w.pastTotal === '' || w.total == null || w.total === '') return '';
  const chg = w.totalChangePct ? ` (${w.totalChangePct}%)` : '';
  return `${formatCompactAmount(w.pastTotal)} → ${formatCompactAmount(w.total)}${chg}`;
}

/** Coins that added or removed the most bid+ask, with the signed amount. */
export function sizeDriverLine(w?: MarketPerpDepthShareWindow | null): string {
  const parts = (w?.sizeDrivers ?? [])
    .filter((d) => d.base || d.symbol)
    .slice(0, 3)
    .map((d) => {
      const n = Number(d.change);
      const name = d.base || d.symbol || '';
      if (!Number.isFinite(n)) return name;
      const body = formatCompactAmount(Math.abs(n));
      if (n > 0) return `${name} +${body}`;
      if (n < 0) return `${name} -${body}`;
      return `${name} ${body}`;
    });
  return parts.filter(Boolean).join(', ');
}

export function driverNames(w?: MarketPerpDepthShareWindow | null): string {
  const names = (w?.drivers ?? [])
    .filter((d) => d.base || d.symbol)
    .slice(0, 3)
    .map((d) => d.base || d.symbol || '');
  return names.filter(Boolean).join(', ');
}

export function distanceBands(data?: MarketPerpDepthOverview | null): MarketPerpDepthOverviewBand[] {
  return (data?.bands ?? []).filter((b) => Number(b.total) > 0);
}

export function bandShareLabel(b?: MarketPerpDepthOverviewBand | null, headline?: string | null): string {
  if (!b) return '—';
  const amt = formatCompactAmount(b.total);
  if (!b.ofHeadlinePct || b.rangePct === headline) return amt;
  return `${amt} · ${b.ofHeadlinePct}%`;
}

export function bandLeaderLabel(b?: MarketPerpDepthOverviewBand | null): string {
  return (b?.leaders ?? []).filter(Boolean).slice(0, 3).join(', ');
}
