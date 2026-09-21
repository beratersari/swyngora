import type { WithLoadingProps } from '@/components/types';
import type { MarketBreadth } from '@/libs/api';

export type BreadthOverviewProps = WithLoadingProps & {
  data?: MarketBreadth | null;
  error?: string | null;
};
