export type CategoriesPageViewModel = {
  title: string;
  search: string;
  onSearchChange: (q: string) => void;
  searchPlaceholder: string;
  featuredTitle: string;
  featuredTags: string[];
  allTitle: string;
  tags: string[];
  selectedTag: string | null;
  isLoading: boolean;
  isSearchDebouncing: boolean;
  errorMessage: string | null;
  emptyMessage: string | null;
  formatLabel: (tag: string) => string;
  onSelectTag: (tag: string) => void;
  onRetry: () => void;
  onBack: () => void;
  retryLabel: string;
  backLabel: string;
};

export type CategoriesPageProps = {
  viewModel?: CategoriesPageViewModel;
};
