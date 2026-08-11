import type { ScannerResult } from '@/libs/api';

export type SignalsHitsTableProps = {
  items: ScannerResult[];
  loading?: boolean;
  onOpen?: (exchange: string, symbol: string) => void;
};
