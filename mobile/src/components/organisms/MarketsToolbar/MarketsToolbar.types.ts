export type MarketsToolbarProps = {
  search: string;
  onSearchChange: (q: string) => void;
  isSearchDebouncing?: boolean;
  activeFilterCount: number;
  onOpenFilters: () => void;
  /** Show only favorited pairs (client-side) */
  favoritesOnly?: boolean;
  onToggleFavoritesOnly?: () => void;
  favoritesCount?: number;
};
