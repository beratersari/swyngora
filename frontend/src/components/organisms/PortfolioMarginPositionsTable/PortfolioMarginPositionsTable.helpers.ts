import type { MarginPosition } from '@/libs/api';
import type { MarginBracketUpdate } from './PortfolioMarginPositionsTable.types';

export type MarginBracketIssue = 'stop' | 'takeProfit' | 'unchanged';

export type MarginBracketDraft = {
  side: MarginPosition['side'];
  entryPrice: number | undefined;
  openedStop: number | null | undefined;
  openedTakeProfit: number | null | undefined;
  stopLoss: number | null;
  takeProfit: number | null;
};

const PRICE_EPS = 1e-8;

function finitePositive(value: number | null | undefined): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0;
}

function samePrice(a: number, b: number): boolean {
  const scale = Math.max(1, Math.abs(a), Math.abs(b));
  return Math.abs(a - b) <= PRICE_EPS * scale;
}

function onCorrectSide(
  kind: 'stop' | 'takeProfit',
  side: MarginPosition['side'],
  entry: number | undefined,
  price: number,
): boolean {
  if ((side !== 'long' && side !== 'short') || !finitePositive(entry)) return false;
  if (kind === 'stop') return side === 'long' ? price < entry : price > entry;
  return side === 'long' ? price > entry : price < entry;
}

type FieldResult =
  { kind: 'keep' } | { kind: 'clear' } | { kind: 'set'; price: number } | { kind: 'bad' };

function fieldResult(
  kind: 'stop' | 'takeProfit',
  draft: MarginBracketDraft,
  opened: number | null | undefined,
  next: number | null,
): FieldResult {
  const had = finitePositive(opened) ? opened : null;
  if (next == null) return had == null ? { kind: 'keep' } : { kind: 'clear' };
  if (!finitePositive(next)) return { kind: 'bad' };
  if (had != null && samePrice(had, next)) return { kind: 'keep' };
  if (!onCorrectSide(kind, draft.side, draft.entryPrice, next)) return { kind: 'bad' };
  return { kind: 'set', price: next };
}

/**
 * Diff an open position's stop and take profit against the edit draft.
 * A cleared price removes that order. An unchanged price is left off the request.
 */
export function marginBracketChange(
  draft: MarginBracketDraft,
): { change: Omit<MarginBracketUpdate, 'id'> } | { issue: MarginBracketIssue } {
  const stop = fieldResult('stop', draft, draft.openedStop, draft.stopLoss);
  const take = fieldResult('takeProfit', draft, draft.openedTakeProfit, draft.takeProfit);
  if (stop.kind === 'bad') return { issue: 'stop' };
  if (take.kind === 'bad') return { issue: 'takeProfit' };
  const change: Omit<MarginBracketUpdate, 'id'> = {};
  if (stop.kind === 'clear') change.clearStopLoss = true;
  if (stop.kind === 'set') change.stopLoss = stop.price;
  if (take.kind === 'clear') change.clearTakeProfit = true;
  if (take.kind === 'set') change.takeProfit = take.price;
  if (
    change.stopLoss == null &&
    change.takeProfit == null &&
    !change.clearStopLoss &&
    !change.clearTakeProfit
  ) {
    return { issue: 'unchanged' };
  }
  return { change };
}

/** Smallest paper close. Matches the backend minimum trade size. */
export const MIN_MARGIN_CLOSE_QUANTITY = 1e-8;

export type MarginClosePlan =
  { full: true } | { full: false; quantity: number; remain: number } | { issue: 'quantity' };

function sameSize(a: number, b: number): boolean {
  const scale = Math.max(1, Math.abs(a), Math.abs(b));
  return Math.abs(a - b) <= 1e-12 * scale;
}

/**
 * Size to send on a margin close. The full open size omits quantity.
 * A smaller size is a partial close. A remainder below the minimum is rejected.
 */
export function marginCloseQuantity(
  openQty: number | undefined,
  requested: number | null,
): MarginClosePlan {
  if (!finitePositive(openQty) || !finitePositive(requested)) return { issue: 'quantity' };
  if (requested < MIN_MARGIN_CLOSE_QUANTITY) return { issue: 'quantity' };
  if (requested > openQty && !sameSize(requested, openQty)) return { issue: 'quantity' };
  if (requested >= openQty || sameSize(requested, openQty)) return { full: true };
  const remain = openQty - requested;
  if (remain < MIN_MARGIN_CLOSE_QUANTITY) return { issue: 'quantity' };
  return { full: false, quantity: requested, remain };
}
