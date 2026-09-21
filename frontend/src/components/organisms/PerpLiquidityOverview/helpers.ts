import type {
  MarketPerpDepthOverview,
  MarketPerpDepthOverviewBand,
  MarketPerpDepthShareWindow,
} from '@/libs/api';
import { formatCompactAmount } from '@/libs/utils';
import { PERP_LIQUIDITY_LEADERS } from './constants';

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
