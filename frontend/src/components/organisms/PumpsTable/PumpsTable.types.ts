import type { PumpScanRow } from '@/libs/utils/pumpScan';

export type PumpsTableProps = {
  rows: PumpScanRow[];
  loading?: boolean;
  hasScanned: boolean;
  emptyHint: string;
  emptyTitle: string;
  columns: {
    symbol: string;
    returnPct: string;
    volumeRatio: string;
    time: string;
    events: string;
  };
  onRowOpen: (exchange: string, symbol: string) => void;
  locale?: string;
};
