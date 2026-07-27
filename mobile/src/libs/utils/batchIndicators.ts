import {
  BATCH_MAX_SYMBOLS,
  DEFAULT_BATCH_EMA_PERIODS,
  DEFAULT_BATCH_INTERVAL,
  DEFAULT_BATCH_RSI_PERIOD,
} from '@/config/batchIndicatorsConstants';
import type { MarketExchange } from '@/libs/api';
import { normalizeExchange } from './spotQuery';
import { watchKey, type WatchlistPair } from './watchlistKey';

export type BatchIndicatorsArg = {
  exchange: MarketExchange;
  symbols: string[];
  interval?: string;
  rsiPeriod?: number;
  emaPeriods?: string;
};

export type BatchIndicatorItem = {
  symbol?: string;
  rsi?: number | null;
  ema?: Record<string, number>;
  error?: string;
};

export type BatchIndicatorsResponse = {
  exchange?: string;
  interval?: string;
  items?: BatchIndicatorItem[];
  note?: string;
};

export type RsiRowFields = {
  rsiLabel: string;
  rsiTone: 'success' | 'warning' | 'error' | 'secondary';
  rsiLoading?: boolean;
};

/** Split array into chunks of at most `max` (default 50). */
export function chunkSymbols(symbols: string[], max: number = BATCH_MAX_SYMBOLS): string[][] {
  const size = Math.max(1, max);
  const out: string[][] = [];
  for (let i = 0; i < symbols.length; i += size) {
    out.push(symbols.slice(i, i + size));
  }
  return out;
}

/**
 * Group pairs by normalized exchange → unique uppercase symbols.
 * Preserves first-seen order per exchange.
 */
export function groupPairsByExchange(
  pairs: Pick<WatchlistPair, 'exchange' | 'symbol'>[],
): Record<MarketExchange, string[]> {
  const order: MarketExchange[] = ['binance', 'coinbase', 'bybit'];
  const maps: Record<MarketExchange, string[]> = {
    binance: [],
    coinbase: [],
    bybit: [],
  };
  const seen: Record<MarketExchange, Set<string>> = {
    binance: new Set(),
    coinbase: new Set(),
    bybit: new Set(),
  };

  for (const p of pairs) {
    const ex = normalizeExchange(p.exchange);
    const sym = String(p.symbol ?? '')
      .trim()
      .toUpperCase();
    if (!sym) continue;
    if (seen[ex].has(sym)) continue;
    seen[ex].add(sym);
    maps[ex].push(sym);
  }

  // Ensure keys exist even when empty (callers use skip)
  for (const ex of order) {
    if (!maps[ex]) maps[ex] = [];
  }
  return maps;
}

export function buildBatchIndicatorsArg(input: {
  exchange: string;
  symbols: string[];
  interval?: string;
  rsiPeriod?: number;
  emaPeriods?: string;
  maxSymbols?: number;
}): BatchIndicatorsArg {
  const max = input.maxSymbols ?? BATCH_MAX_SYMBOLS;
  const unique: string[] = [];
  const seen = new Set<string>();
  for (const raw of input.symbols) {
    const sym = String(raw ?? '')
      .trim()
      .toUpperCase();
    if (!sym || seen.has(sym)) continue;
    seen.add(sym);
    unique.push(sym);
    if (unique.length >= max) break;
  }
  return {
    exchange: normalizeExchange(input.exchange),
    symbols: unique,
    interval: input.interval ?? DEFAULT_BATCH_INTERVAL,
    rsiPeriod: input.rsiPeriod ?? DEFAULT_BATCH_RSI_PERIOD,
    emaPeriods: input.emaPeriods ?? DEFAULT_BATCH_EMA_PERIODS,
  };
}

/** Index batch items by uppercase symbol. */
export function indexBatchItemsBySymbol(
  items: BatchIndicatorItem[] | undefined | null,
): Map<string, BatchIndicatorItem> {
  const map = new Map<string, BatchIndicatorItem>();
  if (!items) return map;
  for (const item of items) {
    const sym = String(item.symbol ?? '')
      .trim()
      .toUpperCase();
    if (!sym) continue;
    map.set(sym, item);
  }
  return map;
}

/** Join key for multi-exchange maps. */
export function batchItemKey(exchange: string, symbol: string): string {
  return watchKey(exchange, symbol);
}

export function formatRsi(value: number | null | undefined): string {
  if (value == null || Number.isNaN(Number(value))) return '—';
  return `RSI ${Number(value).toFixed(1)}`;
}

/**
 * Visual tone only (not a signal): high → warning, low → success, mid → secondary.
 */
export function rsiTone(
  value: number | null | undefined,
): 'success' | 'warning' | 'error' | 'secondary' {
  if (value == null || Number.isNaN(Number(value))) return 'secondary';
  const n = Number(value);
  if (n >= 70) return 'warning';
  if (n <= 30) return 'success';
  return 'secondary';
}

export function rsiFieldsFromItem(
  item: BatchIndicatorItem | undefined,
  loading: boolean,
): RsiRowFields {
  if (loading && !item) {
    return { rsiLabel: '…', rsiTone: 'secondary', rsiLoading: true };
  }
  if (!item || item.error || item.rsi == null) {
    return { rsiLabel: '—', rsiTone: 'secondary', rsiLoading: false };
  }
  return {
    rsiLabel: formatRsi(item.rsi),
    rsiTone: rsiTone(item.rsi),
    rsiLoading: false,
  };
}

/** Build RTK POST body (omit undefined). */
export function buildBatchIndicatorsBody(arg: BatchIndicatorsArg): {
  exchange: string;
  interval: string;
  symbols: string[];
  rsiPeriod: number;
  emaPeriods: string;
} {
  return {
    exchange: arg.exchange,
    interval: arg.interval ?? DEFAULT_BATCH_INTERVAL,
    symbols: arg.symbols,
    rsiPeriod: arg.rsiPeriod ?? DEFAULT_BATCH_RSI_PERIOD,
    emaPeriods: arg.emaPeriods ?? DEFAULT_BATCH_EMA_PERIODS,
  };
}
