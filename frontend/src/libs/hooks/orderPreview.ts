import type { PreviewPortfolioOrderArg } from '@/libs/api';

/** Wait before pricing a ticket so each keystroke does not call the API. */
export const ORDER_PREVIEW_DEBOUNCE_MS = 300;

/** Refresh an open preview while the tab is visible. */
export const ORDER_PREVIEW_POLL_MS = 15_000;

export type SpotOrderPreviewDraft = {
  product: 'spot';
  exchange: string;
  symbol: string;
  type: string;
  side: 'buy' | 'sell';
  quantity: number;
  triggerPrice?: number;
  takeProfitPrice?: number;
  stopLossPrice?: number;
  trailType?: 'percent' | 'offset';
  trailValue?: number;
  lotMethod?: 'fifo' | 'lifo';
};

export type MarginOrderPreviewDraft = {
  product: 'margin';
  exchange: string;
  symbol: string;
  type: 'market' | 'limit';
  side: 'long' | 'short';
  quantity: number;
  leverage: number;
  limitPrice?: number;
  stopLossPrice?: number;
  takeProfitPrice?: number;
};

export type OrderPreviewDraft = SpotOrderPreviewDraft | MarginOrderPreviewDraft;

function positive(value: number | undefined): boolean {
  return value != null && Number.isFinite(value) && value > 0;
}

/** True when the draft has every field the preview endpoint needs. */
export function previewDraftReady(draft: OrderPreviewDraft | null | undefined): draft is OrderPreviewDraft {
  if (!draft || !draft.symbol.trim()) return false;
  if (!Number.isFinite(draft.quantity) || draft.quantity < 1e-8) return false;
  if (draft.product === 'margin') {
    if (draft.side !== 'long' && draft.side !== 'short') return false;
    if (!Number.isInteger(draft.leverage) || draft.leverage < 1 || draft.leverage > 10) return false;
    if (draft.type !== 'market' && draft.type !== 'limit') return false;
    if (draft.type === 'limit' && !positive(draft.limitPrice)) return false;
    return true;
  }
  switch (draft.type) {
    case 'market':
      return draft.side === 'buy' || draft.side === 'sell';
    case 'limit_buy':
    case 'limit_sell':
    case 'stop_loss':
      return positive(draft.triggerPrice);
    case 'trailing_stop':
      return (draft.trailType === 'percent' || draft.trailType === 'offset') && positive(draft.trailValue);
    case 'oco':
      return positive(draft.takeProfitPrice) && positive(draft.stopLossPrice);
    case 'bracket':
      return positive(draft.triggerPrice) && positive(draft.takeProfitPrice) && positive(draft.stopLossPrice);
    default:
      return false;
  }
}

/** Map a complete draft onto the preview request. Optional prices stay off the wire when unset. */
export function toPreviewArg(draft: OrderPreviewDraft, portfolioId?: string): PreviewPortfolioOrderArg {
  const book = portfolioId?.trim() || undefined;
  const symbol = draft.symbol.trim().toUpperCase();
  if (draft.product === 'margin') {
    return {
      portfolioId: book,
      exchange: draft.exchange,
      symbol,
      product: 'margin',
      type: draft.type,
      side: draft.side,
      quantity: draft.quantity,
      leverage: draft.leverage,
      limitPrice: positive(draft.limitPrice) ? draft.limitPrice : undefined,
      stopLossPrice: positive(draft.stopLossPrice) ? draft.stopLossPrice : undefined,
      takeProfitPrice: positive(draft.takeProfitPrice) ? draft.takeProfitPrice : undefined,
    };
  }
  return {
    portfolioId: book,
    exchange: draft.exchange,
    symbol,
    product: 'spot',
    type: draft.type,
    side: draft.side,
    quantity: draft.quantity,
    triggerPrice: positive(draft.triggerPrice) ? draft.triggerPrice : undefined,
    takeProfitPrice: positive(draft.takeProfitPrice) ? draft.takeProfitPrice : undefined,
    stopLossPrice: positive(draft.stopLossPrice) ? draft.stopLossPrice : undefined,
    trailType: draft.trailType,
    trailValue: positive(draft.trailValue) ? draft.trailValue : undefined,
    lotMethod: draft.lotMethod,
  };
}
