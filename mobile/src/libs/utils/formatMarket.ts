import { formatPrice } from './formatPrice';

const DASH = '—';

export function formatChangePercent(value: string | number | null | undefined): string {
  if (value === null || value === undefined || value === '') return DASH;
  const n = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(n)) return DASH;
  const sign = n > 0 ? '+' : '';
  return `${sign}${n.toFixed(2)}%`;
}

export function changeTone(
  value: string | number | null | undefined,
): 'success' | 'error' | 'secondary' {
  if (value === null || value === undefined || value === '') return 'secondary';
  const n = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(n) || n === 0) return 'secondary';
  return n > 0 ? 'success' : 'error';
}

/** Compact USD-style volume / mcap (e.g. 1.2B, 45.3M). */
export function formatCompactUsd(value: string | number | null | undefined): string {
  if (value === '∞') return '∞';
  if (value === null || value === undefined || value === '') return DASH;
  const n = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(n)) return DASH;
  if (n === 0) return '0';
  const abs = Math.abs(n);
  const sign = n < 0 ? '-' : '';
  if (abs >= 1e12) return `${sign}${(abs / 1e12).toFixed(2)}T`;
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(2)}B`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(2)}K`;
  return formatPrice(n);
}
