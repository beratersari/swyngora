import type { PortfolioEquityPoint, PortfolioPerformancePeriod } from '@/libs/api';

export type PortfolioEquityChartProps = {
  points?: PortfolioEquityPoint[];
  /** Used to seed a visible 2-point series when the API returns a single live mark. */
  startEquity?: number;
  startAt?: string;
  period: PortfolioPerformancePeriod;
  onPeriodChange: (p: PortfolioPerformancePeriod) => void;
  isLoading?: boolean;
  isError?: boolean;
  height?: number;
};
