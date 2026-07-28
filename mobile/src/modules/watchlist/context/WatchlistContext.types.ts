import type { WatchlistPair } from '@/libs/utils';

export type WatchlistContextValue = {
  /** Hydration finished (local + first server attempt) */
  isReady: boolean;
  items: WatchlistPair[];
  count: number;
  error: string | null;
  /** Last mutation / max-items error for toast-style UI */
  actionError: string | null;
  clearActionError: () => void;
  isWatched: (exchange: string, symbol: string) => boolean;
  toggle: (exchange: string, symbol: string) => Promise<void>;
  refresh: () => Promise<void>;
};
