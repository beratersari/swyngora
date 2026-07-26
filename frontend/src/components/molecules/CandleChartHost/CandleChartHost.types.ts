import type { ChartCandle } from '@/libs/utils';
import type { WithLoadingProps } from '@/components/types';

export type CandleChartHostProps = WithLoadingProps & {
  data: ChartCandle[];
  height?: number;
  className?: string;
};
