import type { WithLoadingProps } from '@/components/types';
import type { MarketNetPosition } from '@/libs/api';

export type PositionSidePanelProps = WithLoadingProps & {
  data?: MarketNetPosition | null;
  error?: string | null;
};
