import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
// favorites filter is client-side over loaded rows
import { useIsFocused, useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import {
  rtkErrorMessage,
  useListExchangesQuery,
  useListSpotMarketsQuery,
  type SpotMarket,
} from '@/libs/api';
import { useAppStateActive, useDebouncedValue } from '@/libs/hooks';
import { toSpotListQuery } from '@/libs/utils';
import { useMarketsContext } from '../../context';
import { useWatchlist } from '@/modules/watchlist';
import { MarketsScreens, type MarketsStackParamList } from '../../navigation';
import {
  DEFAULT_LIMIT,
  DEFAULT_ORDER,
  DEFAULT_QUOTE,
  DEFAULT_SORT,
  FALLBACK_EXCHANGES,
  SEARCH_DEBOUNCE_MS,
  SPOT_POLL_MS,
} from './MarketsPage.constants';
import {
  buildFavoritesOnlyRows,
  favoritesEmptyMessage,
  favoritesForExchange,
  favoritesSummaryLabel,
  mapSpotMarketToRow,
} from './MarketsPage.helpers';
import type { MarketRowViewModel } from '@/components/organisms/market-row';
import type { MarketsPageViewModel } from './MarketsPage.types';

function mergeUniqueRows(
  prev: MarketRowViewModel[],
  nextItems: SpotMarket[],
): MarketRowViewModel[] {
  const seen = new Set(prev.map((r) => r.id));
  const appended: MarketRowViewModel[] = [];
  for (const item of nextItems) {
    const row = mapSpotMarketToRow(item);
    if (!seen.has(row.id)) {
      seen.add(row.id);
      appended.push(row);
    }
  }
  return appended.length === 0 ? prev : [...prev, ...appended];
}

export function useMarketsPageViewModel(): MarketsPageViewModel {
  const navigation =
    useNavigation<NativeStackNavigationProp<MarketsStackParamList>>();
  const markets = useMarketsContext();
  const watchlist = useWatchlist();
  const active = useAppStateActive();
  const focused = useIsFocused();

  const debouncedSearch = useDebouncedValue(markets.search, SEARCH_DEBOUNCE_MS);
  const isSearchDebouncing = markets.search.trim() !== debouncedSearch.trim();

  // Reset list when debounced search settles to a new value
  const lastDebouncedRef = useRef(debouncedSearch);
  useEffect(() => {
    if (lastDebouncedRef.current !== debouncedSearch) {
      lastDebouncedRef.current = debouncedSearch;
      markets.notifyFiltersChanged();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only when debounced search changes
  }, [debouncedSearch]);

  const [offset, setOffset] = useState(0);
  const [rows, setRows] = useState<MarketRowViewModel[]>([]);
  const [total, setTotal] = useState(0);
  const [favoritesOnly, setFavoritesOnly] = useState(false);

  // Reset pagination when filter revision changes
  useEffect(() => {
    setOffset(0);
    setRows([]);
    setTotal(0);
  }, [markets.filterRevision]);

  const filterState = useMemo(
    () => ({
      exchange: markets.exchange,
      q: markets.search,
      quote: markets.quote,
      tags: markets.selectedTags,
      sort: markets.sort,
      order: markets.order,
      limit: DEFAULT_LIMIT,
      offset,
    }),
    [
      markets.exchange,
      markets.search,
      markets.quote,
      markets.selectedTags,
      markets.sort,
      markets.order,
      offset,
    ],
  );

  const spotArgs = useMemo(
    () => toSpotListQuery(filterState, debouncedSearch),
    [filterState, debouncedSearch],
  );

  // Poll only the first page while user has not scrolled into more pages
  const pollingEnabled = active && focused && offset === 0 && rows.length <= DEFAULT_LIMIT;

  const spotQuery = useListSpotMarketsQuery(spotArgs, {
    pollingInterval: pollingEnabled ? SPOT_POLL_MS : 0,
    refetchOnFocus: false,
  });

  const exchangesQuery = useListExchangesQuery();

  // Merge query results into accumulated rows
  useEffect(() => {
    if (!spotQuery.data || spotQuery.isError) return;
    const items = spotQuery.data.items ?? [];
    const nextTotal = spotQuery.data.total ?? 0;
    setTotal(nextTotal);

    if (offset === 0) {
      setRows(items.map(mapSpotMarketToRow));
    } else {
      setRows((prev) => mergeUniqueRows(prev, items));
    }
  }, [spotQuery.data, spotQuery.isError, offset]);

  const exchanges =
    exchangesQuery.data?.exchanges?.length
      ? exchangesQuery.data.exchanges
      : FALLBACK_EXCHANGES;

  const hasMore = rows.length < total;
  const isLoading =
    (spotQuery.isLoading || spotQuery.isFetching) && rows.length === 0 && !isSearchDebouncing;
  const isLoadingMore =
    spotQuery.isFetching && offset > 0 && rows.length > 0;
  const isRefreshing =
    spotQuery.isFetching && offset === 0 && rows.length > 0;

  const errorMessage = spotQuery.isError
    ? rtkErrorMessage(spotQuery.error, { resource: 'markets' })
    : null;

  const emptyMessage =
    !errorMessage && !isLoading && !isSearchDebouncing && rows.length === 0
      ? 'No markets match filters'
      : null;

  const onLoadMore = useCallback(() => {
    if (!hasMore || spotQuery.isFetching || isLoading || isSearchDebouncing) return;
    setOffset((prev) => prev + DEFAULT_LIMIT);
  }, [hasMore, spotQuery.isFetching, isLoading, isSearchDebouncing]);

  const onRefresh = useCallback(() => {
    setOffset(0);
    markets.notifyFiltersChanged();
    void spotQuery.refetch();
  }, [markets, spotQuery]);

  const onRetry = useCallback(() => {
    void spotQuery.refetch();
    void exchangesQuery.refetch();
  }, [spotQuery, exchangesQuery]);

  const onOpenFilters = useCallback(() => {
    navigation.navigate(MarketsScreens.Filters);
  }, [navigation]);

  const onSearchChange = useCallback(
    (q: string) => {
      markets.setSearch(q);
    },
    [markets],
  );

  const onPressRow = useCallback(
    (symbol: string) => {
      navigation.navigate(MarketsScreens.Detail, {
        exchange: markets.exchange,
        symbol,
      });
    },
    [navigation, markets.exchange],
  );

  const summaryLabel =
    total > 0
      ? `Showing ${rows.length} of ${total}`
      : rows.length === 0
        ? null
        : `${rows.length} markets`;

  const filterSummaryParts: string[] = [];
  if (markets.quote !== DEFAULT_QUOTE) filterSummaryParts.push(markets.quote);
  if (markets.sort !== DEFAULT_SORT) filterSummaryParts.push(markets.sort);
  if (markets.order !== DEFAULT_ORDER) filterSummaryParts.push(markets.order);
  if (markets.selectedTags.length > 0) {
    filterSummaryParts.push(`${markets.selectedTags.length} tags`);
  }
  const filterSummary =
    filterSummaryParts.length > 0 ? filterSummaryParts.join(' · ') : null;

  const isWatched = useCallback(
    (symbol: string) => watchlist.isWatched(markets.exchange, symbol),
    [watchlist, markets.exchange],
  );

  const onStarPress = useCallback(
    (symbol: string) => {
      void watchlist.toggle(markets.exchange, symbol);
    },
    [watchlist, markets.exchange],
  );

  const onToggleFavoritesOnly = useCallback(() => {
    setFavoritesOnly((v) => !v);
  }, []);

  /** Favorites for the active exchange (badge + filter). */
  const favoritesOnExchange = useMemo(
    () => favoritesForExchange(watchlist.items, markets.exchange),
    [watchlist.items, markets.exchange],
  );

  const displayRows = useMemo(() => {
    if (!favoritesOnly) return rows;
    return buildFavoritesOnlyRows(favoritesOnExchange, rows);
  }, [favoritesOnly, rows, favoritesOnExchange]);

  const displayEmpty =
    favoritesEmptyMessage(favoritesOnly, errorMessage, isLoading, displayRows.length) ??
    emptyMessage;
  const displaySummary = favoritesSummaryLabel(
    favoritesOnly,
    displayRows.length,
    summaryLabel,
  );

  return {
    title: 'Markets',
    exchanges,
    selectedExchange: markets.exchange,
    onSelectExchange: markets.setExchange,
    exchangesLoading: exchangesQuery.isLoading,

    search: markets.search,
    onSearchChange,
    isSearchDebouncing,

    activeFilterCount: markets.activeFilterCount,
    filterSummary,
    onOpenFilters,

    favoritesOnly,
    favoritesCount: favoritesOnExchange.length,
    onToggleFavoritesOnly,

    rows: displayRows,
    total: favoritesOnly ? displayRows.length : total,
    hasMore: favoritesOnly ? false : hasMore,
    isLoading,
    isLoadingMore: favoritesOnly ? false : isLoadingMore,
    isRefreshing,
    isPollingPaused: !pollingEnabled,
    errorMessage,
    emptyMessage: displayEmpty,
    summaryLabel: displaySummary,
    detailHint: favoritesOnly
      ? 'Showing favorites only — tap ★ again to remove'
      : null,
    actionError: watchlist.actionError,

    onLoadMore,
    onRetry,
    onRefresh,
    onPressRow,
    isWatched,
    onStarPress,
  };
}
