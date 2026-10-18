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
  const compare = ready ? query.currentData : undefined;
  return {
    compare,
    loading: ready && query.isFetching && compare == null,
    failed: ready && query.isError && compare == null,
  };
}
