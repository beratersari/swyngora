import type { MarketExchange, Ticker24h } from '@/libs/api';
import {
  CROSS_EXCHANGE_MAX_CANDIDATES,
  CROSS_EXCHANGE_VENUES,
} from '@/config/crossExchangeConstants';
import { changeTone, formatChangePercent, formatCompactUsd } from './formatMarket';
import { formatPrice } from './formatPrice';
import { isMarketExchange } from './spotQuery';

export type ParsedMarketSymbol = {
  base: string;
  quote: string | null;
};

export type CrossExchangePlanRow = {
  exchange: MarketExchange;
  /** Ordered symbols to try (first is preferred). Source venue has one entry. */
  candidates: string[];
  isSource: boolean;
};

export type CrossExchangeRowStatus = 'ok' | 'loading' | 'unavailable' | 'error';

export type CrossExchangeRowModel = {
  id: string;
  exchange: MarketExchange;
  symbol: string;
  isSource: boolean;
  lastPriceLabel: string;
  changePercentLabel: string;
  changeTone: 'success' | 'error' | 'secondary';
  quoteVolumeLabel: string;
  status: CrossExchangeRowStatus;
  errorMessage?: string;
};

const STABLE_QUOTES = [
  'USDT',
  'USDC',
  'USD',
  'BUSD',
  'FDUSD',
  'EUR',
  'GBP',
  'BTC',
  'ETH',
  'TRY',
  'BRL',
] as const;

/**
 * Parse a venue symbol into base + quote when possible.
 * Supports compact (BTCUSDT) and hyphenated (BTC-USD) forms.
 */
export function parseMarketSymbol(
  exchange: string,
  symbol: string,
): ParsedMarketSymbol | null {
  const raw = String(symbol ?? '').trim().toUpperCase();
  if (!raw) return null;

  if (raw.includes('-')) {
    const [base, quote] = raw.split('-').map((p) => p.trim());
    if (!base) return null;
    return { base, quote: quote || null };
  }

  // Compact: try known quote suffixes longest-first
  const sorted = [...STABLE_QUOTES].sort((a, b) => b.length - a.length);
  for (const q of sorted) {
    if (raw.endsWith(q) && raw.length > q.length) {
      return { base: raw.slice(0, -q.length), quote: q };
    }
  }

  // Coinbase-style without hyphen after normalize miss: BTCUSD
  if (exchange === 'coinbase' && raw.endsWith('USD') && raw.length > 3) {
    return { base: raw.slice(0, -3), quote: 'USD' };
  }

  // Unknown quote — treat whole as base
  return { base: raw, quote: null };
}

/** Ordered symbol candidates for a target venue given a base asset. */
export function symbolCandidatesForExchange(
  base: string,
  targetExchange: MarketExchange | string,
): string[] {
  const b = base.trim().toUpperCase();
  if (!b) return [];
  const ex = String(targetExchange).toLowerCase();

  let list: string[];
  if (ex === 'coinbase') {
    list = [`${b}-USD`, `${b}-USDT`, `${b}-USDC`];
  } else if (ex === 'bybit') {
    list = [`${b}USDT`, `${b}USDC`, `${b}USD`];
  } else {
    // binance default
    list = [`${b}USDT`, `${b}USDC`, `${b}USD`, `${b}FDUSD`];
  }

  return list.slice(0, CROSS_EXCHANGE_MAX_CANDIDATES);
}

/**
 * Build fetch plan for all product venues from the current detail route.
 */
export function buildCrossExchangePlan(
  sourceExchange: string,
  sourceSymbol: string,
  venues: readonly MarketExchange[] = CROSS_EXCHANGE_VENUES,
): CrossExchangePlanRow[] {
  const srcEx = isMarketExchange(sourceExchange)
    ? sourceExchange
    : ('binance' as MarketExchange);
  const sym = String(sourceSymbol ?? '').trim().toUpperCase();
  const parsed = parseMarketSymbol(srcEx, sym);
  const base = parsed?.base ?? '';

  return venues.map((exchange) => {
    const isSource = exchange === srcEx;
    if (isSource) {
      return {
        exchange,
        candidates: sym ? [sym] : [],
        isSource: true,
      };
    }
    return {
      exchange,
      candidates: base ? symbolCandidatesForExchange(base, exchange) : [],
      isSource: false,
    };
  });
}

export function mapTickerToCrossExchangeRow(
  plan: CrossExchangePlanRow,
  resolvedSymbol: string,
  ticker: Ticker24h | undefined,
  opts: {
    status: CrossExchangeRowStatus;
    errorMessage?: string;
  },
): CrossExchangeRowModel {
  const symbol = resolvedSymbol || plan.candidates[0] || '—';
  if (opts.status !== 'ok' || !ticker) {
    return {
      id: `${plan.exchange}|${symbol}`,
      exchange: plan.exchange,
      symbol,
      isSource: plan.isSource,
      lastPriceLabel: '—',
      changePercentLabel: '—',
      changeTone: 'secondary',
      quoteVolumeLabel: '—',
      status: opts.status,
      errorMessage: opts.errorMessage,
    };
  }

  return {
    id: `${plan.exchange}|${symbol}`,
    exchange: plan.exchange,
    symbol: ticker.symbol ?? symbol,
    isSource: plan.isSource,
    lastPriceLabel: formatPrice(ticker.lastPrice),
    changePercentLabel: formatChangePercent(ticker.priceChangePercent),
    changeTone: changeTone(ticker.priceChangePercent),
    quoteVolumeLabel: formatCompactUsd(ticker.quoteVolume),
    status: 'ok',
  };
}

/** Prefer lower last price as "cheapest" highlight helper (optional UI). */
export function cheapestExchangeId(rows: CrossExchangeRowModel[]): string | null {
  let best: { id: string; n: number } | null = null;
  for (const r of rows) {
    if (r.status !== 'ok') continue;
    const n = Number(String(r.lastPriceLabel).replace(/,/g, ''));
    if (!Number.isFinite(n) || n <= 0) continue;
    if (!best || n < best.n) best = { id: r.id, n };
  }
  return best?.id ?? null;
}
