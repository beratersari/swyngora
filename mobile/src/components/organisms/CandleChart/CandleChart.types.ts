import type { ChartCandle, ChartLinePoint } from '@/libs/utils';

export type CandleChartOverlay = {
  id: string;
  title: string;
  color: string;
  data: ChartLinePoint[];
};

export type CandleChartProps = {
  candles: ChartCandle[];
  overlays?: CandleChartOverlay[];
  height?: number;
  isLoading?: boolean;
  errorMessage?: string | null;
  emptyMessage?: string;
};
