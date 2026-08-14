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
  submitError?: unknown;
  onModeChange?: (mode: MarginMode) => Promise<void>;
  onSubmit: (values: PaperMarginFormValues) => Promise<void>;
};
