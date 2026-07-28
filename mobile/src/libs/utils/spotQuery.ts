import type { MarketExchange, SpotListQuery, SpotSortField, SpotSortOrder } from '@/libs/api';

export type MarketsFilterState = {
  exchange: MarketExchange;
  q: string;
  quote: string;
  /** Selected tags (OR). Serialized as comma-separated `tags` query. */
  tags: string[];
  sort: SpotSortField;
  order: SpotSortOrder;
  limit: number;
  offset: number;
};

export const DEFAULT_MARKETS_FILTER: MarketsFilterState = {
  exchange: 'binance',
  q: '',
  quote: 'USDT',
  tags: [],
  sort: 'quoteVolume',
  order: 'desc',
  limit: 30,
  offset: 0,
};

const EXCHANGES = new Set(['binance', 'coinbase', 'bybit']);
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

export function isMarketExchange(value: string): value is MarketExchange {
  return EXCHANGES.has(String(value).toLowerCase());
}

export function isSpotSortField(value: string): value is SpotSortField {
  return SORTS.has(value as SpotSortField);
}

/** Build RTK listSpotMarkets args from filter state + debounced q. */
export function toSpotListQuery(
  state: MarketsFilterState,
  debouncedQ: string,
): SpotListQuery {
  const tagsJoined = state.tags.map((t) => t.trim()).filter(Boolean).join(',');
  return {
    exchange: state.exchange,
    q: debouncedQ.trim() || undefined,
    quote: state.quote || undefined,
    tags: tagsJoined || undefined,
    sort: state.sort,
    order: state.order,
    limit: state.limit,
    offset: state.offset,
    status: 'TRADING',
  };
}

export function normalizeExchange(value: string | undefined): MarketExchange {
  if (!value) return DEFAULT_MARKETS_FILTER.exchange;
  const lower = String(value).toLowerCase();
  if (isMarketExchange(lower)) return lower;
  return DEFAULT_MARKETS_FILTER.exchange;
}
