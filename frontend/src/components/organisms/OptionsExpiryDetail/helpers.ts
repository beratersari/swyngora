import type { MarketOptionsExpiryStrike } from '@/libs/api';

export function maxPayout(strikes: MarketOptionsExpiryStrike[] | undefined): number {
  let max = 0;
  for (const s of strikes ?? []) {
    const n = Number(s.payout);
    if (Number.isFinite(n) && n > max) max = n;
  }
  return max;
}

export function payoutPct(value: string | undefined, max: number): number {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0 || max <= 0) return 0;
  return Math.min(100, (n / max) * 100);
}

export function formatMovePct(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return `±${n.toFixed(1)}%`;
}
