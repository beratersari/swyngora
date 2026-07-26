import type { ChartLinePoint } from '@/libs/utils';
import type { WithLoadingProps } from '@/components/types';

export type IndicatorChartHostProps = WithLoadingProps & {
  /** RSI (or similar 0–100) series */
  data: ChartLinePoint[];
  height?: number;
  className?: string;
  /** Horizontal guide lines (default 30 / 70) */
  bands?: number[];
};
