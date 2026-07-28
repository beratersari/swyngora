export type SortOption = { value: string; label: string };

export type MarketsFilterPageViewModel = {
  title: string;

  quote: string;
  quoteOptions: string[];
  onQuoteChange: (quote: string) => void;

  sort: string;
  order: 'asc' | 'desc';
  sortOptions: SortOption[];
  onSortChange: (sort: string) => void;
  onOrderChange: (order: 'asc' | 'desc') => void;

  availableTags: string[];
  draftTags: string[];
  isLoadingTags: boolean;
  tagsError: string | null;
  searchTag: string;
  onSearchTagChange: (q: string) => void;
  onToggleTag: (tag: string) => void;
  onClearTags: () => void;
  onSelectAllVisible: () => void;

  onResetAll: () => void;
  onApply: () => void;
  onCancel: () => void;
  selectedTagsCount: number;
};

export type MarketsFilterPageProps = {
  viewModel?: MarketsFilterPageViewModel;
};
