import type { CoinSituation } from '@/libs/api';

export type PositionSituationProps = {
  entry?: CoinSituation | null;
  now?: CoinSituation | null;
  nowLoading?: boolean;
};
