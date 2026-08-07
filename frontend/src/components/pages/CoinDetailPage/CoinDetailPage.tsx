import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, message } from 'antd';
import { useParams, useSearchParams } from 'react-router-dom';
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
  rtkErrorMessage,
  useAddWatchlistItemMutation,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useGetSupplyQuery,
  useGetTicker24hQuery,
  useListDelistScheduleQuery,
  useGetWatchlistQuery,
  useLazyGetCandlesQuery,
  useLazyGetPumpEventsQuery,
  useListIntervalsQuery,
  useRemoveWatchlistItemMutation,
  type MarketExchange,
  type PumpEventDto,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
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
} from '@/config/constants';
import { ChartCard, ChartTitleRow, PageStack } from './CoinDetailPage.styles';
import { mergePumpEvents, pumpEventsToChartMarkers } from './CoinDetailPage.helpers';

/**
 * Coin detail: 24h ticker, supply, OHLCV chart (EMA overlays), RSI/EMA analysis.
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

  // New pair / interval → drop paged history and restart from the live window.
  useEffect(() => {
    historyRequestIdRef.current += 1;
    setHistoryCandles([]);
    setHistoryPumpEvents([]);
    setHistoryExhausted(false);
    setHistoryLoading(false);
  }, [exchange, symbol, interval]);

  // Threshold / marker toggle: drop history pumps (API minReturnPct or skip changed).
  // Candles stay; live pumps refetch via RTK args; re-pan reloads history markers.
  useEffect(() => {
    historyRequestIdRef.current += 1;
    setHistoryPumpEvents([]);
  }, [pumpThresholdPct, showPumpMarkers]);

  useEffect(() => {
    if (!supportedIntervals?.length) return;
    if (supportedIntervals.includes(urlState.interval)) return;
    const next = resolveInterval(urlState.interval, supportedIntervals);
    setSearchParams(detailStateToSearchParams({ interval: next }), { replace: true });
  }, [supportedIntervals, urlState.interval, setSearchParams]);

  const skipSeries = skip || waitingForIntervals;
  const supplyAsset = toSupplyAsset(symbol);
  const seriesKey = `${exchange}|${symbol}|${interval}`;

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
  const watched = Boolean(
    watchlistQuery.data?.items?.some(
      (it) => it.exchange === exchange && it.symbol === symbol,
    ),
  );

  const tickerQuery = useGetTicker24hQuery(
    { exchange: exchangeArg, symbol },
    {
      skip,
      pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
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
    if (!showPumpMarkers) return [];
    // Live (right edge) + history-page pumps (older bars loaded while panning left).
    const allEvents = mergePumpEvents(pumpsQuery.data?.events, historyPumpEvents);
    const raw = pumpEventsToChartMarkers(allEvents, pumpThresholdPct);
    const barSec = intervalToSeconds(interval);
    const maxDist = barSec > 0 ? barSec * 1.5 : 3600;
    return snapMarkersToCandleTimes(raw, chartData, { maxDistanceSec: maxDist });
  }, [
    showPumpMarkers,
    pumpsQuery.data?.events,
    historyPumpEvents,
    pumpThresholdPct,
    chartData,
    interval,
  ]);  const patchUrl = (patch: Partial<{ interval: string }>) => {
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
    void supplyQuery.refetch();
    void candlesQuery.refetch();
    void indicatorsQuery.refetch();
    if (showPumpMarkers) {
      void pumpsQuery.refetch();
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
          onRefresh={refreshAll}
          isFetching={
            seriesFetching ||
            historyLoading ||
            (showPumpMarkers && pumpsQuery.isFetching)
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
              height={360}
            />
          </>
        )}
      </ChartCard>

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
