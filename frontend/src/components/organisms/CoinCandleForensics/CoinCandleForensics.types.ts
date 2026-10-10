import type { ForensicsReport } from '@/libs/types/forensics.types';

export type CoinCandleForensicsProps = {
  /** UTC minute label of the clicked candle. Null means no candle is selected. */
  timeLabel: string | null;
  report?: ForensicsReport;
  isLoading?: boolean;
  error?: string | null;
  /** Route to the full forensics desk for this candle. */
  fullReadTo?: string | null;
  onClear: () => void;
  /** Open the forensics read for a similar past move. */
  onOpenMove: (at: string) => void;
};
