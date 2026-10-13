import type { CoinSituation } from '@/libs/api';
import type { SituationComparison } from '@/components/molecules/CoinSituation';
import type { OpenLotReadView } from './helpers';

export type PositionSituationProps = {
  entry?: CoinSituation | null;
  now?: CoinSituation | null;
  nowLoading?: boolean;
  comparison?: SituationComparison | null;
  lots?: OpenLotReadView[] | null;
};
