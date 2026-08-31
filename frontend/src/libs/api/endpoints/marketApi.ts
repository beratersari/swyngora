import { baseApi } from '../baseApi';
import {
  candleTagId,
  compactParams,
  indicatorTagId,
  intervalTagId,
  productTagId,
  spotListTagId,
  supplyTagId,
  holdersTagId,
  tickerTagId,
  orderBookTagId,
  orderHeatmapTagId,
  liqHeatmapTagId,
  transformExchangesResponse,
  transformIntervalsResponse,
  transformProductTagsResponse,
} from './marketApi.helpers';
import type {
  CandlesQuery,
  CandlesResponse,
  ExchangesResponse,
  FxRatesResponse,
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
  RSIHeatmapQuery,
  RSIHeatmapResponse,
  SpotListQuery,
  SpotListResponse,
  Supply,
  SupplyQuery,
  AssetHolders,
  AssetProfile,
  AssetProfileQuery,
  HoldersQuery,
  MarketOpenInterest,
  MarketLiquidations,
  MarketLiquidationOverview,
  MarketLiquidationLevels,
  LiquidationLevelsQuery,
  MarketLiquidationCascade,
  MarketLiquidationCascadeScan,
  LiquidationCascadeQuery,
  LiquidationCascadeScanQuery,
  MarketCvd,
  OpenInterestQuery,
  LiquidationsQuery,
  LiquidationOverviewQuery,
  CvdQuery,
  Ticker24h,
  Ticker24hQuery,
  SpotOrderBook,
  OrderBookQuery,
  OrderBookHeatmap,
  OrderBookHeatmapQuery,
  LiquidationHuntHeatmap,
  LiquidationHuntHeatmapQuery,
  LiquidationHunt,
  LiquidationHuntQuery,
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
  AssetHolders,
  AssetProfile,
  HoldersQuery,
  MarketOpenInterest,
  MarketLiquidations,
  MarketLiquidationOverview,
  MarketLiquidationLevels,
  MarketLiquidationCascade,
  MarketLiquidationCascadeScan,
  MarketCvd,
  CandlesQuery,
  Ticker24hQuery,
  SpotOrderBook,
  OrderBookLevel,
  OrderBookQuery,
  OrderBookHeatmap,
  OrderBookHeatmapQuery,
  LiquidationHuntHeatmap,
  LiquidationHuntHeatmapQuery,
  LiquidationHunt,
  LiquidationHuntQuery,
  SupplyQuery,
  IntervalsQuery,
  IndicatorsQuery,
  IntervalsResponse,
  IndicatorsResponse,
  ExchangesResponse,
  FxRatesResponse,
  ProductTagsResponse,
  PumpEventsQuery,
  PumpEventsResponse,
  PumpEventDto,
  PumpScanHitDto,
  ScanPumpEventsQuery,
  ScanPumpEventsResponse,
  RSIHeatmapQuery,
  RSIHeatmapResponse,
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
  orderBookTagId,
  supplyTagId,
  holdersTagId,
  indicatorTagId,
} from './marketApi.helpers';

export const marketApi = baseApi.injectEndpoints({
  endpoints: (build) => ({
    getFxRates: build.query<FxRatesResponse, void>({
      query: () => '/api/v1/market/fx',
    }),

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

    getSpotOrderBook: build.query<SpotOrderBook, OrderBookQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [{ type: 'OrderBook' as const, id: orderBookTagId(arg) }],
    }),

    getSpotOrderBookHeatmap: build.query<OrderBookHeatmap, OrderBookHeatmapQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/heatmap',
        params: compactParams({ ...arg }),
      }),
      providesTags: (_r, _e, arg) => [
        { type: 'OrderHeatmap' as const, id: orderHeatmapTagId(arg) },
      ],
    }),

    getSupply: build.query<Supply, SupplyQuery>({
      query: (arg) => ({
        url: '/api/v1/market/supply',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: (_r, _e, arg) => [{ type: 'Supply' as const, id: supplyTagId(arg) }],
    }),

    getAssetProfile: build.query<AssetProfile, AssetProfileQuery>({
      query: (arg) => ({
        url: '/api/v1/market/asset-profile',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getHolders: build.query<AssetHolders, HoldersQuery>({
      query: (arg) => ({
        url: '/api/v1/market/holders',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      providesTags: (_r, _e, arg) => [{ type: 'Holders' as const, id: holdersTagId(arg) }],
    }),

    getOpenInterest: build.query<MarketOpenInterest, OpenInterestQuery>({
      query: (arg) => ({
        url: '/api/v1/market/open-interest',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketLiquidations: build.query<MarketLiquidations, LiquidationsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/liquidations',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketLiquidationOverview: build.query<
      MarketLiquidationOverview,
      LiquidationOverviewQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/liquidations/overview',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketLiquidationLevels: build.query<
      MarketLiquidationLevels,
      LiquidationLevelsQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-levels',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketLiquidationCascade: build.query<
      MarketLiquidationCascade,
      LiquidationCascadeQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-cascade',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketLiquidationCascadeScan: build.query<
      MarketLiquidationCascadeScan,
      LiquidationCascadeScanQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-cascade/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketLiquidationHunt: build.query<LiquidationHunt, LiquidationHuntQuery>({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-hunt',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketLiquidationHuntHeatmap: build.query<
      LiquidationHuntHeatmap,
      LiquidationHuntHeatmapQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-hunt/heatmap',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      keepUnusedDataFor: 60,
      providesTags: (_r, _e, arg) => [
        { type: 'LiqHeatmap' as const, id: liqHeatmapTagId(arg) },
      ],
    }),

    getMarketCvd: build.query<MarketCvd, CvdQuery>({
      query: (arg) => ({
        url: '/api/v1/market/cvd',
        params: compactParams({ ...(arg ?? {}) }),
      }),
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

    getPostDelist: build.query<
      import('./marketApi.types').PostDelistResponse,
      import('./marketApi.types').PostDelistQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/post-delist',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      keepUnusedDataFor: 300,
    }),

    getRSIHeatmap: build.query<RSIHeatmapResponse, RSIHeatmapQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/rsi-heatmap',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      keepUnusedDataFor: 90,
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
  useGetFxRatesQuery,
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  useListDelistScheduleQuery,
  useGetPostDelistQuery,
  useLazyListSpotMarketsQuery,
  useListIntervalsQuery,
  useGetCandlesQuery,
  useLazyGetCandlesQuery,
  useGetTicker24hQuery,
  useGetSpotOrderBookQuery,
  useGetSpotOrderBookHeatmapQuery,
  useGetSupplyQuery,
  useGetHoldersQuery,
  useGetAssetProfileQuery,
  useGetOpenInterestQuery,
  useGetMarketLiquidationsQuery,
  useGetMarketLiquidationOverviewQuery,
  useGetMarketLiquidationLevelsQuery,
  useGetMarketLiquidationCascadeQuery,
  useGetMarketLiquidationCascadeScanQuery,
  useGetMarketLiquidationHuntQuery,
  useGetMarketLiquidationHuntHeatmapQuery,
  useGetMarketCvdQuery,
  useGetIndicatorsQuery,
  useLazyGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useLazyGetPumpEventsQuery,
  useScanPumpEventsQuery,
  useGetRSIHeatmapQuery,
  usePostIndicatorsBatchMutation,
} = marketApi;
