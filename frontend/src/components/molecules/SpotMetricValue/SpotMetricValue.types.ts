import type { SpotMarket } from '@/libs/api';
import type { SpotMetricDef } from '@/libs/utils';

export type SpotMetricValueProps = {
  metric: SpotMetricDef;
  spot: SpotMarket | undefined | null;
  exchange?: string;
  isLoading?: boolean;
};
