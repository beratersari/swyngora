import { baseApi } from '../baseApi';
import {
  candleTagId,
  compactParams,
  indicatorTagId,
  intervalTagId,
  productTagId,
  spotListTagId,
  supplyTagId,
  tickerTagId,
  transformExchangesResponse,
  transformIntervalsResponse,
  transformProductTagsResponse,
} from './marketApi.helpers';
import type {
  CandlesQuery,
  CandlesResponse,
  ExchangesResponse,
  IndicatorsQuery,
  IndicatorsResponse,
  IntervalsQuery,
  IntervalsResponse,
  MarketExchange,
  ProductTagsResponse,
  PumpEventsQuery,
  PumpEventsResponse,
  ScanPumpEventsQuery,
  ScanPumpEventsResponse,
  SpotListQuery,
  SpotListResponse,
  Supply,
  SupplyQuery,
  Ticker24h,
  Ticker24hQuery,
} from './marketApi.types';
// delist types re-exported below

export type {
  SpotMarket,
  SpotListResponse,
  SpotListQuery,
  SpotSortField,
  SpotSortOrder,
  MarketExchange,
  CandlesResponse,
  Candle,
  Ticker24h,
  Supply,
  CandlesQuery,
  Ticker24hQuery,
  SupplyQuery,
  IntervalsQuery,
  IndicatorsQuery,
  IntervalsResponse,
  IndicatorsResponse,
  ExchangesResponse,
  ProductTagsResponse,
  PumpEventsQuery,
  PumpEventsResponse,
  PumpEventDto,
  PumpScanHitDto,
  ScanPumpEventsQuery,
  ScanPumpEventsResponse,
} from './marketApi.types';
// delist types re-exported below

export {
  compactParams,
  transformExchangesResponse,
  transformProductTagsResponse,
  transformIntervalsResponse,
  spotListTagId,
  productTagId,
  intervalTagId,
  candleTagId,
  tickerTagId,
  supplyTagId,
  indicatorTagId,
} from './marketApi.helpers';

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
          exchange: arg && 'exchange' in arg ? arg.exchange : undefined,
        }),
      }),
      transformResponse: (
        raw: { exchange?: string; tags?: string[] },
        _meta,
        arg,
      ): ProductTagsResponse => transformProductTagsResponse(raw, arg),
      providesTags: (_r, _e, arg) => [{ type: 'ProductTag' as const, id: productTagId(arg) }],
    }),

    listSpotMarkets: build.query<SpotListResponse, SpotListQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/spot',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: (_r, _e, arg) => [
        { type: 'SpotList' as const, id: spotListTagId(arg) },
        { type: 'SpotList' as const, id: 'LIST' },
      ],
    }),

    listIntervals: build.query<IntervalsResponse, IntervalsQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/intervals',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      transformResponse: (
        raw: { exchange?: string; intervals?: string[] },
        _meta,
        arg,
      ): IntervalsResponse => transformIntervalsResponse(raw, arg),
      providesTags: (_r, _e, arg) => [{ type: 'Interval' as const, id: intervalTagId(arg) }],
    }),

    getCandles: build.query<CandlesResponse, CandlesQuery>({
      query: (arg) => ({
        url: '/api/v1/market/candles',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [{ type: 'Candle' as const, id: candleTagId(arg) }],
    }),

    getTicker24h: build.query<Ticker24h, Ticker24hQuery>({
      query: (arg) => ({
        url: '/api/v1/market/ticker/24h',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [{ type: 'Ticker' as const, id: tickerTagId(arg) }],
    }),

    getSupply: build.query<Supply, SupplyQuery>({
      query: (arg) => ({
        url: '/api/v1/market/supply',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: (_r, _e, arg) => [{ type: 'Supply' as const, id: supplyTagId(arg) }],
    }),

    getIndicators: build.query<IndicatorsResponse, IndicatorsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/indicators',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [{ type: 'Indicator' as const, id: indicatorTagId(arg) }],
    }),

    getPumpEvents: build.query<PumpEventsResponse, PumpEventsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/pumps',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Pump' as const,
          id: `${arg.exchange ?? 'binance'}:${arg.symbol}:${arg.interval ?? '1h'}`,
        },
      ],
    }),

    scanPumpEvents: build.query<ScanPumpEventsResponse, ScanPumpEventsQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/pumps/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: (_r, _e, arg) => [
        {
          type: 'Pump' as const,
          id: `scan:${arg && typeof arg === 'object' ? (arg.exchange ?? 'binance') : 'binance'}`,
        },
      ],
    }),

    listDelistSchedule: build.query<
      import('./marketApi.types').DelistScheduleResponse,
      { exchange?: import('./marketApi.types').MarketExchange } | void
    >({
      query: (arg) => ({
        url: '/api/v1/market/delist-schedule',
        params: {
          exchange: arg && typeof arg === 'object' && 'exchange' in arg ? arg.exchange : undefined,
        },
      }),
      keepUnusedDataFor: 300,
    }),

    postIndicatorsBatch: build.mutation<
      {
        exchange?: string;
        interval?: string;
        items?: {
          symbol?: string;
          rsi?: number | null;
          ema?: Record<string, number>;
          error?: string;
        }[];
        note?: string;
      },
      {
        exchange?: string;
        interval?: string;
        symbols: string[];
        rsiPeriod?: number;
        emaPeriods?: string;
      }
    >({
      query: (body) => ({
        url: '/api/v1/market/indicators/batch',
        method: 'POST',
        body,
      }),
    }),
  }),
});

export const {
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  useListDelistScheduleQuery,
  useLazyListSpotMarketsQuery,
  useListIntervalsQuery,
  useGetCandlesQuery,
  useLazyGetCandlesQuery,
  useGetTicker24hQuery,
  useGetSupplyQuery,
  useGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useLazyGetPumpEventsQuery,
  useScanPumpEventsQuery,
  usePostIndicatorsBatchMutation,
} = marketApi;
