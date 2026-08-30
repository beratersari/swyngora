import type { MarketLiquidationLevels } from '@/libs/api';

export type LiquidationBarChartProps = {
  data?: MarketLiquidationLevels | null;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
};

export type LevelRow = {
  price: number;
  longN: number;
  shortN: number;
  totalN: number;
};

export type TimeRow = {
  t: number;
  label: string;
  longN: number;
  shortN: number;
  totalN: number;
};

export type BarHover = {
  x: number;
  y: number;
  title: string;
  longN: number;
  shortN: number;
  totalN: number;
};
