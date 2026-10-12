import type { CoinSituation } from '@/libs/api';
import type { SituationComparison } from '@/components/molecules/CoinSituation';

export type PositionSituationProps = {
  entry?: CoinSituation | null;
  now?: CoinSituation | null;
  nowLoading?: boolean;
  comparison?: SituationComparison | null;
};
