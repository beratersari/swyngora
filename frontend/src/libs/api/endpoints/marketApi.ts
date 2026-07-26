import { baseApi } from '../baseApi';
import type { components, operations } from '../generated/schema';

export type SpotMarket = components['schemas']['SpotMarket'];
export type SpotListResponse = components['schemas']['SpotListResponse'];
export type SpotListQuery = NonNullable<operations['listSpotMarkets']['parameters']['query']>;
export type SpotSortField = NonNullable<SpotListQuery['sort']>;
export type SpotSortOrder = NonNullable<SpotListQuery['order']>;
export type MarketExchange = NonNullable<SpotListQuery['exchange']>;

export type ExchangesResponse = {
  exchanges: string[];
  default: string;
};

export type ProductTagsResponse = {
  exchange: string;
  tags: string[];
};

/** Drop undefined / empty-string query values so RTK does not send noise. */
function compactParams<T extends Record<string, unknown>>(
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

export const marketApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listExchanges: build.query<ExchangesResponse, void>({
      query: () => '/api/v1/market/exchanges',
      transformResponse: (raw: { exchanges?: string[]; default?: string }): ExchangesResponse => ({
        exchanges: raw.exchanges ?? [],
        default: raw.default ?? 'binance',
      }),
      providesTags: ['Exchange'],
    }),

    listProductTags: build.query<ProductTagsResponse, { exchange?: MarketExchange } | void>({
      query: (arg) => ({
        url: '/api/v1/market/tags',
        params: compactParams({ exchange: arg && 'exchange' in arg ? arg.exchange : undefined }),
      }),
      transformResponse: (
        raw: { exchange?: string; tags?: string[] },
        _meta,
        arg,
      ): ProductTagsResponse => ({
        exchange:
          raw.exchange ??
          (arg && typeof arg === 'object' && 'exchange' in arg
            ? (arg.exchange ?? 'binance')
            : 'binance'),
        tags: raw.tags ?? [],
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'ProductTag' as const,
          id: arg && typeof arg === 'object' && arg.exchange ? arg.exchange : 'binance',
        },
      ],
    }),

    listSpotMarkets: build.query<SpotListResponse, SpotListQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/spot',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: ['SpotList'],
    }),
  }),
});

export const {
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  useLazyListSpotMarketsQuery,
} = marketApi;
