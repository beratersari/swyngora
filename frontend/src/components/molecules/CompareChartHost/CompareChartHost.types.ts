import type { ChartLinePoint } from '@/libs/utils';

export type CompareSeries = {
  id: string;
  title: string;
  color: string;
  data: ChartLinePoint[];
};

export type CompareChartHostProps = {
  series: CompareSeries[];
  height?: number;
  isLoading?: boolean;
  className?: string;
};
