export type CategorySectionProps = {
  title: string;
  actionLabel?: string;
  onAction?: () => void;
  tags: string[];
  onSelectTag: (tag: string) => void;
  isLoading?: boolean;
  errorMessage?: string | null;
  emptyMessage?: string | null;
  onRetry?: () => void;
  retryLabel?: string;
  formatLabel?: (tag: string) => string;
};
