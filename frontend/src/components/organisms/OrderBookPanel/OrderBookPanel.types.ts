import type { OrderBookLevel, SpotOrderBook } from '@/libs/api';

export type OrderBookPanelProps = {
  book?: SpotOrderBook;
  group: string;
  onGroupChange: (group: string) => void;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
};

export type DepthRow = {
  level: OrderBookLevel;
  side: 'bid' | 'ask';
  depthPct: number;
};
