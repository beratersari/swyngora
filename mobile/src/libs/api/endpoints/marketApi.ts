import { baseApi } from '../baseApi';
import type { components, operations } from '../generated/schema';

export type SpotMarket = components['schemas']['SpotMarket'];
export type SpotListResponse = components['schemas']['SpotListResponse'];
export type SpotListQuery = NonNullable<operations['listSpotMarkets']['parameters']['query']>;
export type SpotSortField = NonNullable<SpotListQuery['sort']>;
export type SpotSortOrder = NonNullable<SpotListQuery['order']>;
export type MarketExchange = NonNullable<SpotListQuery['exchange']>;

export type CandlesResponse = components['schemas']['CandlesResponse'];
export type Candle = components['schemas']['Candle'];
export type Ticker24h = components['schemas']['Ticker24h'];
export type Supply = components['schemas']['Supply'];

export type CandlesQuery = NonNullable<operations['getCandles']['parameters']['query']>;
export type Ticker24hQuery = NonNullable<operations['getTicker24h']['parameters']['query']>;
export type SupplyQuery = NonNullable<operations['getSupply']['parameters']['query']>;
export type IntervalsQuery = NonNullable<operations['listIntervals']['parameters']['query']>;
export type IndicatorsQuery = NonNullable<operations['getIndicators']['parameters']['query']>;

export type IntervalsResponse = {
  exchange: string;
  intervals: string[];
};

export type IndicatorsResponse = {
  exchange?: string;
  symbol?: string;
  interval?: string;
  rsiPeriod?: number;
  emaPeriods?: number[];
  latest?: {
    rsi?: number | null;
    ema?: Record<string, number>;
  };
  points?: {
    openTime?: string;
    close?: number;
    rsi?: number | null;
    ema?: Record<string, number>;
  }[];
  note?: string;
};

export type ExchangesResponse = {
  exchanges: string[];
  default: string;
};

export type ProductTagsResponse = {
  exchange: string;
  tags: string[];
};

/** Drop undefined / empty-string query values so RTK does not send noise. */
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
}): ExchangesResponse {
  return {
    exchanges: raw.exchanges ?? [],
    default: raw.default ?? 'binance',
  };
}

export function transformProductTagsResponse(
  raw: { exchange?: string; tags?: string[] },
  arg?: { exchange?: MarketExchange } | void,
): ProductTagsResponse {
  return {
    exchange:
      raw.exchange ??
      (arg && typeof arg === 'object' && arg.exchange ? arg.exchange : 'binance'),
    tags: raw.tags ?? [],
  };
}

export function transformIntervalsResponse(
  raw: { exchange?: string; intervals?: string[] },
  arg?: IntervalsQuery | void,
): IntervalsResponse {
  return {
    exchange:
      raw.exchange ??
      (arg && typeof arg === 'object' && arg.exchange ? arg.exchange : 'binance'),
    intervals: raw.intervals ?? [],
  };
}

export const marketApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    listExchanges: build.query<ExchangesResponse, void>({
      query: () => '/api/v1/market/exchanges',
      transformResponse: transformExchangesResponse,
      providesTags: ['Exchange'],
    }),

    listProductTags: build.query<ProductTagsResponse, { exchange?: MarketExchange } | void>({
      query: (arg) => ({
        url: '/api/v1/market/tags',
        params: compactParams({
          exchange: arg && typeof arg === 'object' && 'exchange' in arg ? arg.exchange : undefined,
        }),
      }),
      transformResponse: (raw, _meta, arg) => transformProductTagsResponse(raw, arg),
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

    listIntervals: build.query<IntervalsResponse, IntervalsQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/intervals',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      transformResponse: (raw, _meta, arg) => transformIntervalsResponse(raw, arg),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Interval' as const,
          id: arg && typeof arg === 'object' && arg.exchange ? arg.exchange : 'binance',
        },
      ],
    }),

    getCandles: build.query<CandlesResponse, CandlesQuery>({
      query: (arg) => ({
        url: '/api/v1/market/candles',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Candle' as const,
          id: `${arg.exchange ?? 'binance'}:${arg.symbol}:${arg.interval ?? '1h'}:${arg.limit ?? 100}`,
        },
      ],
    }),

    getTicker24h: build.query<Ticker24h, Ticker24hQuery>({
      query: (arg) => ({
        url: '/api/v1/market/ticker/24h',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [
        { type: 'Ticker' as const, id: `${arg.exchange ?? 'binance'}:${arg.symbol}` },
      ],
    }),

    getSupply: build.query<Supply, SupplyQuery>({
      query: (arg) => ({
        url: '/api/v1/market/supply',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Supply' as const,
          id: (arg && (arg.asset || arg.symbol)) || 'unknown',
        },
      ],
    }),

    getIndicators: build.query<IndicatorsResponse, IndicatorsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/indicators',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Indicator' as const,
          id: `${arg.exchange ?? 'binance'}:${arg.symbol}:${arg.interval ?? '1h'}:${arg.limit ?? 100}:${arg.rsiPeriod ?? 14}:${arg.emaPeriods ?? '12,26'}`,
        },
      ],
    }),
  }),
});

export const {
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  useListIntervalsQuery,
  useGetCandlesQuery,
  useGetTicker24hQuery,
  useGetSupplyQuery,
  useGetIndicatorsQuery,
} = marketApi;
