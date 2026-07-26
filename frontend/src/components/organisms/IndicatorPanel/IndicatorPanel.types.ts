import type { IndicatorsResponse } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

export type IndicatorPanelProps = WithLoadingProps & {
  data?: IndicatorsResponse | null;
  errorMessage?: string | null;
  showEmaOnChart?: boolean;
  onToggleEma?: (next: boolean) => void;
};
