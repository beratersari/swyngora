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
  /** Percent of portfolio equity to lose if the stop fills. Sizes the buy. */
  riskPercent?: number;
  /** Clauses this buy stores. Omit to leave the request unset. An empty list stores no read. */
  reason?: string[];
  /** Set when exchange is all. The venue the whole order is compared with. */
  compareExchange?: string;
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
  /** Percent of portfolio equity to lose if the stop fills. Sizes the open. */
  riskPercent?: number;
};

export type OrderPreviewDraft = SpotOrderPreviewDraft | MarginOrderPreviewDraft;

function positive(value: number | undefined): boolean {
  return value != null && Number.isFinite(value) && value > 0;
}

/** True when the draft asks the server to size the order from a stop. */
export function riskSizingActive(riskPercent: number | null | undefined): riskPercent is number {
  return (
    riskPercent != null && Number.isFinite(riskPercent) && riskPercent > 0 && riskPercent <= 100
  );
}

/** Quantity from a preview whose risk percent still matches the field the user typed. */
export function sizedQuantity(
  riskPercent: number | null | undefined,
  sizing: { riskPercent?: number; quantity?: number } | null | undefined,
): number | null {
  if (!riskSizingActive(riskPercent) || sizing?.quantity == null) return null;
  if (!Number.isFinite(sizing.quantity) || sizing.quantity <= 0) return null;
  if (sizing.riskPercent == null || Math.abs(sizing.riskPercent - riskPercent) > 1e-6) return null;
  return sizing.quantity;
}

/** True when the draft has every field the preview endpoint needs. */
export function previewDraftReady(
  draft: OrderPreviewDraft | null | undefined,
): draft is OrderPreviewDraft {
  if (!draft || !draft.symbol.trim()) return false;
  const sizing = riskSizingActive(draft.riskPercent);
  if (sizing && !positive(draft.stopLossPrice)) return false;
  if (!sizing && (!Number.isFinite(draft.quantity) || draft.quantity < 1e-8)) return false;
  if (draft.product === 'margin') {
    if (draft.side !== 'long' && draft.side !== 'short') return false;
    if (!Number.isInteger(draft.leverage) || draft.leverage < 1 || draft.leverage > 10)
      return false;
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
      return (
        (draft.trailType === 'percent' || draft.trailType === 'offset') &&
        positive(draft.trailValue)
      );
    case 'oco':
      return positive(draft.takeProfitPrice) && positive(draft.stopLossPrice);
    case 'bracket':
      return (
        positive(draft.triggerPrice) &&
        positive(draft.takeProfitPrice) &&
        positive(draft.stopLossPrice)
      );
    default:
      return false;
  }
}

function sizingQuantity(draft: OrderPreviewDraft): number | undefined {
  if (riskSizingActive(draft.riskPercent)) return undefined;
  return draft.quantity;
}

/** Map a complete draft onto the preview request. Optional prices stay off the wire when unset. */
export function toPreviewArg(
  draft: OrderPreviewDraft,
  portfolioId?: string,
): PreviewPortfolioOrderArg {
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
      quantity: sizingQuantity(draft),
      riskPercent: riskSizingActive(draft.riskPercent) ? draft.riskPercent : undefined,
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
    quantity: sizingQuantity(draft),
    riskPercent: riskSizingActive(draft.riskPercent) ? draft.riskPercent : undefined,
    triggerPrice: positive(draft.triggerPrice) ? draft.triggerPrice : undefined,
    takeProfitPrice: positive(draft.takeProfitPrice) ? draft.takeProfitPrice : undefined,
    stopLossPrice: positive(draft.stopLossPrice) ? draft.stopLossPrice : undefined,
    trailType: draft.trailType,
    trailValue: positive(draft.trailValue) ? draft.trailValue : undefined,
    lotMethod: draft.lotMethod,
    reason: draft.reason,
    compareExchange: draft.compareExchange,
  };
}
