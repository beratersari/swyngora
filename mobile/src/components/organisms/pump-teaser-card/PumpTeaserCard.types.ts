export type PumpTeaserItem = {
  id: string;
  exchange: string;
  symbol: string;
  returnLabel: string;
  returnTone: 'success' | 'error' | 'secondary';
  metaLabel: string;
};

export type PumpTeaserCardProps = {
  title: string;
  actionLabel?: string;
  onAction?: () => void;
  items: PumpTeaserItem[];
  isLoading?: boolean;
  errorMessage?: string | null;
  emptyMessage?: string | null;
  disclaimer?: string | null;
  onPressItem?: (exchange: string, symbol: string) => void;
  onRetry?: () => void;
  retryLabel?: string;
};
