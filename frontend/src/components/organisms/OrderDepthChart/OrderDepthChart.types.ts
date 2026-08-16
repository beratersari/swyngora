import type { SpotOrderBook } from '@/libs/api';

export type DepthMetric = 'base' | 'notional';

export type OrderDepthChartProps = {
  book?: SpotOrderBook;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string | null;
};

export type DepthPoint = {
  price: number;
  depth: number;
};

export type DepthSeries = {
  mid: number;
  bids: DepthPoint[];
  asks: DepthPoint[];
  maxDepth: number;
  minPrice: number;
  maxPrice: number;
};

export type DepthLayout = {
  plotX: number;
  plotY: number;
  plotW: number;
  plotH: number;
  minPrice: number;
  maxPrice: number;
  maxDepth: number;
};

export type DepthHover = {
  x: number;
  y: number;
  side: 'bid' | 'ask';
  price: number;
  depth: number;
};
