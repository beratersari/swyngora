import { useCallback, useMemo, useState } from 'react';
import { useIsFocused, useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { useTranslation } from 'react-i18next';
import {
  rtkErrorMessage,
  useListExchangesQuery,
  useScanPumpEventsQuery,
  type MarketExchange,
} from '@/libs/api';
import { useAppStateActive } from '@/libs/hooks';
import {
  buildScanQuery,
  defaultPumpScanFilters,
  formatPumpReturnPct,
  formatVolumeRatio,
  isMarketExchange,
  pumpReturnTone,
  type PumpDirection,
  type PumpScanFilterState,
} from '@/libs/utils';
import {
  PUMP_DIRECTION_OPTIONS,
  PUMP_DISCLAIMER,
  PUMP_LOOKBACK_OPTIONS,
  PUMP_THRESHOLD_OPTIONS,
} from '@/config/pumpConstants';
import type { PumpHitRowViewModel } from '@/components/organisms/pump-hit-row';
import { PumpsScreens, type PumpsStackParamList } from '../../navigation';
import type { PumpsScanPageViewModel } from './PumpsScanPage.types';

const FALLBACK_EXCHANGES = ['binance', 'coinbase', 'bybit'];

function mapHitToRow(
  hit: {
    symbol?: string;
    exchange?: string;
    interval?: string;
    bestReturnPct?: number;
    events?: { volumeRatio?: number }[];
  },
  eventsLabel: string,
): PumpHitRowViewModel {
  const symbol = hit.symbol ?? '—';
  const exchange = (hit.exchange ?? 'binance').toLowerCase();
  const events = hit.events ?? [];
  const vol =
    events[0]?.volumeRatio != null ? formatVolumeRatio(events[0].volumeRatio) : '';
  return {
    id: `${exchange}|${symbol}`,
    symbol,
    exchange,
    bestReturnLabel: formatPumpReturnPct(hit.bestReturnPct),
    bestReturnTone: pumpReturnTone(hit.bestReturnPct),
    eventsLabel,
    metaLabel: [hit.interval, vol].filter(Boolean).join(' · '),
  };
}

export function usePumpsScanPageViewModel(): PumpsScanPageViewModel {
  const { t } = useTranslation(['pumps', 'common']);
  const navigation =
    useNavigation<NativeStackNavigationProp<PumpsStackParamList>>();
  const active = useAppStateActive();
  const focused = useIsFocused();

  const [filters, setFilters] = useState<PumpScanFilterState>(() =>
    defaultPumpScanFilters('binance'),
  );

  const exchangesQuery = useListExchangesQuery();
  const exchanges =
    exchangesQuery.data?.exchanges?.length
      ? exchangesQuery.data.exchanges
      : FALLBACK_EXCHANGES;

  const scanArgs = useMemo(() => buildScanQuery(filters), [filters]);

  // Skip when backgrounded so we do not keep heavy scans running
  const skip = !active || !focused;
  const scanQuery = useScanPumpEventsQuery(scanArgs, {
    skip,
    refetchOnFocus: false,
    pollingInterval: 0,
  });

  const rows = useMemo(
    () =>
      (scanQuery.data?.hits ?? []).map((hit) => {
        const count = hit.events?.length ?? 0;
        return mapHitToRow(
          hit,
          t('pumps:events', { count }),
        );
      }),
    [scanQuery.data?.hits, t],
  );

  const isLoading = scanQuery.isLoading || (scanQuery.isFetching && rows.length === 0);
  const isRefreshing = scanQuery.isFetching && rows.length > 0;
  const errorMessage = scanQuery.isError
    ? rtkErrorMessage(scanQuery.error, { resource: 'pumps scan' })
    : null;
  const emptyMessage =
    !errorMessage && !isLoading && rows.length === 0
      ? t('pumps:empty')
      : null;

  const summaryLabel = useMemo(() => {
    const parts = [
      filters.quote,
      filters.interval,
      t('pumps:hours', { hours: filters.lookbackHours }),
      t('pumps:threshold', { pct: filters.minReturnPct }),
      filters.direction,
    ];
    const count = scanQuery.data?.hitCount;
    if (typeof count === 'number') parts.push(String(count));
    return parts.join(' · ');
  }, [filters, scanQuery.data?.hitCount, t]);

  const onSelectExchange = useCallback((exchange: string) => {
    const ex: MarketExchange = isMarketExchange(exchange) ? exchange : 'binance';
    setFilters((prev) => ({
      ...defaultPumpScanFilters(ex),
      // keep user threshold/direction/lookback when switching venue
      minReturnPct: prev.minReturnPct,
      direction: prev.direction,
      lookbackHours: prev.lookbackHours,
      exchange: ex,
      quote: defaultPumpScanFilters(ex).quote,
    }));
  }, []);

  const onSelectLookback = useCallback((hours: number) => {
    setFilters((prev) => ({ ...prev, lookbackHours: hours }));
  }, []);

  const onSelectThreshold = useCallback((pct: number) => {
    setFilters((prev) => ({ ...prev, minReturnPct: pct }));
  }, []);

  const onSelectDirection = useCallback((direction: string) => {
    const d = (['up', 'down', 'both'] as PumpDirection[]).includes(
      direction as PumpDirection,
    )
      ? (direction as PumpDirection)
      : 'up';
    setFilters((prev) => ({ ...prev, direction: d }));
  }, []);

  const onRetry = useCallback(() => {
    void scanQuery.refetch();
    void exchangesQuery.refetch();
  }, [scanQuery, exchangesQuery]);

  const onRefresh = useCallback(() => {
    void scanQuery.refetch();
  }, [scanQuery]);

  const onPressRow = useCallback(
    (exchange: string, symbol: string) => {
      navigation.navigate(PumpsScreens.Detail, { exchange, symbol });
    },
    [navigation],
  );

  return {
    title: t('pumps:title'),
    exchanges,
    selectedExchange: filters.exchange,
    onSelectExchange,
    exchangesLoading: exchangesQuery.isLoading,

    lookbackHours: filters.lookbackHours,
    lookbackOptions: [...PUMP_LOOKBACK_OPTIONS],
    onSelectLookback,

    minReturnPct: filters.minReturnPct,
    thresholdOptions: [...PUMP_THRESHOLD_OPTIONS],
    onSelectThreshold,

    direction: filters.direction,
    directionOptions: PUMP_DIRECTION_OPTIONS.map((d) => ({
      value: d.value,
      label: t(`pumps:directions.${d.value}`),
    })),
    onSelectDirection,

    summaryLabel,
    disclaimer:
      scanQuery.data?.note ??
      t('common:disclaimer.pumps', { defaultValue: PUMP_DISCLAIMER }),

    rows,
    isLoading: skip ? false : isLoading,
    isRefreshing: skip ? false : isRefreshing,
    errorMessage: skip ? null : errorMessage,
    emptyMessage: skip ? t('pumps:paused') : emptyMessage,

    onRetry,
    onRefresh,
    onPressRow,
  };
}
