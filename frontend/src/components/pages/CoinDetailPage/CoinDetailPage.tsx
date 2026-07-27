import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert } from 'antd';
import { useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { CandleChartHost } from '@/components/molecules/CandleChartHost';
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
  useGetWatchlistQuery,
  useListIntervalsQuery,
  useRemoveWatchlistItemMutation,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import {
  apiCandlesToChart,
  detailStateToSearchParams,
  indicatorPointsToEmaLine,
  marketsBackPath,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  sortedEmaKeys,
  toSupplyAsset,
} from '@/libs/utils';
import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
  DEFAULT_DETAIL_PUMP_THRESHOLD_PCT,
  DEFAULT_DETAIL_SERIES_POLL_MS,
  DEFAULT_DETAIL_TICKER_POLL_MS,
  DEFAULT_EMA_PERIODS,
  DEFAULT_RSI_PERIOD,
  DETAIL_CANDLE_MAX_LIMIT,
  DETAIL_CANDLE_PAGE_SIZE,
} from '@/config/constants';
import { palette } from '@/styles/tokens';
import { ChartCard, ChartTitleRow, PageStack } from './CoinDetailPage.styles';

/**
 * Coin detail: 24h ticker, supply, OHLCV chart (EMA overlays), RSI/EMA analysis.
 * Candle history grows automatically as the user pans left on the chart.
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
  /** Progressive bar count — not a user control; grows when chart scrolls left. */
  const [candleLimit, setCandleLimit] = useState(DEFAULT_DETAIL_CANDLE_LIMIT);
  /** |return %| threshold for pump/dump markers on the chart. */
  const [pumpThresholdPct, setPumpThresholdPct] = useState(
    DEFAULT_DETAIL_PUMP_THRESHOLD_PCT,
  );
  const [showPumpMarkers, setShowPumpMarkers] = useState(true);

  const skip = !symbol;
  const intervalsQuery = useListIntervalsQuery({ exchange });
  const supportedIntervals = intervalsQuery.data?.intervals;
  // Only block series on the first intervals load. On error, fall through with
  // resolveInterval defaults so candles are not stuck on skeleton forever.
  const waitingForIntervals =
    !skip &&
    !supportedIntervals?.length &&
    intervalsQuery.isLoading &&
    !intervalsQuery.isError;

  const interval = resolveInterval(urlState.interval, supportedIntervals);

  // New pair / interval → start from a fresh window of bars.
  useEffect(() => {
    setCandleLimit(DEFAULT_DETAIL_CANDLE_LIMIT);
  }, [exchange, symbol, interval]);

  useEffect(() => {
    if (!supportedIntervals?.length) return;
    if (supportedIntervals.includes(urlState.interval)) return;
    const next = resolveInterval(urlState.interval, supportedIntervals);
    setSearchParams(detailStateToSearchParams({ interval: next }), { replace: true });
  }, [supportedIntervals, urlState.interval, setSearchParams]);

  const skipSeries = skip || waitingForIntervals;
  const supplyAsset = toSupplyAsset(symbol);
  const seriesKey = `${exchange}|${symbol}|${interval}`;

  const watchlistQuery = useGetWatchlistQuery();
  const [addWatch, addWatchState] = useAddWatchlistItemMutation();
  const [removeWatch, removeWatchState] = useRemoveWatchlistItemMutation();
  const watched = Boolean(
    watchlistQuery.data?.items?.some(
      (it) => it.exchange === exchange && it.symbol === symbol,
    ),
  );

  // Use the same bar count as the chart so event times land inside loaded candles.
  const pumpsQuery = useGetPumpEventsQuery(
    {
      exchange,
      symbol,
      interval,
      limit: candleLimit,
      minReturnPct: pumpThresholdPct,
      direction: 'both',
      maxEvents: 40,
    },
    { skip: skipSeries || !showPumpMarkers },
  );

  const tickerQuery = useGetTicker24hQuery(
    { exchange, symbol },
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

  const candlesQuery = useGetCandlesQuery(
    { exchange, symbol, interval, limit: candleLimit },
    {
      skip: skipSeries,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const indicatorsQuery = useGetIndicatorsQuery(
    {
      exchange,
      symbol,
      interval,
      limit: candleLimit,
      rsiPeriod: DEFAULT_RSI_PERIOD,
      emaPeriods: DEFAULT_EMA_PERIODS,
    },
    {
      skip: skipSeries,
      pollingInterval: visible ? DEFAULT_DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const chartData = useMemo(() => {
    const raw = candlesQuery.data?.candles ?? [];
    return apiCandlesToChart(
      raw.flatMap((c) =>
        c.openTime && c.open && c.high && c.low && c.close
          ? [
              {
                openTime: c.openTime,
                open: c.open,
                high: c.high,
                low: c.low,
                close: c.close,
                volume: c.volume ?? '0',
                closeTime: c.closeTime,
              },
            ]
          : [],
      ),
    );
  }, [candlesQuery.data?.candles]);

  const returnedBars = candlesQuery.data?.candles?.length ?? 0;
  // If we asked for N and got fewer, the venue has no older history in this window.
  const hasMoreHistory =
    candleLimit < DETAIL_CANDLE_MAX_LIMIT &&
    returnedBars >= candleLimit &&
    candlesQuery.isSuccess;

  const isLoadingMore =
    candlesQuery.isFetching && chartData.length > 0 && candleLimit > DEFAULT_DETAIL_CANDLE_LIMIT;

  const onNeedMoreHistory = useCallback(() => {
    setCandleLimit((prev) => {
      if (prev >= DETAIL_CANDLE_MAX_LIMIT) return prev;
      return Math.min(DETAIL_CANDLE_MAX_LIMIT, prev + DETAIL_CANDLE_PAGE_SIZE);
    });
  }, []);

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
    // API events have signed returnPct (no direction field on the DTO).
    const events =
      (pumpsQuery.data as { events?: { openTime?: string; returnPct?: number }[] } | undefined)
        ?.events ?? [];
    const out: CandleChartMarker[] = [];
    for (const ev of events) {
      if (!ev.openTime) continue;
      const ms = Date.parse(ev.openTime);
      if (!Number.isFinite(ms)) continue;
      const ret = Number(ev.returnPct);
      // Extra client filter if API returns near-threshold noise.
      if (Number.isFinite(ret) && Math.abs(ret) < pumpThresholdPct) continue;
      const up = !Number.isFinite(ret) || ret >= 0;
      out.push({
        time: Math.floor(ms / 1000),
        position: up ? 'belowBar' : 'aboveBar',
        color: up ? palette.mountainMeadow : '#E07A7A',
        shape: up ? 'arrowUp' : 'arrowDown',
        text: up ? `↑${Number.isFinite(ret) ? ret.toFixed(1) : ''}` : `↓${Number.isFinite(ret) ? Math.abs(ret).toFixed(1) : ''}`,
      });
    }
    return out;
  }, [pumpsQuery.data, showPumpMarkers, pumpThresholdPct]);

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
    void supplyQuery.refetch();
    void candlesQuery.refetch();
    void indicatorsQuery.refetch();
  };

  const backTo = marketsBackPath(exchange);

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
        onToggleWatch={() => {
          if (watched) {
            void removeWatch({ exchange, symbol });
          } else {
            void addWatch({ exchange, symbol });
          }
        }}
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
              bars: chartData.length || candleLimit,
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
          isFetching={seriesFetching || (showPumpMarkers && pumpsQuery.isFetching)}
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
