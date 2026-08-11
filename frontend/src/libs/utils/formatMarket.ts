import { formatPrice } from './formatPrice';

const DASH = '—';

/**
 * Compact UTC delist label for tags, e.g. "17 Aug 2026 03:00 UTC".
 */
export function formatDelistDate(
  value: string | number | Date | null | undefined,
  locale?: string,
): string {
  if (value === null || value === undefined || value === '') return '';
  const d = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(d.getTime())) return '';
  const loc = locale || undefined;
  const datePart = d.toLocaleDateString(loc, {
    timeZone: 'UTC',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  });
  const timePart = d.toLocaleTimeString(loc, {
    timeZone: 'UTC',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
  return `${datePart} ${timePart} UTC`;
}

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

/** Compact quantity (volume / mcap) without a currency prefix, e.g. 1.2B, 45.3M. */
export function formatCompactAmount(value: string | number | null | undefined): string {
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

/** Compact USD mcap/quote volume. Does not add a `$` prefix. */
export function formatCompactUsd(value: string | number | null | undefined): string {
  return formatCompactAmount(value);
}

/** Compact amount plus optional asset code, e.g. "10.00K BTC". */
export function formatCompactAsset(
  value: string | number | null | undefined,
  asset?: string | null,
): string {
  const num = formatCompactAmount(value);
  if (num === DASH) return DASH;
  const code = (asset ?? '').trim().toUpperCase();
  return code ? `${num} ${code}` : num;
}

export function formatTradeCount(
  value: number | null | undefined,
  exchange: string | undefined,
): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return DASH;
  if (value === 0 && exchange && exchange !== 'binance') return DASH;
  if (value === 0) return '0';
  return value.toLocaleString();
}

export function formatMarketCapMax(value: number | '∞' | null | undefined): string {
  if (value === '∞') return '∞';
  return formatCompactUsd(value);
}


export function formatDateTime(
  value: string | number | Date | null | undefined,
  locale?: string,
): string {
  if (value === null || value === undefined || value === '') return '—';
  const d = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString(locale || undefined, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}
