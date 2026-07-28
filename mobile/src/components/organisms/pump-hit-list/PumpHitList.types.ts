import type { ReactElement } from 'react';
import type { PumpHitRowViewModel } from '@/components/organisms/pump-hit-row';

export type PumpHitListProps = {
  rows: PumpHitRowViewModel[];
  isLoading: boolean;
  emptyMessage: string | null;
  errorMessage: string | null;
  onRetry: () => void;
  onPressRow: (exchange: string, symbol: string) => void;
  ListHeaderComponent?: ReactElement | null;
  refreshing?: boolean;
  onRefresh?: () => void;
};
