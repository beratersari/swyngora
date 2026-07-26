import { useEffect, useMemo, useState } from 'react';
import { Alert } from 'antd';
import { useParams, useSearchParams } from 'react-router-dom';
import { Text } from '@/components/atoms/Text';
import { CandleChartHost } from '@/components/molecules/CandleChartHost';
import type { CandleChartOverlay } from '@/components/molecules/CandleChartHost/CandleChartHost.types';
import { DetailChartToolbar } from '@/components/organisms/DetailChartToolbar';
import { DetailHeader } from '@/components/organisms/DetailHeader';
import { DetailStats } from '@/components/organisms/DetailStats';
import { IndicatorPanel, emaColor } from '@/components/organisms/IndicatorPanel';
import {
  rtkErrorMessage,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useGetSupplyQuery,
  useGetTicker24hQuery,
  useListIntervalsQuery,
} from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import {
  apiCandlesToChart,
  detailStateToSearchParams,
  indicatorPointsToEmaLine,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  sortedEmaKeys,
} from '@/libs/utils';
import {
  DEFAULT_DETAIL_SERIES_POLL_MS,
  DEFAULT_DETAIL_TICKER_POLL_MS,
  DEFAULT_EMA_PERIODS,
  DEFAULT_RSI_PERIOD,
} from '@/config/constants';
import { ChartCard, ChartTitleRow, PageStack } from './CoinDetailPage.styles';

/**
 * Coin detail: 24h ticker, supply, OHLCV chart (EMA overlays), RSI/EMA analysis.
 * Route: /markets/:exchange/:symbol
 */
export function CoinDetailPage() {
  const { exchange: exchangeParam, symbol: symbolParam } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();

  const exchange = parseExchangeParam(exchangeParam);
  const symbol = parseSymbolParam(symbolParam);
  const urlState = useMemo(() => parseDetailSearchParams(searchParams), [searchParams]);

  const [showEma, setShowEma] = useState(true);

  const intervalsQuery = useListIntervalsQuery({ exchange });
  const supportedIntervals = intervalsQuery.data?.intervals;
  const intervalsReady = Boolean(supportedIntervals?.length);

  // Clamp interval to venue-supported set once intervals load (DET-A §5.4)
  const interval = resolveInterval(urlState.interval, supportedIntervals);
  const limit = urlState.limit;

  useEffect(() => {
    if (!supportedIntervals?.length) return;
    if (supportedIntervals.includes(urlState.interval)) return;
    const next = resolveInterval(urlState.interval, supportedIntervals);
    setSearchParams(
      detailStateToSearchParams({ interval: next, limit: urlState.limit }),
      { replace: true },
    );
  }, [supportedIntervals, urlState.interval, urlState.limit, setSearchParams]);

  const skip = !symbol;
  /** Avoid candles/indicators 400 on venue-invalid interval before A1 resolves */
  const skipSeries = skip || !intervalsReady;

  const tickerQuery = useGetTicker24hQuery(
    { exchange, symbol },
    {
      skip,
      pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const supplyQuery = useGetSupplyQuery(
    { symbol },
    {
      skip,
      pollingInterval: visible ? DEFAULT_DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );

  const candlesQuery = useGetCandlesQuery(
    { exchange, symbol, interval, limit },
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
      limit,
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
    // DET-A: OHLCV mapping requires openTime + OHLC; volume optional for chart bars
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

  const overlays: CandleChartOverlay[] = useMemo(() => {
    if (!showEma) return [];
    const keys = sortedEmaKeys(indicatorsQuery.data?.latest?.ema);
    return keys.map((key, i) => ({
      id: `ema-${key}`,
      title: `EMA ${key}`,
      color: emaColor(key, i),
      data: indicatorPointsToEmaLine(indicatorsQuery.data?.points, key),
    }));
  }, [showEma, indicatorsQuery.data]);

  const patchUrl = (patch: Partial<{ interval: string; limit: number }>) => {
    setSearchParams(
      detailStateToSearchParams({
        interval: patch.interval ?? interval,
        limit: patch.limit ?? limit,
      }),
      { replace: true },
    );
  };

  const refreshAll = () => {
    void tickerQuery.refetch();
    void supplyQuery.refetch();
    void candlesQuery.refetch();
    void indicatorsQuery.refetch();
  };

  if (!symbol) {
    return (
      <PageStack>
        <Alert type="error" showIcon message="Missing symbol" description="No trading pair in the URL." />
      </PageStack>
    );
  }

  const headerLoading = tickerQuery.isLoading && !tickerQuery.data;
  const statsLoading = (tickerQuery.isLoading || supplyQuery.isLoading) && !tickerQuery.data;
  const seriesLoading =
    (candlesQuery.isLoading || indicatorsQuery.isLoading) && chartData.length === 0;
  const seriesFetching =
    candlesQuery.isFetching || indicatorsQuery.isFetching || tickerQuery.isFetching;

  return (
    <PageStack>
      <DetailHeader
        symbol={symbol}
        exchange={exchange}
        lastPrice={tickerQuery.data?.lastPrice}
        priceChangePercent={tickerQuery.data?.priceChangePercent}
        assetName={supplyQuery.data?.name}
        isLoading={headerLoading}
      />

      <DetailStats
        exchange={exchange}
        ticker={tickerQuery.data}
        supply={supplyQuery.data}
        isLoading={statsLoading}
        tickerError={
          tickerQuery.isError
            ? rtkErrorMessage(tickerQuery.error, { resource: '24h ticker' })
            : null
        }
        supplyError={
          supplyQuery.isError
            ? rtkErrorMessage(supplyQuery.error, {
                resource: 'supply',
                statusMessages: {
                  404: 'Supply snapshot not available for this asset (Binance marketing list coverage).',
                },
              })
            : null
        }
      />

      <ChartCard>
        <ChartTitleRow>
          <Text variant="h4" color="cream">
            Price chart
          </Text>
          <Text variant="caption" color="steel">
            {interval} · {limit} bars · EMA overlay optional
          </Text>
        </ChartTitleRow>

        <DetailChartToolbar
          intervals={supportedIntervals ?? []}
          interval={interval}
          limit={limit}
          intervalsLoading={intervalsQuery.isLoading}
          onIntervalChange={(iv) => patchUrl({ interval: iv })}
          onLimitChange={(n) => patchUrl({ limit: n })}
          onRefresh={refreshAll}
          isFetching={seriesFetching}
        />

        {candlesQuery.isError ? (
          <Alert
            type="error"
            showIcon
            message="Could not load candles"
            description={rtkErrorMessage(candlesQuery.error, { resource: 'candles' })}
          />
        ) : !seriesLoading && candlesQuery.isSuccess && chartData.length === 0 ? (
          <Alert type="info" showIcon message="No candle data" description="Empty series for this interval." />
        ) : (
          <CandleChartHost
            data={chartData}
            overlays={overlays}
            isLoading={seriesLoading || skipSeries}
            height={360}
          />
        )}
      </ChartCard>

      <IndicatorPanel
        data={indicatorsQuery.data}
        isLoading={indicatorsQuery.isLoading && !indicatorsQuery.data}
        errorMessage={
          indicatorsQuery.isError
            ? rtkErrorMessage(indicatorsQuery.error, { resource: 'indicators' })
            : null
        }
        showEmaOnChart={showEma}
        onToggleEma={setShowEma}
      />

      {!visible ? (
        <Text variant="caption" color="secondary">
          Live refresh paused while this tab is hidden.
        </Text>
      ) : null}
    </PageStack>
  );
}
