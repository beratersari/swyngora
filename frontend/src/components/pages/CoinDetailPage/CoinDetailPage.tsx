import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Select, message } from 'antd';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { CandleChartHost } from '@/components/molecules/CandleChartHost';
import { snapMarkersToCandleTimes } from '@/components/molecules/CandleChartHost/CandleChartHost.markers';
import {
  bookHeatColumns,
  formatBookReachPct,
  perpBookCells,
  perpBookReach,
  CANDLE_FOOTPRINT_DEFAULT_CANDLES,
  CHART_BOOK_LIVE_MAX,
  candleFootprintSpan,
  candleFootprintView,
  chartBookWindow,
  chartBotMarks,
  chartProfileRead,
  chartSweepMarks,
  chartSweepVenue,
  chartTapeSpan,
  sweepSpanOutside,
  anchoredVenueSeries,
  chartAnchors,
  chartPaneBucket,
  chartPaneFromSec,
  crowdGapLines,
  crowdStoredLines,
  crowdStoredSeries,
  footprintLevels,
  footprintWindow,
  formatRecordedSpan,
  fundingRateLines,
  futuresPaneColor,
  recordedParts,
  liqHeatCells,
  liqHeatNotes,
  openLevelAnchorLines,
  openLevelDelta,
  openLevelNotes,
  openLevelPoints,
  openLevelSegments,
  openLevelSizeSince,
  liquidationBucketBars,
  liquidationProfile,
  openInterestLines,
  positioningClickLines,
  venuesWithoutTop,
  type LiqHeatSide,
  type LiqHeatVenue,
} from './chartDesk';
import type {
  CandleChartLowerSeries,
  CandleChartMarker,
  CandleChartOverlay,
  CandleChartPriceLine,
} from '@/components/molecules/CandleChartHost/CandleChartHost.types';
import { DetailChartToolbar, type CvdChartRange } from '@/components/organisms/DetailChartToolbar';
import {
  formatSignedCompactUsd,
  pickOpenLevelBook,
} from '@/components/organisms/OpenLiquidationLevels/OpenLiquidationLevels.helpers';
import { DetailHeader } from '@/components/organisms/DetailHeader';
import { DetailStats } from '@/components/organisms/DetailStats';
import { HolderPanel } from '@/components/organisms/HolderPanel';
import { PostDelistPanel } from '@/components/organisms/PostDelistPanel';
import { TapePanel } from '@/components/organisms/TapePanel';
import { UnusualContext, UNUSUAL_POLL_MS } from '@/components/organisms/UnusualContext';
import {
  DEFAULT_LIQ_HEATMAP_RANGE,
  LIQ_HEATMAP_POLL_MS,
  liqMapReadsArchive,
} from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.constants';
import type { LiqHeatRange, LiqHeatSource } from '@/components/organisms/LiquidationHeatmap/LiquidationHeatmap.types';
import { PERP_LIQUIDITY_LIMIT } from '@/components/organisms/PerpLiquidityOverview';
import { IndicatorPanel, emaColor } from '@/components/organisms/IndicatorPanel';
import { BookGapsCard } from '@/components/organisms/BookGapsCard';
import { SpreadOpenNote } from './SpreadOpenNote';
import {
  OrderBookPanel,
  ORDER_BOOK_LEVELS,
  ORDER_BOOK_POLL_MS,
} from '@/components/organisms/OrderBookPanel';
import { OrderDepthChart } from '@/components/organisms/OrderDepthChart';
import {
  DEFAULT_ORDER_HEATMAP_WINDOW,
  ORDER_HEATMAP_HISTORY_POLL_MS,
  ORDER_HEATMAP_POLL_MS,
  OrderHeatmap,
} from '@/components/organisms/OrderHeatmap';
import { PaperTradeForm, type PaperTradeFormValues } from '@/components/organisms/PaperTradeForm';
import {
  rtkErrorMessage,
  useAddWatchlistItemMutation,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useLazyGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useGetSupplyQuery,
  useGetHoldersQuery,
  useGetAssetProfileQuery,
  useGetOpenInterestQuery,
  useGetMarketNetPositionQuery,
  useGetMarketLiquidationsQuery,
  useGetMarketCvdQuery,
  useGetMarketHistorySpanQuery,
  useGetMarketFuturesHistoryQuery,
  useGetMarketPositioningAnchorQuery,
  useGetMarketFlowAtPricesQuery,
  useGetMarketLiquiditySweepsQuery,
  useGetMarketVolumeProfileQuery,
  useGetMarketCandleFootprintQuery,
  useGetMarketUnusualQuery,
  useGetMarketLiquidationLevelsQuery,
  useGetMarketLiquidationHuntHeatmapQuery,
  useGetMarketLiquidationPrintHeatmapQuery,
  useGetMarketLiquidationOpenLevelsQuery,
  useGetMarketVolumeShareQuery,
  useGetMarketPerpDepthOverviewQuery,
  useGetMarketParticipationQuery,
  useGetMarketPrintSizeQuery,
  useGetMarketAbsorptionQuery,
  useGetMarketPerpDepthQuery,
  useGetMarketPerpBookHistoryQuery,
  useGetMarketBotTrackerQuery,
  useGetMarketOrderCountQuery,
  useGetMarketPressureQuery,
  useGetMarketSituationQuery,
  useGetMarketForensicsQuery,
  useGetMarketSnapshotQuery,
  useGetTicker24hQuery,
  useGetSpotOrderBookQuery,
  useGetSpotOrderBookHeatmapQuery,
  useGetPostDelistQuery,
  useListDelistScheduleQuery,
  useGetWatchlistQuery,
  useLazyGetCandlesQuery,
  useLazyGetPumpEventsQuery,
  useListIntervalsQuery,
  useListPortfoliosQuery,
  useGetPortfolioQuery,
  useGetPortfolioSituationChangeQuery,
  useListScannerResultsQuery,
  usePlacePortfolioOrderMutation,
  useRemoveWatchlistItemMutation,
  type MarketExchange,
  type PumpEventDto,
  type IndicatorsResponse,
} from '@/libs/api';
import { useDisplayCurrency, useDocumentVisible, useMediaQuery } from '@/libs/hooks';
import { usePriceSubscription, usePortfolioSubscription } from '@/libs/realtime';
import {
  formatDelistDay,
  formatPrice,
  indicatorPointsToEmaLine,
  mergeIndicatorPoints,
  newPaperIdempotencyKey,
  rtkCurrent,
  rtkCurrentPending,
} from '@/libs/utils';
import { mediaQueries, semanticColors } from '@/styles/tokens';
import {
  apiCandlesToChart,
  DEFAULT_DETAIL_TAB,
  detailStateToSearchParams,
  filterValidApiCandles,
  preferLongerCandleSeries,
  candleIntervalSeconds,
  intervalToSeconds,
  marketsBackPathFromSession,
  mergeCandleHistory,
  oldestCandleOpenTimeMs,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  pairQuote,

  trimCandlesToMax,
  type ApiCandle,
  type DetailTab,
} from '@/libs/utils';
import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
  DETAIL_CANDLE_FIRST_LIMIT,
  DEFAULT_DETAIL_PUMP_THRESHOLD_PCT,
  DEFAULT_DETAIL_SERIES_POLL_MS,
  DEFAULT_DETAIL_TICKER_POLL_MS,
  DEFAULT_EMA_PERIODS,
  DEFAULT_RSI_PERIOD,
  DETAIL_API_BAR_MAX,
  DETAIL_CANDLE_MAX_LIMIT,
  DETAIL_CANDLE_PAGE_SIZE,
  DETAIL_INDICATOR_LIMIT,
  DESK_CHART_HEIGHT,
  PHONE_CHART_HEIGHT,
} from '@/config/constants';
import {
  ChartAndBook,
  ChartCard,
  ChartTitleRow,
  DeskTabs,
  PageStack,
  PaperTradeCard,
  SideStack,
  TabStack,
} from './CoinDetailPage.styles';
import {
  appendCandlesAfter,
  delistCandleQueryEndTime,
  delistEventsToVertLines,


  mergeChartMarkers,
  mergePumpEvents,
  livePumpEventsForPair,
  pumpEventsToChartMarkers,
  scannerResultsToChartMarkers,
} from './CoinDetailPage.helpers';
import {
  anchoredCvdByCandle,
  cvdAnchorStartsLate,
  cvdBarSeconds,
  cvdRangeForAnchor,
  deltaByCandle,
  formatUtcMinute,
  futuresCvdSeries,
  futuresSizeSeries,
  sizedCvdByCandle,
  snapCandleOpen,
  spotCvdSeries,
  liquidationChartLevels,
  runningCvdByCandle,
  type CvdChartSeries,
} from './chartTape';
import { formatSignedTapeNum } from '@/components/organisms/TapePanel/TapePanel.helpers';
import {
  absorptionDeskNote,
  chartPresetLayers,
  chartWallNotes,
  formatDeskPct,
  isChartPreset,
  liquidationLevelReview,
  matchingChartPreset,
  perpBidShareNote,
  printSizeDeskNote,
  type ChartDeskLayers,
} from './chartPresets';
import { CoinSituation } from '@/components/molecules/CoinSituation';
import { openPositionFor, PositionSituation } from '@/components/molecules/PositionSituation';
import {
  clickSecFromForensicsAt,
  forensicsKnownUntil,
  COIN_FORENSICS_DURING,
  COIN_FORENSICS_EXCHANGE,
  COIN_FORENSICS_WINDOW,
  coinForensicsHref,
  CoinCandleForensics,
} from '@/components/organisms/CoinCandleForensics';
import { CoinCandleSituation, situationCandleCloseMs } from '@/components/organisms/CoinCandleSituation';
import type { ForensicsReport } from '@/libs/types/forensics.types';
import { CoinFlowTab } from './CoinFlowTab';
import { PastOrderBook } from './PastOrderBook';
import { CoinStructureTab } from './CoinStructureTab';

/**
 * Coin detail: header + 24h/supply strip, then CoinMarketCap-style tabs
 * (overview chart, order book, tape, flow, structure, holders, indicators, paper trade).

 * Live candles poll a short window; pan-left pages older bars via endTime.
 * Pump/dump markers: live window is polled; each history page fetches pumps for
 * the same endTime range so markers exist on older candles, not only the right edge.
 * Route: /markets/:exchange/:symbol
 */
export function CoinDetailPage() {
  const { t } = useTranslation(['detail', 'common']);
  const { convert, currency: displayCurrency, rates: fxRates } = useDisplayCurrency();
  const { exchange: exchangeParam, symbol: symbolParam } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();
  const isPhone = useMediaQuery(mediaQueries.phone);

  const exchange = parseExchangeParam(exchangeParam);
  const symbol = parseSymbolParam(symbolParam);
  const urlState = useMemo(() => parseDetailSearchParams(searchParams), [searchParams]);

  const [showEma, setShowEma] = useState(true);
  /** Older pages (endTime fetches), merged with the live window for the chart. */
  const [historyCandles, setHistoryCandles] = useState<ApiCandle[]>([]);
  /** Pump events for those history pages (live pumps stay on RTK query). */
  const [historyPumpEvents, setHistoryPumpEvents] = useState<PumpEventDto[]>([]);
  const [historyIndicatorPoints, setHistoryIndicatorPoints] = useState<
    NonNullable<IndicatorsResponse['points']>
  >([]);
  const [historyExhausted, setHistoryExhausted] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  /** |return %| threshold for pump/dump markers on the chart. */
  const [pumpThresholdPct, setPumpThresholdPct] = useState(
    DEFAULT_DETAIL_PUMP_THRESHOLD_PCT,
  );
  const [showPumpMarkers, setShowPumpMarkers] = useState(true);
  /** Backend group size; empty until the first book returns a default. */
  const [orderBookGroup, setOrderBookGroup] = useState('');
  const [orderHeatmapWindow, setOrderHeatmapWindow] = useState(DEFAULT_ORDER_HEATMAP_WINDOW);
  const [showSignalMarkers, setShowSignalMarkers] = useState(true);
  const [layers, setLayers] = useState<ChartDeskLayers>(() => chartPresetLayers('all'));
  const chartPreset = matchingChartPreset(layers);
  const {
    showDelta,
    showCvd,
    showLiqLevels,
    showBook,
    showFootprint,
    showProfile,
    showSweeps,
    showBots,
    showCrowd,
    showOi,
    showFunding,
    showLiqNotional,
    showLiqMap,
    showOpenLevels,
  } = layers;
  const setLayer = (key: keyof ChartDeskLayers) => (value: boolean) => {
    setLayers((prev) => ({ ...prev, [key]: value }));
  };
  const choosePreset = (value: string) => {
    if (isChartPreset(value)) setLayers(chartPresetLayers(value));
  };
  const [cvdRange, setCvdRange] = useState<CvdChartRange>('24h');
  /** Candle open, unix seconds, where CVD restarts. Null keeps the day and 4h lines only. */
  const [cvdClickSec, setCvdClickSec] = useState<number | null>(null);
  useEffect(() => {
    setCvdClickSec(null);
  }, [exchange, symbol]);
  const [liqMapRange, setLiqMapRange] = useState<LiqHeatRange>(DEFAULT_LIQ_HEATMAP_RANGE);
  const [liqMapVenue, setLiqMapVenue] = useState<LiqHeatVenue>('combined');
  const [liqMapSide, setLiqMapSide] = useState<LiqHeatSide>('totals');
  const [liqMapSource, setLiqMapSource] = useState<LiqHeatSource>('prints');
  const [footWindow, setFootWindow] = useState<'5m' | '15m' | '1h' | '4h' | '1d' | '3d' | '7d' | '30d'>('1h');
  const [footSpan, setFootSpan] = useState<{ from: number; to: number } | null>(null);
  const [bookWindow, setBookWindow] = useState(CHART_BOOK_LIVE_MAX);
  const [tapeSpan, setTapeSpan] = useState<{ from: number; to: number } | null>(null);
  /** Guards against applying history pages after exchange/symbol/interval change. */
  const historyRequestIdRef = useRef(0);

  const skip = !symbol || !exchange;
  // RTK args require a concrete exchange; skipped when path venue is invalid.
  const exchangeArg = exchange ?? 'binance';
  const isEquity = exchange === 'nasdaq' || exchange === 'bist';
  const situationQuery = useGetMarketSituationQuery(
    { symbol: symbol ?? '' },
    {
      skip: skip || isEquity,
      pollingInterval: visible ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const forensicsAt =
    !isEquity && cvdClickSec != null ? new Date(cvdClickSec * 1000).toISOString() : undefined;
  const forensicsAsOf = forensicsAt
    ? forensicsKnownUntil(forensicsAt, COIN_FORENSICS_DURING)
    : undefined;
  const forensicsQuery = useGetMarketForensicsQuery(
    {
      symbol: symbol ?? '',
      at: forensicsAt ?? '',
      asOf: forensicsAsOf,
      exchange: COIN_FORENSICS_EXCHANGE,
      window: COIN_FORENSICS_WINDOW,
      during: COIN_FORENSICS_DURING,
    },
    {
      skip: skip || isEquity || !symbol || forensicsAt == null || !visible,
      refetchOnFocus: true,
    },
  );
  const intervalsQuery = useListIntervalsQuery(
    { exchange: exchangeArg },
    { skip: !exchange },
  );
  const supportedIntervals = intervalsQuery.data?.intervals;
  // Only block series on the first intervals load. On error, fall through with
  // resolveInterval defaults so candles are not stuck on skeleton forever.
  const waitingForIntervals =
    !skip &&
    !supportedIntervals?.length &&
    intervalsQuery.isLoading &&
    !intervalsQuery.isError;

  const interval = resolveInterval(urlState.interval, supportedIntervals);
  const seriesKey = `${exchange}|${symbol}|${interval}`;

  // Reset paged history during render when the series identity changes so the
  // first paint after navigation is not mixed old-history + new live candles.
  const [historySeriesKey, setHistorySeriesKey] = useState(seriesKey);
  if (historySeriesKey !== seriesKey) {
    setHistorySeriesKey(seriesKey);
    historyRequestIdRef.current += 1;
    setHistoryCandles([]);
    setHistoryPumpEvents([]);
    setHistoryIndicatorPoints([]);
    setHistoryExhausted(false);
    setHistoryLoading(false);
    setFootSpan(null);
    setTapeSpan(null);
  }

  useEffect(() => {
    setOrderBookGroup('');
  }, [exchange, symbol]);

  // Threshold / marker toggle: drop history pumps (API minReturnPct or skip changed).
  // Candles stay; live pumps refetch via RTK args; re-pan reloads history markers.
  // Clear historyLoading so an in-flight page cannot leave the loader stuck.
  useEffect(() => {
    historyRequestIdRef.current += 1;
    setHistoryPumpEvents([]);
    setHistoryLoading(false);
  }, [pumpThresholdPct, showPumpMarkers]);

  useEffect(() => {
    if (!supportedIntervals?.length) return;
    if (supportedIntervals.includes(urlState.interval)) return;
    const next = resolveInterval(urlState.interval, supportedIntervals);
    setSearchParams(detailStateToSearchParams({ interval: next, tab: urlState.tab }), {
      replace: true,
    });
  }, [supportedIntervals, urlState.interval, urlState.tab, setSearchParams]);

  // Candles start immediately with the URL/default interval. History / pumps /
  // indicators wait until the venue interval list is known so we do not page
  // the wrong series.
  const skipSeries = skip || waitingForIntervals;
  const supplyAsset = symbol;

  const watchlistQuery = useGetWatchlistQuery(undefined, { refetchOnFocus: true });
  const delistQuery = useListDelistScheduleQuery(
    { exchange: (exchange ?? 'binance') as MarketExchange },
    { skip: !exchange || isEquity },
  );
  const delistSchedule = rtkCurrent(delistQuery);
  const delistHit = useMemo(() => {
    if (!symbol || !delistSchedule?.items?.length) return null;
    return (
      delistSchedule.items.find(
        (it) => (it.symbol ?? '').toUpperCase() === String(symbol).toUpperCase(),
      ) ?? null
    );
  }, [delistSchedule?.items, symbol]);
  const delistTime = delistHit?.delistTime ?? null;
  const announcedAt = delistHit?.announcedAt ?? null;
  const candleEndTime = useMemo(() => delistCandleQueryEndTime(delistTime), [delistTime]);
  const [addWatch, addWatchState] = useAddWatchlistItemMutation();
  const [removeWatch, removeWatchState] = useRemoveWatchlistItemMutation();
  const [fetchOlderCandles] = useLazyGetCandlesQuery();
  const [fetchOlderIndicators] = useLazyGetIndicatorsQuery();
  const [fetchOlderPumps] = useLazyGetPumpEventsQuery();
  const booksQuery = useListPortfoliosQuery(undefined, { refetchOnFocus: true });
  const books = booksQuery.data?.portfolios ?? [];
  const [paperBookId, setPaperBookId] = useState(() => {
    try {
      return localStorage.getItem('swyngora.portfolioBookId') ?? '';
    } catch {
      return '';
    }
  });
  useEffect(() => {
    if (paperBookId) return;
    if (books.length === 1 && books[0]?.id) setPaperBookId(books[0].id);
  }, [books, paperBookId]);
  usePortfolioSubscription(paperBookId || undefined, visible && Boolean(paperBookId));
  const paperPortfolio = useGetPortfolioQuery(
    paperBookId ? { portfolioId: paperBookId } : undefined,
    { skip: !paperBookId, refetchOnFocus: true },
  );
  const heldPosition = openPositionFor(
    rtkCurrent(paperPortfolio)?.positions,
    exchange ?? '',
    symbol ?? '',
  );
  const situationChange = useGetPortfolioSituationChangeQuery(
    {
      symbol: symbol ?? '',
      exchange: exchangeArg,
      portfolioId: paperBookId || undefined,
    },
    {
      skip: skip || isEquity || !paperBookId || !heldPosition?.entrySituation,
      pollingInterval: visible ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const [placePaperOrder, placePaperState] = usePlacePortfolioOrderMutation();
  const scannerResultsQuery = useListScannerResultsQuery(
    { limit: 100, offset: 0 },
    { skip, pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0, refetchOnFocus: true },
  );
  const watched = Boolean(
    watchlistQuery.data?.items?.some(
      (it) => it.exchange === exchange && it.symbol === symbol,
    ),
  );

  const { connected: livePrices } = usePriceSubscription(
    symbol && exchangeArg ? [{ exchange: exchangeArg, symbol }] : [],
    Boolean(visible && !skip),
  );

  const orderBookQuery = useGetSpotOrderBookQuery(
    {
      exchange: exchangeArg,
      symbol,
      group: orderBookGroup || undefined,
      limit: ORDER_BOOK_LEVELS,
    },
    {
      skip: skip || isEquity,
      pollingInterval: visible ? ORDER_BOOK_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const orderHeatmapQuery = useGetSpotOrderBookHeatmapQuery(
    {
      exchange: exchangeArg,
      symbol,
      group: orderBookGroup || undefined,
      window: orderHeatmapWindow,
    },
    {
      skip: skip || isEquity,
      pollingInterval: visible
        ? orderHeatmapWindow > 1800
          ? ORDER_HEATMAP_HISTORY_POLL_MS
          : ORDER_HEATMAP_POLL_MS
        : 0,
      refetchOnFocus: true,
    },
  );
  const suggestedBookGroups = rtkCurrent(orderBookQuery)?.suggestedGroupSizes;
  useEffect(() => {
    if (!orderBookGroup || !suggestedBookGroups?.length) return;
    if (!suggestedBookGroups.includes(orderBookGroup)) {
      setOrderBookGroup('');
    }
  }, [orderBookGroup, suggestedBookGroups]);

  const tickerQuery = useGetTicker24hQuery(
    { exchange: exchangeArg, symbol },
    {
      skip,
      pollingInterval: visible && !livePrices ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const pastDelist = Boolean(rtkCurrent(tickerQuery)?.halted);
  const postDelistQuery = useGetPostDelistQuery(
    {
      exchange: (exchange ?? 'binance') as MarketExchange,
      symbol: symbol ?? '',
      interval,
      limit: DEFAULT_DETAIL_CANDLE_LIMIT,
    },
    {
      skip: skip || isEquity || !symbol || !exchange || !pastDelist,
      pollingInterval: visible ? 120_000 : 0,
      refetchOnFocus: true,
    },
  );

  const supplyQuery = useGetSupplyQuery(
    { asset: supplyAsset },
    {
      skip: skip || !supplyAsset || isEquity,
      pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const holdersQuery = useGetHoldersQuery(
    { asset: supplyAsset },
    {
      skip: skip || !supplyAsset || isEquity,
      refetchOnFocus: true,
    },
  );
  const profileQuery = useGetAssetProfileQuery(
    { asset: supplyAsset },
    {
      skip: skip || !supplyAsset || isEquity,
      refetchOnFocus: true,
    },
  );

  const perpSymbol = symbol;
  const tapeQueryArg = { exchange: 'all' as const, symbol: perpSymbol };
  const skipTape = skip || isEquity || !perpSymbol;
  const netPositionQuery = useGetMarketNetPositionQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const openInterestQuery = useGetOpenInterestQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const liquidationsQuery = useGetMarketLiquidationsQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const cvdFetchRange = cvdRangeForAnchor(
    cvdRange,
    showCvd ? cvdClickSec : null,
    Math.floor(Date.now() / 1000),
  );
  const cvdClickIso =
    showCvd && cvdClickSec != null ? new Date(cvdClickSec * 1000).toISOString() : undefined;
  const cvdQuery = useGetMarketCvdQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const cvdChartQuery = useGetMarketCvdQuery(
    { exchange: 'all', symbol: perpSymbol, range: cvdFetchRange, anchor: cvdClickIso },
    {
      skip: skipTape || cvdFetchRange === '24h' || (!showCvd && !showDelta),
      pollingInterval: visible && cvdFetchRange !== '24h' ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const unusualQuery = useGetMarketUnusualQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? UNUSUAL_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const chartBookQuery = useGetSpotOrderBookHeatmapQuery(
    { exchange: exchangeArg, symbol, window: bookWindow },
    {
      skip: skipTape || !showBook,
      pollingInterval:
        visible && showBook
          ? bookWindow > CHART_BOOK_LIVE_MAX
            ? ORDER_HEATMAP_HISTORY_POLL_MS
            : ORDER_HEATMAP_POLL_MS
          : 0,
      refetchOnFocus: true,
    },
  );
  const chartProfileQuery = useGetMarketVolumeProfileQuery(
    tapeSpan
      ? {
          exchange: 'all',
          symbol: perpSymbol,
          startTime: new Date(tapeSpan.from * 1000).toISOString(),
          endTime: new Date(tapeSpan.to * 1000).toISOString(),
        }
      : { exchange: 'all', symbol: perpSymbol, window: '24h' },
    {
      skip: skipTape || !showProfile,
      pollingInterval: visible && showProfile ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const sweepVenue = chartSweepVenue(exchangeArg);
  const chartSweepQuery = useGetMarketLiquiditySweepsQuery(
    { exchange: sweepVenue, symbol: perpSymbol },
    {
      skip: skipTape || !showSweeps,
      pollingInterval: visible && showSweeps ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const footprintQuery = useGetMarketFlowAtPricesQuery(
    { exchange: 'all', symbol: perpSymbol, window: footWindow },
    {
      skip: skipTape || !showFootprint,
      pollingInterval: visible && showFootprint ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const historySpanQuery = useGetMarketHistorySpanQuery(
    { symbol: perpSymbol },
    { skip: skipTape, refetchOnFocus: true },
  );
  const liqMapActive = showLiqMap && urlState.tab === 'overview';
  const liqMapModelQuery = useGetMarketLiquidationHuntHeatmapQuery(
    { symbol: perpSymbol, range: liqMapRange, exchange: 'all' },
    {
      skip: skipTape || !liqMapActive || liqMapSource !== 'model',
      pollingInterval: visible && liqMapActive && liqMapSource === 'model' ? LIQ_HEATMAP_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const liqMapPrintQuery = useGetMarketLiquidationPrintHeatmapQuery(
    { symbol: perpSymbol, range: liqMapRange, exchange: 'all' },
    {
      skip: skipTape || !liqMapActive || liqMapSource !== 'prints',
      pollingInterval: visible && liqMapActive && liqMapSource === 'prints' ? LIQ_HEATMAP_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const liqMapQuery = liqMapSource === 'prints' ? liqMapPrintQuery : liqMapModelQuery;
  const openLevelsQuery = useGetMarketLiquidationOpenLevelsQuery(
    { symbol: perpSymbol, range: liqMapRange, exchange: 'all' },
    {
      skip: skipTape || !showOpenLevels || urlState.tab !== 'overview',
      pollingInterval:
        visible && showOpenLevels && urlState.tab === 'overview' ? LIQ_HEATMAP_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const liqLevelsQuery = useGetMarketLiquidationLevelsQuery(
    { symbol: perpSymbol, exchange: 'all', range: '24h' },
    {
      skip: skipTape || !showLiqLevels,
      pollingInterval: visible && showLiqLevels ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const volumeShareQuery = useGetMarketVolumeShareQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const perpLiqQuery = useGetMarketPerpDepthOverviewQuery(
    { exchange: 'all', limit: PERP_LIQUIDITY_LIMIT, rangePct: 1 },
    {
      skip: skipTape,
      pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const participationQuery = useGetMarketParticipationQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const printSizeQuery = useGetMarketPrintSizeQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const absorptionQuery = useGetMarketAbsorptionQuery(
    { symbol: perpSymbol, exchange: 'all' },
    {
      skip: skipTape || chartPreset !== 'orderFlow',
      pollingInterval: visible && chartPreset === 'orderFlow' ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const perpDepthQuery = useGetMarketPerpDepthQuery(
    { symbol: perpSymbol, exchange: 'all', rangePct: 1 },
    {
      skip: skipTape || chartPreset !== 'liquidity',
      pollingInterval: visible && chartPreset === 'liquidity' ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const perpBandsOn = !isEquity && (showLiqMap || chartPreset === 'liquidity');
  const perpBookQuery = useGetMarketPerpBookHistoryQuery(
    { symbol: perpSymbol, exchange: 'all', windowSec: bookWindow },
    {
      skip: skipTape || !perpBandsOn,
      pollingInterval: visible && perpBandsOn ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const botTrackerQuery = useGetMarketBotTrackerQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const orderCountQuery = useGetMarketOrderCountQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const pressureQuery = useGetMarketPressureQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
    refetchOnFocus: true,
  });
  const snapshotQuery = useGetMarketSnapshotQuery(tapeQueryArg, {
    skip: skipTape,
    pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
    refetchOnFocus: true,
  });

  // First-paint slice + full live window in parallel. The short request usually
  // returns first so the chart can draw before the 300-bar poll window arrives.
  const firstCandlesQuery = useGetCandlesQuery(
    {
      exchange: exchangeArg,
      symbol,
      interval,
      limit: DETAIL_CANDLE_FIRST_LIMIT,
      ...(candleEndTime ? { endTime: candleEndTime } : {}),
    },
    { skip, refetchOnFocus: true },
  );
  const candlesQuery = useGetCandlesQuery(
    {
      exchange: exchangeArg,
      symbol,
      interval,
      limit: DEFAULT_DETAIL_CANDLE_LIMIT,
      ...(candleEndTime ? { endTime: candleEndTime } : {}),
    },
    {
      skip,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const liveHeadReady = Boolean(
    rtkCurrent(firstCandlesQuery)?.candles?.length || rtkCurrent(candlesQuery)?.candles?.length,
  );

  const liveCandleRows = preferLongerCandleSeries(
    rtkCurrent(firstCandlesQuery)?.candles,
    rtkCurrent(candlesQuery)?.candles,
  );
  const liveCandles = useMemo(
    () => filterValidApiCandles(liveCandleRows),
    [liveCandleRows],
  );

  const mergedCandles = useMemo(
    () =>
      trimCandlesToMax(
        mergeCandleHistory(historyCandles, liveCandles),
        DETAIL_CANDLE_MAX_LIMIT,
      ),
    [historyCandles, liveCandles],
  );
  const postDelistView = rtkCurrent(postDelistQuery);
  const chartSeriesKey = `${seriesKey}|${delistTime ?? ''}|${postDelistView?.source ?? ''}`;
  const offVenueApiCandles = useMemo(
    () => filterValidApiCandles(postDelistView?.candles),
    [postDelistView?.candles],
  );
  const allCandles = useMemo(() => {
    if (!pastDelist || !postDelistView?.available) return mergedCandles;
    return appendCandlesAfter(mergedCandles, offVenueApiCandles);
  }, [mergedCandles, offVenueApiCandles, pastDelist, postDelistView?.available]);
  const postDelistLast = useMemo(() => {
    const px = postDelistView?.lastPrice;
    if (!px || displayCurrency === 'native') return px;
    const converted = convert(Number(px), postDelistView?.quote || 'USD');
    return converted == null ? px : String(converted);
  }, [convert, displayCurrency, postDelistView?.lastPrice, postDelistView?.quote]);

  const chartData = useMemo(() => {
    const raw = apiCandlesToChart(allCandles);
    if (displayCurrency === 'native') return raw;
    const q = pairQuote(symbol, exchange);
    const out = [];
    for (const bar of raw) {
      const open = convert(bar.open, q);
      const high = convert(bar.high, q);
      const low = convert(bar.low, q);
      const close = convert(bar.close, q);
      if (open == null || high == null || low == null || close == null) continue;
      out.push({ ...bar, open, high, low, close });
    }
    return out;
  }, [allCandles, convert, displayCurrency, exchange, symbol]);

  const paneBarSec = candleIntervalSeconds(interval);
  const paneSpanSec =
    chartData.length > 0 && paneBarSec > 0
      ? chartData[chartData.length - 1]!.time + paneBarSec - chartData[0]!.time
      : 0;
  const paneBucket = chartPaneBucket(paneSpanSec);
  const paneFromSec =
    chartData[0] != null
      ? chartPaneFromSec(chartData[0].time, Math.floor(Date.now() / 1000), paneBucket)
      : undefined;
  const futuresPaneBase = {
    symbol: perpSymbol,
    exchange: 'all' as const,
    from: paneFromSec != null ? new Date(paneFromSec * 1000).toISOString() : undefined,
    bucket: paneBucket,
  };
  const crowdHistQuery = useGetMarketFuturesHistoryQuery(
    { ...futuresPaneBase, metric: 'long_short' },
    {
      skip: skipTape || !showCrowd || paneFromSec == null,
      pollingInterval: visible && showCrowd ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const oiHistQuery = useGetMarketFuturesHistoryQuery(
    { ...futuresPaneBase, metric: 'open_interest' },
    {
      skip: skipTape || !showOi || paneFromSec == null,
      pollingInterval: visible && showOi ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const fundingHistQuery = useGetMarketFuturesHistoryQuery(
    { ...futuresPaneBase, metric: 'funding' },
    {
      skip: skipTape || !showFunding || paneFromSec == null,
      pollingInterval: visible && showFunding ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const liqHistQuery = useGetMarketFuturesHistoryQuery(
    { ...futuresPaneBase, metric: 'liquidations' },
    {
      skip: skipTape || !showLiqNotional || paneFromSec == null,
      pollingInterval: visible && showLiqNotional ? 60_000 : 0,
      refetchOnFocus: true,
    },
  );
  const positioningClickOn = !isEquity && (showOi || showFunding || showLiqNotional);
  const positioningAnchorQuery = useGetMarketPositioningAnchorQuery(
    {
      symbol: perpSymbol ?? '',
      exchange: 'all',
      anchor: cvdClickSec != null ? new Date(cvdClickSec * 1000).toISOString() : '',
      from: futuresPaneBase.from,
      bucket: paneBucket,
    },
    {
      skip: skipTape || !positioningClickOn || cvdClickSec == null || !perpSymbol || paneFromSec == null,
      refetchOnFocus: true,
    },
  );
  const positioningClickText = useMemo(() => {
    if (!positioningClickOn || cvdClickSec == null) return '';
    const report = rtkCurrent(positioningAnchorQuery);
    const venueName = (id: string) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id });
    const join = (ids?: string[]) => (ids ?? []).map(venueName).filter(Boolean).join(', ');
    const changes = (rows?: { exchange?: string; change?: string }[]) =>
      (rows ?? [])
        .filter((row) => row.exchange && row.change)
        .map((row) => `${venueName(row.exchange ?? '')} ${row.change}`)
        .join(', ');
    const parts = [t('detail:chart.positioningClickSince', { time: formatUtcMinute(cvdClickSec) })];
    const oi = changes(report?.openInterest);
    const funding = changes(report?.funding);
    const liq = changes(report?.liquidations);
    if (oi) parts.push(t('detail:chart.positioningClickOi', { changes: oi }));
    if (funding) parts.push(t('detail:chart.positioningClickFunding', { changes: funding }));
    if (liq) parts.push(t('detail:chart.positioningClickLiq', { changes: liq }));
    const missingOi = join(report?.missingOpenInterest);
    const missingFunding = join(report?.missingFunding);
    const missingLiq = join(report?.missingLiquidations);
    const late = join(report?.lateLiquidations);
    if (missingOi) parts.push(t('detail:chart.positioningClickOiMissing', { venues: missingOi }));
    if (missingFunding) parts.push(t('detail:chart.positioningClickFundingMissing', { venues: missingFunding }));
    if (missingLiq) parts.push(t('detail:chart.positioningClickLiqMissing', { venues: missingLiq }));
    if (late) parts.push(t('detail:chart.positioningClickLiqLate', { venues: late }));
    return parts.join(' ');
  }, [positioningClickOn, cvdClickSec, positioningAnchorQuery, t]);

  // Live pump window matches the polled candle head (API max 1000).
  const pumpsQuery = useGetPumpEventsQuery(
    {
      exchange: exchangeArg,
      symbol,
      interval,
      limit: DEFAULT_DETAIL_CANDLE_LIMIT,
      minReturnPct: pumpThresholdPct,
      direction: 'both',
      maxEvents: 40,
    },
    {
      skip: skipSeries || !showPumpMarkers || isEquity || !liveHeadReady,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const indicatorsQuery = useGetIndicatorsQuery(
    {
      exchange: exchangeArg,
      symbol,
      interval,
      // Indicators stay on a fixed recent window (not full multi-page history).
      limit: Math.min(DETAIL_INDICATOR_LIMIT, DETAIL_API_BAR_MAX),
      rsiPeriod: DEFAULT_RSI_PERIOD,
      emaPeriods: DEFAULT_EMA_PERIODS,
    },
    {
      skip: skipSeries || !liveHeadReady,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const hasMoreHistory =
    !historyExhausted &&
    mergedCandles.length > 0 &&
    mergedCandles.length < DETAIL_CANDLE_MAX_LIMIT &&
    (candlesQuery.isSuccess || historyCandles.length > 0);

  const isLoadingMore = historyLoading;

  const onNeedMoreHistory = useCallback(() => {
    if (historyLoading || historyExhausted || skipSeries || !symbol || !exchange) return;
    if (mergedCandles.length >= DETAIL_CANDLE_MAX_LIMIT) {
      setHistoryExhausted(true);
      return;
    }
    const oldestMs = oldestCandleOpenTimeMs(mergedCandles);
    if (oldestMs == null) return;

    const endTime = new Date(oldestMs - 1).toISOString();
    const requestId = ++historyRequestIdRef.current;
    const seriesKey = `${exchange}|${symbol}|${interval}`;
    setHistoryLoading(true);

    const candleReq = fetchOlderCandles({
      exchange,
      symbol,
      interval,
      limit: DETAIL_CANDLE_PAGE_SIZE,
      endTime,
    }).unwrap();

    const indicatorReq = showEma
      ? fetchOlderIndicators({
          exchange,
          symbol,
          interval,
          limit: Math.min(DETAIL_CANDLE_PAGE_SIZE, DETAIL_API_BAR_MAX),
          endTime,
          rsiPeriod: DEFAULT_RSI_PERIOD,
          emaPeriods: DEFAULT_EMA_PERIODS,
        })
          .unwrap()
          .catch(() => ({ points: [] as NonNullable<IndicatorsResponse['points']> }))
      : Promise.resolve({ points: [] as NonNullable<IndicatorsResponse['points']> });

    // Same time window as the candle page so markers exist on older bars.
    const pumpReq =
      showPumpMarkers
        ? fetchOlderPumps({
            exchange,
            symbol,
            interval,
            limit: DETAIL_CANDLE_PAGE_SIZE,
            endTime,
            minReturnPct: pumpThresholdPct,
            direction: 'both',
            maxEvents: 40,
          })
            .unwrap()
            .catch(() => ({ events: [] as PumpEventDto[] }))
        : Promise.resolve({ events: [] as PumpEventDto[] });

    void Promise.all([candleReq, pumpReq, indicatorReq])
      .then(([candleRes, pumpRes, indicatorRes]) => {
        // Drop stale responses after pair/interval change mid-flight.
        if (requestId !== historyRequestIdRef.current) return;
        if (`${exchange}|${symbol}|${interval}` !== seriesKey) return;

        const batch = filterValidApiCandles(candleRes.candles);
        if (batch.length === 0) {
          setHistoryExhausted(true);
          return;
        }
        setHistoryCandles((prev) => {
          const mergedHist = mergeCandleHistory(batch, prev);
          const room = Math.max(0, DETAIL_CANDLE_MAX_LIMIT - DEFAULT_DETAIL_CANDLE_LIMIT);
          return room > 0 ? trimCandlesToMax(mergedHist, room) : mergedHist;
        });
        const pageEvents = pumpRes.events ?? [];
        if (pageEvents.length > 0) {
          setHistoryPumpEvents((prev) => mergePumpEvents(prev, pageEvents));
        }
        const pagePoints = indicatorRes.points ?? [];
        if (pagePoints.length > 0) {
          setHistoryIndicatorPoints((prev) => mergeIndicatorPoints(pagePoints, prev));
        }
        if (batch.length < DETAIL_CANDLE_PAGE_SIZE) {
          setHistoryExhausted(true);
        }
      })
      .catch(() => {
        // Keep hasMore true so the user can retry by panning again.
      })
      .finally(() => {
        if (requestId === historyRequestIdRef.current) {
          setHistoryLoading(false);
        }
      });
  }, [
    mergedCandles,
    exchange,
    fetchOlderCandles,
    fetchOlderIndicators,
    fetchOlderPumps,
    historyExhausted,
    historyLoading,
    interval,
    pumpThresholdPct,
    showEma,
    showPumpMarkers,
    skipSeries,
    symbol,
  ]);

  const liveIndicators = rtkCurrent(indicatorsQuery);
  const emaPeriods = useMemo(() => {
    const fromApi = (liveIndicators?.emaPeriods ?? []).filter(
      (n): n is number => Number.isInteger(n) && n >= 2,
    );
    return fromApi.length > 0 ? [...fromApi].sort((a, b) => a - b) : [12, 26];
  }, [liveIndicators?.emaPeriods]);
  const overlayPoints = useMemo(
    () => mergeIndicatorPoints(historyIndicatorPoints, liveIndicators?.points),
    [historyIndicatorPoints, liveIndicators?.points],
  );
  const overlays: CandleChartOverlay[] = useMemo(() => {
    const lines: CandleChartOverlay[] = [];
    if (showEma) {
      for (let i = 0; i < emaPeriods.length; i += 1) {
        const period = emaPeriods[i]!;
        lines.push({
          id: `ema-${period}`,
          title: t('detail:indicators.emaLabel', { period }),
          color: emaColor(String(period), i),
          data: indicatorPointsToEmaLine(overlayPoints, String(period)),
        });
      }
    }
    const realCount = mergedCandles.length;
    const lastReal = realCount > 0 ? chartData[realCount - 1] : undefined;
    const lastBar = chartData[chartData.length - 1];
    if (lastReal && lastBar && lastBar.time > lastReal.time) {
      lines.push({
        id: 'delist-last-price',
        title: t('detail:chart.haltedLast'),
        color: semanticColors.status.warning,
        data: [
          { time: lastReal.time, value: lastReal.close },
          { time: lastBar.time, value: lastBar.close },
        ],
      });
    }
    return lines;
  }, [showEma, emaPeriods, overlayPoints, chartData, mergedCandles.length, t]);

  const barDurationSec = candleIntervalSeconds(interval);
  const situationCloseMs =
    cvdClickSec == null ? null : situationCandleCloseMs(cvdClickSec, barDurationSec, Date.now());
  const situationAt = situationCloseMs == null ? undefined : new Date(situationCloseMs).toISOString();
  const situationPastQuery = useGetMarketSituationQuery(
    { symbol: symbol ?? '', at: situationAt },
    {
      skip: skip || isEquity || situationAt == null || !visible,
      refetchOnFocus: true,
    },
  );
  const onCandleAnchor = useCallback(
    (timeSec: number) => {
      if (barDurationSec <= 0) return;
      const open = snapCandleOpen(chartData, timeSec, barDurationSec);
      if (open == null) return;
      setCvdClickSec((prev) => (prev === open ? null : open));
    },
    [barDurationSec, chartData],
  );
  const activeFootSpan = historySeriesKey === seriesKey ? footSpan : null;
  const candleFootRange = useMemo(() => {
    if (!(barDurationSec > 0)) return null;
    if (activeFootSpan) return activeFootSpan;
    const nowSec = Date.now() / 1000;
    const to = Math.ceil(nowSec / barDurationSec) * barDurationSec;
    return { from: to - CANDLE_FOOTPRINT_DEFAULT_CANDLES * barDurationSec, to };
  }, [barDurationSec, activeFootSpan]);
  const candleFootLive =
    candleFootRange != null &&
    candleFootRange.to * 1000 >= Date.now() - 2 * barDurationSec * 1000;
  const candleFootprintQuery = useGetMarketCandleFootprintQuery(
    {
      exchange: 'all',
      symbol: perpSymbol,
      interval,
      startTime: candleFootRange ? new Date(candleFootRange.from * 1000).toISOString() : undefined,
      endTime: candleFootRange ? new Date(candleFootRange.to * 1000).toISOString() : undefined,
    },
    {
      skip: skipTape || !showFootprint || candleFootRange == null,
      pollingInterval: visible && showFootprint && candleFootLive ? 30_000 : 0,
      refetchOnFocus: true,
    },
  );
  const chartCvdReport = cvdFetchRange === '24h' ? rtkCurrent(cvdQuery) : rtkCurrent(cvdChartQuery);
  const cvdExchange = exchange === 'bybit' || exchange === 'okx' ? exchange : 'binance';
  const tapeCvd = useMemo(
    () => futuresCvdSeries(chartCvdReport, cvdExchange),
    [chartCvdReport, cvdExchange],
  );
  const tapeSize = useMemo(
    () => futuresSizeSeries(chartCvdReport, cvdExchange),
    [chartCvdReport, cvdExchange],
  );
  const spotExchange = exchange === 'bybit' ? 'bybit' : 'binance';
  const spotCvd = useMemo(
    () => spotCvdSeries(chartCvdReport, spotExchange),
    [chartCvdReport, spotExchange],
  );
  const clickCvd = useMemo(() => {
    if (isEquity || !showCvd || cvdClickSec == null || barDurationSec <= 0) return null;
    const line = anchoredCvdByCandle(chartData, tapeCvd.points, cvdClickSec, barDurationSec);
    const last = line.length > 0 ? line[line.length - 1]!.value : null;
    return {
      time: formatUtcMinute(cvdClickSec),
      amount: last == null ? '' : formatSignedTapeNum(last),
      exchange: t(`detail:liqHeatmap.venues.${tapeCvd.source}`, { defaultValue: tapeCvd.source }),
      late: cvdAnchorStartsLate(tapeCvd.points, cvdClickSec, cvdBarSeconds(tapeCvd.bar)),
      empty: line.length === 0,
    };
  }, [isEquity, showCvd, cvdClickSec, barDurationSec, chartData, tapeCvd, t]);
  const positioningRead = useMemo(() => {
    const anchors = chartAnchors(Math.floor(Date.now() / 1000));
    const venueName = (id: string) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id });
    const crowd = crowdStoredLines(rtkCurrent(crowdHistQuery)?.items);
    const oiLines = openInterestLines(rtkCurrent(oiHistQuery)?.items);
    return {
      crowd,
      noTop: venuesWithoutTop(crowd),
      oiLines,
      funding: fundingRateLines(rtkCurrent(fundingHistQuery)?.items),
      liqBars: liquidationBucketBars(rtkCurrent(liqHistQuery)?.items),
      oiAnchor: anchoredVenueSeries(
        oiLines,
        anchors,
        {
          day: (id) => t('detail:chart.oiDay', { venue: venueName(id) }),
          block: (id) => t('detail:chart.oi4h', { venue: venueName(id) }),
        },
        0,
        'oi',
      ),
      gapAnchor: anchoredVenueSeries(
        crowdGapLines(crowd),
        anchors,
        {
          day: (id) => t('detail:chart.gapDay', { venue: venueName(id) }),
          block: (id) => t('detail:chart.gap4h', { venue: venueName(id) }),
        },
        0,
        'gap',
      ),
    };
  }, [crowdHistQuery, oiHistQuery, fundingHistQuery, liqHistQuery, t]);
  const lowerSeries: CandleChartLowerSeries[] = useMemo(() => {
    if (isEquity || barDurationSec <= 0) return [];
    const series: CandleChartLowerSeries[] = [];
    let pane = 1;
    if (showDelta) {
      const delta = deltaByCandle(chartData, tapeCvd.points, barDurationSec);
      if (delta.length > 0) {
        series.push({
          id: 'delta',
          pane,
          kind: 'histogram',
          title: t('detail:chart.delta'),
          color: semanticColors.chart.up,
          downColor: semanticColors.chart.down,
          data: delta,
        });
        pane += 1;
      }
    }
    if (showCvd) {
      const cvdLayers: {
        prefix: string;
        tape: CvdChartSeries;
        market: string;
        running: string;
        day: string;
        block: string;
      }[] = [
        {
          prefix: '',
          tape: tapeCvd,
          market: t('detail:chart.cvdFutures'),
          running: semanticColors.text.link,
          day: semanticColors.accent.default,
          block: semanticColors.status.warning,
        },
        {
          prefix: 'spot-',
          tape: spotCvd,
          market: t('detail:chart.cvdSpot'),
          running: semanticColors.chart.down,
          day: semanticColors.chart.up,
          block: semanticColors.text.link,
        },
      ];
      for (const layer of cvdLayers) {
        const venue = t(`detail:liqHeatmap.venues.${layer.tape.source}`, {
          defaultValue: layer.tape.source,
        });
        if (cvdRange !== '24h') {
          const running = runningCvdByCandle(chartData, layer.tape.points, barDurationSec);
          if (running.length > 0) {
            series.push({
              id: `${layer.prefix}cvd-range`,
              pane,
              kind: 'line',
              title: t('detail:chart.cvdRunningNamed', {
                market: layer.market,
                venue,
                range: cvdRange,
              }),
              color: layer.running,
              data: running,
            });
            pane += 1;
          }
        }
        const day =
          layer.tape.dayFrom != null
            ? anchoredCvdByCandle(chartData, layer.tape.points, layer.tape.dayFrom, barDurationSec)
            : [];
        const block =
          layer.tape.blockFrom != null
            ? anchoredCvdByCandle(chartData, layer.tape.points, layer.tape.blockFrom, barDurationSec)
            : [];
        const click =
          cvdClickSec != null
            ? anchoredCvdByCandle(chartData, layer.tape.points, cvdClickSec, barDurationSec)
            : [];
        if (day.length > 0 || block.length > 0 || click.length > 0) {
          if (day.length > 0) {
            series.push({
              id: `${layer.prefix}cvd-day`,
              pane,
              kind: 'line',
              title: t('detail:chart.cvdDayNamed', { market: layer.market, venue }),
              color: layer.day,
              data: day,
            });
          }
          if (block.length > 0) {
            series.push({
              id: `${layer.prefix}cvd-4h`,
              pane,
              kind: 'line',
              title: t('detail:chart.cvd4hNamed', { market: layer.market, venue }),
              color: layer.block,
              data: block,
            });
          }
          if (click.length > 0) {
            series.push({
              id: `${layer.prefix}cvd-click`,
              pane,
              kind: 'line',
              title: t('detail:chart.cvdClickNamed', { market: layer.market, venue }),
              color: semanticColors.chart.rsi,
              data: click,
            });
          }
          pane += 1;
        }
        if (layer.prefix === '') {
          const sizeVenue = t(`detail:liqHeatmap.venues.${tapeSize.source}`, {
            defaultValue: tapeSize.source,
          });
          const drawSize = (
            id: string,
            points: typeof tapeSize.whale,
            titles: { running: string; day: string; block: string; click: string },
          ) => {
            if (points.length === 0) return;
            const holdSec = cvdBarSeconds(layer.tape.bar);
            if (cvdRange !== '24h') {
              const running = sizedCvdByCandle(chartData, points, null, barDurationSec, holdSec);
              if (running.some((point) => !point.gap)) {
                series.push({
                  id: `${id}-range`,
                  pane,
                  kind: 'line',
                  title: titles.running,
                  color: semanticColors.text.link,
                  data: running,
                });
                pane += 1;
              }
            }
            const sizeDay =
              layer.tape.dayFrom != null
                ? sizedCvdByCandle(chartData, points, layer.tape.dayFrom, barDurationSec, holdSec)
                : [];
            const sizeBlock =
              layer.tape.blockFrom != null
                ? sizedCvdByCandle(chartData, points, layer.tape.blockFrom, barDurationSec, holdSec)
                : [];
            const sizeClick =
              cvdClickSec != null
                ? sizedCvdByCandle(chartData, points, cvdClickSec, barDurationSec, holdSec)
                : [];
            if (sizeDay.length === 0 && sizeBlock.length === 0 && sizeClick.length === 0) return;
            if (sizeDay.length > 0) {
              series.push({
                id: `${id}-day`,
                pane,
                kind: 'line',
                title: titles.day,
                color: semanticColors.accent.default,
                data: sizeDay,
              });
            }
            if (sizeBlock.length > 0) {
              series.push({
                id: `${id}-4h`,
                pane,
                kind: 'line',
                title: titles.block,
                color: semanticColors.status.warning,
                data: sizeBlock,
              });
            }
            if (sizeClick.length > 0) {
              series.push({
                id: `${id}-click`,
                pane,
                kind: 'line',
                title: titles.click,
                color: semanticColors.chart.rsi,
                data: sizeClick,
              });
            }
            pane += 1;
          };
          drawSize('whale', tapeSize.whale, {
            running: t('detail:chart.cvdWhaleRunningNamed', { venue: sizeVenue, range: cvdRange }),
            day: t('detail:chart.cvdWhaleDayNamed', { venue: sizeVenue }),
            block: t('detail:chart.cvdWhale4hNamed', { venue: sizeVenue }),
            click: t('detail:chart.cvdWhaleClickNamed', { venue: sizeVenue }),
          });
          drawSize('retail', tapeSize.retail, {
            running: t('detail:chart.cvdRetailRunningNamed', { venue: sizeVenue, range: cvdRange }),
            day: t('detail:chart.cvdRetailDayNamed', { venue: sizeVenue }),
            block: t('detail:chart.cvdRetail4hNamed', { venue: sizeVenue }),
            click: t('detail:chart.cvdRetailClickNamed', { venue: sizeVenue }),
          });
        }
      }
    }
    if (showCrowd) {
      const venueName = (id: string) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id });
      const before = series.length;
      series.push(
        ...crowdStoredSeries(
          positioningRead.crowd,
          {
            crowd: t('detail:chart.crowdLine'),
            top: t('detail:chart.topLine'),
            venueCrowd: (venue) => t('detail:chart.crowdLineVenue', { venue: venueName(venue) }),
            venueTop: (venue) => t('detail:chart.topLineVenue', { venue: venueName(venue) }),
          },
          {
            crowd: semanticColors.accent.default,
            top: semanticColors.status.warning,
            crowdAlt: semanticColors.chart.up,
            topAlt: semanticColors.text.link,
          },
          pane,
        ),
      );
      if (series.length > before) pane += 1;
      if (positioningRead.gapAnchor.series.length > 0) {
        for (const line of positioningRead.gapAnchor.series) {
          series.push({ ...line, pane });
        }
        pane += 1;
      }
    }
    if (showOi) {
      if (positioningRead.oiLines.length > 0) {
        for (const line of positioningRead.oiLines) {
          series.push({
            id: `oi-${line.exchange}`,
            pane,
            kind: 'line',
            title: t('detail:chart.oiLine', {
              exchange: t(`detail:liqHeatmap.venues.${line.exchange}`, { defaultValue: line.exchange }),
            }),
            color: futuresPaneColor(line.exchange),
            data: line.data,
          });
        }
        pane += 1;
      }
      if (positioningRead.oiAnchor.series.length > 0) {
        for (const line of positioningRead.oiAnchor.series) {
          series.push({ ...line, pane });
        }
        pane += 1;
      }
      if (cvdClickSec != null) {
        const venueName = (id: string) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id });
        const lines = positioningClickLines(
          rtkCurrent(positioningAnchorQuery)?.openInterest,
          (id) => t('detail:chart.oiClick', { venue: venueName(id) }),
          pane,
          'oi-click',
        );
        if (lines.length > 0) {
          series.push(...lines);
          pane += 1;
        }
      }
    }
    if (showFunding) {
      if (positioningRead.funding.length > 0) {
        for (const line of positioningRead.funding) {
          series.push({
            id: `funding-${line.exchange}`,
            pane,
            kind: 'line',
            title: t(
              line.exchange === 'hyperliquid' ? 'detail:chart.fundingLineHourly' : 'detail:chart.fundingLine',
              {
                exchange: t(`detail:liqHeatmap.venues.${line.exchange}`, { defaultValue: line.exchange }),
              },
            ),
            color: futuresPaneColor(line.exchange),
            data: line.data,
          });
        }
        pane += 1;
      }
      if (cvdClickSec != null) {
        const venueName = (id: string) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id });
        const lines = positioningClickLines(
          rtkCurrent(positioningAnchorQuery)?.funding,
          (id) => t('detail:chart.fundingClick', { venue: venueName(id) }),
          pane,
          'funding-click',
        );
        if (lines.length > 0) {
          series.push(...lines);
          pane += 1;
        }
      }
    }
    if (showLiqNotional) {
      const bars = positioningRead.liqBars;
      if (bars.length > 0) {
        series.push({
          id: 'liq-notional',
          pane,
          kind: 'histogram',
          title: t('detail:chart.liqNotional'),
          color: semanticColors.chart.up,
          downColor: semanticColors.chart.down,
          data: bars,
        });
        pane += 1;
      }
      if (cvdClickSec != null) {
        const venueName = (id: string) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id });
        const lines = positioningClickLines(
          rtkCurrent(positioningAnchorQuery)?.liquidations,
          (id) => t('detail:chart.liqClick', { venue: venueName(id) }),
          pane,
          'liq-click',
        );
        if (lines.length > 0) {
          series.push(...lines);
          pane += 1;
        }
      }
    }
    if (showOpenLevels) {
      const report = rtkCurrent(openLevelsQuery);
      const delta = openLevelDelta(report, liqMapVenue);
      if (delta.length > 0) {
        series.push({
          id: 'open-liq-delta',
          pane,
          kind: 'line',
          title: t('detail:chart.openLevelDelta'),
          color: semanticColors.accent.default,
          data: delta,
        });
        pane += 1;
      }
      const anchored = openLevelAnchorLines(report, liqMapVenue, cvdClickSec);
      const rows = [
        anchored.day.length > 0
          ? {
              id: 'open-liq-day',
              title: t('detail:chart.openLevelDay'),
              color: semanticColors.accent.default,
              data: anchored.day,
            }
          : null,
        anchored.block.length > 0
          ? {
              id: 'open-liq-4h',
              title: t('detail:chart.openLevel4h'),
              color: semanticColors.status.warning,
              data: anchored.block,
            }
          : null,
        anchored.click.length > 0
          ? {
              id: 'open-liq-click',
              title: t('detail:chart.openLevelClick'),
              color: semanticColors.text.link,
              data: anchored.click,
            }
          : null,
      ];
      for (const row of rows) {
        if (!row) continue;
        series.push({ pane, kind: 'line', ...row });
      }
    }
    return series;
  }, [
    isEquity,
    barDurationSec,
    showDelta,
    showCvd,
    cvdRange,
    showCrowd,
    showOi,
    showFunding,
    showLiqNotional,
    showOpenLevels,
    liqMapVenue,
    chartData,
    tapeCvd,
    tapeSize,
    spotCvd,
    cvdClickSec,
    positioningRead,
    positioningAnchorQuery,
    openLevelsQuery,
    t,
  ]);

  const profileRead = useMemo(
    () => (isEquity || !showProfile ? null : chartProfileRead(rtkCurrent(chartProfileQuery), exchangeArg)),
    [isEquity, showProfile, chartProfileQuery, exchangeArg],
  );

  const chartPriceLines: CandleChartPriceLine[] = useMemo(() => {
    const lines: CandleChartPriceLine[] = [];
    if (!isEquity && showLiqLevels) {
      const levels = rtkCurrent(liqLevelsQuery);
      const sideLabel = (side: 'long' | 'short') => t(`detail:chart.liqSide.${side}`);
      for (const line of liquidationChartLevels(levels)) {
        const review = liquidationLevelReview(levels, cvdExchange, line.side, line.leverages);
        const base =
          line.leverages.length > 0
            ? `${line.leverages.map((n) => `${n}×`).join(' · ')} ${sideLabel(line.side)}`
            : sideLabel(line.side);
        lines.push({
          id: line.id,
          price: line.price,
          color: line.side === 'long' ? semanticColors.chart.down : semanticColors.chart.up,
          title: review
            ? `${base} · ${t('detail:chart.levelReview', {
                venue: t(`detail:liqHeatmap.venues.${cvdExchange}`, { defaultValue: cvdExchange }),
                hit: `${Math.round(review.hitRate * 100)}%`,
                kept: `${Math.round(review.sizeScale * 100)}%`,
              })}`
            : base,
        });
      }
    }
    if (profileRead) {
      for (const line of profileRead.lines) {
        lines.push({
          id: line.id,
          price: line.price,
          color: line.role === 'poc' ? semanticColors.chart.emaFast : semanticColors.chart.neutral,
          lineWidth: line.role === 'poc' ? 2 : 1,
          title: t(`detail:chart.${line.role}`),
        });
      }
    }
    return lines;
  }, [isEquity, showLiqLevels, liqLevelsQuery, cvdExchange, profileRead, t]);

  const chartBookHeat = useMemo(
    () => (isEquity || !showBook ? [] : bookHeatColumns(rtkCurrent(chartBookQuery))),
    [isEquity, showBook, chartBookQuery],
  );
  const chartCandleFootprint = useMemo(
    () =>
      isEquity || !showFootprint
        ? { candles: [], stepSec: 0, definite: false, truncated: false, missing: [] as string[], exchange: '' }
        : candleFootprintView(rtkCurrent(candleFootprintQuery), exchangeArg),
    [isEquity, showFootprint, candleFootprintQuery, exchangeArg],
  );
  const chartFootprint = useMemo(
    () =>
      isEquity || !showFootprint || chartCandleFootprint.candles.length > 0
        ? []
        : footprintLevels(rtkCurrent(footprintQuery)),
    [isEquity, showFootprint, chartCandleFootprint.candles.length, footprintQuery],
  );
  const chartLiqProfile = useMemo(
    () => (isEquity || !showLiqLevels ? [] : liquidationProfile(rtkCurrent(liqLevelsQuery))),
    [isEquity, showLiqLevels, liqLevelsQuery],
  );
  const chartLiqHeat = useMemo(() => {
    if (isEquity) return [];
    // Book bands go down first. Liquidation cells paint over them, so a pocket
    // stays readable and the resting size still shows beside it.
    const book = perpBandsOn ? perpBookCells(rtkCurrent(perpBookQuery)) : [];
    const liq = showLiqMap ? liqHeatCells(rtkCurrent(liqMapQuery), liqMapVenue, liqMapSide) : [];
    return [...book, ...liq];
  }, [isEquity, perpBandsOn, showLiqMap, liqMapQuery, liqMapVenue, liqMapSide, perpBookQuery]);
  const perpReach = useMemo(
    () => (perpBandsOn ? perpBookReach(rtkCurrent(perpBookQuery)) : null),
    [perpBandsOn, perpBookQuery],
  );
  const chartOpenLevels = useMemo(
    () =>
      isEquity || !showOpenLevels
        ? []
        : openLevelSegments(rtkCurrent(openLevelsQuery), liqMapVenue),
    [isEquity, showOpenLevels, openLevelsQuery, liqMapVenue],
  );
  const liqMapNote = useMemo(
    () => (isEquity || !showLiqMap ? null : liqHeatNotes(rtkCurrent(liqMapQuery), liqMapVenue)),
    [isEquity, showLiqMap, liqMapQuery, liqMapVenue],
  );
  const openLevelNote = useMemo(
    () => (isEquity || !showOpenLevels ? null : openLevelNotes(rtkCurrent(openLevelsQuery), liqMapVenue)),
    [isEquity, showOpenLevels, openLevelsQuery, liqMapVenue],
  );
  const openAnchorRead = useMemo(() => {
    if (isEquity || !showOpenLevels) return null;
    const report = rtkCurrent(openLevelsQuery);
    const lines = openLevelAnchorLines(report, liqMapVenue, cvdClickSec);
    const points = openLevelPoints(report, liqMapVenue);
    const book = pickOpenLevelBook(report, liqMapVenue);
    const textFor = (id: 'day' | '4h') => {
      const row = book?.anchors?.find((anchor) => anchor.anchor === id);
      if (!row?.complete) return null;
      return {
        long: formatSignedCompactUsd(row.longChange),
        short: formatSignedCompactUsd(row.shortChange),
        delta: formatSignedCompactUsd(row.deltaChange),
      };
    };
    const click = cvdClickSec != null ? openLevelSizeSince(points, cvdClickSec) : null;
    return {
      lines,
      day: textFor('day'),
      block: textFor('4h'),
      click:
        click == null || cvdClickSec == null
          ? null
          : {
              time: formatUtcMinute(cvdClickSec),
              long: formatSignedCompactUsd(click.long),
              short: formatSignedCompactUsd(click.short),
              delta: formatSignedCompactUsd(click.delta),
            },
      clickMissing: cvdClickSec != null && click == null && points.length > 0,
    };
  }, [isEquity, showOpenLevels, openLevelsQuery, liqMapVenue, cvdClickSec]);
  const chartMix = showLiqMap && liqMapNote?.mix ? liqMapNote.mix : openLevelNote?.mix ?? '';
  const chartMissing =
    showLiqMap && liqMapNote && liqMapNote.missingVenues.length > 0
      ? liqMapNote.missingVenues
      : !showLiqMap && openLevelNote && openLevelNote.missingVenues.length > 0
        ? openLevelNote.missingVenues
        : [];
  const liqMapNeedsPan = useMemo(() => {
    const firstBar = chartData[0]?.time;
    if (firstBar == null || chartLiqHeat.length === 0) return false;
    let earliest = chartLiqHeat[0]?.time ?? firstBar;
    for (const cell of chartLiqHeat) earliest = Math.min(earliest, cell.time);
    return earliest + 60 < firstBar;
  }, [chartData, chartLiqHeat]);
  const onVisibleSpan = useCallback((from: number, to: number) => {
    const span = to - from;
    const next = footprintWindow(span);
    setFootWindow((prev) => (prev === next ? prev : next));
    const nextBook = chartBookWindow(span);
    setBookWindow((prev) => (prev === nextBook ? prev : nextBook));
    const nextFoot = candleFootprintSpan(barDurationSec, from, to);
    setFootSpan((prev) => {
      if (!nextFoot) return prev;
      if (prev && prev.from === nextFoot.from && prev.to === nextFoot.to) return prev;
      return nextFoot;
    });
    const nextTape = chartTapeSpan(barDurationSec, from, to);
    setTapeSpan((prev) => {
      if (!nextTape) return prev;
      if (prev && prev.from === nextTape.from && prev.to === nextTape.to) return prev;
      return nextTape;
    });
  }, [barDurationSec]);
  const recordedLine = useMemo(() => {
    if (isEquity) return '';
    const rows = recordedParts(rtkCurrent(historySpanQuery));
    if (rows.length === 0) {
      return historySpanQuery.isSuccess || historySpanQuery.isError
        ? t('detail:chart.recordedEmpty')
        : '';
    }
    const parts = rows
      .map((row) =>
        t('detail:chart.recordedPart', {
          kind: t(`detail:chart.recordedKind.${row.kind}`),
          exchange: row.exchange,
          span: formatRecordedSpan(row.spanSec),
        }),
      )
      .join(', ');
    return t('detail:chart.recorded', { parts });
  }, [isEquity, historySpanQuery, t]);

  const chartMarkers: CandleChartMarker[] = useMemo(() => {
    const barSec = intervalToSeconds(interval);
    const maxDist = barSec > 0 ? barSec * 1.5 : 3600;
    const livePumps = rtkCurrent(pumpsQuery);
    const pumpRaw =
      showPumpMarkers && exchange && symbol
        ? pumpEventsToChartMarkers(
            mergePumpEvents(
              livePumpEventsForPair(livePumps, exchange, symbol),
              historyPumpEvents,
            ),
            pumpThresholdPct,
          )
        : [];
    const signalRaw =
      showSignalMarkers && exchange && symbol
        ? scannerResultsToChartMarkers(rtkCurrent(scannerResultsQuery)?.results, exchange, symbol)
        : [];
    const pumpSnapped = snapMarkersToCandleTimes(pumpRaw, chartData, { maxDistanceSec: maxDist });
    const signalSnapped = snapMarkersToCandleTimes(signalRaw, chartData, { maxDistanceSec: maxDist });
    const sweepSource = !isEquity && showSweeps ? chartSweepMarks(rtkCurrent(chartSweepQuery), sweepVenue, tapeSpan) : [];
    const botSource = !isEquity && showBots ? chartBotMarks(rtkCurrent(botTrackerQuery), tapeSpan) : [];
    const sweepRaw: CandleChartMarker[] = sweepSource.map((mark) => ({
            time: mark.time,
            position: mark.kind === 'sweep-high' ? 'aboveBar' : 'belowBar',
            color: mark.kind === 'sweep-high' ? semanticColors.chart.down : semanticColors.chart.up,
            shape: mark.kind === 'sweep-high' ? 'arrowDown' : 'arrowUp',
            text: t(mark.kind === 'sweep-high' ? 'detail:chart.sweepHigh' : 'detail:chart.sweepLow'),
          }));
    const botRaw: CandleChartMarker[] = botSource.map((mark) => ({
      time: mark.time,
      position: 'aboveBar' as const,
      color: mark.kind === 'bot-buy' ? semanticColors.chart.up : semanticColors.chart.down,
      shape: 'square' as const,
      text: t(mark.kind === 'bot-buy' ? 'detail:chart.botBuy' : 'detail:chart.botSell', {
        count: mark.count ?? 0,
      }),
    }));
    const sweepSnapped = snapMarkersToCandleTimes(sweepRaw, chartData, { maxDistanceSec: maxDist });
    const botSnapped = snapMarkersToCandleTimes(botRaw, chartData, { maxDistanceSec: maxDist });
    const splitRaw: CandleChartMarker[] =
      !isEquity && showCvd && tapeCvd.split
        ? [
            {
              time: tapeCvd.split.time,
              position: tapeCvd.split.kind === 'price_up_cvd_down' ? 'aboveBar' : 'belowBar',
              color: semanticColors.status.warning,
              shape: 'circle',
              text: t(
                tapeCvd.split.kind === 'price_up_cvd_down'
                  ? 'detail:chart.cvdSplitUp'
                  : 'detail:chart.cvdSplitDown',
              ),
            },
          ]
        : [];
    const splitSnapped = snapMarkersToCandleTimes(splitRaw, chartData, { maxDistanceSec: maxDist });
    return mergeChartMarkers(
      mergeChartMarkers(mergeChartMarkers(mergeChartMarkers(pumpSnapped, signalSnapped), sweepSnapped), botSnapped),
      splitSnapped,
    );
  }, [
    showPumpMarkers,
    showSignalMarkers,
    showSweeps,
    showBots,
    showCvd,
    tapeCvd.split,
    isEquity,
    pumpsQuery,
    historyPumpEvents,
    pumpThresholdPct,
    scannerResultsQuery.data?.results,
    chartSweepQuery,
    botTrackerQuery,
    sweepVenue,
    tapeSpan,
    exchange,
    symbol,
    chartData,
    interval,
    t,
  ]);

  const chartVertLines = useMemo(() => {
    const lines = delistEventsToVertLines({
      announcedAt,
      delistTime,
      announcedLabel: t('detail:chart.delistAnnounced'),
      delistLabel: t('detail:chart.delistHalt'),
      announcedColor: semanticColors.status.warning,
      delistColor: semanticColors.status.error,
    });
    if (!isEquity && cvdClickSec != null && (showCvd || showOi || showFunding || showLiqNotional)) {
      lines.push({
        id: 'cvd-click',
        time: cvdClickSec,
        color: semanticColors.chart.rsi,
        label: showCvd ? t('detail:chart.cvd') : t('detail:chart.positioningClickMark'),
        labelOffsetY: lines.length > 0 ? 18 : 0,
      });
    }
    return lines;
  }, [announcedAt, delistTime, t, isEquity, showCvd, showOi, showFunding, showLiqNotional, cvdClickSec]);

  const tab: DetailTab = useMemo(() => {
    if (
      isEquity &&
      (urlState.tab === 'orderbook' ||
        urlState.tab === 'holders' ||
        urlState.tab === 'tape' ||
        urlState.tab === 'flow' ||
        urlState.tab === 'structure')
    ) {
      return DEFAULT_DETAIL_TAB;
    }
    return urlState.tab;
  }, [isEquity, urlState.tab]);

  const patchUrl = (patch: Partial<{ interval: string; tab: DetailTab }>) => {
    setSearchParams(
      detailStateToSearchParams({
        interval: patch.interval ?? interval,
        tab: patch.tab ?? tab,
      }),
      { replace: true },
    );
  };

  const refreshAll = () => {
    void intervalsQuery.refetch();
    void tickerQuery.refetch();
    void orderBookQuery.refetch();
    void orderHeatmapQuery.refetch();
    void supplyQuery.refetch();
    void holdersQuery.refetch();
    void firstCandlesQuery.refetch();
    void candlesQuery.refetch();
    void indicatorsQuery.refetch();
    if (showPumpMarkers) {
      void pumpsQuery.refetch();
    }
    if (showSignalMarkers) {
      void scannerResultsQuery.refetch();
    }
    if (showCrowd) void crowdHistQuery.refetch();
    if (showOi) void oiHistQuery.refetch();
    if (showFunding) void fundingHistQuery.refetch();
    if (showLiqNotional) void liqHistQuery.refetch();
    if (chartPreset === 'orderFlow') {
      void absorptionQuery.refetch();
      void printSizeQuery.refetch();
    }
    if (chartPreset === 'liquidity') void perpDepthQuery.refetch();
    if (perpBandsOn) void perpBookQuery.refetch();
  };

  if (!exchange) {
    return (
      <PageStack>
        <Alert
          type="error"
          showIcon
          message={t('detail:invalidExchangeTitle')}
          description={t('detail:invalidExchangeBody')}
        />
      </PageStack>
    );
  }

  if (!symbol) {
    return (
      <PageStack>
        <Alert
          type="error"
          showIcon
          message={t('detail:missingSymbolTitle')}
          description={t('detail:missingSymbolBody')}
        />
      </PageStack>
    );
  }

  const backTo = marketsBackPathFromSession(exchange);

  const headerLoading = rtkCurrentPending(tickerQuery);
  const statsLoading = rtkCurrentPending(tickerQuery) || rtkCurrentPending(supplyQuery);
  const seriesLoading =
    chartData.length === 0 &&
    rtkCurrentPending(firstCandlesQuery) &&
    rtkCurrentPending(candlesQuery);
  const seriesFetching =
    firstCandlesQuery.isFetching ||
    candlesQuery.isFetching ||
    indicatorsQuery.isFetching ||
    tickerQuery.isFetching ||
    intervalsQuery.isFetching;

  const absorptionNote =
    chartPreset === 'orderFlow'
      ? absorptionDeskNote(rtkCurrent(absorptionQuery) as Parameters<typeof absorptionDeskNote>[0], cvdExchange)
      : null;
  const printSizeNote =
    chartPreset === 'orderFlow' ? printSizeDeskNote(rtkCurrent(printSizeQuery), cvdExchange) : null;
  const perpBid =
    chartPreset === 'liquidity'
      ? perpBidShareNote(rtkCurrent(perpDepthQuery) as Parameters<typeof perpBidShareNote>[0])
      : null;
  const liveWalls =
    chartPreset === 'liquidity' ? chartWallNotes(rtkCurrent(orderBookQuery)?.analysis?.walls) : [];

  return (
    <PageStack>
      <Text variant="caption" color="secondary">
        {t('detail:eyebrow')}
      </Text>
      <DetailHeader
        symbol={symbol}
        exchange={exchange}
        lastPrice={rtkCurrent(tickerQuery)?.lastPrice}
        priceChangePercent={rtkCurrent(tickerQuery)?.priceChangePercent}
        assetName={rtkCurrent(supplyQuery)?.name ?? rtkCurrent(profileQuery)?.name}
        logoUrl={rtkCurrent(profileQuery)?.logoUrl}
        listingDate={rtkCurrent(profileQuery)?.listingDate}
        contractLabel={
          rtkCurrent(profileQuery)?.contracts?.[0]
            ? `${rtkCurrent(profileQuery)?.contracts?.[0]?.chain ?? ''} ${rtkCurrent(profileQuery)?.contracts?.[0]?.address ?? ''}`.trim()
            : null
        }
        backTo={backTo}
        isLoading={headerLoading}
        watched={watched}
        watchLoading={addWatchState.isLoading || removeWatchState.isLoading}
        alertTo={exchange && symbol ? `/alerts?exchange=${encodeURIComponent(exchange)}&symbol=${encodeURIComponent(symbol)}` : undefined}
        compareTo={exchange && symbol ? `/compare?pairs=${encodeURIComponent(`${exchange}:${symbol}`)}` : undefined}
        signalsTo="/signals"
        onToggleWatch={() => {
          const run = watched
            ? removeWatch({ exchange, symbol }).unwrap()
            : addWatch({ exchange, symbol }).unwrap();
          void run.catch((err) => {
            void message.error(
              rtkErrorMessage(err, { resource: t('detail:watchFailed') }),
            );
          });
        }}
      delistTime={delistTime}
      announcedAt={announcedAt}
      halted={Boolean(rtkCurrent(tickerQuery)?.halted)}
      />

      <DetailStats
        exchange={exchange}
        ticker={rtkCurrent(tickerQuery)}
        supply={rtkCurrent(supplyQuery)}
        isLoading={statsLoading}
        tickerError={
          tickerQuery.isError
            ? rtkErrorMessage(tickerQuery.error, {
                resource: t('detail:resource.ticker'),
              })
            : null
        }
        supplyError={
          isEquity
            ? t('detail:errors.supplyEquity')
            : supplyQuery.isError
              ? rtkErrorMessage(supplyQuery.error, {
                  resource: t('detail:resource.supply'),
                  codeMessages: {
                    supply_unmapped: t('detail:errors.supply404'),
                  },
                  statusMessages: {
                    404: t('detail:errors.supply404'),
                  },
                })
              : null
        }
      />

      {!isEquity && exchange && symbol ? (
        <CoinSituation
          report={rtkCurrent(situationQuery)}
          isLoading={situationQuery.isLoading}
          error={
            situationQuery.isError
              ? rtkErrorMessage(situationQuery.error, {
                  resource: t('detail:situation.title'),
                })
              : null
          }
        />
      ) : null}

      <DeskTabs
        activeKey={tab}
        onChange={(key) => patchUrl({ tab: key as DetailTab })}
        items={[
          {
            key: 'overview',
            label: t('detail:tabs.overview'),
            children: (
      <TabStack>
      {pastDelist && !isEquity ? (
        <PostDelistPanel
          view={postDelistView}
          lastPrice={postDelistLast}
          isLoading={rtkCurrentPending(postDelistQuery)}
          error={
            postDelistQuery.isError
              ? rtkErrorMessage(postDelistQuery.error, {
                  resource: t('detail:resource.postDelist'),
                })
              : null
          }
        />
      ) : null}
      <ChartCard>
        {pastDelist && postDelistView?.available ? (
          <Alert
            type="info"
            showIcon
            message={t('detail:chart.postDelistBanner', {
              date: formatDelistDay(delistTime) || delistTime,
              source: postDelistView.sourceLabel || postDelistView.source,
              exchange,
            })}
          />
        ) : null}
        <ChartTitleRow>
          <Text variant="h4" color="primary">
            {t('detail:chart.title')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('detail:chart.meta', {
              interval,
              bars: chartData.length || DEFAULT_DETAIL_CANDLE_LIMIT,
            })}
          </Text>
        </ChartTitleRow>

        <DetailChartToolbar
          chartPreset={isEquity ? null : chartPreset}
          presetOptions={
            isEquity
              ? undefined
              : (['all', 'positioning', 'orderFlow', 'liquidity'] as const).map((id) => ({
                  value: id,
                  label: t(`detail:chart.presets.${id}`),
                }))
          }
          onChartPresetChange={isEquity ? undefined : choosePreset}
          intervals={supportedIntervals ?? []}
          interval={interval}
          intervalsLoading={intervalsQuery.isLoading && !supportedIntervals?.length}
          onIntervalChange={(iv) => patchUrl({ interval: iv })}
          pumpThresholdPct={isEquity ? undefined : pumpThresholdPct}
          onPumpThresholdChange={isEquity ? undefined : setPumpThresholdPct}
          showPumpMarkers={showPumpMarkers}
          onShowPumpMarkersChange={isEquity ? undefined : setShowPumpMarkers}
          showSignalMarkers={showSignalMarkers}
          onShowSignalMarkersChange={isEquity ? undefined : setShowSignalMarkers}
          showDelta={showDelta}
          onShowDeltaChange={isEquity ? undefined : setLayer('showDelta')}
          showCvd={showCvd}
          onShowCvdChange={isEquity ? undefined : setLayer('showCvd')}
          cvdRange={cvdRange}
          onCvdRangeChange={isEquity ? undefined : setCvdRange}
          showLiqLevels={showLiqLevels}
          onShowLiqLevelsChange={isEquity ? undefined : setLayer('showLiqLevels')}
          showBook={showBook}
          onShowBookChange={isEquity ? undefined : setLayer('showBook')}
          showFootprint={showFootprint}
          onShowFootprintChange={isEquity ? undefined : setLayer('showFootprint')}
          showProfile={showProfile}
          onShowProfileChange={isEquity ? undefined : setLayer('showProfile')}
          showSweeps={showSweeps}
          onShowSweepsChange={isEquity ? undefined : setLayer('showSweeps')}
          showBots={showBots}
          onShowBotsChange={isEquity ? undefined : setLayer('showBots')}
          showCrowd={showCrowd}
          onShowCrowdChange={isEquity ? undefined : setLayer('showCrowd')}
          showOi={showOi}
          onShowOiChange={isEquity ? undefined : setLayer('showOi')}
          showFunding={showFunding}
          onShowFundingChange={isEquity ? undefined : setLayer('showFunding')}
          showLiqNotional={showLiqNotional}
          onShowLiqNotionalChange={isEquity ? undefined : setLayer('showLiqNotional')}
          showLiqMap={showLiqMap}
          onShowLiqMapChange={isEquity ? undefined : setLayer('showLiqMap')}
          showOpenLevels={showOpenLevels}
          onShowOpenLevelsChange={isEquity ? undefined : setLayer('showOpenLevels')}
          liqMapRange={liqMapRange}
          onLiqMapRangeChange={isEquity ? undefined : setLiqMapRange}
          liqMapVenue={liqMapVenue}
          onLiqMapVenueChange={isEquity ? undefined : setLiqMapVenue}
          liqMapSide={liqMapSide}
          onLiqMapSideChange={isEquity ? undefined : setLiqMapSide}
          liqMapSource={liqMapSource}
          onLiqMapSourceChange={isEquity ? undefined : setLiqMapSource}
          onRefresh={refreshAll}
          isFetching={
            seriesFetching ||
            historyLoading ||
            (showPumpMarkers && pumpsQuery.isFetching) ||
            (showSignalMarkers && scannerResultsQuery.isFetching) ||
            (showCrowd && crowdHistQuery.isFetching) ||
            (showOi && oiHistQuery.isFetching) ||
            (showFunding && fundingHistQuery.isFetching) ||
            (showLiqNotional && liqHistQuery.isFetching) ||
            (showLiqMap && liqMapQuery.isFetching) ||
            (showOpenLevels && openLevelsQuery.isFetching) ||
            (showProfile && chartProfileQuery.isFetching) ||
            (showSweeps && chartSweepQuery.isFetching) ||
            (cvdRange !== '24h' && (showCvd || showDelta) && cvdChartQuery.isFetching)
          }
        />

        {intervalsQuery.isError && !supportedIntervals?.length ? (
          <Alert
            type="warning"
            showIcon
            message={t('detail:chart.intervalsErrorTitle')}
            description={rtkErrorMessage(intervalsQuery.error, {
              resource: t('detail:resource.intervals'),
            })}
          />
        ) : null}

        {(firstCandlesQuery.isError && candlesQuery.isError && chartData.length === 0) ? (
          <Alert
            type="error"
            showIcon
            message={t('detail:chart.loadErrorTitle')}
            description={rtkErrorMessage(candlesQuery.error ?? firstCandlesQuery.error, {
              resource: t('detail:resource.candles'),
            })}
          />
        ) : !seriesLoading &&
          (firstCandlesQuery.isSuccess || candlesQuery.isSuccess) &&
          chartData.length === 0 ? (
          <Alert
            type="info"
            showIcon
            message={t('detail:chart.emptyTitle')}
            description={t('detail:chart.emptyBody')}
          />
        ) : (
          <>
            {candlesQuery.isError && chartData.length > 0 ? (
              <Alert
                type="warning"
                showIcon
                message={t('detail:chart.loadErrorTitle')}
                description={rtkErrorMessage(candlesQuery.error, {
                  resource: t('detail:resource.candles'),
                })}
              />
            ) : null}
            {!isEquity &&
            (showDelta ||
              showCvd ||
              showLiqLevels ||
              showBook ||
              showFootprint ||
              showProfile ||
              showSweeps ||
              showBots ||
              showCrowd ||
              showOi ||
              showFunding ||
              showLiqNotional ||
              showLiqMap ||
              showOpenLevels) ? (
              <Text variant="caption" color="secondary">
                {t('detail:chart.tapeHint')}
              </Text>
            ) : null}
            {!isEquity && (showOi || showFunding || showLiqNotional) ? (
              <Text variant="caption" color="secondary" data-testid="chart-hyperliquid-limit">
                {t('detail:chart.hyperliquidLimit')}
              </Text>
            ) : null}
            {!isEquity && showCrowd && positioningRead.noTop.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-no-top">
                {t('detail:chart.noTopVenue', {
                  venues: positioningRead.noTop
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showOi && positioningRead.oiAnchor.missingDay.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-oi-day-missing">
                {t('detail:chart.oiDayMissing', {
                  venues: positioningRead.oiAnchor.missingDay
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showOi && positioningRead.oiAnchor.missingBlock.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-oi-4h-missing">
                {t('detail:chart.oi4hMissing', {
                  venues: positioningRead.oiAnchor.missingBlock
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showCrowd && positioningRead.gapAnchor.missingDay.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-gap-day-missing">
                {t('detail:chart.gapDayMissing', {
                  venues: positioningRead.gapAnchor.missingDay
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showCrowd && positioningRead.gapAnchor.missingBlock.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-gap-4h-missing">
                {t('detail:chart.gap4hMissing', {
                  venues: positioningRead.gapAnchor.missingBlock
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity &&
            cvdRange !== '24h' &&
            tapeCvd.points.length > 0 &&
            !tapeCvd.complete ? (
              <Text variant="caption" color="secondary" data-testid="chart-cvd-partial">
                {t('detail:chart.cvdPartialNamed', {
                  market: t('detail:chart.cvdFutures'),
                  range: cvdRange,
                })}
              </Text>
            ) : null}
            {!isEquity &&
            showCvd &&
            cvdRange !== '24h' &&
            spotCvd.points.length > 0 &&
            !spotCvd.complete ? (
              <Text variant="caption" color="secondary" data-testid="chart-spot-cvd-partial">
                {t('detail:chart.cvdPartialNamed', {
                  market: t('detail:chart.cvdSpot'),
                  range: cvdRange,
                })}
              </Text>
            ) : null}
            {!isEquity && showCvd && tapeCvd.split ? (
              <Text variant="caption" color="secondary" data-testid="chart-cvd-split">
                {t(tapeCvd.withheld.length > 0 ? 'detail:chart.cvdSplitWithheld' : 'detail:chart.cvdSplit', {
                  kind: t(
                    tapeCvd.split.kind === 'price_down_cvd_up'
                      ? 'detail:chart.cvdSplitDown'
                      : 'detail:chart.cvdSplitUp',
                  ),
                  duration: tapeCvd.split.duration,
                  exchange: t(`detail:liqHeatmap.venues.${tapeCvd.source}`, {
                    defaultValue: tapeCvd.source,
                  }),
                  venues: tapeCvd.withheld
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && tapeCvd.points.length > 0 && tapeCvd.withheld.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-cvd-venue">
                {t('detail:chart.cvdVenue', {
                  exchange: t(`detail:liqHeatmap.venues.${tapeCvd.source}`, {
                    defaultValue: tapeCvd.source,
                  }),
                  venues: tapeCvd.withheld
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showCvd && (tapeSize.whale.length > 0 || tapeSize.retail.length > 0) ? (
              <Text variant="caption" color="secondary" data-testid="chart-cvd-size">
                {t('detail:chart.cvdSize')}
              </Text>
            ) : null}
            {!isEquity &&
            showCvd &&
            tapeSize.source !== '' &&
            tapeSize.source !== 'combined' &&
            tapeSize.withheld.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-cvd-size-venue">
                {t('detail:chart.cvdSizeVenue', {
                  exchange: t(`detail:liqHeatmap.venues.${tapeSize.source}`, {
                    defaultValue: tapeSize.source,
                  }),
                  venues: tapeSize.withheld
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showCvd && spotCvd.points.length > 0 && spotCvd.withheld.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-spot-cvd-venue">
                {t('detail:chart.cvdVenueSpot', {
                  exchange: t(`detail:liqHeatmap.venues.${spotCvd.source}`, {
                    defaultValue: spotCvd.source,
                  }),
                  venues: spotCvd.withheld
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showCvd && cvdClickSec == null ? (
              <Text variant="caption" color="secondary" data-testid="chart-cvd-click-hint">
                {t('detail:chart.cvdClickHint')}
              </Text>
            ) : null}
            {!isEquity && positioningClickOn && cvdClickSec == null ? (
              <Text variant="caption" color="secondary" data-testid="chart-positioning-click-hint">
                {t('detail:chart.positioningClickHint')}
              </Text>
            ) : null}
            {!isEquity && showCvd && clickCvd ? (
              <Text variant="caption" color="secondary" data-testid="chart-cvd-click">
                {t(
                  clickCvd.empty
                    ? 'detail:chart.cvdClickEmpty'
                    : clickCvd.late
                      ? 'detail:chart.cvdClickLate'
                      : 'detail:chart.cvdClick',
                  {
                    time: clickCvd.time,
                    amount: clickCvd.amount,
                    exchange: clickCvd.exchange,
                  },
                )}{' '}
                <Button
                  type="link"
                  size="small"
                  onClick={() => setCvdClickSec(null)}
                  data-testid="chart-cvd-click-clear"
                >
                  {t('detail:chart.cvdClickClear')}
                </Button>
              </Text>
            ) : null}
            {!isEquity && positioningClickOn && cvdClickSec != null ? (
              <Text variant="caption" color="secondary" data-testid="chart-positioning-click">
                {positioningClickText}{' '}
                <Button
                  type="link"
                  size="small"
                  onClick={() => setCvdClickSec(null)}
                  data-testid="chart-positioning-click-clear"
                >
                  {t('detail:chart.cvdClickClear')}
                </Button>
              </Text>
            ) : null}
            {!isEquity && showBook && bookWindow > CHART_BOOK_LIVE_MAX ? (
              <Text variant="caption" color="secondary" data-testid="chart-book-stored">
                {t('detail:chart.bookStored')}
              </Text>
            ) : null}
            {!isEquity && showProfile && (profileRead?.lines.length ?? 0) > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-profile">
                {t('detail:chart.profileCaption')}
              </Text>
            ) : null}
            {!isEquity && showProfile && profileRead && !profileRead.definite && profileRead.missing.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-profile-venue">
                {t('detail:chart.profileVenue', {
                  venues: profileRead.missing
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                  exchange: t(`detail:liqHeatmap.venues.${profileRead.exchange}`, {
                    defaultValue: profileRead.exchange,
                  }),
                })}
              </Text>
            ) : null}
            {!isEquity && showSweeps && sweepSpanOutside(tapeSpan, Date.now() / 1000) ? (
              <Text variant="caption" color="secondary" data-testid="chart-sweeps-outside">
                {t('detail:chart.sweepOutside')}
              </Text>
            ) : null}
            {!isEquity &&
            showSweeps &&
            !sweepSpanOutside(tapeSpan, Date.now() / 1000) &&
            chartSweepMarks(rtkCurrent(chartSweepQuery), sweepVenue, tapeSpan).length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-sweeps">
                {t('detail:chart.sweepCaption', {
                  exchange: t(`detail:liqHeatmap.venues.${sweepVenue}`, { defaultValue: sweepVenue }),
                })}
              </Text>
            ) : null}
            {!isEquity && showBots && chartBotMarks(rtkCurrent(botTrackerQuery), tapeSpan).length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-bots">
                {t('detail:chart.botCaption')}
              </Text>
            ) : null}
            {!isEquity && showFootprint && chartCandleFootprint.candles.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-candle-footprint">
                {t('detail:chart.candleFootprint')}
              </Text>
            ) : null}
            {!isEquity &&
            showFootprint &&
            chartCandleFootprint.candles.length > 0 &&
            !chartCandleFootprint.definite &&
            chartCandleFootprint.missing.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-candle-footprint-venue">
                {t('detail:chart.candleFootprintVenue', {
                  venues: chartCandleFootprint.missing
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                  exchange: t(`detail:liqHeatmap.venues.${chartCandleFootprint.exchange}`, {
                    defaultValue: chartCandleFootprint.exchange,
                  }),
                })}
              </Text>
            ) : null}
            {!isEquity && showFootprint && chartCandleFootprint.truncated && chartCandleFootprint.candles.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-candle-footprint-truncated">
                {t('detail:chart.candleFootprintTruncated')}
              </Text>
            ) : null}
            {!isEquity && recordedLine ? (
              <Text variant="caption" color="secondary" data-testid="chart-recorded">
                {recordedLine}
              </Text>
            ) : null}
            {!isEquity && showLiqMap && liqMapSource === 'prints' ? (
              <Text variant="caption" color="secondary" data-testid="chart-liq-prints">
                {t('detail:liqHeatmap.subtitlePrints')}
              </Text>
            ) : null}
            {!isEquity && showLiqMap && liqMapSource === 'model' && liqMapReadsArchive(liqMapRange) ? (
              <Text variant="caption" color="secondary" data-testid="chart-liq-archive">
                {t('detail:liqHeatmap.archive')}
              </Text>
            ) : null}
            {!isEquity && chartMix ? (
              <Text variant="caption" color="secondary" data-testid="chart-liq-mix">
                {t('detail:liqHeatmap.leverageMix', { mix: chartMix })}
              </Text>
            ) : null}
            {!isEquity && showLiqMap && liqMapNote && liqMapNote.evenVenues.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-liq-even">
                {t('detail:liqHeatmap.evenBlend', {
                  venues: liqMapNote.evenVenues
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && showLiqMap && liqMapSource === 'prints' && liqMapNote && liqMapNote.gapVenues.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-liq-gap">
                {t('detail:liqHeatmap.gap', {
                  venues: liqMapNote.gapVenues
                    .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && chartMissing.length > 0 ? (
              <Text variant="caption" color="secondary" data-testid="chart-liq-missing">
                {t(
                  showLiqMap && liqMapSource === 'prints'
                    ? 'detail:liqHeatmap.missingPrints'
                    : 'detail:liqHeatmap.missing',
                  {
                    venues: chartMissing
                      .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                      .join(', '),
                  },
                )}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-caption">
                {t('detail:openLevels.caption')}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openAnchorRead?.day ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-day">
                {t('detail:openLevels.sinceDay', openAnchorRead.day)}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openAnchorRead?.lines.missingDay ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-day-missing">
                {t('detail:openLevels.dayMissing')}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openAnchorRead?.block ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-4h">
                {t('detail:openLevels.since4h', openAnchorRead.block)}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openAnchorRead?.lines.missingBlock ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-4h-missing">
                {t('detail:openLevels.blockMissing')}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && cvdClickSec == null && !showCvd && !positioningClickOn ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-click-hint">
                {t('detail:openLevels.clickHint')}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openAnchorRead?.click ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-click">
                {t('detail:openLevels.sinceClick', openAnchorRead.click)}{' '}
                {!showCvd && !positioningClickOn ? (
                  <Button
                    type="link"
                    size="small"
                    onClick={() => setCvdClickSec(null)}
                    data-testid="chart-open-click-clear"
                  >
                    {t('detail:chart.cvdClickClear')}
                  </Button>
                ) : null}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openAnchorRead?.clickMissing && cvdClickSec != null ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-click-missing">
                {t('detail:openLevels.clickMissing', { time: formatUtcMinute(cvdClickSec) })}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openLevelNote?.withheld ? (
              <Text variant="caption" color="secondary" data-testid="chart-open-withheld">
                {t('detail:openLevels.withheld')}
              </Text>
            ) : null}
            {!isEquity && showOpenLevels && openLevelsQuery.isError ? (
              <Text variant="caption" color="secondary">
                {rtkErrorMessage(openLevelsQuery.error, {
                  resource: t('detail:resource.openLevels'),
                })}
              </Text>
            ) : null}
            {!isEquity && showLiqMap && liqMapNeedsPan ? (
              <Text variant="caption" color="secondary" data-testid="chart-liq-pan">
                {t('detail:chart.liqMapPan')}
              </Text>
            ) : null}
            {!isEquity && showLiqMap && liqMapQuery.isError ? (
              <Text variant="caption" color="secondary">
                {rtkErrorMessage(liqMapQuery.error, {
                  resource: t('detail:resource.liqHeatmap'),
                })}
              </Text>
            ) : null}
            {!isEquity && showLiqLevels && rtkCurrent(liqLevelsQuery)?.review?.summary ? (
              <Text variant="caption" color="secondary" data-testid="chart-level-review">
                {rtkCurrent(liqLevelsQuery)?.review?.summary}
              </Text>
            ) : null}
            {!isEquity && chartPreset === 'orderFlow' && absorptionNote ? (
              <Text variant="caption" color="secondary" data-testid="chart-preset-absorption">
                {absorptionNote.text}
                {absorptionNote.exchange !== 'combined' && absorptionNote.withheld.length > 0
                  ? ` ${t('detail:chart.presetAbsorptionWithheld', {
                      venues: absorptionNote.withheld
                        .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                        .join(', '),
                    })}`
                  : ''}
              </Text>
            ) : null}
            {!isEquity && chartPreset === 'orderFlow' && absorptionQuery.isError ? (
              <Text variant="caption" color="secondary" data-testid="chart-preset-absorption-error">
                {rtkErrorMessage(absorptionQuery.error, { resource: t('detail:resource.tape') })}
              </Text>
            ) : null}
            {!isEquity && chartPreset === 'orderFlow' && printSizeNote ? (
              <Text variant="caption" color="secondary" data-testid="chart-preset-print-size">
                {printSizeNote.text}
                {printSizeNote.exchange !== 'combined' && printSizeNote.withheld.length > 0
                  ? ` ${t('detail:chart.presetPrintSizeWithheld', {
                      venues: printSizeNote.withheld
                        .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                        .join(', '),
                    })}`
                  : ''}
              </Text>
            ) : null}
            {!isEquity && chartPreset === 'liquidity' && perpBid ? (
              <Text variant="caption" color="secondary" data-testid="chart-preset-bid">
                {perpBid.definite && perpBid.bidPct != null
                  ? t('detail:chart.presetBid', {
                      pct: formatDeskPct(perpBid.bidPct),
                      range: perpBid.rangePct,
                    })
                  : perpBid.missing.length > 0
                    ? t('detail:chart.presetBidMissing', {
                        venues: perpBid.missing
                          .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                          .join(', '),
                      })
                    : t('detail:chart.presetBidUncovered')}
              </Text>
            ) : null}
            {!isEquity && chartPreset === 'liquidity' && perpBid && perpBid.recorded.length > 0 && !perpBid.definite ? (
              <Text variant="caption" color="secondary" data-testid="chart-preset-bid-recorded">
                {t('detail:chart.presetBidRecorded', {
                  lines: perpBid.recorded
                    .map(
                      (row) =>
                        `${t(`detail:liqHeatmap.venues.${row.exchange}`, { defaultValue: row.exchange })} ${formatDeskPct(row.bidPct)}%`,
                    )
                    .join(', '),
                })}
              </Text>
            ) : null}
            {!isEquity && perpBandsOn ? (
              <Text variant="caption" color="secondary" data-testid="chart-perp-book">
                {t('detail:chart.perpBook')}
                {(rtkCurrent(perpBookQuery)?.gaps?.length ?? 0) > 0
                  ? ` ${t('detail:chart.perpBookGap', {
                      venues: [
                        ...new Set(
                          (rtkCurrent(perpBookQuery)?.gaps ?? []).flatMap((gap) => [
                            ...(gap.missing ?? []),
                            ...(gap.stale ?? []),
                          ]),
                        ),
                      ]
                        .map((id) => t(`detail:liqHeatmap.venues.${id}`, { defaultValue: id }))
                        .join(', '),
                    })}`
                  : ''}
              </Text>
            ) : null}
            {!isEquity && perpReach ? (
              <Text variant="caption" color="secondary" data-testid="chart-perp-book-reach">
                {t('detail:chart.perpBookReach', {
                  bid: formatBookReachPct(perpReach.bidPct),
                  ask: formatBookReachPct(perpReach.askPct),
                })}
              </Text>
            ) : null}
            {!isEquity && chartPreset === 'liquidity' && perpDepthQuery.isError ? (
              <Text variant="caption" color="secondary" data-testid="chart-preset-bid-error">
                {rtkErrorMessage(perpDepthQuery.error, { resource: t('detail:resource.tape') })}
              </Text>
            ) : null}
            {!isEquity && chartPreset === 'liquidity' && !rtkCurrentPending(orderBookQuery) ? (
              <Text variant="caption" color="secondary" data-testid="chart-preset-walls">
                {liveWalls.length > 0
                  ? t('detail:chart.presetWalls', {
                      walls: liveWalls
                        .map((wall) =>
                          t('detail:chart.presetWall', {
                            side: t(`detail:chart.wallSide.${wall.side}`),
                            price: wall.price,
                            behavior: wall.behavior
                              ? t(`detail:chart.wallBehavior.${wall.behavior}`, { defaultValue: wall.behavior })
                              : '',
                          }),
                        )
                        .join('; '),
                    })
                  : t('detail:chart.presetWallsEmpty')}
              </Text>
            ) : null}
            <CandleChartHost
              data={chartData}
              overlays={overlays}
              lowerSeries={lowerSeries}
              priceLines={chartPriceLines}
              bookHeat={chartBookHeat}
              footprint={chartFootprint}
              candleFootprints={chartCandleFootprint.candles}
              footprintStepSec={
                chartCandleFootprint.stepSec > 0 ? chartCandleFootprint.stepSec : barDurationSec
              }
              liqProfile={chartLiqProfile}
              liqHeat={chartLiqHeat}
              openLevels={chartOpenLevels}
              onVisibleSpan={isEquity ? undefined : onVisibleSpan}
              onBarTimeClick={isEquity ? undefined : onCandleAnchor}
              rightPadding={
                !isEquity &&
                (showLiqLevels || (showFootprint && chartCandleFootprint.candles.length === 0))
                  ? 16
                  : undefined
              }
              markers={chartMarkers}
              vertLines={chartVertLines}
              barDurationSec={barDurationSec}
              isLoading={seriesLoading}
              seriesKey={chartSeriesKey}
              isLoadingMore={isLoadingMore}
              hasMoreHistory={hasMoreHistory}
              onNeedMoreHistory={onNeedMoreHistory}
              anchorEndIndex={
                pastDelist && postDelistView?.available
                  ? allCandles.length
                  : mergedCandles.length
              }
              height={
                (isPhone ? PHONE_CHART_HEIGHT : DESK_CHART_HEIGHT) +
                (lowerSeries.some((s) => s.id === 'delta') ? (isPhone ? 72 : 120) : 0) +
                (lowerSeries.some((s) => s.id === 'cvd-range') ? (isPhone ? 72 : 120) : 0) +
                (lowerSeries.some((s) => s.id === 'cvd-day' || s.id === 'cvd-4h' || s.id === 'cvd-click')
                  ? isPhone
                    ? 72
                    : 120
                  : 0) +
                (lowerSeries.some((s) => s.id === 'whale-range') ? (isPhone ? 72 : 120) : 0) +
                (lowerSeries.some((s) => s.id === 'whale-day' || s.id === 'whale-4h' || s.id === 'whale-click')
                  ? isPhone
                    ? 72
                    : 120
                  : 0) +
                (lowerSeries.some((s) => s.id === 'retail-range') ? (isPhone ? 72 : 120) : 0) +
                (lowerSeries.some((s) => s.id === 'retail-day' || s.id === 'retail-4h' || s.id === 'retail-click')
                  ? isPhone
                    ? 72
                    : 120
                  : 0) +
                (lowerSeries.some((s) => s.id === 'spot-cvd-range') ? (isPhone ? 72 : 120) : 0) +
                (lowerSeries.some(
                  (s) => s.id === 'spot-cvd-day' || s.id === 'spot-cvd-4h' || s.id === 'spot-cvd-click',
                )
                  ? isPhone
                    ? 72
                    : 120
                  : 0) +
                (lowerSeries.some((s) => s.id.startsWith('crowd-long') || s.id.startsWith('top-long'))
                  ? isPhone
                    ? 72
                    : 100
                  : 0) +
                (lowerSeries.some((s) => s.id.startsWith('gap-day-') || s.id.startsWith('gap-4h-'))
                  ? isPhone
                    ? 72
                    : 100
                  : 0) +
                (lowerSeries.some(
                  (s) => s.id === 'oi-binance' || s.id === 'oi-bybit' || s.id === 'oi-okx',
                )
                  ? isPhone
                    ? 72
                    : 100
                  : 0) +
                (lowerSeries.some((s) => s.id.startsWith('oi-day-') || s.id.startsWith('oi-4h-'))
                  ? isPhone
                    ? 72
                    : 100
                  : 0) +
                (lowerSeries.some((s) => s.id.startsWith('oi-click-')) ? (isPhone ? 72 : 100) : 0) +
                (lowerSeries.some((s) => s.id.startsWith('funding-') && !s.id.startsWith('funding-click-'))
                  ? isPhone
                    ? 72
                    : 100
                  : 0) +
                (lowerSeries.some((s) => s.id.startsWith('funding-click-')) ? (isPhone ? 72 : 100) : 0) +
                (lowerSeries.some((s) => s.id === 'liq-notional') ? (isPhone ? 72 : 120) : 0) +
                (lowerSeries.some((s) => s.id.startsWith('liq-click-')) ? (isPhone ? 72 : 100) : 0) +
                (lowerSeries.some((s) => s.id === 'open-liq-delta') ? (isPhone ? 72 : 100) : 0) +
                (lowerSeries.some(
                  (s) => s.id === 'open-liq-day' || s.id === 'open-liq-4h' || s.id === 'open-liq-click',
                )
                  ? isPhone
                    ? 72
                    : 100
                  : 0)
              }
            />
            {!isEquity && symbol ? (
              <>
              {cvdClickSec != null ? (
                <CoinCandleSituation
                  timeLabel={
                    situationAt != null
                      ? formatUtcMinute(Math.floor(new Date(situationAt).getTime() / 1000))
                      : null
                  }
                  report={rtkCurrent(situationPastQuery)}
                  isLoading={rtkCurrentPending(situationPastQuery)}
                  error={
                    situationPastQuery.isError
                      ? rtkErrorMessage(situationPastQuery.error, {
                          resource: t('detail:situation.candle.title'),
                        })
                      : null
                  }
                />
              ) : null}
              <CoinCandleForensics
                timeLabel={cvdClickSec != null ? formatUtcMinute(cvdClickSec) : null}
                report={rtkCurrent(forensicsQuery) as ForensicsReport | undefined}
                isLoading={rtkCurrentPending(forensicsQuery)}
                error={
                  forensicsQuery.isError
                    ? rtkErrorMessage(forensicsQuery.error, {
                        resource: t('detail:chart.forensicsTitle'),
                      })
                    : null
                }
                fullReadTo={
                  symbol && forensicsAt ? coinForensicsHref(symbol, forensicsAt) : null
                }
                onClear={() => setCvdClickSec(null)}
                onOpenMove={(at) => {
                  const sec = clickSecFromForensicsAt(at);
                  if (sec == null || sec === cvdClickSec) return;
                  setCvdClickSec(sec);
                }}
              />
              </>
            ) : null}
          </>
        )}
      </ChartCard>
      </TabStack>
            ),
          },
          ...(!isEquity
            ? [
                {
                  key: 'orderbook',
                  label: t('detail:tabs.orderbook'),
                  children: (
                    <TabStack>
                      <ChartAndBook>
                        <OrderDepthChart
                          book={rtkCurrent(orderBookQuery)}
                          isLoading={rtkCurrentPending(orderBookQuery)}
                          isFetching={orderBookQuery.isFetching}
                          errorMessage={
                            orderBookQuery.isError
                              ? rtkErrorMessage(orderBookQuery.error, {
                                  resource: t('detail:resource.orderBook'),
                                })
                              : null
                          }
                        />
                        <SideStack>
                          <SpreadOpenNote exchange={exchangeArg} symbol={symbol} />
                          <OrderBookPanel
                            book={rtkCurrent(orderBookQuery)}
                            group={orderBookGroup || rtkCurrent(orderBookQuery)?.groupSize || ''}
                            onGroupChange={setOrderBookGroup}
                            isLoading={rtkCurrentPending(orderBookQuery)}
                            isFetching={orderBookQuery.isFetching}
                            errorMessage={
                              orderBookQuery.isError
                                ? rtkErrorMessage(orderBookQuery.error, {
                                    resource: t('detail:resource.orderBook'),
                                  })
                                : null
                            }
                          />
                        </SideStack>
                      </ChartAndBook>
                      <BookGapsCard exchange={exchangeArg} symbol={symbol} />
                      <OrderHeatmap
                        data={rtkCurrent(orderHeatmapQuery)}
                        windowSeconds={orderHeatmapWindow}
                        onWindowChange={setOrderHeatmapWindow}
                        isLoading={rtkCurrentPending(orderHeatmapQuery)}
                        isFetching={orderHeatmapQuery.isFetching}
                        errorMessage={
                          orderHeatmapQuery.isError
                            ? rtkErrorMessage(orderHeatmapQuery.error, {
                                resource: t('detail:resource.orderBook'),
                              })
                            : null
                        }
                      />
                      <PastOrderBook symbol={symbol} exchange={exchangeArg} />
                    </TabStack>
                  ),
                },
                {
                  key: 'tape',
                  label: t('detail:tabs.tape'),
                  children: (
                    <TabStack>
                      <UnusualContext
                        report={rtkCurrent(unusualQuery)}
                        isLoading={rtkCurrentPending(unusualQuery)}
                        error={
                          unusualQuery.isError
                            ? rtkErrorMessage(unusualQuery.error, {
                                resource: t('detail:resource.tape'),
                              })
                            : null
                        }
                      />
                      <TapePanel
                        openInterest={rtkCurrent(openInterestQuery)}
                        netPosition={rtkCurrent(netPositionQuery)}
                        netPositionError={
                          netPositionQuery.isError
                            ? rtkErrorMessage(netPositionQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  404: t('detail:tape.none'),
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        openInterestError={
                          openInterestQuery.isError
                            ? rtkErrorMessage(openInterestQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  404: t('detail:tape.none'),
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        liquidations={rtkCurrent(liquidationsQuery)}
                        liquidationsError={
                          liquidationsQuery.isError
                            ? rtkErrorMessage(liquidationsQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  404: t('detail:tape.none'),
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        cvd={rtkCurrent(cvdQuery)}
                        cvdError={
                          cvdQuery.isError
                            ? rtkErrorMessage(cvdQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        snapshot={rtkCurrent(snapshotQuery)}
                        snapshotError={
                          snapshotQuery.isError
                            ? rtkErrorMessage(snapshotQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        volumeShare={rtkCurrent(volumeShareQuery)}
                        volumeShareError={
                          volumeShareQuery.isError
                            ? rtkErrorMessage(volumeShareQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        symbol={perpSymbol}
                        perpLiquidity={rtkCurrent(perpLiqQuery)}
                        perpLiquidityError={
                          perpLiqQuery.isError
                            ? rtkErrorMessage(perpLiqQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        participation={rtkCurrent(participationQuery)}
                        participationError={
                          participationQuery.isError
                            ? rtkErrorMessage(participationQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        printSize={rtkCurrent(printSizeQuery)}
                        orderCount={rtkCurrent(orderCountQuery)}
                        orderCountError={
                          orderCountQuery.isError
                            ? rtkErrorMessage(orderCountQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        botTracker={rtkCurrent(botTrackerQuery)}
                        botTrackerError={
                          botTrackerQuery.isError
                            ? rtkErrorMessage(botTrackerQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        printSizeError={
                          printSizeQuery.isError
                            ? rtkErrorMessage(printSizeQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        pressure={rtkCurrent(pressureQuery)}
                        pressureError={
                          pressureQuery.isError
                            ? rtkErrorMessage(pressureQuery.error, {
                                resource: t('detail:resource.tape'),
                                statusMessages: {
                                  400: t('detail:tape.unsupported'),
                                },
                              })
                            : null
                        }
                        isLoading={
                          rtkCurrentPending(openInterestQuery) ||
                          rtkCurrentPending(liquidationsQuery) ||
                          rtkCurrentPending(cvdQuery) ||
                          rtkCurrentPending(snapshotQuery) ||
                          rtkCurrentPending(volumeShareQuery) ||
                          rtkCurrentPending(perpLiqQuery) ||
                          rtkCurrentPending(participationQuery) ||
                          rtkCurrentPending(botTrackerQuery) ||
                          rtkCurrentPending(orderCountQuery) ||
                          rtkCurrentPending(printSizeQuery) ||
                          rtkCurrentPending(pressureQuery)
                        }
                      />
                    </TabStack>
                  ),
                },
                {
                  key: 'flow',
                  label: t('detail:tabs.flow'),
                  children:
                    tab === 'flow' ? (
                      <TabStack>
                        <CoinFlowTab symbol={symbol} exchange={exchangeArg} />
                      </TabStack>
                    ) : null,
                },
                {
                  key: 'structure',
                  label: t('detail:tabs.structure'),
                  children:
                    tab === 'structure' ? (
                      <TabStack>
                        <CoinStructureTab symbol={symbol} exchange={exchangeArg} />
                      </TabStack>
                    ) : null,
                },
                {
                  key: 'holders',
                  label: t('detail:tabs.holders'),
                  children: (
                    <HolderPanel
                      holders={rtkCurrent(holdersQuery)}
                      isLoading={rtkCurrentPending(holdersQuery)}
                      error={
                        holdersQuery.isError
                          ? rtkErrorMessage(holdersQuery.error, {
                              resource: t('detail:resource.holders'),
                              codeMessages: {
                                catalog_unmapped: t('detail:errors.holdersUnmapped'),
                                holders_unpublished: t('detail:errors.holders404'),
                              },
                              statusMessages: {
                                404: t('detail:errors.holders404'),
                              },
                            })
                          : null
                      }
                    />
                  ),
                },
              ]
            : []),
          {
            key: 'indicators',
            label: t('detail:tabs.indicators'),
            children: (
              <IndicatorPanel
                data={liveIndicators}
                priceQuote={pairQuote(symbol, exchange)}
                isLoading={rtkCurrentPending(indicatorsQuery)}
                errorMessage={
                  indicatorsQuery.isError
                    ? rtkErrorMessage(indicatorsQuery.error, {
                        resource: t('detail:resource.indicators'),
                      })
                    : null
                }
                showEmaOnChart={showEma}
                onToggleEma={setShowEma}
              />
            ),
          },
          {
            key: 'trade',
            label: t('detail:tabs.trade'),
            children: (
      <PaperTradeCard>
        <Text variant="h4" color="primary">
          {t('detail:paperTrade.title')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('detail:paperTrade.hint')}
        </Text>
        {books.length === 0 ? (
          <>
            <Text variant="body" color="secondary">
              {t('detail:paperTrade.noBook')}
            </Text>
            <Link to="/portfolio">
              <Button type="link" size="small">
                {t('detail:paperTrade.openPortfolio')}
              </Button>
            </Link>
          </>
        ) : (
          <>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center' }}>
              <Text variant="caption" color="secondary">
                {t('detail:paperTrade.book')}
              </Text>
              <Select
                size="small"
                style={{ minWidth: 160 }}
                value={paperBookId || undefined}
                options={books.map((b) => ({
                  value: b.id ?? '',
                  label: b.name || b.id,
                }))}
                onChange={(id) => {
                  setPaperBookId(id);
                  try {
                    localStorage.setItem('swyngora.portfolioBookId', id);
                  } catch {
                    /* ignore */
                  }
                }}
                placeholder={t('detail:paperTrade.selectBook')}
              />
              <Link to={`/portfolio?book=${encodeURIComponent(paperBookId || '')}`}>
                <Button type="link" size="small">
                  {t('detail:paperTrade.openPortfolio')}
                </Button>
              </Link>
            </div>
            {rtkCurrent(paperPortfolio)?.availableCash != null ? (
              <Text variant="caption" color="secondary">
                {t('detail:paperTrade.availableCash', {
                  amount: formatPrice(rtkCurrent(paperPortfolio)?.availableCash),
                })}
              </Text>
            ) : null}
            {heldPosition && !isEquity ? (
              <PositionSituation
                entry={heldPosition.entrySituation}
                now={
                  situationChange.data?.liveOk
                    ? situationChange.data.live
                    : rtkCurrent(situationQuery)
                }
                nowLoading={
                  situationChange.isLoading && situationChange.data == null
                    ? true
                    : rtkCurrentPending(situationQuery) && !situationChange.data?.liveOk
                }
                comparison={
                  situationChange.data?.liveOk ? situationChange.data.comparison : undefined
                }
              />
            ) : null}
            <PaperTradeForm
              lockedExchange={exchangeArg}
              lockedSymbol={symbol}
              compact
              advanced={false}
              showLotMethod={false}
              isSubmitting={placePaperState.isLoading}
              disabled={books.length > 1 && !paperBookId}
              submitError={placePaperState.isError ? placePaperState.error : undefined}
              onSubmit={async (values: PaperTradeFormValues) => {
                if (books.length > 1 && !paperBookId) return;
                await placePaperOrder({
                  portfolioId: paperBookId || undefined,
                  exchange: values.exchange,
                  symbol: values.symbol,
                  side: values.side,
                  type: values.orderType,
                  quantity: values.quantity,
                  triggerPrice: values.triggerPrice,
                  timeInForce: values.timeInForce,
                  lotMethod: values.lotMethod,
                  idempotencyKey: newPaperIdempotencyKey('detail'),
                }).unwrap();
                void message.success(
                  values.orderType !== 'market'
                    ? t('detail:paperTrade.successPending', {
                        defaultValue: 'Pending paper order placed',
                      })
                    : values.side === 'buy'
                      ? t('detail:paperTrade.successBuy')
                      : t('detail:paperTrade.successSell'),
                );
              }}
            />
          </>
        )}
      </PaperTradeCard>
            ),
          },
        ]}
      />

      {!visible ? (
        <Text variant="caption" color="secondary">
          {t('detail:pollingPaused')}
        </Text>
      ) : null}
    </PageStack>
  );
}
