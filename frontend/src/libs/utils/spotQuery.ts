import type { SpotListQuery, SpotSortField, SpotSortOrder, MarketExchange } from '@/libs/api';
import { venueQuote } from './displayCurrency';

export type MarketsUrlState = {
  exchange: MarketExchange;
  q: string;
  quote: string;
  tag: string;
  sort: SpotSortField;
  order: SpotSortOrder;
  limit: number;
  offset: number;
};

export const DEFAULT_MARKETS_STATE: MarketsUrlState = {
  exchange: 'binance',
  q: '',
  quote: '',
  tag: '',
  sort: 'quoteVolume',
  order: 'desc',
  limit: 50,
  offset: 0,
};

const EXCHANGES = new Set(['binance', 'coinbase', 'bybit', 'nasdaq', 'bist']);
const SORTS = new Set<SpotSortField>([
  'quoteVolume',
  'volume',
  'priceChangePercent',
  'lastPrice',
  'tradeCount',
  'symbol',
  'baseAsset',
  'marketCapCirculating',
  'marketCapTotal',
  'marketCapMax',
  'tags',
]);

/** Venue default quote. Prefer API `venueQuotes`; last-resort matches Go QuoteForVenue. */
export function defaultQuoteForExchange(
  exchange: string,
  quotes?: Record<string, string> | null,
): string {
  return venueQuote(exchange, quotes);
}

function parseIntParam(raw: string | null, fallback: number, min: number, max: number): number {
  if (raw === null || raw === '') return fallback;
  const n = Number(raw);
  if (!Number.isFinite(n)) return fallback;
  return Math.min(max, Math.max(min, Math.floor(n)));
}

/** Parse /markets URL search params into state with defaults. */
export function parseMarketsSearchParams(
  params: URLSearchParams,
  venueQuotes?: Record<string, string> | null,
): MarketsUrlState {
  const exchangeRaw = (params.get('exchange') ?? DEFAULT_MARKETS_STATE.exchange).toLowerCase();
  const exchange = (
    EXCHANGES.has(exchangeRaw) ? exchangeRaw : DEFAULT_MARKETS_STATE.exchange
  ) as MarketExchange;

  const sortRaw = (params.get('sort') ?? DEFAULT_MARKETS_STATE.sort) as SpotSortField;
  const sort = SORTS.has(sortRaw) ? sortRaw : DEFAULT_MARKETS_STATE.sort;

  const orderRaw = params.get('order') ?? DEFAULT_MARKETS_STATE.order;
  const order: SpotSortOrder =
    orderRaw === 'asc' || orderRaw === 'desc' ? orderRaw : DEFAULT_MARKETS_STATE.order;

  // Missing quote → venue default (not a global USDT, which starves Coinbase).
  const quoteParam = params.get('quote');
  const quote =
    quoteParam !== null && quoteParam !== ''
      ? quoteParam
      : defaultQuoteForExchange(exchange, venueQuotes);

  return {
    exchange,
    q: params.get('q') ?? '',
    quote,
    tag: params.get('tag') ?? '',
    sort,
    order,
    limit: parseIntParam(params.get('limit'), DEFAULT_MARKETS_STATE.limit, 1, 500),
    offset: parseIntParam(params.get('offset'), DEFAULT_MARKETS_STATE.offset, 0, 1_000_000),
  };
}

/** Serialize state to URLSearchParams (omit defaults where sensible). */
export function marketsStateToSearchParams(
  state: MarketsUrlState,
  venueQuotes?: Record<string, string> | null,
): URLSearchParams {
  const p = new URLSearchParams();
  if (state.exchange !== DEFAULT_MARKETS_STATE.exchange) p.set('exchange', state.exchange);
  if (state.q.trim()) p.set('q', state.q.trim());
  // Omit quote when it matches this exchange's primary default.
  if (state.quote && state.quote !== defaultQuoteForExchange(state.exchange, venueQuotes)) {
    p.set('quote', state.quote);
  }
  if (state.tag.trim()) p.set('tag', state.tag.trim());
  if (state.sort !== DEFAULT_MARKETS_STATE.sort) p.set('sort', state.sort);
  if (state.order !== DEFAULT_MARKETS_STATE.order) p.set('order', state.order);
  if (state.limit !== DEFAULT_MARKETS_STATE.limit) p.set('limit', String(state.limit));
  if (state.offset !== DEFAULT_MARKETS_STATE.offset) p.set('offset', String(state.offset));
  return p;
}

/**
 * Align list query with debounced search: when `debouncedQ` is ahead of URL `q`,
 * force `offset: 0` so the first RTK request is not page N of the new query.
 * (URL offset is reset in a later effect — without this, render uses stale offset.)
 */
export function effectiveMarketsStateForQuery(
  state: MarketsUrlState,
  debouncedQ: string,
): MarketsUrlState {
  if (debouncedQ === state.q) {
    return state;
  }
  return { ...state, q: debouncedQ, offset: 0 };
}

/** Build RTK listSpotMarkets args from UI state + debounced q. */
export function toSpotListQuery(state: MarketsUrlState, debouncedQ: string): SpotListQuery {
  const effective = effectiveMarketsStateForQuery(state, debouncedQ);
  return {
    exchange: effective.exchange,
    q: debouncedQ.trim() || undefined,
    quote: effective.quote || undefined,
    tag: effective.tag.trim() || undefined,
    sort: effective.sort,
    order: effective.order,
    limit: effective.limit,
    offset: effective.offset,
    status: 'TRADING',
  };
}
