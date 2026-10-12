import type { SituationRecord } from '@/libs/api';

export type PortfolioSituationRecordProps = {
  record?: SituationRecord;
  loading?: boolean;
  failed?: boolean;
};
