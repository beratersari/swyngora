export type MarketsToolbarProps = {
  search: string;
  onSearchChange: (q: string) => void;
  isSearchDebouncing?: boolean;
  activeFilterCount: number;
  onOpenFilters: () => void;
};
