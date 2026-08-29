import type { ScannerSetup } from '@/libs/api';

export type SignalsSetupGridProps = {
  setups: ScannerSetup[];
  loading?: boolean;
  emptyText?: string;
  onOpen?: (exchange: string, symbol: string) => void;
};
