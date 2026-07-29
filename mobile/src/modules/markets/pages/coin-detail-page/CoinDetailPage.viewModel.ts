import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useIsFocused, useNavigation, useRoute } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { RouteProp } from '@react-navigation/native';
import { useTranslation } from 'react-i18next';
import { AiScreens } from '@/modules/ai';
import type { CandleChartOverlay } from '@/components/organisms/candle-chart';
import {
  rtkErrorMessage,
  useGetCandlesQuery,
  useGetIndicatorsQuery,
  useGetPumpEventsQuery,
  useGetSupplyQuery,
  useGetTicker24hQuery,
  useLazyGetCandlesQuery,
  useListIntervalsQuery,
  type MarketExchange,
} from '@/libs/api';
import { useAppStateActive } from '@/libs/hooks';
import {
  apiCandlesToChart,
  buildDetailPumpQuery,
  changeTone,
  emaColor,
  endTimeBeforeOldestCandle,
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
  cheapestExchangeId,
  isMarketExchange,
  mergeChartCandles,
  pumpEventsToChartMarkers,
  pumpEventsToMarginLines,
  pumpModeLabel,
  pumpReturnTone,
  resolveInterval,
  sortedEmaKeys,
  type ChartCandle,
} from '@/libs/utils';
import {
  DEFAULT_PUMP_DETAIL_DIRECTION,
  DEFAULT_PUMP_DETAIL_LOOKBACK_HOURS,
  DEFAULT_PUMP_DETAIL_MAX_EVENTS,
  DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT,
  PUMP_DISCLAIMER,
} from '@/config/pumpConstants';
import { MarketsScreens, type MarketsStackParamList } from '../../navigation';
import { useOptionalWatchlist } from '@/modules/watchlist';
import {
  DEFAULT_DETAIL_CANDLE_LIMIT,
  DEFAULT_DETAIL_INTERVAL,
  DEFAULT_EMA_PERIODS,
  DEFAULT_RSI_PERIOD,
  DETAIL_HISTORY_EDGE_BARS,
  DETAIL_HISTORY_PAGE_SIZE,
  DETAIL_MAX_CANDLES,
  DETAIL_SERIES_POLL_MS,
  DETAIL_TICKER_POLL_MS,
} from './CoinDetailPage.constants';
import type { CoinDetailPageViewModel } from './CoinDetailPage.types';
import { useCrossExchangeRows } from './useCrossExchangeRows';

function mapApiCandlesToChart(
  raw: {
    openTime?: string;
    open?: string;
    high?: string;
    low?: string;
    close?: string;
    volume?: string;
    closeTime?: string;
  }[],
): ChartCandle[] {
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
}

export function useCoinDetailPageViewModel(): CoinDetailPageViewModel {
  const { t } = useTranslation(['detail', 'common', 'pumps']);
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
  const [showPumps, setShowPumps] = useState(true);
  const [showPumpMargin, setShowPumpMargin] = useState(false);
  /** Older bars loaded by panning left (prepended to the latest RTK window). */
  const [olderCandles, setOlderCandles] = useState<ChartCandle[]>([]);
  const [isLoadingOlder, setIsLoadingOlder] = useState(false);
  const [historyExhausted, setHistoryExhausted] = useState(false);
  const loadingOlderRef = useRef(false);
  /** Prevents re-fetching the same endTime page while state catches up. */
  const lastHistoryEndTimeRef = useRef<string | null>(null);
  /** Oldest open time (seconds) currently in the merged series. */
  const oldestCandleTimeRef = useRef<number | null>(null);

  const intervalsQuery = useListIntervalsQuery({ exchange });
  const supported = intervalsQuery.data?.intervals;
  const resolvedInterval = resolveInterval(
    interval,
    supported,
    DEFAULT_DETAIL_INTERVAL,
  );

  const seriesKey = `${exchange}|${symbol}|${resolvedInterval}`;

  const skip = !symbol;
  const skipSeries = skip || !(supported?.length);

  // Reset accumulated history when the series identity changes.
  useEffect(() => {
    setOlderCandles([]);
    setHistoryExhausted(false);
    setIsLoadingOlder(false);
    loadingOlderRef.current = false;
    lastHistoryEndTimeRef.current = null;
    oldestCandleTimeRef.current = null;
  }, [seriesKey]);

  const tickerQuery = useGetTicker24hQuery(
    { exchange, symbol },
    {
      skip,
      pollingInterval: polling ? DETAIL_TICKER_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );

  const crossExchange = useCrossExchangeRows({
    sourceExchange: exchange,
    sourceSymbol: symbol,
    sourceTicker: tickerQuery.data,
    sourceLoading: tickerQuery.isLoading || tickerQuery.isFetching,
    sourceError: tickerQuery.error,
    sourceIsError: tickerQuery.isError,
    polling,
    skip,
  });

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

  const [fetchOlderCandles] = useLazyGetCandlesQuery();

  const latestCandles = useMemo(
    () => mapApiCandlesToChart(candlesQuery.data?.candles ?? []),
    [candlesQuery.data?.candles],
  );

  const candles = useMemo(
    () => mergeChartCandles(olderCandles, latestCandles).slice(-DETAIL_MAX_CANDLES),
    [olderCandles, latestCandles],
  );

  useEffect(() => {
    const nextOldest = candles[0]?.time ?? null;
    if (nextOldest !== oldestCandleTimeRef.current) {
      oldestCandleTimeRef.current = nextOldest;
      // Oldest advanced (or reset) — allow the next history page.
      lastHistoryEndTimeRef.current = null;
    }
  }, [candles]);

  const seriesLimit = useMemo(
    () =>
      Math.min(
        DETAIL_MAX_CANDLES,
        Math.max(DEFAULT_DETAIL_CANDLE_LIMIT, candles.length || DEFAULT_DETAIL_CANDLE_LIMIT),
      ),
    [candles.length],
  );

  const indicatorsQuery = useGetIndicatorsQuery(
    {
      exchange,
      symbol,
      interval: resolvedInterval,
      limit: seriesLimit,
      rsiPeriod: DEFAULT_RSI_PERIOD,
      emaPeriods: DEFAULT_EMA_PERIODS,
    },
    {
      skip: skipSeries,
      pollingInterval: polling ? DETAIL_SERIES_POLL_MS : 0,
      refetchOnFocus: false,
    },
  );

  const onRequestOlderHistory = useCallback(async () => {
    if (
      skipSeries ||
      loadingOlderRef.current ||
      historyExhausted ||
      candles.length === 0 ||
      candles.length >= DETAIL_MAX_CANDLES
    ) {
      return;
    }
    const oldestTime = oldestCandleTimeRef.current ?? candles[0]?.time;
    if (oldestTime == null || !Number.isFinite(oldestTime)) return;
    const oldestExisting = candles.find((c) => c.time === oldestTime) ?? candles[0];
    const endTime = endTimeBeforeOldestCandle(
      oldestExisting ?? { time: oldestTime, open: 0, high: 0, low: 0, close: 0 },
    );
    if (!endTime) return;
    // Same page already in flight or just requested — wait for merge.
    if (lastHistoryEndTimeRef.current === endTime) return;

    loadingOlderRef.current = true;
    lastHistoryEndTimeRef.current = endTime;
    setIsLoadingOlder(true);
    try {
      const data = await fetchOlderCandles({
        exchange,
        symbol,
        interval: resolvedInterval,
        limit: DETAIL_HISTORY_PAGE_SIZE,
        endTime,
      }).unwrap();
      const mapped = mapApiCandlesToChart(data.candles ?? []);
      // Keep only bars strictly older than what we already show.
      const strictlyOlder = mapped.filter((c) => c.time < oldestTime);
      if (strictlyOlder.length === 0) {
        setHistoryExhausted(true);
        return;
      }
      setOlderCandles((prev) => {
        const merged = mergeChartCandles(strictlyOlder, prev);
        // Cap total with room for the live window.
        return merged.slice(
          Math.max(0, merged.length - (DETAIL_MAX_CANDLES - DEFAULT_DETAIL_CANDLE_LIMIT)),
        );
      });
      // lastHistoryEndTimeRef clears when oldestCandleTimeRef advances (effect above).
    } catch {
      // Allow retry on the same endTime after a failure.
      lastHistoryEndTimeRef.current = null;
    } finally {
      loadingOlderRef.current = false;
      setIsLoadingOlder(false);
    }
  }, [
    skipSeries,
    historyExhausted,
    candles,
    fetchOlderCandles,
    exchange,
    symbol,
    resolvedInterval,
  ]);

  const candleOverlays: CandleChartOverlay[] = useMemo(() => {
    if (!showEma) return [];
    const keys = sortedEmaKeys(indicatorsQuery.data?.latest?.ema);
    return keys.map((key, i) => ({
      id: `ema-${key}`,
      title: t('detail:emaTitle', { period: key }),
      color: emaColor(key, i),
      data: indicatorPointsToEmaLine(indicatorsQuery.data?.points, key),
    }));
  }, [showEma, indicatorsQuery.data, t]);

  const rsiPoints = useMemo(
    () => indicatorPointsToRsi(indicatorsQuery.data?.points),
    [indicatorsQuery.data?.points],
  );

  const latestRsi = indicatorsQuery.data?.latest?.rsi ?? null;

  const emaLatestLabels = useMemo(() => {
    const ema = indicatorsQuery.data?.latest?.ema;
    if (!ema) return [];
    return sortedEmaKeys(ema).map((k) =>
      t('detail:emaLatest', { period: k, value: formatPrice(ema[k]) }),
    );
  }, [indicatorsQuery.data?.latest?.ema, t]);

  const ticker = tickerQuery.data;
  const supply = supplyQuery.data;

  const statsItems = useMemo(
    () => [
      { label: t('detail:stats.open'), value: formatPrice(ticker?.openPrice) },
      { label: t('detail:stats.high24h'), value: formatPrice(ticker?.highPrice) },
      { label: t('detail:stats.low24h'), value: formatPrice(ticker?.lowPrice) },
      { label: t('detail:stats.baseVol'), value: formatCompactUsd(ticker?.volume) },
      { label: t('detail:stats.quoteVol'), value: formatCompactUsd(ticker?.quoteVolume) },
      {
        label: t('detail:stats.trades'),
        value: formatTradeCount(ticker?.tradeCount, exchange),
      },
      {
        label: t('detail:stats.circSupply'),
        value: formatSupplyNum(supply?.circulatingSupply),
      },
      {
        label: t('detail:stats.totalSupply'),
        value: formatSupplyNum(supply?.totalSupply),
      },
      {
        label: t('detail:stats.circMcap'),
        value: formatCompactUsd(
          supply?.circulatingSupply != null && ticker?.lastPrice
            ? Number(ticker.lastPrice) * Number(supply.circulatingSupply)
            : null,
        ),
      },
    ],
    [ticker, supply, exchange, t],
  );

  const supplyError =
    supplyQuery.isError && (supplyQuery.error as { status?: number })?.status !== 404
      ? rtkErrorMessage(supplyQuery.error, { resource: 'supply' })
      : supplyQuery.isError
        ? t('detail:supplyUnavailable')
        : null;

  const onBack = useCallback(() => {
    navigation.goBack();
  }, [navigation]);

  const onAskAi = useCallback(() => {
    const parent = navigation.getParent() as
      | {
          navigate: (
            name: string,
            params?: {
              screen?: string;
              params?: {
                exchange?: string;
                symbol?: string;
                interval?: string;
              };
            },
          ) => void;
        }
      | undefined;
    parent?.navigate('AskTab', {
      screen: AiScreens.Chat,
      params: {
        exchange,
        symbol,
        interval: resolvedInterval,
      },
    });
  }, [navigation, exchange, symbol, resolvedInterval]);

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

  const chartMarkers = useMemo(
    () =>
      showPumps ? pumpEventsToChartMarkers(pumpQuery.data?.events) : [],
    [showPumps, pumpQuery.data?.events],
  );

  const chartPriceLines = useMemo(
    () =>
      showPumpMargin ? pumpEventsToMarginLines(pumpQuery.data?.events) : [],
    [showPumpMargin, pumpQuery.data?.events],
  );

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
      t('pumps:threshold', {
        pct: pumpQuery.data.minReturnPct ?? DEFAULT_PUMP_DETAIL_MIN_RETURN_PCT,
      }),
      pumpQuery.data.direction ?? DEFAULT_PUMP_DETAIL_DIRECTION,
      pumpQuery.data.interval ?? resolvedInterval,
    ];
    if (pumpQuery.data.eventCount != null) {
      parts.push(t('detail:pumpEventsCount', { count: pumpQuery.data.eventCount }));
    }
    return parts.join(' · ');
  }, [pumpQuery.data, resolvedInterval, t]);

  const onRetry = useCallback(() => {
    void tickerQuery.refetch();
    void supplyQuery.refetch();
    void candlesQuery.refetch();
    void indicatorsQuery.refetch();
    void intervalsQuery.refetch();
    void pumpQuery.refetch();
    crossExchange.refetchAll();
  }, [
    tickerQuery,
    supplyQuery,
    candlesQuery,
    indicatorsQuery,
    intervalsQuery,
    pumpQuery,
    crossExchange,
  ]);

  const onSelectInterval = useCallback((next: string) => {
    setInterval(next);
  }, []);

  const onPressCrossExchangeRow = useCallback(
    (nextExchange: string, nextSymbol: string) => {
      if (
        nextExchange === exchange &&
        nextSymbol.toUpperCase() === symbol.toUpperCase()
      ) {
        return;
      }
      navigation.navigate(MarketsScreens.Detail, {
        exchange: nextExchange,
        symbol: nextSymbol,
      });
    },
    [navigation, exchange, symbol],
  );

  const watched = watchlist?.isWatched(exchange, symbol) ?? false;
  const onStarPress = useCallback(() => {
    void watchlist?.toggle(exchange, symbol);
  }, [watchlist, exchange, symbol]);

  const crossExchangeCheapestId = useMemo(
    () => cheapestExchangeId(crossExchange.rows),
    [crossExchange.rows],
  );

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

    crossExchangeTitle: t('detail:crossExchangeTitle'),
    crossExchangeRows: crossExchange.rows,
    crossExchangeDisclaimer: t('detail:crossExchangeDisclaimer'),
    crossExchangeUnavailableLabel: t('detail:crossExchangeUnavailable'),
    crossExchangeSourceLabel: t('detail:crossExchangeSource'),
    crossExchangeCheapestLabel: t('detail:crossExchangeCheapest'),
    crossExchangeCheapestId,
    onPressCrossExchangeRow,

    intervals: supported ?? [],
    intervalsLoading: intervalsQuery.isLoading,
    interval: resolvedInterval,
    onSelectInterval,
    showEma,
    onToggleEma: () => setShowEma((v) => !v),
    showPumps,
    onTogglePumps: () => setShowPumps((v) => !v),
    showPumpMargin,
    onTogglePumpMargin: () => setShowPumpMargin((v) => !v),

    candles,
    candleOverlays,
    chartMarkers,
    chartPriceLines,
    candlesLoading:
      (candlesQuery.isLoading || candlesQuery.isFetching) && candles.length === 0,
    candlesLoadingOlder: isLoadingOlder,
    candlesError: candlesQuery.isError
      ? rtkErrorMessage(candlesQuery.error, { resource: 'candles' })
      : null,
    chartSeriesKey: seriesKey,
    canLoadOlderHistory:
      !historyExhausted && candles.length < DETAIL_MAX_CANDLES && candles.length > 0,
    historyEdgeBars: DETAIL_HISTORY_EDGE_BARS,
    onRequestOlderHistory,

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
    pumpDisclaimer:
      pumpQuery.data?.note ??
      t('common:disclaimer.pumps', { defaultValue: PUMP_DISCLAIMER }),

    onBack,
    onRetry,
    askAiLabel: t('detail:askAi'),
    onAskAi,
  };
}
