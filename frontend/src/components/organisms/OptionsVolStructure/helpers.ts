import type { MarketOptionsVolPoint } from '@/libs/api';

export function maxAtmIv(points: MarketOptionsVolPoint[] | undefined): number {
  let max = 0;
  for (const p of points ?? []) {
    const n = Number(p.atmIv);
    if (Number.isFinite(n) && n > max) max = n;
  }
  return max;
}

export function ivBarPct(value: string | undefined, max: number): number {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0 || max <= 0) return 0;
  return Math.min(100, (n / max) * 100);
}

export function formatIvPct(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  return `${n.toFixed(1)}%`;
}

export function formatSkewPts(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  const sign = n > 0 ? '+' : '';
  return `${sign}${n.toFixed(1)}`;
}

export function maxCallPutIv(points: MarketOptionsVolPoint[] | undefined): number {
  let max = 0;
  for (const p of points ?? []) {
    for (const raw of [p.callIv, p.putIv, p.atmIv]) {
      const n = Number(raw);
      if (Number.isFinite(n) && n > max) max = n;
    }
  }
  return max;
}
