import type { TFunction } from 'i18next';
import type { ExecutionQuote } from '@/libs/api';
import type { ExecutionStyleId } from './constants';

/** Plain sentence for a path that would not fill. */
export function unfilledReason(
  t: TFunction<'portfolio'>,
  status: ExecutionQuote['status'],
  schedule: ExecutionQuote['schedule'],
): string {
  if (status === 'schedule') {
    if (schedule === 'interval') return t('compare.schedule.interval');
    if (schedule === 'size') return t('compare.schedule.size');
    return t('compare.schedule.range');
  }
  if (status === 'slippage_limited') return t('compare.status.slippage_limited');
  if (status === 'inventory_short') return t('compare.status.inventory_short');
  if (status === 'missing') return t('compare.status.missing');
  return t('compare.status.exhausted');
}

/** Percent with up to three decimals. 0.5 means 0.5%. */
export function formatComparePct(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) return '—';
  const rounded = Math.round(value * 1000) / 1000;
  return `${rounded}%`;
}

function cashScore(side: string | undefined, row: ExecutionQuote): number {
  if (side === 'sell') return -(row.cashCredit ?? 0);
  return row.cashDebit ?? 0;
}

/**
 * Styles that spend the least on a buy, or return the most on a sell.
 * A single fillable path, or a tie across every fillable path, is not marked.
 */
export function cheapestStyles(
  side: string | undefined,
  rows: ExecutionQuote[] | null | undefined,
): Set<ExecutionStyleId> {
  const filled = (rows ?? []).filter((row) => row.status === 'filled' && row.style);
  if (filled.length < 2) return new Set();
  const best = Math.min(...filled.map((row) => cashScore(side, row)));
  const winners = filled.filter((row) => Math.abs(cashScore(side, row) - best) < 1e-8);
  if (winners.length === filled.length) return new Set();
  return new Set(winners.map((row) => row.style as ExecutionStyleId));
}

/** Venue names that would receive size, in walk order. */
export function legVenues(row: ExecutionQuote): string {
  const names: string[] = [];
  for (const leg of row.legs ?? []) {
    const name = leg.exchange?.trim();
    if (name && !names.includes(name)) names.push(name);
  }
  return names.join(', ');
}
