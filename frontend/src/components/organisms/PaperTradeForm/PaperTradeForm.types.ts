import type { MarketExchange, PaperOrderType } from '@/libs/api';

export type PaperTradeFormValues = {
  exchange: MarketExchange;
  symbol: string;
  /** API order type */
  orderType: PaperOrderType;
  /** Required for market; ignored for limit_buy/limit_sell (derived from type) */
  side: 'buy' | 'sell';
  quantity: number;
  triggerPrice?: number;
  takeProfitPrice?: number;
  stopLossPrice?: number;
  trailType?: 'percent' | 'offset';
  trailValue?: number;
  lotMethod?: 'fifo' | 'lifo';
  timeInForce?: 'gtc' | 'ioc' | 'fok';
};

export type PaperTradeFormProps = {
  /** When set, exchange/symbol fields are fixed (detail page). */
  lockedExchange?: MarketExchange;
  lockedSymbol?: string;
  defaultExchange?: MarketExchange;
  defaultSymbol?: string;
  defaultSide?: 'buy' | 'sell';
  showLotMethod?: boolean;
  /** Hide advanced types (OCO/bracket/trailing); still allow market + limit + stop. */
  advanced?: boolean;
  isSubmitting?: boolean;
  submitError?: unknown;
  compact?: boolean;
  onSubmit: (values: PaperTradeFormValues) => Promise<void>;
};
