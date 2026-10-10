import type { MarketDerivativesGrid } from '@/libs/api';
import type { GridMetric, GridRange } from './constants';

export type GridTone = 'up' | 'down' | 'heat' | 'flat' | 'blank';

export type DerivativesTimeGridProps = {
  exchange: string;
  metric: GridMetric;
  range: GridRange;
  grid?: MarketDerivativesGrid;
  loading?: boolean;
};
