import type { CoinSituation as CoinSituationReport } from '@/libs/api';

export type CoinSituationProps = {
  report?: CoinSituationReport | null;
  isLoading?: boolean;
  error?: string | null;
  /** Opens the alerts form with the definite clauses on this read already selected. */
  alertHref?: string | null;
};
