import type { WithLoadingProps } from '@/components/types';

export type CoinInsightStat = { label: string; value: string };

export type CoinInsightPanelProps = WithLoadingProps & {
  title: string;
  summary?: string | null;
  stats?: CoinInsightStat[];
  error?: string | null;
};
