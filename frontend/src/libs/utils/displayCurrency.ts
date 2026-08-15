import { formatCompactAmount } from './formatMarket';
import { formatPrice } from './formatPrice';

export const DISPLAY_CURRENCY_STORAGE_KEY = 'swyngora.displayCurrency';

export const DISPLAY_CURRENCIES = ['native', 'USD', 'TRY', 'EUR'] as const;
export type DisplayCurrency = (typeof DISPLAY_CURRENCIES)[number];

export type FxRatesMap = Record<string, number>;

export function isDisplayCurrency(raw: string | null | undefined): raw is DisplayCurrency {
  return DISPLAY_CURRENCIES.includes((raw ?? '') as DisplayCurrency);
}

export function loadDisplayCurrency(): DisplayCurrency {
  try {
    const raw = localStorage.getItem(DISPLAY_CURRENCY_STORAGE_KEY);
    if (isDisplayCurrency(raw)) return raw;
  } catch {
    /* ignore */
  }
  return 'native';
}

export function saveDisplayCurrency(next: DisplayCurrency): void {
  try {
    localStorage.setItem(DISPLAY_CURRENCY_STORAGE_KEY, next);
  } catch {
    /* ignore */
  }
}

/** Native quote for last/open/high/low/quoteVolume. */
export function venueQuote(exchange?: string | null): string {
  switch ((exchange ?? '').toLowerCase()) {
    case 'bist':
      return 'TRY';
    case 'nasdaq':
    case 'coinbase':
      return 'USD';
    default:
      return 'USDT';
  }
}

/** Market-cap fields: BIST is TRY; crypto and Nasdaq are USD. */
export function marketCapQuote(exchange?: string | null): string {
  return venueQuote(exchange) === 'TRY' ? 'TRY' : 'USD';
}

export function aliasFxCode(code?: string | null): string {
  const c = (code ?? '').trim().toUpperCase();
  if (c === 'USDT' || c === 'USDC' || c === 'BUSD') return 'USD';
  return c;
}

export function convertAmount(
  value: string | number | null | undefined,
  from: string,
  to: string,
  rates?: FxRatesMap | null,
): number | null {
  if (value === null || value === undefined || value === '') return null;
  const n = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(n)) return null;
  const src = aliasFxCode(from);
  const dst = aliasFxCode(to);
  if (!src || !dst || src === dst) return n;
  const fromPerUsd = src === 'USD' ? 1 : rates?.[src];
  const toPerUsd = dst === 'USD' ? 1 : rates?.[dst];
  if (!fromPerUsd || fromPerUsd <= 0 || !toPerUsd || toPerUsd <= 0) return null;
  return (n / fromPerUsd) * toPerUsd;
}

export function resolveDisplayCode(preference: DisplayCurrency, nativeQuote: string): string {
  if (preference === 'native') return (nativeQuote || 'USD').toUpperCase();
  return preference;
}

export function formatConvertedPrice(
  value: string | number | null | undefined,
  nativeQuote: string,
  preference: DisplayCurrency,
  rates?: FxRatesMap | null,
): string {
  const to = resolveDisplayCode(preference, nativeQuote);
  const n = preference === 'native' ? Number(value) : convertAmount(value, nativeQuote, to, rates);
  if (n === null || !Number.isFinite(n)) {
    if (preference === 'native') return `${formatPrice(value)} ${to}`.trim();
    return '—';
  }
  return `${formatPrice(n)} ${to}`;
}

/** Scale a price-axis series (EMA/MA overlays) into the display currency. */
export function scalePriceSeries(
  points: { time: number; value: number }[],
  nativeQuote: string,
  preference: DisplayCurrency,
  rates?: FxRatesMap | null,
): { time: number; value: number }[] {
  if (!points.length || preference === 'native') return points;
  return points.map((p) => {
    const next = convertAmount(p.value, nativeQuote, preference, rates);
    return next === null ? p : { ...p, value: next };
  });
}

export function formatConvertedCompact(
  value: string | number | null | undefined,
  nativeQuote: string,
  preference: DisplayCurrency,
  rates?: FxRatesMap | null,
): string {
  const to = resolveDisplayCode(preference, nativeQuote);
  const n = preference === 'native' ? Number(value) : convertAmount(value, nativeQuote, to, rates);
  if (preference === 'native') {
    const num = formatCompactAmount(value);
    return num === '—' ? num : `${num} ${to}`;
  }
  if (n === null || !Number.isFinite(n)) return '—';
  return `${formatCompactAmount(n)} ${to}`;
}
