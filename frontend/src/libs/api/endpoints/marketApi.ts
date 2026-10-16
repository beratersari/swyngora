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
  MarketBasis,
  BasisQuery,
  MarketOpenInterest,
  MarketOptions,
  MarketOptionsExpiry,
  MarketOptionsExpiryStrike,
  MarketOptionsATM,
  MarketOptionsVolPoint,
  MarketOptionsSkew,
  MarketOptionsGamma,
  MarketOptionsGammaLevel,
  MarketOptionsFinding,
  MarketOptionsVsMarket,
  MarketOptionsHistory,
  MarketOptionsHistoryWindow,
  MarketOptionsHistoryPoint,
  MarketOptionsMove,
  MarketOptionsMoves,
  MarketOptionsStrikeOI,
  MarketOptionsStrikeOIWindow,
  MarketOptionsStrikeOIRow,
  OptionsQuery,
  MarketLiquidations,
  MarketLiquidationOverview,
  MarketLiquidationLevels,
  LiquidationLevelsQuery,
  MarketLiquidationCascade,
  MarketLiquidationCascadeScan,
  LiquidationCascadeQuery,
  LiquidationCascadeScanQuery,
  MarketCvd,
  MarketUnusual,
  MarketUnusualScan,
  UnusualQuery,
  UnusualScanQuery,
  TapeBacktestQuery,
  TapeBacktestReport,
  OpenInterestQuery,
  LiquidationsQuery,
  LiquidationOverviewQuery,
  CvdQuery,
  MarketVolumeShare,
  VolumeShareQuery,
  SpotPerpLiquidity,
  SpotPerpQuery,
  MarketPerpDepthOverview,
  PerpDepthOverviewQuery,
  MarketPerpBookHistory,
  PerpBookHistoryQuery,
  MarketPerpDepthShareWindow,
  MarketPerpDepthShareDriver,
  MarketParticipation,
  ParticipationQuery,
  MarketPrintSize,
  MarketPrintSizeOverview,
  PrintSizeCoin,
  PrintSizeMarketWindow,
  PrintSizeOverviewQuery,
  PrintSizeQuery,
  MarketBotTracker,
  BotCluster,
  BotTrackerQuery,
  MarketOrderCount,
  MarketOrderCountOverview,
  OrderCountCoin,
  OrderCountMarketWindow,
  OrderCountOverviewQuery,
  OrderCountQuery,
  MarketDerivativesBoard,
  DerivativesBoardQuery,
  MarketDerivativesGrid,
  MarketDerivativesGridRow,
  MarketDerivativesGridCell,
  DerivativesGridQuery,
  ImpactScan,
  ImpactScanHit,
  ImpactScanQuery,
  WallScan,
  WallScanQuery,
  BookGaps,
  BookGapsQuery,
  BookGapScan,
  BookGapScanQuery,
  SweepRecoveryReport,
  WideSpread,
  WideSpreadQuery,
  WideSpreadScan,
  WideSpreadScanQuery,
  MarketPressure,
  PressureQuery,
  Ticker24h,
  Ticker24hQuery,
  SpotOrderBook,
  OrderBookQuery,
  OrderBookHeatmap,
  OrderBookHeatmapQuery,
  BookHistoryReport,
  BookHistoryQuery,
  LiquidationHuntHeatmap,
  LiquidationHuntHeatmapQuery,
  LiquidationPrintHeatmapQuery,
  LiquidationOpenLevels,
  LiquidationOpenLevelsQuery,
  LiquidationHunt,
  LiquidationHuntQuery,
  LiquidationMaxPain,
  LiquidationMaxPainQuery,
  MarketForensics,
  MarketForensicsQuery,
  MarketForensicsScan,
  MarketForensicsScanQuery,
  MarketForensicsRadarQuery,
  MarketBreadth,
  MarketBreadthQuery,
  MarketNetPosition,
  NetPositionQuery,
  FearGreedReport,
  CoinSituation,
  CoinSituationQuery,
  MarketVolumeSurgeScan,
  MarketVolumeSurgeScanQuery,
  MarketDeskReport,
  VenueDivergenceReport,
  SymbolMarketQuery,
  VolumeProfileReport,
  VolumeProfileQuery,
  LiquiditySweepReport,
  FlowAtPricesReport,
  CandleFootprintReport,
  CandleFootprintQuery,
  MarketLongShortRatio,
  LongShortQuery,
  HistorySpan,
  HistorySpanQuery,
  MarketFuturesHistory,
  FuturesHistoryQuery,
  MarketPositioningAnchor,
  PositioningAnchorQuery,
} from './marketApi.types';
import type { ForensicsRadarReview } from '@/libs/types/forensics.types';

export type { TapeBacktestVenue, TapeBacktestHorizon, TapeBacktestSignal } from './marketApi.types';
// delist types re-exported below

export type {
  DerivativesCoin,
  DerivativesVenue,
  CandleFootprintBook,
  CandleFootprintCandle,
  CandleFootprintRow,
} from './marketApi.types';

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
  MarketOptions,
  MarketOptionsExpiry,
  MarketOptionsExpiryStrike,
  MarketOptionsATM,
  MarketOptionsVolPoint,
  MarketOptionsSkew,
  MarketOptionsGamma,
  MarketOptionsGammaLevel,
  MarketOptionsFinding,
  MarketOptionsVsMarket,
  MarketOptionsHistory,
  MarketOptionsHistoryWindow,
  MarketOptionsHistoryPoint,
  MarketOptionsMove,
  MarketOptionsMoves,
  MarketOptionsStrikeOI,
  MarketOptionsStrikeOIWindow,
  MarketOptionsStrikeOIRow,
  OptionsQuery,
  MarketLiquidations,
  MarketLiquidationOverview,
  MarketLiquidationLevels,
  MarketLongShortRatio,
  HistorySpan,
  MarketFuturesHistory,
  MarketLiquidationCascade,
  MarketLiquidationCascadeScan,
  MarketCvd,
  MarketUnusual,
  MarketUnusualScan,
  MarketUnusualScanHit,
  UnusualQuery,
  UnusualScanQuery,
  TapeBacktestQuery,
  TapeBacktestReport,
  MarketVolumeShare,
  SpotPerpLiquidity,
  SpotPerpBook,
  SpotPerpVenue,
  SpotPerpQuery,
  MarketPerpDepthOverview,
  MarketPerpDepthCoin,
  MarketPerpDepthShareWindow,
  MarketPerpDepthShareDriver,
  MarketPerpDepthOverviewBand,
  MarketPerpDepthOverviewBandCoin,
  MarketPerpBookHistory,
  PerpBookHistoryQuery,
  MarketBreadth,
  MarketBreadthWindow,
  MarketNetPosition,
  MarketNetPositionWindow,
  FearGreedReport,
  CoinSituation,
  CoinSituationPriceCVD,
  CoinSituationOpenInterest,
  CoinSituationLiquidation,
  CoinSituationPrints,
  CoinSituationPrintAnchor,
  CoinSituationFunding,
  CoinSituationCrowd,
  CoinSituationSize,
  FearGreedFactor,
  FearGreedSample,
  FearGreedWindow,
  MarketVolumeSurgeScan,
  MarketVolumeSurgeHit,
  MarketDeskReport,
  VenueDivergenceReport,
  VolumeProfileReport,
  VolumeProfileQuery,
  LiquiditySweepReport,
  VolumeProfileVenue,
  VolumeProfileBin,
  FlowAtPricesReport,
  FlowAtPricesVenue,
  FlowAtPricesLevel,
  CandleFootprintReport,
  CandleFootprintQuery,
  MarketParticipation,
  MarketPrintSize,
  MarketPrintSizeOverview,
  PrintSizeCoin,
  PrintSizeMarketWindow,
  PrintSizeOverviewQuery,
  MarketBotTracker,
  BotCluster,
  MarketOrderCount,
  MarketOrderCountOverview,
  OrderCountCoin,
  OrderCountMarketWindow,
  OrderCountOverviewQuery,
  MarketDerivativesBoard,
  DerivativesBoardQuery,
  MarketDerivativesGrid,
  MarketDerivativesGridRow,
  MarketDerivativesGridCell,
  MarketDerivativesGridVenue,
  DerivativesGridQuery,
  ImpactScan,
  ImpactScanHit,
  ImpactScanQuery,
  WallScan,
  WallScanQuery,
  BookGaps,
  BookGapsQuery,
  BookGapScan,
  BookGapScanQuery,
  SweepRecoveryReport,
  WideSpread,
  WideSpreadQuery,
  WideSpreadScan,
  WideSpreadScanQuery,
  WideSpreadHit,
  MarketPressure,
  MarketForensics,
  MarketForensicsQuery,
  MarketForensicsScan,
  MarketForensicsScanQuery,
  CandlesQuery,
  Ticker24hQuery,
  SpotOrderBook,
  OrderBookLevel,
  OrderBookQuery,
  OrderBookHeatmap,
  OrderBookHeatmapQuery,
  LiquidationHuntHeatmap,
  LiquidationHuntHeatmapQuery,
  LiquidationPrintHeatmapQuery,
  LiquidationOpenLevels,
  LiquidationOpenLevelBook,
  LiquidationOpenLevelsQuery,
  LiquidationHunt,
  LiquidationHuntQuery,
  LiquidationMaxPain,
  LiquidationMaxPainQuery,
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

    getOrderBookHistory: build.query<BookHistoryReport, BookHistoryQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/history',
        params: compactParams({ ...arg }),
      }),
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

    getMarketBasis: build.query<MarketBasis, BasisQuery>({
      query: (arg) => ({
        url: '/api/v1/market/basis',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getOpenInterest: build.query<MarketOpenInterest, OpenInterestQuery>({
      query: (arg) => ({
        url: '/api/v1/market/open-interest',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketOptions: build.query<MarketOptions, OptionsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/options',
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

    getMarketLiquidationMaxPain: build.query<LiquidationMaxPain, LiquidationMaxPainQuery>({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-max-pain',
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

    getMarketLiquidationPrintHeatmap: build.query<
      LiquidationHuntHeatmap,
      LiquidationPrintHeatmapQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-prints/heatmap',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      keepUnusedDataFor: 60,
      providesTags: (_r, _e, arg) => [
        { type: 'LiqPrintHeatmap' as const, id: liqHeatmapTagId(arg) },
      ],
    }),

    getMarketLiquidationOpenLevels: build.query<
      LiquidationOpenLevels,
      LiquidationOpenLevelsQuery
    >({
      query: (arg) => ({
        url: '/api/v1/market/liquidation-open-levels',
        params: compactParams({ ...(arg ?? {}) }),
      }),
      keepUnusedDataFor: 60,
      providesTags: (_r, _e, arg) => [
        { type: 'LiqOpenLevels' as const, id: liqHeatmapTagId(arg) },
      ],
    }),

    getMarketLongShortRatio: build.query<MarketLongShortRatio, LongShortQuery>({
      query: (arg) => ({
        url: '/api/v1/market/long-short-ratio',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketHistorySpan: build.query<HistorySpan, HistorySpanQuery>({
      query: (arg) => ({
        url: '/api/v1/market/history-span',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketFuturesHistory: build.query<MarketFuturesHistory, FuturesHistoryQuery>({
      query: (arg) => ({
        url: '/api/v1/market/futures-history',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketPositioningAnchor: build.query<MarketPositioningAnchor, PositioningAnchorQuery>({
      query: (arg) => ({
        url: '/api/v1/market/positioning-anchor',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketCvd: build.query<MarketCvd, CvdQuery>({
      query: (arg) => ({
        url: '/api/v1/market/cvd',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketUnusual: build.query<MarketUnusual, UnusualQuery>({
      query: (arg) => ({
        url: '/api/v1/market/unusual',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    scanMarketUnusual: build.query<MarketUnusualScan, UnusualScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/unusual/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketTapeBacktest: build.query<TapeBacktestReport, TapeBacktestQuery>({
      query: (arg) => ({
        url: '/api/v1/market/tape-backtest',
        params: compactParams({ ...arg }),
      }),
    }),

    getMarketVolumeShare: build.query<MarketVolumeShare, VolumeShareQuery>({
      query: (arg) => ({
        url: '/api/v1/market/volume-share',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketPerpBookHistory: build.query<MarketPerpBookHistory, PerpBookHistoryQuery>({
      query: (arg) => ({
        url: '/api/v1/market/perp-depth/book',
        params: compactParams({ ...arg }),
      }),
    }),

    getMarketPerpDepthOverview: build.query<MarketPerpDepthOverview, PerpDepthOverviewQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/perp-depth/overview',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketBreadth: build.query<MarketBreadth, MarketBreadthQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/breadth',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketFearGreed: build.query<FearGreedReport, void>({
      query: () => ({ url: '/api/v1/market/fear-greed' }),
    }),

    getMarketSituation: build.query<CoinSituation, CoinSituationQuery>({
      query: (arg) => ({
        url: '/api/v1/market/situation',
        params: compactParams({ symbol: arg.symbol, at: arg.at }),
      }),
    }),

    scanMarketVolumeSurges: build.query<MarketVolumeSurgeScan, MarketVolumeSurgeScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/volume-surge/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getSpotPerpLiquidity: build.query<SpotPerpLiquidity, SpotPerpQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/spot-perp',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketPerpDepth: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/perp-depth',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketVolumeSurge: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/volume-surge',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketTakerFlow: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/taker-flow',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketNetPosition: build.query<MarketNetPosition, NetPositionQuery>({
      query: (arg) => ({
        url: '/api/v1/market/net-position',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketFlowAtPrices: build.query<FlowAtPricesReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/flow-at-prices',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketCandleFootprint: build.query<CandleFootprintReport, CandleFootprintQuery>({
      query: (arg) => ({
        url: '/api/v1/market/flow-at-prices/candles',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketVWAP: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/vwap',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketVolumeProfile: build.query<VolumeProfileReport, VolumeProfileQuery>({
      query: (arg) => ({
        url: '/api/v1/market/volume-profile',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketAbsorption: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/absorption',
        params: compactParams({ ...arg }),
      }),
    }),
    getSweepRecovery: build.query<SweepRecoveryReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/recovery',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketLiquiditySweeps: build.query<LiquiditySweepReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/liquidity-sweeps',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketWhales: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/whales',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketIcebergs: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/icebergs',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketBookLiquidityFlow: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/liquidity-flow',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketCorrelation: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/correlation',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketVolatility: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/volatility',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketLevels: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/levels',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketSqueezeRisk: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/squeeze-risk',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketPositioning: build.query<MarketDeskReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/positioning',
        params: compactParams({ ...arg }),
      }),
    }),
    getMarketVenueDivergence: build.query<VenueDivergenceReport, SymbolMarketQuery>({
      query: (arg) => ({
        url: '/api/v1/market/venue-divergence',
        params: compactParams({ ...arg }),
      }),
    }),

    getMarketParticipation: build.query<MarketParticipation, ParticipationQuery>({
      query: (arg) => ({
        url: '/api/v1/market/participation',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketPrintSizeOverview: build.query<MarketPrintSizeOverview, PrintSizeOverviewQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/print-size/overview',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    getMarketPrintSize: build.query<MarketPrintSize, PrintSizeQuery>({
      query: (arg) => ({
        url: '/api/v1/market/print-size',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketBotTracker: build.query<MarketBotTracker, BotTrackerQuery>({
      query: (arg) => ({
        url: '/api/v1/market/bot-tracker',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    scanMarketImpact: build.query<ImpactScan, ImpactScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/impact/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    scanSuspiciousWalls: build.query<WallScan, WallScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/walls/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    getBookGaps: build.query<BookGaps, BookGapsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/gaps',
        params: compactParams({ ...arg }),
      }),
    }),
    scanBookGaps: build.query<BookGapScan, BookGapScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/gaps/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    getWideSpread: build.query<WideSpread, WideSpreadQuery>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/spread',
        params: compactParams({ ...arg }),
      }),
    }),
    scanWideSpreads: build.query<WideSpreadScan, WideSpreadScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/orderbook/spreads/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    getMarketDerivativesBoard: build.query<MarketDerivativesBoard, DerivativesBoardQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/derivatives-board',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    getMarketDerivativesGrid: build.query<MarketDerivativesGrid, DerivativesGridQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/derivatives-grid',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    getMarketOrderCountOverview: build.query<MarketOrderCountOverview, OrderCountOverviewQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/order-count/overview',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),
    getMarketOrderCount: build.query<MarketOrderCount, OrderCountQuery>({
      query: (arg) => ({
        url: '/api/v1/market/order-count',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketPressure: build.query<MarketPressure, PressureQuery>({
      query: (arg) => ({
        url: '/api/v1/market/pressure',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    getMarketForensics: build.query<MarketForensics, MarketForensicsQuery>({
      query: (arg) => ({
        url: '/api/v1/market/forensics',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    scanMarketForensics: build.query<MarketForensicsScan, MarketForensicsScanQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/forensics/scan',
        params: compactParams({ ...(arg ?? {}) }),
      }),
    }),

    reviewSetupRadar: build.query<ForensicsRadarReview, MarketForensicsRadarQuery | void>({
      query: (arg) => ({
        url: '/api/v1/market/forensics/radar',
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
  useGetOrderBookHistoryQuery,
  useGetSupplyQuery,
  useGetHoldersQuery,
  useGetAssetProfileQuery,
  useGetMarketBasisQuery,
  useGetOpenInterestQuery,
  useGetMarketOptionsQuery,
  useGetMarketLiquidationsQuery,
  useGetMarketLiquidationOverviewQuery,
  useGetMarketLiquidationLevelsQuery,
  useGetMarketLiquidationCascadeQuery,
  useGetMarketLiquidationCascadeScanQuery,
  useGetMarketLiquidationHuntQuery,
  useGetMarketLiquidationMaxPainQuery,
  useGetMarketLiquidationHuntHeatmapQuery,
  useGetMarketLiquidationPrintHeatmapQuery,
  useGetMarketLiquidationOpenLevelsQuery,
  useGetMarketLongShortRatioQuery,
  useGetMarketHistorySpanQuery,
  useGetMarketFuturesHistoryQuery,
  useGetMarketPositioningAnchorQuery,
  useGetMarketCvdQuery,
  useGetMarketUnusualQuery,
  useScanMarketUnusualQuery,
  useGetMarketTapeBacktestQuery,
  useGetMarketVolumeShareQuery,
  useGetMarketPerpDepthOverviewQuery,
  useGetMarketPerpBookHistoryQuery,
  useGetMarketBreadthQuery,
  useGetMarketFearGreedQuery,
  useGetMarketSituationQuery,
  useScanMarketVolumeSurgesQuery,
  useLazyScanMarketVolumeSurgesQuery,
  useGetSpotPerpLiquidityQuery,
  useGetMarketPerpDepthQuery,
  useGetMarketVolumeSurgeQuery,
  useGetMarketTakerFlowQuery,
  useGetMarketNetPositionQuery,
  useGetMarketFlowAtPricesQuery,
  useGetMarketCandleFootprintQuery,
  useGetMarketVWAPQuery,
  useGetMarketVolumeProfileQuery,
  useGetMarketAbsorptionQuery,
  useGetMarketLiquiditySweepsQuery,
  useGetSweepRecoveryQuery,
  useGetMarketWhalesQuery,
  useGetMarketIcebergsQuery,
  useGetMarketBookLiquidityFlowQuery,
  useGetMarketCorrelationQuery,
  useGetMarketVolatilityQuery,
  useGetMarketLevelsQuery,
  useGetMarketSqueezeRiskQuery,
  useGetMarketPositioningQuery,
  useGetMarketVenueDivergenceQuery,
  useGetMarketParticipationQuery,
  useGetMarketPrintSizeOverviewQuery,
  useGetMarketPrintSizeQuery,
  useGetMarketBotTrackerQuery,
  useScanMarketImpactQuery,
  useScanSuspiciousWallsQuery,
  useGetBookGapsQuery,
  useScanBookGapsQuery,
  useGetWideSpreadQuery,
  useScanWideSpreadsQuery,
  useGetMarketDerivativesBoardQuery,
  useGetMarketDerivativesGridQuery,
  useGetMarketOrderCountOverviewQuery,
  useGetMarketOrderCountQuery,
  useGetMarketPressureQuery,
  useGetMarketForensicsQuery,
  useScanMarketForensicsQuery,
  useReviewSetupRadarQuery,
  useGetIndicatorsQuery,
  useLazyGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useLazyGetPumpEventsQuery,
  useScanPumpEventsQuery,
  useGetRSIHeatmapQuery,
  usePostIndicatorsBatchMutation,
} = marketApi;
