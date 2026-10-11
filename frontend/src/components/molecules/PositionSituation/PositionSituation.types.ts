import type { CoinSituation } from '@/libs/api';

export type PositionSituationProps = {
  /** Shown above the two reads, for example a margin side and leverage. */
  lead?: string;
  entry?: CoinSituation | null;
  now?: CoinSituation | null;
  nowLoading?: boolean;
};
