import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import type { MarketExchange, SpotSortField, SpotSortOrder } from '@/libs/api';
import { isSpotSortField, normalizeExchange } from '@/libs/utils';
import {
  DEFAULT_ORDER,
  DEFAULT_QUOTE,
  DEFAULT_SORT,
} from '../pages/MarketsPage/MarketsPage.constants';
import type { MarketsContextValue, MarketsListFilters } from './MarketsContext.types';

const MarketsContext = createContext<MarketsContextValue | null>(null);

export function MarketsProvider({ children }: { children: ReactNode }) {
  const [exchange, setExchangeState] = useState<MarketExchange>('binance');
  const [search, setSearchState] = useState('');
  const [quote, setQuoteState] = useState(DEFAULT_QUOTE);
  const [selectedTags, setSelectedTagsState] = useState<string[]>([]);
  const [sort, setSortState] = useState<SpotSortField>(DEFAULT_SORT);
  const [order, setOrderState] = useState<SpotSortOrder>(DEFAULT_ORDER);
  const [filterRevision, setFilterRevision] = useState(0);

  const notifyFiltersChanged = useCallback(() => {
    setFilterRevision((n) => n + 1);
  }, []);

  const setExchange = useCallback(
    (next: string) => {
      setExchangeState(normalizeExchange(next));
      // Tags are exchange-scoped; clear on switch
      setSelectedTagsState([]);
      notifyFiltersChanged();
    },
    [notifyFiltersChanged],
  );

  const setSearch = useCallback((next: string) => {
    setSearchState(next);
  }, []);

  const applyListFilters = useCallback(
    (filters: MarketsListFilters) => {
      setQuoteState(filters.quote);
      setSelectedTagsState(filters.selectedTags);
      if (isSpotSortField(filters.sort)) {
        setSortState(filters.sort);
      }
      setOrderState(filters.order);
      notifyFiltersChanged();
    },
    [notifyFiltersChanged],
  );

  const activeFilterCount = useMemo(() => {
    let n = 0;
    if (quote !== DEFAULT_QUOTE) n += 1;
    if (selectedTags.length > 0) n += 1;
    if (sort !== DEFAULT_SORT) n += 1;
    if (order !== DEFAULT_ORDER) n += 1;
    return n;
  }, [quote, selectedTags, sort, order]);

  const value = useMemo<MarketsContextValue>(
    () => ({
      exchange,
      search,
      quote,
      selectedTags,
      sort,
      order,
      filterRevision,
      setExchange,
      setSearch,
      applyListFilters,
      notifyFiltersChanged,
      activeFilterCount,
    }),
    [
      exchange,
      search,
      quote,
      selectedTags,
      sort,
      order,
      filterRevision,
      setExchange,
      setSearch,
      applyListFilters,
      notifyFiltersChanged,
      activeFilterCount,
    ],
  );

  return <MarketsContext.Provider value={value}>{children}</MarketsContext.Provider>;
}

export function useMarketsContext(): MarketsContextValue {
  const ctx = useContext(MarketsContext);
  if (!ctx) {
    throw new Error('useMarketsContext must be used within MarketsProvider');
  }
  return ctx;
}

export function useOptionalMarketsContext(): MarketsContextValue | null {
  return useContext(MarketsContext);
}
