import { useCallback, useEffect, useMemo, useState } from 'react';
import { useIsFocused, useNavigation, useRoute } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { RouteProp } from '@react-navigation/native';
import { useTranslation } from 'react-i18next';
import {
  rtkErrorMessage,
  useListExchangesQuery,
  useListSpotMarketsQuery,
  type MarketExchange,
  type SpotMarket,
} from '@/libs/api';
import { useAppStateActive } from '@/libs/hooks';
import {
  buildLeaderboardSpotQuery,
  isMarketExchange,
  mapSpotListToLeaderboardRows,
} from '@/libs/utils';
import {
  defaultQuoteForLeaderboard,
  FALLBACK_LEADERBOARD_EXCHANGES,
  isLeaderboardKind,
  LEADERBOARD_DEFAULT_EXCHANGE,
  LEADERBOARD_KINDS,
  LEADERBOARD_PAGE_SIZE,
  LEADERBOARD_POLL_MS,
  LEADERBOARD_QUOTE_OPTIONS,
  type LeaderboardKind,
} from '@/config/leaderboardConstants';
import type { MarketRowViewModel } from '@/components/organisms/market-row';
import { MarketsScreens, type MarketsStackParamList } from '../../navigation';
import type { LeaderboardsPageViewModel } from './LeaderboardsPage.types';

function mergeUniqueRows(
  prev: MarketRowViewModel[],
  nextItems: SpotMarket[],
  offset: number,
): MarketRowViewModel[] {
  const mapped = mapSpotListToLeaderboardRows(nextItems, offset);
  const seen = new Set(prev.map((r) => r.id));
  const appended: MarketRowViewModel[] = [];
  for (const row of mapped) {
    if (!seen.has(row.id)) {
      seen.add(row.id);
      appended.push(row);
    }
  }
  return appended.length === 0 ? prev : [...prev, ...appended];
}

export function useLeaderboardsPageViewModel(): LeaderboardsPageViewModel {
  const { t } = useTranslation(['leaderboards', 'common', 'markets']);
  const navigation =
    useNavigation<NativeStackNavigationProp<MarketsStackParamList>>();
  const route =
    useRoute<RouteProp<MarketsStackParamList, typeof MarketsScreens.Leaderboards>>();
  const active = useAppStateActive();
  const focused = useIsFocused();

  const initialBoard = isLeaderboardKind(route.params?.board)
    ? route.params.board
    : 'gainers';

  const [board, setBoard] = useState<LeaderboardKind>(initialBoard);
  const [exchange, setExchange] = useState<MarketExchange>(LEADERBOARD_DEFAULT_EXCHANGE);
  const [quote, setQuote] = useState(defaultQuoteForLeaderboard(LEADERBOARD_DEFAULT_EXCHANGE));
  const [offset, setOffset] = useState(0);
  const [rows, setRows] = useState<MarketRowViewModel[]>([]);
  const [total, setTotal] = useState(0);
  const [filterRevision, setFilterRevision] = useState(0);

  // Sync board from navigation params when route changes
  useEffect(() => {
    if (isLeaderboardKind(route.params?.board) && route.params.board !== board) {
      setBoard(route.params.board);
      setFilterRevision((n) => n + 1);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only when route board changes
  }, [route.params?.board]);

  useEffect(() => {
    setOffset(0);
    setRows([]);
    setTotal(0);
  }, [filterRevision]);

  const spotArgs = useMemo(
    () =>
      buildLeaderboardSpotQuery({
        board,
        exchange,
        quote,
        limit: LEADERBOARD_PAGE_SIZE,
        offset,
      }),
    [board, exchange, quote, offset],
  );

  const pollingEnabled = active && focused && offset === 0 && rows.length <= LEADERBOARD_PAGE_SIZE;

  const spotQuery = useListSpotMarketsQuery(spotArgs, {
    pollingInterval: pollingEnabled ? LEADERBOARD_POLL_MS : 0,
    refetchOnFocus: false,
  });

  const exchangesQuery = useListExchangesQuery();

  useEffect(() => {
    if (!spotQuery.data || spotQuery.isError) return;
    const items = spotQuery.data.items ?? [];
    const nextTotal = spotQuery.data.total ?? 0;
    setTotal(nextTotal);
    if (offset === 0) {
      setRows(mapSpotListToLeaderboardRows(items, 0));
    } else {
      setRows((prev) => mergeUniqueRows(prev, items, offset));
    }
  }, [spotQuery.data, spotQuery.isError, offset]);

  const exchanges =
    exchangesQuery.data?.exchanges?.length
      ? exchangesQuery.data.exchanges
      : [...FALLBACK_LEADERBOARD_EXCHANGES];

  const hasMore = rows.length < total;
  const isLoading =
    (spotQuery.isLoading || spotQuery.isFetching) && rows.length === 0;
  const isLoadingMore = spotQuery.isFetching && offset > 0 && rows.length > 0;
  const isRefreshing = spotQuery.isFetching && offset === 0 && rows.length > 0;

  const errorMessage = spotQuery.isError
    ? rtkErrorMessage(spotQuery.error, { resource: 'leaderboards' })
    : null;

  const emptyMessage =
    !errorMessage && !isLoading && rows.length === 0
      ? t('leaderboards:empty', { board: t(`leaderboards:boards.${board}`) })
      : null;

  const boardOptions = LEADERBOARD_KINDS.map((k) => ({
    value: k,
    label: t(`leaderboards:boards.${k}`),
  }));

  const onSelectBoard = useCallback((next: string) => {
    if (!isLeaderboardKind(next)) return;
    setBoard(next);
    setFilterRevision((n) => n + 1);
  }, []);

  const onSelectExchange = useCallback((next: string) => {
    const ex = isMarketExchange(next) ? next : LEADERBOARD_DEFAULT_EXCHANGE;
    setExchange(ex);
    setQuote(defaultQuoteForLeaderboard(ex));
    setFilterRevision((n) => n + 1);
  }, []);

  const onSelectQuote = useCallback((next: string) => {
    setQuote(next);
    setFilterRevision((n) => n + 1);
  }, []);

  const onLoadMore = useCallback(() => {
    if (!hasMore || spotQuery.isFetching || isLoading) return;
    setOffset((prev) => prev + LEADERBOARD_PAGE_SIZE);
  }, [hasMore, spotQuery.isFetching, isLoading]);

  const onRefresh = useCallback(() => {
    setOffset(0);
    setFilterRevision((n) => n + 1);
    void spotQuery.refetch();
  }, [spotQuery]);

  const onRetry = useCallback(() => {
    void spotQuery.refetch();
    void exchangesQuery.refetch();
  }, [spotQuery, exchangesQuery]);

  const onPressRow = useCallback(
    (symbol: string) => {
      navigation.navigate(MarketsScreens.Detail, {
        exchange,
        symbol,
      });
    },
    [navigation, exchange],
  );

  const onBack = useCallback(() => {
    if (navigation.canGoBack()) navigation.goBack();
    else navigation.navigate(MarketsScreens.List);
  }, [navigation]);

  const summaryLabel =
    total > 0
      ? t('leaderboards:summary', { count: rows.length, total })
      : null;

  return {
    title: t('leaderboards:title'),
    board,
    boardOptions,
    onSelectBoard,
    exchanges,
    selectedExchange: exchange,
    onSelectExchange,
    exchangesLoading: exchangesQuery.isLoading,
    quote,
    quoteOptions: [...LEADERBOARD_QUOTE_OPTIONS],
    onSelectQuote,
    rows,
    isLoading,
    isLoadingMore,
    isRefreshing,
    hasMore,
    isPollingPaused: !pollingEnabled,
    errorMessage,
    emptyMessage,
    summaryLabel,
    onLoadMore,
    onRetry,
    onRefresh,
    onPressRow,
    onBack,
    backLabel: t('common:actions.back'),
    retryLabel: t('common:actions.retry'),
  };
}
