import type { ReactNode } from 'react';
import type { WatchlistItem } from '@/libs/api';
import type { SpotMetricDef } from '@/libs/utils';

export type WatchlistMetricRenderArgs = {
  exchange: string;
  symbol: string;
  metric: SpotMetricDef;
};

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
  /**
   * Page-owned live metric cell (RTK / polling must stay outside organisms).
   * When omitted, metric cells show an em dash.
   */
  renderMetric?: (args: WatchlistMetricRenderArgs) => ReactNode;
};