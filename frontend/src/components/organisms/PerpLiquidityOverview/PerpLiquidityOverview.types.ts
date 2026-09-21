import type { WithLoadingProps } from '@/components/types';
import type { MarketPerpDepthOverview } from '@/libs/api';

export type PerpLiquidityOverviewProps = WithLoadingProps & {
  data?: MarketPerpDepthOverview | null;
  error?: string | null;
};
