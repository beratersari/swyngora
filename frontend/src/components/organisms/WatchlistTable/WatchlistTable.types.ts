import type { WatchlistItem } from '@/libs/api';
import type { SpotMetricDef } from '@/libs/utils';

export type WatchlistTableProps = {
  items: WatchlistItem[];
  loading?: boolean;
  onRemove: (exchange: string, symbol: string) => void;
  removeLoading?: boolean;
  onOpen: (exchange: string, symbol: string) => void;
  /**
   * Ordered metric columns (shared catalog / column picker).
   * When omitted, uses watchlist defaults.
   */
  metrics?: SpotMetricDef[];
};
