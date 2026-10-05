import type { DerivativesCoin } from '@/libs/api';
import type { HeatMetric } from './constants';

export type DerivativesHeatmapProps = {
  exchange: string;
  coins: DerivativesCoin[];
  metric: HeatMetric;
  loading?: boolean;
};

export type HeatTone = 'up' | 'down' | 'heat' | 'flat' | 'blank';

export type HeatCell = {
  symbol: string;
  label: string;
  text: string;
  tone: HeatTone;
  /** 0 to 1. Blank and neutral cells are 0. */
  strength: number;
  path: string;
};
