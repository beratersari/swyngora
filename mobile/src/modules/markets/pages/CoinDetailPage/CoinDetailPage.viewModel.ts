import { useCallback, useMemo, useState } from 'react';
import { useIsFocused, useNavigation, useRoute } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { RouteProp } from '@react-navigation/native';
import type { CandleChartOverlay } from '@/components/organisms/CandleChart';
import {
  rtkErrorMessage,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useGetSupplyQuery,
  useGetTicker24hQuery,
  useListIntervalsQuery,
  type MarketExchange,
} from '@/libs/api';
import { useAppStateActive } from '@/libs/hooks';
import {
  apiCandlesToChart,
  changeTone,
  emaColor,
  formatChangePercent,
  formatCompactUsd,
  formatPrice,
  formatSupplyNum,
  formatTradeCount,
  indicatorPointsToEmaLine,
  indicatorPointsToRsi,
  isMarketExchange,
  resolveInterval,
  sortedEmaKeys,
} from '@/libs/utils';
import type { MarketsStackParamList } from '../../navigation';
import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
  DEFAULT_DETAIL_INTERVAL,
  DEFAULT_EMA_PERIODS,
  DEFAULT_RSI_PERIOD,
  DETAIL_SERIES_POLL_MS,
  DETAIL_TICKER_POLL_MS,
} from './CoinDetailPage.constants';
import type { CoinDetailPageViewModel } from './CoinDetailPage.types';

export function useCoinDetailPageViewModel(): CoinDetailPageViewModel {
  const navigation =
    useNavigation<NativeStackNavigationProp<MarketsStackParamList>>();
  const route = useRoute<RouteProp<MarketsStackParamList, 'CoinDetail'>>();
  const active = useAppStateActive();
  const focused = useIsFocused();
  const polling = active && focused;

  const rawExchange = route.params?.exchange ?? 'binance';
  const exchange: MarketExchange = isMarketExchange(rawExchange)
    ? rawExchange
    : 'binance';
  const symbol = (route.params?.symbol ?? '').toUpperCase();

  const [interval, setInterval] = useState(DEFAULT_DETAIL_INTERVAL);
  const [showEma, setShowEma] = useState(true);

  const intervalsQuery = useListIntervalsQuery({ exchange });
  const supported = intervalsQuery.data?.intervals;
  const resolvedInterval = resolveInterval(
    interval,
    supported,
    DEFAULT_DETAIL_INTERVAL,
  );

  const skip = !symbol;
  const skipSeries = skip || !(supported?.length);

  const tickerQuery = useGetTicker24hQuery(
    { exchange, symbol },
    {
      skip,
      pollingInterval: polling ? DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );

  const supplyQuery = useGetSupplyQuery(
    { symbol },
    {
      skip,
      pollingInterval: polling ? DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );

  const candlesQuery = useGetCandlesQuery(
    {
      exchange,
      symbol,
      interval: resolvedInterval,
      limit: DEFAULT_DETAIL_CANDLE_LIMIT,
    },
    {
      skip: skipSeries,
      pollingInterval: polling ? DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );

  const indicatorsQuery = useGetIndicatorsQuery(
    {
      exchange,
      symbol,
      interval: resolvedInterval,
      limit: DEFAULT_DETAIL_CANDLE_LIMIT,
      rsiPeriod: DEFAULT_RSI_PERIOD,
      emaPeriods: DEFAULT_EMA_PERIODS,
    },
    {
      skip: skipSeries,
      pollingInterval: polling ? DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );

  const candles = useMemo(() => {
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

  const candleOverlays: CandleChartOverlay[] = useMemo(() => {
    if (!showEma) return [];
    const keys = sortedEmaKeys(indicatorsQuery.data?.latest?.ema);
    return keys.map((key, i) => ({
      id: `ema-${key}`,
      title: `EMA ${key}`,
      color: emaColor(key, i),
      data: indicatorPointsToEmaLine(indicatorsQuery.data?.points, key),
    }));
  }, [showEma, indicatorsQuery.data]);

  const rsiPoints = useMemo(
    () => indicatorPointsToRsi(indicatorsQuery.data?.points),
    [indicatorsQuery.data?.points],
  );

  const latestRsi = indicatorsQuery.data?.latest?.rsi ?? null;

  const emaLatestLabels = useMemo(() => {
    const ema = indicatorsQuery.data?.latest?.ema;
    if (!ema) return [];
    return sortedEmaKeys(ema).map((k) => `EMA ${k}: ${formatPrice(ema[k])}`);
  }, [indicatorsQuery.data?.latest?.ema]);

  const ticker = tickerQuery.data;
  const supply = supplyQuery.data;

  const statsItems = useMemo(
    () => [
      { label: 'Open', value: formatPrice(ticker?.openPrice) },
      { label: 'High 24h', value: formatPrice(ticker?.highPrice) },
      { label: 'Low 24h', value: formatPrice(ticker?.lowPrice) },
      { label: 'Base vol', value: formatCompactUsd(ticker?.volume) },
      { label: 'Quote vol', value: formatCompactUsd(ticker?.quoteVolume) },
      {
        label: 'Trades',
        value: formatTradeCount(ticker?.tradeCount, exchange),
      },
      { label: 'Circ. supply', value: formatSupplyNum(supply?.circulatingSupply) },
      { label: 'Total supply', value: formatSupplyNum(supply?.totalSupply) },
      {
        label: 'Circ. mcap',
        value: formatCompactUsd(
          supply?.circulatingSupply != null && ticker?.lastPrice
            ? Number(ticker.lastPrice) * Number(supply.circulatingSupply)
            : null,
        ),
      },
    ],
    [ticker, supply, exchange],
  );

  const supplyError =
    supplyQuery.isError && (supplyQuery.error as { status?: number })?.status !== 404
      ? rtkErrorMessage(supplyQuery.error, { resource: 'supply' })
      : supplyQuery.isError
        ? 'Supply not available for this asset'
        : null;

  const onBack = useCallback(() => {
    navigation.goBack();
  }, [navigation]);

  const onRetry = useCallback(() => {
    void tickerQuery.refetch();
    void supplyQuery.refetch();
    void candlesQuery.refetch();
    void indicatorsQuery.refetch();
    void intervalsQuery.refetch();
  }, [tickerQuery, supplyQuery, candlesQuery, indicatorsQuery, intervalsQuery]);

  const onSelectInterval = useCallback((next: string) => {
    setInterval(next);
  }, []);

  return {
    symbol,
    exchange,
    lastPriceLabel: formatPrice(ticker?.lastPrice),
    changePercentLabel: formatChangePercent(ticker?.priceChangePercent),
    changeTone: changeTone(ticker?.priceChangePercent),
    headerLoading: tickerQuery.isLoading,

    statsItems,
    statsLoading: tickerQuery.isLoading || supplyQuery.isLoading,
    tickerError: tickerQuery.isError
      ? rtkErrorMessage(tickerQuery.error, { resource: 'ticker' })
      : null,
    supplyError,

    intervals: supported ?? [],
    intervalsLoading: intervalsQuery.isLoading,
    interval: resolvedInterval,
    onSelectInterval,
    showEma,
    onToggleEma: () => setShowEma((v) => !v),

    candles,
    candleOverlays,
    candlesLoading: candlesQuery.isLoading || candlesQuery.isFetching,
    candlesError: candlesQuery.isError
      ? rtkErrorMessage(candlesQuery.error, { resource: 'candles' })
      : null,

    rsiPoints,
    latestRsi: latestRsi === undefined ? null : latestRsi,
    indicatorsLoading: indicatorsQuery.isLoading || indicatorsQuery.isFetching,
    indicatorsError: indicatorsQuery.isError
      ? rtkErrorMessage(indicatorsQuery.error, { resource: 'indicators' })
      : null,
    emaLatestLabels,

    onBack,
    onRetry,
  };
}
