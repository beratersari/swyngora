import type { SwingDecision } from '@/libs/api';

export type SwingEngineGridProps = {
  items: SwingDecision[];
  loading?: boolean;
  emptyText?: string;
  onOpen?: (exchange: string, symbol: string) => void;
};
