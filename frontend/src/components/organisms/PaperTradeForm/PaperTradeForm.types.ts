import type { MarketExchange } from '@/libs/api';

export type PaperTradeFormValues = {
  exchange: MarketExchange;
  symbol: string;
  side: 'buy' | 'sell';
  quantity: number;
  lotMethod?: 'fifo' | 'lifo';
};

export type PaperTradeFormProps = {
  /** When set, exchange/symbol fields are fixed (detail page). */
  lockedExchange?: MarketExchange;
  lockedSymbol?: string;
  defaultExchange?: MarketExchange;
  defaultSymbol?: string;
  defaultSide?: 'buy' | 'sell';
  showLotMethod?: boolean;
  isSubmitting?: boolean;
  submitError?: unknown;
  compact?: boolean;
  onSubmit: (values: PaperTradeFormValues) => Promise<void>;
};
