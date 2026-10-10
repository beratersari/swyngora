import type { CoinSituation as CoinSituationReport } from '@/libs/api';

export type CoinSituationProps = {
  report?: CoinSituationReport | null;
  isLoading?: boolean;
  error?: string | null;
};
