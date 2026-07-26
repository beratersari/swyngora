import type { ChartCandle, ChartLinePoint } from '@/libs/utils';
import type { WithLoadingProps } from '@/components/types';

export type CandleChartOverlay = {
  id: string;
  data: ChartLinePoint[];
  color: string;
  title?: string;
};

export type CandleChartHostProps = WithLoadingProps & {
  data: ChartCandle[];
  /** Optional EMA (or other) line overlays on the price scale */
  overlays?: CandleChartOverlay[];
  height?: number;
  className?: string;
};
