import type { MarketExchange, SpotSortField, SpotSortOrder } from '@/libs/api';

export type MarketsListFilters = {
  quote: string;
  selectedTags: string[];
  sort: SpotSortField;
  order: SpotSortOrder;
};

export type MarketsFilterState = MarketsListFilters & {
  exchange: MarketExchange;
  search: string;
};

export type MarketsContextValue = MarketsFilterState & {
  setExchange: (exchange: string) => void;
  setSearch: (search: string) => void;
  /** Commit quote, sort, order, tags and reset the list. */
  applyListFilters: (filters: MarketsListFilters) => void;
  /**
   * Discovery path: single tag, default quote/sort, clear search.
   * Replaces any multi-tag filter.
   */
  selectCategoryTag: (tag: string) => void;
  /** Clear selected tags only (keep quote/sort/search). */
  clearSelectedTags: () => void;
  /** Notify list that filters changed (search debounce). */
  notifyFiltersChanged: () => void;
  filterRevision: number;
  /** Active filter count (quote≠default, tags, sort≠default, order≠default). */
  activeFilterCount: number;
};
