import { useRef } from 'react';
import { useCompareOrderExecutionQuery, type CompareOrderExecutionArg } from '@/libs/api';
import { executionDraftReady, toExecutionArg } from './executionCompare';
import { ORDER_PREVIEW_DEBOUNCE_MS, ORDER_PREVIEW_POLL_MS, type OrderPreviewDraft } from './orderPreview';
import { useDebouncedValue } from './useDebouncedValue';

const idleArg: CompareOrderExecutionArg = {
  symbol: '_',
  exchange: 'binance',
  side: 'buy',
  quantity: 0,
};

export type UseExecutionCompareOptions = {
  portfolioId?: string;
  /** False while the page has no selected book. */
  enabled?: boolean;
  /** False while the tab is hidden. Polling pauses; the last comparison stays. */
  visible?: boolean;
};

/**
 * Debounced fee and slippage for one crypto size, three ways. Nothing is placed.
 * The page owns this hook. The ticket only reports the draft and renders the rows.
 */
export function useExecutionCompare(
  draft: OrderPreviewDraft | null,
  options: UseExecutionCompareOptions = {},
) {
  const enabled = options.enabled !== false;
  const visible = options.visible !== false;
  const debounced = useDebouncedValue(draft, ORDER_PREVIEW_DEBOUNCE_MS);
  const ready = enabled && executionDraftReady(debounced);
  const arg = ready ? toExecutionArg(debounced, options.portfolioId) : idleArg;
  const query = useCompareOrderExecutionQuery(arg, {
    skip: !ready,
    pollingInterval: ready && visible ? ORDER_PREVIEW_POLL_MS : 0,
    refetchOnFocus: true,
  });
  // currentData is only the result for the request in flight. data keeps the previous
  // success while that request runs, including a poll and a changed size.
  const current = ready ? query.currentData : undefined;
  const compare = current ?? (ready ? query.data : undefined);
  const updatedAtRef = useRef<number | null>(null);
  if (!ready) {
    updatedAtRef.current = null;
  } else if (current != null && query.fulfilledTimeStamp != null) {
    updatedAtRef.current = query.fulfilledTimeStamp;
  }
  const refreshing = ready && query.isFetching && compare != null;
  return {
    compare,
    /** No rows yet. A later read keeps the last rows and sets refreshing instead. */
    loading: ready && query.isFetching && compare == null,
    refreshing,
    updatedAt: compare != null ? updatedAtRef.current : null,
    failed: ready && query.isError && compare == null,
  };
}
