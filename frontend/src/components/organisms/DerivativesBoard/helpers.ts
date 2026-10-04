import { formatCompactAmount, formatPrice } from '@/libs/utils';
import type { DerivativesCoin } from '@/libs/api';
import { BOARD_SORT_KEYS, type BoardSort, type BoardSortKey } from './DerivativesBoard.types';

const DASH = '—';

export function coinChartPath(exchange: string, symbol: string): string {
  const venue = !exchange || exchange === 'all' ? 'binance' : exchange;
  return `/markets/${venue}/${symbol}`;
}

export function definiteText(definite: boolean | undefined, value: string | undefined, format: (value: string) => string): string {
  if (!definite || value == null || value === '') return DASH;
  return format(value);
}

export function plainAmount(value: string): string {
  return formatCompactAmount(value);
}

export function signedAmount(value: string): string {
  const n = Number(value);
  if (!Number.isFinite(n)) return DASH;
  const body = formatCompactAmount(Math.abs(n));
  if (n > 0) return `+${body}`;
  if (n < 0) return `-${body}`;
  return body;
}

export function priceText(coin: DerivativesCoin): string {
  if (!coin.price) return DASH;
  return formatPrice(coin.price);
}

export function fundingText(coin: DerivativesCoin): string {
  if (!coin.fundingDefinite || !coin.fundingRatePct) return DASH;
  return `${coin.fundingRatePct}%`;
}

export type RecordedSpan = {
  kind?: string;
  exchange?: string;
  spanSec?: number;
};

const RECORDED_KINDS = ['futures_oi', 'book', 'flow', 'liquidations'] as const;

/** Longest stored span for each series, so the board cell stays one line. */
export function recordedSummary(
  rows: readonly RecordedSpan[] | undefined,
  label: (kind: string) => string,
  span: (sec: number) => string,
): string {
  if (!rows || rows.length === 0) return DASH;
  const parts: string[] = [];
  for (const kind of RECORDED_KINDS) {
    let longest = 0;
    for (const row of rows) {
      if (row.kind !== kind) continue;
      const sec = row.spanSec ?? 0;
      if (sec > longest) longest = sec;
    }
    if (longest > 0) parts.push(`${label(kind)} ${span(longest)}`);
  }
  return parts.length > 0 ? parts.join(' · ') : DASH;
}

/** Per-venue spans for the cell title. */
export function recordedTitle(rows: readonly RecordedSpan[] | undefined, span: (sec: number) => string): string {
  return (rows ?? [])
    .filter((row) => row.kind && row.exchange && (row.spanSec ?? 0) > 0)
    .map((row) => `${row.kind} ${row.exchange} ${span(row.spanSec ?? 0)}`)
    .join(', ');
}

export function formatRecordedSpan(sec: number): string {
  if (!Number.isFinite(sec) || sec <= 0) return '0m';
  if (sec < 3600) return `${Math.max(1, Math.round(sec / 60))}m`;
  if (sec < 48 * 3600) return `${Math.round(sec / 3600)}h`;
  const days = Math.floor(sec / 86400);
  const hours = Math.round((sec % 86400) / 3600);
  return hours > 0 ? `${days}d ${hours}h` : `${days}d`;
}

export function percentText(definite: boolean | undefined, value: string | undefined): string {
  return definiteText(definite, value, (raw) => `${raw}%`);
}

export function isBoardSortKey(value: string): value is BoardSortKey {
  return (BOARD_SORT_KEYS as readonly string[]).includes(value);
}

export function sortBoardCoins(coins: DerivativesCoin[], sort: BoardSort): DerivativesCoin[] {
  const next = [...coins];
  next.sort((a, b) => compareCoins(a, b, sort));
  return next;
}

function compareCoins(a: DerivativesCoin, b: DerivativesCoin, sort: BoardSort): number {
  if (sort.key === 'base') {
    const cmp = (a.base || a.symbol || '').localeCompare(b.base || b.symbol || '');
    return sort.dir === 'asc' ? cmp : -cmp;
  }
  const av = sortNumber(a, sort.key);
  const bv = sortNumber(b, sort.key);
  if (av == null && bv == null) return (a.symbol || '').localeCompare(b.symbol || '');
  if (av == null) return 1;
  if (bv == null) return -1;
  const cmp = av - bv;
  return sort.dir === 'asc' ? cmp : -cmp;
}

function sortNumber(coin: DerivativesCoin, key: BoardSortKey): number | null {
  switch (key) {
    case 'base':
      return null;
    case 'price':
      return finite(coin.price);
    case 'funding':
      return coin.fundingDefinite ? finite(coin.fundingRatePct) : null;
    case 'openInterest':
      return coin.openInterestDefinite ? finite(coin.openInterest) : null;
    case 'change1h':
      return coin.change1hDefinite ? finite(coin.change1h) : null;
    case 'change24h':
      return coin.change24hDefinite ? finite(coin.change24h) : null;
    case 'liqLong':
      return coin.liqDefinite ? finite(coin.liqLong) : null;
    case 'liqShort':
      return coin.liqDefinite ? finite(coin.liqShort) : null;
    case 'longPct':
      return coin.longDefinite ? finite(coin.longPct) : null;
    case 'takerBuyPct':
      return coin.takerDefinite ? finite(coin.takerBuyPct) : null;
    case 'gapPct':
      return coin.gapDefinite ? finite(coin.gapPct) : null;
    case 'cvdDay':
      return coin.cvdDayDefinite ? finite(coin.cvdDay) : null;
    case 'printChange':
      return coin.printDefinite ? finite(coin.printChangePct) : null;
    case 'liqHeat':
      return coin.liqHeatDefinite ? finite(coin.liqHeatRatio) : null;
    case 'bidPct':
      return coin.bidDefinite ? finite(coin.bidPct) : null;
    default:
      return null;
  }
}

function finite(value: string | undefined): number | null {
  if (value == null || value === '') return null;
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}
