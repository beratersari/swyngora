import { useCallback, useMemo, useState } from 'react';
import { useIsFocused, useNavigation, useRoute } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { RouteProp } from '@react-navigation/native';
import type { CandleChartOverlay } from '@/components/organisms/candle-chart';
import {
  rtkErrorMessage,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useGetSupplyQuery,
  useGetTicker24hQuery,
  useListIntervalsQuery,
  type MarketExchange,
} from '@/libs/api';
import { useAppStateActive } from '@/libs/hooks';
import {
  apiCandlesToChart,
  buildDetailPumpQuery,
  changeTone,
  emaColor,
  formatChangePercent,
  formatCompactUsd,
  formatPrice,
  formatPumpEventTime,
  formatPumpReturnPct,
  formatSupplyNum,
  formatTradeCount,
  formatVolumeRatio,
  indicatorPointsToEmaLine,
  indicatorPointsToRsi,
  isMarketExchange,
  pumpModeLabel,
  pumpReturnTone,
  resolveInterval,
  sortedEmaKeys,
} from '@/libs/utils';
import {
  DEFAULT_PUMP_DETAIL_DIRECTION,
  DEFAULT_PUMP_DETAIL_LOOKBACK_HOURS,
  DEFAULT_PUMP_DETAIL_MAX_EVENTS,
  DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT,
  PUMP_DISCLAIMER,
} from '@/config/pumpConstants';
import type { MarketsStackParamList } from '../../navigation';
import { useOptionalWatchlist } from '@/modules/watchlist';
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
  const watchlist = useOptionalWatchlist();
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

  const pumpQueryArgs = useMemo(
    () =>
      buildDetailPumpQuery({
        exchange,
        symbol,
        interval: resolvedInterval,
        lookbackHours: DEFAULT_PUMP_DETAIL_LOOKBACK_HOURS,
        minReturnPct: DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT,
        direction: DEFAULT_PUMP_DETAIL_DIRECTION,
        maxEvents: DEFAULT_PUMP_DETAIL_MAX_EVENTS,
      }),
    [exchange, symbol, resolvedInterval],
  );

  const pumpQuery = useGetPumpEventsQuery(pumpQueryArgs, {
    skip: skip || !symbol,
    refetchOnFocus: false,
    pollingInterval: 0,
  });

  const pumpEventRows = useMemo(() => {
    const events = pumpQuery.data?.events ?? [];
    return events.map((e, i) => {
      const vol = formatVolumeRatio(e.volumeRatio);
      const mode = pumpModeLabel(e.mode);
      return {
        id: `${e.openTime ?? i}-${e.returnPct ?? i}`,
        returnLabel: formatPumpReturnPct(e.returnPct),
        returnTone: pumpReturnTone(e.returnPct),
        timeLabel: formatPumpEventTime(e.openTime),
        metaLabel: [mode, vol].filter(Boolean).join(' · '),
      };
    });
  }, [pumpQuery.data?.events]);

  const pumpEventsSubtitle = useMemo(() => {
    if (!pumpQuery.data) return null;
    const parts = [
      `≥${pumpQuery.data.minReturnPct ?? DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT}%`,
      pumpQuery.data.direction ?? DEFAULT_PUMP_DETAIL_DIRECTION,
      pumpQuery.data.interval ?? resolvedInterval,
    ];
    if (pumpQuery.data.eventCount != null) {
      parts.push(`${pumpQuery.data.eventCount} events`);
    }
    return parts.join(' · ');
  }, [pumpQuery.data, resolvedInterval]);

  const onRetry = useCallback(() => {
    void tickerQuery.refetch();
    void supplyQuery.refetch();
    void candlesQuery.refetch();
    void indicatorsQuery.refetch();
    void intervalsQuery.refetch();
    void pumpQuery.refetch();
  }, [
    tickerQuery,
    supplyQuery,
    candlesQuery,
    indicatorsQuery,
    intervalsQuery,
    pumpQuery,
  ]);

  const onSelectInterval = useCallback((next: string) => {
    setInterval(next);
  }, []);

  const watched = watchlist?.isWatched(exchange, symbol) ?? false;
  const onStarPress = useCallback(() => {
    void watchlist?.toggle(exchange, symbol);
  }, [watchlist, exchange, symbol]);

  return {
    symbol,
    exchange,
    lastPriceLabel: formatPrice(ticker?.lastPrice),
    changePercentLabel: formatChangePercent(ticker?.priceChangePercent),
    changeTone: changeTone(ticker?.priceChangePercent),
    headerLoading: tickerQuery.isLoading,
    watched,
    onStarPress,
    actionError: watchlist?.actionError ?? null,

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

    pumpEventRows,
    pumpEventsLoading: pumpQuery.isLoading || pumpQuery.isFetching,
    pumpEventsError: pumpQuery.isError
      ? rtkErrorMessage(pumpQuery.error, { resource: 'pump events' })
      : null,
    pumpEventsSubtitle,
    pumpDisclaimer: pumpQuery.data?.note ?? PUMP_DISCLAIMER,

    onBack,
    onRetry,
  };
}
