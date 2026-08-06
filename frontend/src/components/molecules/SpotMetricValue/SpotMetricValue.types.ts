import type { SpotMarket } from '@/libs/api';
import type { SpotMetricDef } from '@/libs/utils';
import type { WithLoadingProps } from '@/components/types';

export type SpotMetricValueProps = WithLoadingProps & {
  metric: SpotMetricDef;
  spot?: SpotMarket | null;
  exchange?: string;
  locale?: string;
};
