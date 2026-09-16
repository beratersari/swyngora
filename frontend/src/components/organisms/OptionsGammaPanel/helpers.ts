import type { MarketOptionsGamma, MarketOptionsGammaLevel } from '@/libs/api';

export function formatGex(value: string | undefined): string {
  if (value == null || value === '') return '—';
  const n = Number(value);
  if (!Number.isFinite(n)) return '—';
  const abs = Math.abs(n);
  const sign = n < 0 ? '−' : n > 0 ? '+' : '';
  if (abs >= 1e12) return `${sign}${(abs / 1e12).toFixed(2)}T`;
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(2)}K`;
  if (abs === 0) return '0';
  return `${sign}${abs.toFixed(0)}`;
}

export function formatFlipDist(value: string | undefined): string | undefined {
  if (value == null || value === '') return undefined;
  const n = Number(value);
  if (!Number.isFinite(n) || n === 0) return undefined;
  const sign = n > 0 ? '+' : '−';
  return `${sign}${Math.abs(n).toFixed(2)}%`;
}

export function maxAbsGex(levels: MarketOptionsGammaLevel[] | undefined): number {
  let max = 0;
  for (const row of levels ?? []) {
    const n = Math.abs(Number(row.netUsd));
    if (Number.isFinite(n) && n > max) max = n;
  }
  return max;
}

export function gexBarPct(value: string | undefined, max: number): number {
  const n = Math.abs(Number(value));
  if (!Number.isFinite(n) || n <= 0 || max <= 0) return 0;
  return Math.min(100, (n / max) * 100);
}

export function isNearSpot(spot: string | undefined, index: string | undefined): boolean {
  const s = Number(spot);
  const i = Number(index);
  if (!Number.isFinite(s) || !Number.isFinite(i) || i <= 0) return false;
  return Math.abs(s - i) / i < 0.004;
}

export function isNearFlip(spot: string | undefined, flip: string | undefined): boolean {
  const s = Number(spot);
  const f = Number(flip);
  if (!Number.isFinite(s) || !Number.isFinite(f) || f <= 0) return false;
  return Math.abs(s - f) / f < 0.004;
}

export function gammaHasData(gamma: MarketOptionsGamma | undefined): boolean {
  return Boolean(gamma && (gamma.netUsd || gamma.flip || (gamma.profile?.length ?? 0) > 0));
}
