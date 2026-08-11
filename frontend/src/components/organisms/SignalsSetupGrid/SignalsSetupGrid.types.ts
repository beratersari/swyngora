import type { SwingSetup } from '@/libs/utils';

export type SignalsSetupGridProps = {
  setups: SwingSetup[];
  loading?: boolean;
  emptyText?: string;
  onOpen?: (exchange: string, symbol: string) => void;
};
