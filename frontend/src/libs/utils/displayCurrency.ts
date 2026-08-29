import { formatCompactAmount } from './formatMarket';
import { formatPrice } from './formatPrice';
import { parseTradingPair } from './formatSymbol';

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

/** Native quote from the API `venueQuotes` map. Empty until that payload loads. */
export function venueQuote(
  exchange?: string | null,
  quotes?: Record<string, string> | null,
): string {
  const key = (exchange ?? '').toLowerCase();
  return quotes?.[key] ?? '';
}

/** Pair quote when the symbol splits; otherwise the venue default. */
export function pairQuote(
  symbol?: string | null,
  exchange?: string | null,
  quotes?: Record<string, string> | null,
): string {
  const q = parseTradingPair(symbol).quote;
  return q || venueQuote(exchange, quotes);
}

/** Market-cap currency from the API `marketCapQuotes` map. */
export function marketCapQuote(
  exchange?: string | null,
  quotes?: Record<string, string> | null,
): string {
  const key = (exchange ?? '').toLowerCase();
  return quotes?.[key] ?? '';
}

export function aliasFxCode(
  code?: string | null,
  aliases?: Record<string, string> | null,
): string {
  const c = (code ?? '').trim().toUpperCase();
  return aliases?.[c] ?? c;
}

export function convertAmount(
  value: string | number | null | undefined,
  from: string,
  to: string,
  rates?: FxRatesMap | null,
  aliases?: Record<string, string> | null,
): number | null {
  if (value === null || value === undefined || value === '') return null;
  const n = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(n)) return null;
  const src = aliasFxCode(from, aliases);
  const dst = aliasFxCode(to, aliases);
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
  aliases?: Record<string, string> | null,
): string {
  const to = resolveDisplayCode(preference, nativeQuote);
  const n =
    preference === 'native'
      ? Number(value)
      : convertAmount(value, nativeQuote, to, rates, aliases);
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
  aliases?: Record<string, string> | null,
): { time: number; value: number }[] {
  if (!points.length || preference === 'native') return points;
  const out: { time: number; value: number }[] = [];
  for (const p of points) {
    const next = convertAmount(p.value, nativeQuote, preference, rates, aliases);
    if (next === null) continue;
    out.push({ ...p, value: next });
  }
  return out;
}

export function formatConvertedCompact(
  value: string | number | null | undefined,
  nativeQuote: string,
  preference: DisplayCurrency,
  rates?: FxRatesMap | null,
  aliases?: Record<string, string> | null,
): string {
  const to = resolveDisplayCode(preference, nativeQuote);
  const n =
    preference === 'native'
      ? Number(value)
      : convertAmount(value, nativeQuote, to, rates, aliases);
  if (preference === 'native') {
    const num = formatCompactAmount(value);
    return num === '—' ? num : `${num} ${to}`;
  }
  if (n === null || !Number.isFinite(n)) return '—';
  return `${formatCompactAmount(n)} ${to}`;
}
