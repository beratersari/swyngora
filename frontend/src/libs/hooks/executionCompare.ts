import type { CompareOrderExecutionArg } from '@/libs/api';
import type { OrderPreviewDraft, SpotOrderPreviewDraft } from './orderPreview';

const ROUTE_VENUES = new Set(['binance', 'coinbase', 'bybit']);

/** The single venue a comparison prices, even when the ticket is on best cost. */
export function executionVenue(draft: SpotOrderPreviewDraft): string {
  const raw = draft.exchange.trim().toLowerCase();
  if (raw === 'all' || raw === 'best') return (draft.compareExchange ?? '').trim().toLowerCase();
  return raw;
}

/** True when a crypto market draft has a size the compare endpoint can price. */
export function executionDraftReady(
  draft: OrderPreviewDraft | null | undefined,
): draft is SpotOrderPreviewDraft {
  if (!draft || draft.product !== 'spot' || draft.type !== 'market') return false;
  if (!draft.symbol.trim()) return false;
  if (draft.side !== 'buy' && draft.side !== 'sell') return false;
  if (!Number.isFinite(draft.quantity) || draft.quantity < 1e-8) return false;
  return ROUTE_VENUES.has(executionVenue(draft));
}

/** Map a ready draft onto the compare request. */
export function toExecutionArg(
  draft: SpotOrderPreviewDraft,
  portfolioId?: string,
): CompareOrderExecutionArg {
  const book = portfolioId?.trim() || undefined;
  return {
    portfolioId: book,
    exchange: executionVenue(draft),
    symbol: draft.symbol.trim().toUpperCase(),
    side: draft.side,
    quantity: draft.quantity,
    maxSlippagePct: draft.maxSlippagePct,
    spreadMinutes: draft.spreadMinutes,
    sliceCount: draft.sliceCount,
  };
}
