import type { SituationLeave } from '@/libs/api';

export type PortfolioSituationLeaveProps = {
  record?: SituationLeave;
  loading?: boolean;
  failed?: boolean;
};
