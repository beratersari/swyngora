import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Select, message } from 'antd';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { CandleChartHost } from '@/components/molecules/CandleChartHost';
import { snapMarkersToCandleTimes } from '@/components/molecules/CandleChartHost/CandleChartHost.markers';
import type {
  CandleChartMarker,
  CandleChartOverlay,
} from '@/components/molecules/CandleChartHost/CandleChartHost.types';
import { DetailChartToolbar } from '@/components/organisms/DetailChartToolbar';
import { DetailHeader } from '@/components/organisms/DetailHeader';
import { DetailStats } from '@/components/organisms/DetailStats';
import { HolderPanel } from '@/components/organisms/HolderPanel';
import { PostDelistPanel } from '@/components/organisms/PostDelistPanel';
import { TapePanel } from '@/components/organisms/TapePanel';
import { IndicatorPanel, emaColor } from '@/components/organisms/IndicatorPanel';
import {
  OrderBookPanel,
  ORDER_BOOK_LEVELS,
  ORDER_BOOK_POLL_MS,
} from '@/components/organisms/OrderBookPanel';
import { OrderDepthChart } from '@/components/organisms/OrderDepthChart';
import {
  DEFAULT_ORDER_HEATMAP_WINDOW,
  ORDER_HEATMAP_POLL_MS,
  OrderHeatmap,
} from '@/components/organisms/OrderHeatmap';
import { PaperTradeForm, type PaperTradeFormValues } from '@/components/organisms/PaperTradeForm';
import {
  rtkErrorMessage,
  useAddWatchlistItemMutation,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useGetSupplyQuery,
  useGetHoldersQuery,
  useGetAssetProfileQuery,
  useGetOpenInterestQuery,
  useGetMarketLiquidationsQuery,
  useGetMarketCvdQuery,
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
  useListScannerResultsQuery,
  usePlacePortfolioOrderMutation,
  useRemoveWatchlistItemMutation,
  type MarketExchange,
  type PumpEventDto,
} from '@/libs/api';
import { useDisplayCurrency, useDocumentVisible, useMediaQuery } from '@/libs/hooks';
import { usePriceSubscription, usePortfolioSubscription } from '@/libs/realtime';
import {
  emaLineFromCloses,
  formatDelistDay,
  formatPrice,
  newPaperIdempotencyKey,
  parseEmaPeriods,
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
  intervalToSeconds,
  marketsBackPathFromSession,
  mergeCandleHistory,
  oldestCandleOpenTimeMs,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  aliasFxCode,
  pairQuote,
  toSupplyAsset,
  toPerpSymbol,
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
  isPastDelist,
  postDelistCandleLimit,
  mergeChartMarkers,
  mergePumpEvents,
  livePumpEventsForPair,
  pumpEventsToChartMarkers,
  scannerResultsToChartMarkers,
} from './CoinDetailPage.helpers';

/**
 * Coin detail: header + 24h/supply strip, then CoinMarketCap-style tabs
 * (overview chart, order book, holders, indicators, paper trade).

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
  /** Guards against applying history pages after exchange/symbol/interval change. */
  const historyRequestIdRef = useRef(0);

  const skip = !symbol || !exchange;
  // RTK args require a concrete exchange; skipped when path venue is invalid.
  const exchangeArg = exchange ?? 'binance';
  const isEquity = exchange === 'nasdaq' || exchange === 'bist';
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
    setHistoryExhausted(false);
    setHistoryLoading(false);
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
  const supplyAsset = toSupplyAsset(symbol);

  const watchlistQuery = useGetWatchlistQuery(undefined, { refetchOnFocus: true });
  const delistQuery = useListDelistScheduleQuery(
    { exchange: (exchange ?? 'binance') as MarketExchange },
    { skip: !exchange || isEquity },
  );
  const delistHit = useMemo(() => {
    if (!symbol || !delistQuery.data?.items?.length) return null;
    return (
      delistQuery.data.items.find(
        (it) => (it.symbol ?? '').toUpperCase() === String(symbol).toUpperCase(),
      ) ?? null
    );
  }, [delistQuery.data?.items, symbol]);
  const delistTime = delistHit?.delistTime ?? null;
  const announcedAt = delistHit?.announcedAt ?? null;
  const pastDelist = isPastDelist(delistTime);
  const candleEndTime = useMemo(() => delistCandleQueryEndTime(delistTime), [delistTime]);
  const postDelistQuery = useGetPostDelistQuery(
    {
      exchange: (exchange ?? 'binance') as MarketExchange,
      symbol: symbol ?? '',
      interval,
      limit: postDelistCandleLimit(interval, delistTime),
    },
    {
      skip: skip || isEquity || !symbol || !exchange || !pastDelist,
      pollingInterval: visible ? 120_000 : 0,
      refetchOnFocus: true,
    },
  );
  const [addWatch, addWatchState] = useAddWatchlistItemMutation();
  const [removeWatch, removeWatchState] = useRemoveWatchlistItemMutation();
  const [fetchOlderCandles] = useLazyGetCandlesQuery();
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
      pollingInterval: visible ? ORDER_HEATMAP_POLL_MS : 0,
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

  const perpSymbol = toPerpSymbol(symbol);
  const tapeQueryArg = { exchange: 'all' as const, symbol: perpSymbol };
  const skipTape = skip || isEquity || !perpSymbol;
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
  const cvdQuery = useGetMarketCvdQuery(tapeQueryArg, {
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

    void Promise.all([candleReq, pumpReq])
      .then(([candleRes, pumpRes]) => {
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
    fetchOlderPumps,
    historyExhausted,
    historyLoading,
    interval,
    pumpThresholdPct,
    showPumpMarkers,
    skipSeries,
    symbol,
  ]);

  const liveIndicators = rtkCurrent(indicatorsQuery);
  const emaPeriods = useMemo(() => {
    const fromApi = parseEmaPeriods((liveIndicators?.emaPeriods ?? []).join(','));
    return fromApi.length > 0 ? fromApi : parseEmaPeriods(DEFAULT_EMA_PERIODS);
  }, [liveIndicators?.emaPeriods]);
  const overlays: CandleChartOverlay[] = useMemo(() => {
    const lines: CandleChartOverlay[] = [];
    if (showEma) {
      for (let i = 0; i < emaPeriods.length; i += 1) {
        const period = emaPeriods[i]!;
        lines.push({
          id: `ema-${period}`,
          title: t('detail:indicators.emaLabel', { period }),
          color: emaColor(String(period), i),
          data: emaLineFromCloses(chartData, period),
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
  }, [showEma, emaPeriods, chartData, mergedCandles.length, t]);

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
    return mergeChartMarkers(pumpSnapped, signalSnapped);
  }, [
    showPumpMarkers,
    showSignalMarkers,
    pumpsQuery,
    historyPumpEvents,
    pumpThresholdPct,
    scannerResultsQuery.data?.results,
    exchange,
    symbol,
    chartData,
    interval,
  ]);

  const chartVertLines = useMemo(
    () =>
      delistEventsToVertLines({
        announcedAt,
        delistTime,
        announcedLabel: t('detail:chart.delistAnnounced'),
        delistLabel: t('detail:chart.delistHalt'),
        announcedColor: semanticColors.status.warning,
        delistColor: semanticColors.status.error,
      }),
    [announcedAt, delistTime, t],
  );

  const tab: DetailTab = useMemo(() => {
    if (
      isEquity &&
      (urlState.tab === 'orderbook' || urlState.tab === 'holders' || urlState.tab === 'tape')
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
          onRefresh={refreshAll}
          isFetching={
            seriesFetching ||
            historyLoading ||
            (showPumpMarkers && pumpsQuery.isFetching) ||
            (showSignalMarkers && scannerResultsQuery.isFetching)
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
            <CandleChartHost
              data={chartData}
              overlays={overlays}
              markers={chartMarkers}
              vertLines={chartVertLines}
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
              height={isPhone ? PHONE_CHART_HEIGHT : DESK_CHART_HEIGHT}
            />
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
                    </TabStack>
                  ),
                },
                {
                  key: 'tape',
                  label: t('detail:tabs.tape'),
                  children: (
                    <TapePanel
                      openInterest={rtkCurrent(openInterestQuery)}
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
                      isLoading={
                        rtkCurrentPending(openInterestQuery) ||
                        rtkCurrentPending(liquidationsQuery) ||
                        rtkCurrentPending(cvdQuery)
                      }
                    />
                  ),
                },
                {
                  key: 'holders',
                  label: t('detail:tabs.holders'),
                  children: (
                    <HolderPanel
                      holders={rtkCurrent(holdersQuery)}
                      circulatingSupply={rtkCurrent(supplyQuery)?.circulatingSupply}
                      priceUsd={
                        rtkCurrent(supplyQuery)?.currentPriceUsd ??
                        (aliasFxCode(pairQuote(symbol, exchange)) === 'USD' &&
                        Number.isFinite(Number(rtkCurrent(tickerQuery)?.lastPrice))
                          ? Number(rtkCurrent(tickerQuery)?.lastPrice)
                          : null)
                      }
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
            <PaperTradeForm
              lockedExchange={exchangeArg}
              lockedSymbol={symbol}
              compact
              advanced={false}
              showLotMethod={false}
              isSubmitting={placePaperState.isLoading}
              submitError={placePaperState.isError ? placePaperState.error : undefined}
              onSubmit={async (values: PaperTradeFormValues) => {
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
