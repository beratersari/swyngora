import { useEffect, useMemo, useState } from 'react';
import { Alert } from 'antd';
import { useParams, useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
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
  marketsBackPath,
  parseDetailSearchParams,
  parseExchangeParam,
  parseSymbolParam,
  resolveInterval,
  sortedEmaKeys,
  toSupplyAsset,
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
  const { t } = useTranslation(['detail', 'common']);
  const { exchange: exchangeParam, symbol: symbolParam } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();
  const visible = useDocumentVisible();

  const exchange = parseExchangeParam(exchangeParam);
  const symbol = parseSymbolParam(symbolParam);
  const urlState = useMemo(() => parseDetailSearchParams(searchParams), [searchParams]);

  const [showEma, setShowEma] = useState(true);

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

  const skipSeries = skip || waitingForIntervals;
  const supplyAsset = toSupplyAsset(symbol);

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
  // Keep supply fields in loading state until supply resolves (do not treat
  // "no maxSupply yet" as open/∞ while ticker alone has arrived).
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
            {t('detail:chart.meta', { interval, bars: limit })}
          </Text>
        </ChartTitleRow>

        <DetailChartToolbar
          intervals={supportedIntervals ?? []}
          interval={interval}
          limit={limit}
          intervalsLoading={intervalsQuery.isLoading && !supportedIntervals?.length}
          onIntervalChange={(iv) => patchUrl({ interval: iv })}
          onLimitChange={(n) => patchUrl({ limit: n })}
          onRefresh={refreshAll}
          isFetching={seriesFetching}
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
              isLoading={seriesLoading || waitingForIntervals}
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
