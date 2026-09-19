export type MarketsToolbarProps = {
  search: string;
  onSearchChange: (q: string) => void;
  isSearchDebouncing?: boolean;
  activeFilterCount: number;
  onOpenFilters: () => void;
  /** Open category browse page */
  onOpenCategories?: () => void;
  categoriesLabel?: string;
  categoriesA11y?: string;
  /** Open gainers/losers/volume leaderboards */
  onOpenLeaderboards?: () => void;
  leaderboardsLabel?: string;
  leaderboardsA11y?: string;
  /** Active single category chip */
  activeCategoryLabel?: string | null;
  onClearCategory?: () => void;
  clearCategoryA11y?: string;
  /** Show only favorited pairs (client-side) */
  favoritesOnly?: boolean;
  onToggleFavoritesOnly?: () => void;
  favoritesCount?: number;
};
