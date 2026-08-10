import type { OrderBookLevel } from '@/libs/api';

export function maxNotional(levels: OrderBookLevel[] | undefined): number {
  let max = 0;
  for (const lv of levels ?? []) {
    const n = Number.parseFloat(lv.notional ?? '0');
    if (Number.isFinite(n) && n > max) max = n;
  }
  return max;
}

export function depthPct(notional: string | undefined, max: number): number {
  if (max <= 0) return 0;
  const n = Number.parseFloat(notional ?? '0');
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.min(100, (n / max) * 100);
}

/** Asks high→low so the spread sits in the middle of the ladder. */
export function asksHighToLow(asks: OrderBookLevel[] | undefined): OrderBookLevel[] {
  if (!asks?.length) return [];
  return [...asks].reverse();
}
