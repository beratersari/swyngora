export type SortOption = { value: string; label: string };

export type MarketsFilterBarProps = {
  search: string;
  onSearchChange: (q: string) => void;
  quote: string;
  quoteOptions: string[];
  onQuoteChange: (quote: string) => void;
  availableTags: string[];
  selectedTags: string[];
  onToggleTag: (tag: string) => void;
  onClearTags: () => void;
  sort: string;
  order: 'asc' | 'desc';
  sortOptions: SortOption[];
  onSortChange: (sort: string) => void;
  onOrderChange: (order: 'asc' | 'desc') => void;
};
