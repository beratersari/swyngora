import type { MarketOptionsExpiry } from '@/libs/api';

export function expiryShare(row: MarketOptionsExpiry | undefined): number {
  const n = Number(row?.sharePct);
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.min(100, n);
}

export function callShare(row: MarketOptionsExpiry | undefined): number {
  const call = Number(row?.callOi);
  const put = Number(row?.putOi);
  const total = call + put;
  if (!Number.isFinite(total) || total <= 0) return 0.5;
  return call / total;
}
