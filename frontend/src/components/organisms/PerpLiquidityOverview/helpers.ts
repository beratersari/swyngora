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

/** Minute totals for the depth chart, oldest first. */
export function depthHistoryLine(
  points?: { time?: string; total?: string }[] | null,
): { time: number; value: number }[] {
  const out: { time: number; value: number }[] = [];
  const seen = new Set<number>();
  for (const p of points ?? []) {
    const value = Number(p.total);
    const ms = p.time ? Date.parse(p.time) : NaN;
    if (!Number.isFinite(value) || value <= 0 || !Number.isFinite(ms)) continue;
    const time = Math.floor(ms / 1000);
    if (seen.has(time)) continue;
    seen.add(time);
    out.push({ time, value });
  }
  out.sort((a, b) => a.time - b.time);
  return out;
}

/** BTC closes on the same minutes as the book, oldest first. */
export function depthPriceLine(
  points?: { time?: string; price?: string }[] | null,
): { time: number; value: number }[] {
  const out: { time: number; value: number }[] = [];
  const seen = new Set<number>();
  for (const p of points ?? []) {
    const value = Number(p.price);
    const ms = p.time ? Date.parse(p.time) : NaN;
    if (!Number.isFinite(value) || value <= 0 || !Number.isFinite(ms)) continue;
    const time = Math.floor(ms / 1000);
    if (seen.has(time)) continue;
    seen.add(time);
    out.push({ time, value });
  }
  out.sort((a, b) => a.time - b.time);
  return out;
}

/** Past total bid+ask to now, with the percent change. */
export function depthLine(w?: MarketPerpDepthShareWindow | null): string {
  if (!w?.complete || w.pastTotal == null || w.pastTotal === '' || w.total == null || w.total === '') return '';
  const chg = w.totalChangePct ? ` (${w.totalChangePct}%)` : '';
  return `${formatCompactAmount(w.pastTotal)} → ${formatCompactAmount(w.total)}${chg}`;
}

export type PerpBookMove = {
  kind?: string;
  pastTotal?: string;
  total?: string;
  changePct?: string;
  priceFrom?: string;
  priceTo?: string;
  priceAfter?: string;
  priceChangePct?: string;
  priceAfterPct?: string;
};

/** Past total to the end of a jump or drop, with the percent change. */
export function moveBookLine(m?: PerpBookMove | null): string {
  if (!m?.pastTotal || !m.total) return '';
  const chg = m.changePct ? ` (${m.changePct}%)` : '';
  return `${formatCompactAmount(m.pastTotal)} → ${formatCompactAmount(m.total)}${chg}`;
}

/** BTC from the start of the depth move to the end. */
export function moveDuringLine(m?: PerpBookMove | null): string {
  if (!m?.priceFrom || !m.priceTo) return '';
  const chg = m.priceChangePct ? ` (${m.priceChangePct}%)` : '';
  return `${formatCompactAmount(m.priceFrom)} → ${formatCompactAmount(m.priceTo)}${chg}`;
}

/** Coins that caused most of a jump or drop, with each coin's share of that move. */
export function moveDriverLine(
  m?: { drivers?: { base?: string; symbol?: string; change?: string; sharePct?: string }[] | null } | null,
): string {
  const parts = (m?.drivers ?? [])
    .filter((d) => d.base || d.symbol)
    .slice(0, 3)
    .map((d) => {
      const name = d.base || d.symbol || '';
      const n = Number(d.change);
      const share = d.sharePct ? ` (${d.sharePct}%)` : '';
      if (!Number.isFinite(n)) return `${name}${share}`;
      const body = formatCompactAmount(Math.abs(n));
      if (n > 0) return `${name} +${body}${share}`;
      if (n < 0) return `${name} -${body}${share}`;
      return `${name} ${body}${share}`;
    });
  return parts.filter(Boolean).join(', ');
}

/** BTC about 15 minutes after the depth move ended. */
export function moveAfterLine(m?: PerpBookMove | null): string {
  if (!m?.priceAfter) return '';
  const chg = m.priceAfterPct ? ` (${m.priceAfterPct}%)` : '';
  return `${formatCompactAmount(m.priceAfter)}${chg}`;
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
