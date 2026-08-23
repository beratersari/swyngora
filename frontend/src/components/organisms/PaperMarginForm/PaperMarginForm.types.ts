import type { MarketExchange, MarginMode } from '@/libs/api';

export type PaperMarginFormValues = {
  exchange: MarketExchange;
  symbol: string;
  side: 'long' | 'short';
  orderType: 'market' | 'limit';
  quantity: number;
  leverage: number;
  limitPrice?: number;
  stopLoss?: number;
  takeProfit?: number;
};

export type PaperMarginFormProps = {
  marginMode?: MarginMode | string;
  modeLoading?: boolean;
  isSubmitting?: boolean;
  /** When true, open is blocked (e.g. 2+ books and none selected). */
  disabled?: boolean;
  submitError?: unknown;
  onModeChange?: (mode: MarginMode) => Promise<void>;
  onSubmit: (values: PaperMarginFormValues) => Promise<void>;
};
