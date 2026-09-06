import type { MarketOptions } from '@/libs/api';
import type { OPTIONS_HISTORY_METRICS, OPTIONS_HISTORY_RANGES } from './constants';

export type OptionsHistoryRange = (typeof OPTIONS_HISTORY_RANGES)[number];
export type OptionsHistoryMetric = (typeof OPTIONS_HISTORY_METRICS)[number];

export type OptionsHistoryProps = {
  data?: MarketOptions;
  isLoading?: boolean;
};
