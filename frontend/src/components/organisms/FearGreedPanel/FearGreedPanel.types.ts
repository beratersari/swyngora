import type { WithLoadingProps } from '@/components/types';
import type { FearGreedReport } from '@/libs/api';
import type { TextColor } from '@/styles/tokens';

export type FearGreedPanelProps = WithLoadingProps & {
  data?: FearGreedReport | null;
  error?: string | null;
};

export type FearGreedTone = Extract<TextColor, 'success' | 'error' | 'warning' | 'secondary'>;
