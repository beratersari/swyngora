import type { MarketExchange, SpotListQuery } from './marketApi.types';

/** Drop undefined / empty-string / non string|number query values. */
export function compactParams<T extends Record<string, unknown>>(
  params: T,
): Record<string, string | number> {
  const out: Record<string, string | number> = {};
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue;
    if (typeof value === 'string' || typeof value === 'number') {
      out[key] = value;
    }
  }
  return out;
}

export function transformExchangesResponse(raw: {
  exchanges?: string[];
  default?: string;
}): { exchanges: string[]; default: string } {
  return {
    exchanges: raw.exchanges ?? [],
    default: raw.default ?? 'binance',
  };
}

export function resolveExchangeArg(
  arg: { exchange?: string } | void | undefined,
  rawExchange?: string,
): string {
  if (rawExchange) return rawExchange;
  if (arg && typeof arg === 'object' && arg.exchange) return arg.exchange;
  return 'binance';
}

export function transformProductTagsResponse(
  raw: { exchange?: string; tags?: string[] },
  arg: { exchange?: MarketExchange } | void,
): { exchange: string; tags: string[] } {
  return {
    exchange: resolveExchangeArg(arg, raw.exchange),
    tags: raw.tags ?? [],
  };
}

export function transformIntervalsResponse(
  raw: { exchange?: string; intervals?: string[] },
  arg: { exchange?: string } | void,
): { exchange: string; intervals: string[] } {
  return {
    exchange: resolveExchangeArg(arg, raw.exchange),
    intervals: raw.intervals ?? [],
  };
}

/** Arg-scoped SpotList tag so invalidation does not thrash every list cache. */
export function spotListTagId(arg: SpotListQuery | void): string {
  if (!arg) return 'default';
  const {
    exchange = 'binance',
    q = '',
    quote = '',
    tag = '',
    sort = '',
    order = '',
    limit = '',
    offset = '',
    status = '',
  } = arg;
  return [exchange, q, quote, tag, sort, order, limit, offset, status].join('|');
}

export function productTagId(arg: { exchange?: string } | void): string {
  return arg && typeof arg === 'object' && arg.exchange ? arg.exchange : 'binance';
}

export function intervalTagId(arg: { exchange?: string } | void): string {
  return arg && typeof arg === 'object' && arg.exchange ? arg.exchange : 'binance';
}

export function candleTagId(arg: {
  exchange?: string;
  symbol: string;
  interval?: string;
  limit?: number;
}): string {
  return `${arg.exchange ?? 'binance'}:${arg.symbol}:${arg.interval ?? '1h'}:${arg.limit ?? 100}`;
}

export function tickerTagId(arg: { exchange?: string; symbol: string }): string {
  return `${arg.exchange ?? 'binance'}:${arg.symbol}`;
}

export function orderBookTagId(arg: {
  exchange?: string;
  symbol: string;
  group?: string;
  limit?: number;
}): string {
  return `${arg.exchange ?? 'binance'}:${arg.symbol}:${arg.group ?? ''}:${arg.limit ?? 20}`;
}

export function supplyTagId(arg: { asset?: string; symbol?: string } | void): string {
  return (arg && (arg.asset || arg.symbol)) || 'unknown';
}

export function indicatorTagId(arg: {
  exchange?: string;
  symbol: string;
  interval?: string;
  limit?: number;
  rsiPeriod?: number;
  emaPeriods?: string;
}): string {
  return `${arg.exchange ?? 'binance'}:${arg.symbol}:${arg.interval ?? '1h'}:${arg.limit ?? 100}:${arg.rsiPeriod ?? 14}:${arg.emaPeriods ?? '12,26'}`;
}
