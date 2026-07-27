import type { ChartLinePoint } from '@/libs/utils';

export type IndicatorRsiPaneProps = {
  data: ChartLinePoint[];
  latestRsi: number | null;
  isLoading?: boolean;
  errorMessage?: string | null;
  height?: number;
};
