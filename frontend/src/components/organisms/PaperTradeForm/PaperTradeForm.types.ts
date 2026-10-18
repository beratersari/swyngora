import type { ExecutionCompare, MarketExchange, OrderPreview, PaperOrderType } from '@/libs/api';
import type { SpotOrderPreviewDraft } from '@/libs/hooks/orderPreview';

export type PaperTradeFormValues = {
  exchange: MarketExchange | 'all';
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
  /** Set on a buy that names its clauses. An empty list stores no read. */
  reason?: string[];
  /** Best-cost cap in percent. 0.5 means 0.5% from the best price. */
  maxSlippagePct?: number;
  /** Work the best-cost order as timed slices. */
  spread?: boolean;
  /** Minutes from the first slice to the last. */
  spreadMinutes?: number;
  /** How many slices. */
  sliceCount?: number;
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
  /** When true, submit is blocked (e.g. 2+ books and none selected). */
  disabled?: boolean;
  submitError?: unknown;
  compact?: boolean;
  /** Latest cost and risk for this ticket. The page owns the request. */
  preview?: OrderPreview | null;
  previewLoading?: boolean;
  previewFailed?: boolean;
  /** Fee and slippage for this venue, best cost, and a time split. The page owns the request. */
  execution?: ExecutionCompare | null;
  executionLoading?: boolean;
  executionFailed?: boolean;
  /** Fires when the draft changes so the page can price it. */
  onDraftChange?: (draft: SpotOrderPreviewDraft) => void;
  onSubmit: (values: PaperTradeFormValues) => Promise<void>;
};
