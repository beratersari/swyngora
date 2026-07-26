import type { Supply, Ticker24h } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

export type DetailStatsProps = WithLoadingProps & {
  exchange: string;
  ticker?: Ticker24h | null;
  supply?: Supply | null;
  tickerError?: string | null;
  supplyError?: string | null;
};
