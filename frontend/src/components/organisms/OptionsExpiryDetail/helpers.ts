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

export function formatSignedCompact(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  const abs = Math.abs(n);
  const sign = n < 0 ? '−' : n > 0 ? '+' : '';
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(2)}K`;
  if (abs === 0) return '0';
  return `${sign}${abs.toFixed(0)}`;
}

export function formatMovePct(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return `±${n.toFixed(1)}%`;
}
