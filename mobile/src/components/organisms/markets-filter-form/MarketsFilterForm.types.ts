export type SortOption = { value: string; label: string };

export type MarketsFilterFormProps = {
  quote: string;
  quoteOptions: string[];
  onQuoteChange: (quote: string) => void;

  sort: string;
  order: 'asc' | 'desc';
  sortOptions: SortOption[];
  onSortChange: (sort: string) => void;
  onOrderChange: (order: 'asc' | 'desc') => void;

  availableTags: string[];
  selectedTags: string[];
  isLoadingTags: boolean;
  tagsError: string | null;
  tagSearch: string;
  onTagSearchChange: (q: string) => void;
  onToggleTag: (tag: string) => void;
  onClearTags: () => void;
  onSelectAllVisible: () => void;
  onResetAll: () => void;
};
