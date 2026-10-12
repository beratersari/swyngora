import type { CoinSituation } from '@/libs/api';

export type CoinCandleSituationProps = {
  timeLabel: string | null;
  report?: CoinSituation | null;
  isLoading?: boolean;
  error?: string | null;
};
