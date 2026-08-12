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
import { IndicatorPanel, emaColor } from '@/components/organisms/IndicatorPanel';
import {
  OrderBookPanel,
  ORDER_BOOK_LEVELS,
  ORDER_BOOK_POLL_MS,
} from '@/components/organisms/OrderBookPanel';
import { PaperTradeForm, type PaperTradeFormValues } from '@/components/organisms/PaperTradeForm';
import {
  rtkErrorMessage,
  useAddWatchlistItemMutation,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useGetSupplyQuery,
  useGetTicker24hQuery,
  useGetSpotOrderBookQuery,
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
import { useDocumentVisible, useMediaQuery } from '@/libs/hooks';
import { usePriceSubscription, usePortfolioSubscription } from '@/libs/realtime';
import { formatPrice } from '@/libs/utils';
import { mediaQueries } from '@/styles/tokens';
import {
  apiCandlesToChart,
  detailStateToSearchParams,
  filterValidApiCandles,
  indicatorPointsToEmaLine,
  intervalToSeconds,
  marketsBackPath,
  mergeCandleHistory,
  oldestCandleOpenTimeMs,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  sortedEmaKeys,
  toSupplyAsset,
  trimCandlesToMax,
  type ApiCandle,
} from '@/libs/utils';
import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
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
  PageStack,
  PaperTradeCard,
  SideStack,
} from './CoinDetailPage.styles';
import {
  mergeChartMarkers,
  mergePumpEvents,
  pumpEventsToChartMarkers,
  scannerResultsToChartMarkers,
} from './CoinDetailPage.helpers';

/**
 * Coin detail: 24h ticker, supply, OHLCV chart (EMA overlays), RSI/EMA analysis,
 * and a backend-grouped spot order book (price steps + walls).
 * Live candles poll a short window; pan-left pages older bars via endTime.
 * Pump/dump markers: live window is polled; each history page fetches pumps for
 * the same endTime range so markers exist on older candles, not only the right edge.
 * Route: /markets/:exchange/:symbol
 */
export function CoinDetailPage() {
  const { t } = useTranslation(['detail', 'common']);
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
  const [showSignalMarkers, setShowSignalMarkers] = useState(true);
  /** Guards against applying history pages after exchange/symbol/interval change. */
  const historyRequestIdRef = useRef(0);

  const skip = !symbol || !exchange;
  // RTK args require a concrete exchange; skipped when path venue is invalid.
  const exchangeArg = exchange ?? 'binance';
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
    setSearchParams(detailStateToSearchParams({ interval: next }), { replace: true });
  }, [supportedIntervals, urlState.interval, setSearchParams]);

  const skipSeries = skip || waitingForIntervals;
  const supplyAsset = toSupplyAsset(symbol);

  const watchlistQuery = useGetWatchlistQuery(undefined, { refetchOnFocus: true });
  const delistQuery = useListDelistScheduleQuery(
    { exchange: (exchange ?? 'binance') as MarketExchange },
    { skip: !exchange || exchange !== 'binance' },
  );
  const delistTime = useMemo(() => {
    if (!symbol || !delistQuery.data?.items?.length) return null;
    const hit = delistQuery.data.items.find(
      (it) => (it.symbol ?? '').toUpperCase() === String(symbol).toUpperCase(),
    );
    return hit?.delistTime ?? null;
  }, [delistQuery.data?.items, symbol]);
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
      skip,
      pollingInterval: visible ? ORDER_BOOK_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  const suggestedBookGroups = orderBookQuery.data?.suggestedGroupSizes;
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
      skip: skip || !supplyAsset,
      pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  // Live head only — polled. Deeper history is paged with endTime (see onNeedMoreHistory).
  const candlesQuery = useGetCandlesQuery(
    { exchange: exchangeArg, symbol, interval, limit: DEFAULT_DETAIL_CANDLE_LIMIT },
    {
      skip: skipSeries,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const liveCandles = useMemo(
    () => filterValidApiCandles(candlesQuery.data?.candles),
    [candlesQuery.data?.candles],
  );

  const allCandles = useMemo(
    () =>
      trimCandlesToMax(
        mergeCandleHistory(historyCandles, liveCandles),
        DETAIL_CANDLE_MAX_LIMIT,
      ),
    [historyCandles, liveCandles],
  );

  const chartData = useMemo(() => apiCandlesToChart(allCandles), [allCandles]);

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
      skip: skipSeries || !showPumpMarkers,
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
      skip: skipSeries,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const hasMoreHistory =
    !historyExhausted &&
    allCandles.length > 0 &&
    allCandles.length < DETAIL_CANDLE_MAX_LIMIT &&
    (candlesQuery.isSuccess || historyCandles.length > 0);

  const isLoadingMore = historyLoading;

  const onNeedMoreHistory = useCallback(() => {
    if (historyLoading || historyExhausted || skipSeries || !symbol || !exchange) return;
    if (allCandles.length >= DETAIL_CANDLE_MAX_LIMIT) {
      setHistoryExhausted(true);
      return;
    }
    const oldestMs = oldestCandleOpenTimeMs(allCandles);
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
    allCandles,
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

  const indicatorPoints = indicatorsQuery.data?.points;
  const latestEma = indicatorsQuery.data?.latest?.ema;
  const overlays: CandleChartOverlay[] = useMemo(() => {
    if (!showEma) return [];
    const keys = sortedEmaKeys(latestEma);
    return keys.map((key, i) => ({
      id: `ema-${key}`,
      title: t('detail:indicators.emaLabel', { period: key }),
      color: emaColor(key, i),
      data: indicatorPointsToEmaLine(indicatorPoints, key),
    }));
  }, [showEma, latestEma, indicatorPoints, t]);

  const chartMarkers: CandleChartMarker[] = useMemo(() => {
    const barSec = intervalToSeconds(interval);
    const maxDist = barSec > 0 ? barSec * 1.5 : 3600;
    const pumpRaw =
      showPumpMarkers
        ? pumpEventsToChartMarkers(
            mergePumpEvents(pumpsQuery.data?.events, historyPumpEvents),
            pumpThresholdPct,
          )
        : [];
    const signalRaw =
      showSignalMarkers && exchange && symbol
        ? scannerResultsToChartMarkers(scannerResultsQuery.data?.results, exchange, symbol)
        : [];
    const pumpSnapped = snapMarkersToCandleTimes(pumpRaw, chartData, { maxDistanceSec: maxDist });
    const signalSnapped = snapMarkersToCandleTimes(signalRaw, chartData, { maxDistanceSec: maxDist });
    return mergeChartMarkers(pumpSnapped, signalSnapped);
  }, [
    showPumpMarkers,
    showSignalMarkers,
    pumpsQuery.data?.events,
    historyPumpEvents,
    pumpThresholdPct,
    scannerResultsQuery.data?.results,
    exchange,
    symbol,
    chartData,
    interval,
  ]);

  const patchUrl = (patch: Partial<{ interval: string }>) => {
    setSearchParams(
      detailStateToSearchParams({
        interval: patch.interval ?? interval,
      }),
      { replace: true },
    );
  };

  const refreshAll = () => {
    void intervalsQuery.refetch();
    void tickerQuery.refetch();
    void orderBookQuery.refetch();
    void supplyQuery.refetch();
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

  const backTo = marketsBackPath(exchange);

  const headerLoading = tickerQuery.isLoading && !tickerQuery.data;
  const statsLoading =
    (tickerQuery.isLoading && !tickerQuery.data) ||
    (supplyQuery.isLoading && !supplyQuery.data);
  const seriesLoading =
    ((candlesQuery.isLoading || indicatorsQuery.isLoading) && chartData.length === 0) ||
    waitingForIntervals;
  const seriesFetching =
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
        lastPrice={tickerQuery.data?.lastPrice}
        priceChangePercent={tickerQuery.data?.priceChangePercent}
        assetName={supplyQuery.data?.name}
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
      />

      <DetailStats
        exchange={exchange}
        ticker={tickerQuery.data}
        supply={supplyQuery.data}
        isLoading={statsLoading}
        tickerError={
          tickerQuery.isError
            ? rtkErrorMessage(tickerQuery.error, {
                resource: t('detail:resource.ticker'),
              })
            : null
        }
        supplyError={
          supplyQuery.isError
            ? rtkErrorMessage(supplyQuery.error, {
                resource: t('detail:resource.supply'),
                statusMessages: {
                  404: t('detail:errors.supply404'),
                },
              })
            : null
        }
      />

      <ChartAndBook>
      <ChartCard>
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
          pumpThresholdPct={pumpThresholdPct}
          onPumpThresholdChange={setPumpThresholdPct}
          showPumpMarkers={showPumpMarkers}
          onShowPumpMarkersChange={setShowPumpMarkers}
          showSignalMarkers={showSignalMarkers}
          onShowSignalMarkersChange={setShowSignalMarkers}
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

        {candlesQuery.isError && chartData.length === 0 ? (
          <Alert
            type="error"
            showIcon
            message={t('detail:chart.loadErrorTitle')}
            description={rtkErrorMessage(candlesQuery.error, {
              resource: t('detail:resource.candles'),
            })}
          />
        ) : !seriesLoading && candlesQuery.isSuccess && chartData.length === 0 ? (
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
              isLoading={seriesLoading || waitingForIntervals}
              seriesKey={seriesKey}
              isLoadingMore={isLoadingMore}
              hasMoreHistory={hasMoreHistory}
              onNeedMoreHistory={onNeedMoreHistory}
              height={isPhone ? PHONE_CHART_HEIGHT : DESK_CHART_HEIGHT}
            />
          </>
        )}
      </ChartCard>

      <SideStack>
      <OrderBookPanel
        book={orderBookQuery.data}
        group={orderBookGroup || orderBookQuery.data?.groupSize || ''}
        onGroupChange={setOrderBookGroup}
        isLoading={orderBookQuery.isLoading && !orderBookQuery.data}
        isFetching={orderBookQuery.isFetching}
        errorMessage={
          orderBookQuery.isError
            ? rtkErrorMessage(orderBookQuery.error, {
                resource: t('detail:resource.orderBook'),
              })
            : null
        }
      />
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
            {paperPortfolio.data?.availableCash != null ? (
              <Text variant="caption" color="secondary">
                {t('detail:paperTrade.availableCash', {
                  amount: formatPrice(paperPortfolio.data.availableCash),
                })}
              </Text>
            ) : null}
            <PaperTradeForm
              lockedExchange={exchangeArg}
              lockedSymbol={symbol}
              compact
              showLotMethod={false}
              isSubmitting={placePaperState.isLoading}
              submitError={placePaperState.isError ? placePaperState.error : undefined}
              onSubmit={async (values: PaperTradeFormValues) => {
                await placePaperOrder({
                  portfolioId: paperBookId || undefined,
                  exchange: values.exchange,
                  symbol: values.symbol,
                  side: values.side,
                  type: 'market',
                  quantity: values.quantity,
                  idempotencyKey: `detail-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
                }).unwrap();
                void message.success(
                  values.side === 'buy'
                    ? t('detail:paperTrade.successBuy')
                    : t('detail:paperTrade.successSell'),
                );
              }}
            />
          </>
        )}
      </PaperTradeCard>
      </SideStack>
      </ChartAndBook>

      <IndicatorPanel
        data={indicatorsQuery.data}
        isLoading={indicatorsQuery.isLoading && !indicatorsQuery.data}
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

      {!visible ? (
        <Text variant="caption" color="secondary">
          {t('detail:pollingPaused')}
        </Text>
      ) : null}
    </PageStack>
  );
}
