import type { IndicatorsResponse } from '@/libs/api';
import type { WithLoadingProps } from '@/components/types';

export type IndicatorPanelProps = WithLoadingProps & {
  data?: IndicatorsResponse | null;
  /** Native quote of EMA/MA prices (venue quote). RSI is unitless and ignored. */
  priceQuote?: string;
  errorMessage?: string | null;
  showEmaOnChart?: boolean;
  onToggleEma?: (next: boolean) => void;
};
