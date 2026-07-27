import { useCallback, useMemo } from 'react';
import { useIsFocused, useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { BottomTabNavigationProp } from '@react-navigation/bottom-tabs';
import { useTranslation } from 'react-i18next';
import {
  useAppStateActive,
  useMultiExchangeBatchIndicators,
} from '@/libs/hooks';
import {
  BATCH_FAVORITES_ENRICH_CAP,
  BATCH_INDICATORS_DISCLAIMER,
  BATCH_INDICATORS_POLL_MS,
} from '@/config/batchIndicatorsConstants';
import { WATCHLIST_QUOTE_ENRICH_CAP } from '@/config/watchlistConstants';
import { useWatchlist } from '../../context';
import { WatchlistScreens, type WatchlistStackParamList } from '../../navigation';
import type { WatchlistPageViewModel } from './WatchlistPage.types';

type TabNav = BottomTabNavigationProp<Record<string, object | undefined>>;

export function useWatchlistPageViewModel(): WatchlistPageViewModel {
  const { t } = useTranslation(['watchlist', 'common']);
  const navigation =
    useNavigation<NativeStackNavigationProp<WatchlistStackParamList>>();
  const watchlist = useWatchlist();
  const active = useAppStateActive();
  const focused = useIsFocused();
  const pollQuotes = active && focused;

  const pairs = useMemo(
    () =>
      watchlist.items.slice(0, WATCHLIST_QUOTE_ENRICH_CAP).map((i) => ({
        exchange: i.exchange,
        symbol: i.symbol,
      })),
    [watchlist.items],
  );

  const batchPairs = useMemo(
    () => pairs.slice(0, BATCH_FAVORITES_ENRICH_CAP),
    [pairs],
  );

  const batch = useMultiExchangeBatchIndicators(batchPairs, {
    enabled: pollQuotes && batchPairs.length > 0,
    pollingIntervalMs: BATCH_INDICATORS_POLL_MS,
    enrichCap: BATCH_FAVORITES_ENRICH_CAP,
  });

  const onPressRow = useCallback(
    (exchange: string, symbol: string) => {
      navigation.navigate(WatchlistScreens.Detail, { exchange, symbol });
    },
    [navigation],
  );

  const onUnstar = useCallback(
    (exchange: string, symbol: string) => {
      void watchlist.toggle(exchange, symbol);
    },
    [watchlist],
  );

  const onRefresh = useCallback(() => {
    void watchlist.refresh();
    batch.refetch();
  }, [watchlist, batch]);

  const onRetry = useCallback(() => {
    void watchlist.refresh();
    batch.refetch();
  }, [watchlist, batch]);

  const onOpenMarkets = useCallback(() => {
    // Jump to Markets tab
    const parent = navigation.getParent<TabNav>();
    parent?.navigate('MarketsTab' as never);
  }, [navigation]);

  const emptyMessage =
    watchlist.isReady && watchlist.items.length === 0 && !watchlist.error
      ? t('watchlist:empty')
      : null;

  return {
    title: t('watchlist:title'),
    countLabel:
      watchlist.count > 0
        ? t('watchlist:count', { count: watchlist.count })
        : null,
    isLoading: !watchlist.isReady,
    isRefreshing: false,
    isPollingPaused: !pollQuotes,
    errorMessage: watchlist.error,
    emptyMessage,
    actionError: watchlist.actionError,
    indicatorsError: batch.errorMessage
      ? t('watchlist:indicatorsPrefix', { message: batch.errorMessage })
      : null,
    indicatorsDisclaimer:
      batchPairs.length > 0
        ? (batch.disclaimer ??
          t('common:disclaimer.indicators', {
            defaultValue: BATCH_INDICATORS_DISCLAIMER,
          }))
        : null,
    pairs,
    onRetry,
    onRefresh,
    onPressRow,
    onUnstar,
    onOpenMarkets,
    pollQuotes,
    rsiByKey: batch.byKey,
  };
}
