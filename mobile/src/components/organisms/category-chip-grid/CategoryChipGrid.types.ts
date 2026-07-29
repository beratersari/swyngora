export type CategoryChipGridProps = {
  /** Optional featured block title */
  featuredTitle?: string;
  featuredTags: string[];
  /** All (or search-filtered) tags */
  allTitle?: string;
  tags: string[];
  selectedTag?: string | null;
  onSelectTag: (tag: string) => void;
  isLoading?: boolean;
  emptyMessage?: string | null;
  formatLabel?: (tag: string) => string;
};
