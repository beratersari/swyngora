import { useCallback, useMemo, useState } from 'react';
import { useIsFocused } from '@react-navigation/native';
import {
  rtkErrorMessage,
  useListExchangesQuery,
  useListProductTagsQuery,
  useListSpotMarketsQuery,
  type MarketExchange,
  type SpotSortField,
  type SpotSortOrder,
} from '@/libs/api';
import { useAppStateActive, useDebouncedValue } from '@/libs/hooks';
import {
  DEFAULT_MARKETS_FILTER,
  isSpotSortField,
  normalizeExchange,
  toSpotListQuery,
} from '@/libs/utils';
import {
  DEFAULT_LIMIT,
  FALLBACK_EXCHANGES,
  QUOTE_OPTIONS,
  SEARCH_DEBOUNCE_MS,
  SORT_OPTIONS,
  SPOT_POLL_MS,
} from './MarketsPage.constants';
import { mapSpotMarketToRow, pageRangeLabel } from './MarketsPage.helpers';
import type { MarketsPageViewModel } from './MarketsPage.types';

export function useMarketsPageViewModel(): MarketsPageViewModel {
  const active = useAppStateActive();
  const focused = useIsFocused();
  const pollingEnabled = active && focused;

  const [exchange, setExchange] = useState<MarketExchange>(DEFAULT_MARKETS_FILTER.exchange);
  const [search, setSearch] = useState('');
  const [quote, setQuote] = useState(DEFAULT_MARKETS_FILTER.quote);
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [sort, setSort] = useState<SpotSortField>(DEFAULT_MARKETS_FILTER.sort);
  const [order, setOrder] = useState<SpotSortOrder>(DEFAULT_MARKETS_FILTER.order);
  const [offset, setOffset] = useState(0);
  const [detailHint, setDetailHint] = useState<string | null>(null);

  const debouncedQ = useDebouncedValue(search, SEARCH_DEBOUNCE_MS);

  const exchangesQuery = useListExchangesQuery();
  const tagsQuery = useListProductTagsQuery({ exchange });

  const filterState = useMemo(
    () => ({
      exchange,
      q: search,
      quote,
      tags: selectedTags,
      sort,
      order,
      limit: DEFAULT_LIMIT,
      offset,
    }),
    [exchange, search, quote, selectedTags, sort, order, offset],
  );

  const spotArgs = useMemo(
    () => toSpotListQuery(filterState, debouncedQ),
    [filterState, debouncedQ],
  );

  const spotQuery = useListSpotMarketsQuery(spotArgs, {
    pollingInterval: pollingEnabled ? SPOT_POLL_MS : 0,
    refetchOnFocus: false,
  });

  const exchanges =
    exchangesQuery.data?.exchanges?.length
      ? exchangesQuery.data.exchanges
      : FALLBACK_EXCHANGES;

  const rows = useMemo(
    () => (spotQuery.data?.items ?? []).map(mapSpotMarketToRow),
    [spotQuery.data?.items],
  );

  const total = spotQuery.data?.total ?? 0;
  const limit = spotQuery.data?.limit ?? DEFAULT_LIMIT;
  const canPrev = offset > 0;
  const canNext = offset + limit < total;

  const isLoading = spotQuery.isLoading || (spotQuery.isFetching && rows.length === 0);
  const isRefreshing = spotQuery.isFetching && rows.length > 0;

  const errorMessage = spotQuery.isError
    ? rtkErrorMessage(spotQuery.error, { resource: 'markets' })
    : null;

  const emptyMessage =
    !errorMessage && !isLoading && rows.length === 0
      ? 'No markets match filters'
      : null;

  const onSelectExchange = useCallback((next: string) => {
    setExchange(normalizeExchange(next));
    setOffset(0);
    setSelectedTags([]);
  }, []);

  const onQuoteChange = useCallback((next: string) => {
    setQuote(next);
    setOffset(0);
  }, []);

  const onToggleTag = useCallback((tag: string) => {
    setSelectedTags((prev) => {
      if (prev.includes(tag)) return prev.filter((t) => t !== tag);
      return [...prev, tag];
    });
    setOffset(0);
  }, []);

  const onClearTags = useCallback(() => {
    setSelectedTags([]);
    setOffset(0);
  }, []);

  const onSortChange = useCallback((next: string) => {
    if (isSpotSortField(next)) {
      setSort(next);
      setOffset(0);
    }
  }, []);

  const onOrderChange = useCallback((next: 'asc' | 'desc') => {
    setOrder(next);
    setOffset(0);
  }, []);

  const onNextPage = useCallback(() => {
    setOffset((prev) => prev + DEFAULT_LIMIT);
  }, []);

  const onPrevPage = useCallback(() => {
    setOffset((prev) => Math.max(0, prev - DEFAULT_LIMIT));
  }, []);

  const onRetry = useCallback(() => {
    void spotQuery.refetch();
    void tagsQuery.refetch();
    void exchangesQuery.refetch();
  }, [spotQuery, tagsQuery, exchangesQuery]);

  const onRefresh = useCallback(() => {
    void spotQuery.refetch();
    void tagsQuery.refetch();
  }, [spotQuery, tagsQuery]);

  const onPressRow = useCallback((symbol: string) => {
    setDetailHint(`${symbol}: coin detail coming soon`);
  }, []);

  const onSearchChange = useCallback((q: string) => {
    setSearch(q);
    setOffset(0);
  }, []);

  return {
    title: 'Markets',
    exchanges,
    selectedExchange: exchange,
    onSelectExchange,
    exchangesLoading: exchangesQuery.isLoading,

    search,
    onSearchChange,

    quote,
    quoteOptions: [...QUOTE_OPTIONS],
    onQuoteChange,

    availableTags: tagsQuery.data?.tags ?? [],
    selectedTags,
    onToggleTag,
    onClearTags,

    sort,
    order,
    sortOptions: SORT_OPTIONS.map((o) => ({ value: o.value, label: o.label })),
    onSortChange,
    onOrderChange,

    rows,
    total,
    offset,
    limit,
    onNextPage,
    onPrevPage,
    canNext,
    canPrev,

    isLoading,
    isRefreshing,
    isPollingPaused: !pollingEnabled,
    errorMessage,
    emptyMessage,
    lastUpdatedLabel: detailHint,
    summaryLabel: pageRangeLabel(offset, limit, total),

    onRetry,
    onRefresh,
    onPressRow,
  };
}
