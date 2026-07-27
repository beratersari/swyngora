import { useCallback, useMemo } from 'react';
import { useIsFocused, useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { BottomTabNavigationProp } from '@react-navigation/bottom-tabs';
import { useAppStateActive } from '@/libs/hooks';
import { WATCHLIST_QUOTE_ENRICH_CAP } from '@/config/watchlistConstants';
import { useWatchlist } from '../../context';
import { WatchlistScreens, type WatchlistStackParamList } from '../../navigation';
import type { WatchlistPageViewModel } from './WatchlistPage.types';

type TabNav = BottomTabNavigationProp<Record<string, object | undefined>>;

export function useWatchlistPageViewModel(): WatchlistPageViewModel {
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
  }, [watchlist]);

  const onRetry = useCallback(() => {
    void watchlist.refresh();
  }, [watchlist]);

  const onOpenMarkets = useCallback(() => {
    // Jump to Markets tab
    const parent = navigation.getParent<TabNav>();
    parent?.navigate('MarketsTab' as never);
  }, [navigation]);

  const emptyMessage =
    watchlist.isReady && watchlist.items.length === 0 && !watchlist.error
      ? 'No favorites yet — open Markets and tap ★ on a pair'
      : null;

  return {
    title: 'Favorites',
    countLabel:
      watchlist.count > 0
        ? `${watchlist.count} favorite${watchlist.count === 1 ? '' : 's'}`
        : null,
    isLoading: !watchlist.isReady,
    isRefreshing: false,
    isPollingPaused: !pollQuotes,
    errorMessage: watchlist.error,
    emptyMessage,
    actionError: watchlist.actionError,
    pairs,
    onRetry,
    onRefresh,
    onPressRow,
    onUnstar,
    onOpenMarkets,
    pollQuotes,
  };
}
